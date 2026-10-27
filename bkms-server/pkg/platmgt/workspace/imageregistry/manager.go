/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

// Package imageregistry provides platform administrators with workspace image registry management.
package imageregistry

import (
	"context"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

const (
	// probeTimeout 可用性探测的总时间预算，保存是同步接口
	probeTimeout = 8 * time.Second
	// probeImageName 探测推送权限时拼在命名空间后的镜像名；上传会立即取消，不会真的创建该镜像
	probeImageName = "bkms-registry-probe"
)

var (
	// ErrInvalidRegistryAddress 仓库地址需为 host/namespace，且不带协议前缀
	ErrInvalidRegistryAddress = errors.New("invalid image registry address")
	// ErrPasswordRequired 外部仓库没有可沿用的密码时必须填写密码
	ErrPasswordRequired = errors.New("image registry password is required")
	// ErrRegistryAuthFailed 目标仓库鉴权失败或没有推送权限（401 / 403）
	ErrRegistryAuthFailed = errors.New("image registry authentication failed")
	// ErrRegistryNamespaceInaccessible 目标仓库的命名空间不存在或无权访问（404）
	ErrRegistryNamespaceInaccessible = errors.New("image registry namespace inaccessible")
	// ErrRegistryUnreachable 目标仓库网络不可达或探测超时
	ErrRegistryUnreachable = errors.New("image registry unreachable")
	// ErrBkCICredentialCreateFailed 在空间对应的蓝盾项目下创建镜像凭证失败
	ErrBkCICredentialCreateFailed = errors.New("create bkci image credential failed")
	// ErrBuiltinRegistryNotFound 空间没有平台镜像仓库记录，无法切回平台仓库
	ErrBuiltinRegistryNotFound = errors.New("builtin image registry not found")
)

// UpdateParams 设置空间镜像仓库的参数
type UpdateParams struct {
	// Type 目标仓库类型；builtin 表示切回平台仓库，此时其余字段不生效
	Type bkmsreg.ImageRegistryType
	// Registry 外部仓库地址，形如 host/namespace
	Registry string
	// Username 外部仓库账号
	Username string
	// Password 外部仓库密码，为空表示沿用已有外部仓库记录的密码
	Password string
}

// Manager 平台管理员视角下的空间镜像仓库查看、切换与修改
type Manager struct {
	workspaces *bkmsworkspace.WorkspaceStore
	registries *bkmsreg.ImageRegistryStore
}

// NewManager 创建 Manager
func NewManager(db *mongo.Database) *Manager {
	return &Manager{
		workspaces: store.Of(db, bkmsworkspace.Meta),
		registries: store.Of(db, bkmsreg.ImageRegistryMeta),
	}
}

// Get 返回空间当前生效的镜像仓库；记录缺失时只填类型
func (m *Manager) Get(ctx context.Context, workspaceID string) (*bkmsreg.ImageRegistry, error) {
	ws, err := m.workspaces.Get(ctx, bkmsworkspace.ByWorkspaceID(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s", workspaceID)
	}

	reg, err := m.getRegistry(ctx, ws.ID, ws.ImageRegistryType)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return &bkmsreg.ImageRegistry{WorkspaceID: ws.ID, Type: ws.ImageRegistryType}, nil
	}
	return reg, nil
}

// Update 把空间镜像仓库设置为平台仓库或指定的外部仓库，配置有变化时写一条不含密码的审计
//
// 平台仓库与外部仓库的记录切换后都保留，生效的是哪个由空间的 ImageRegistryType 决定
func (m *Manager) Update(
	ctx context.Context, workspaceID string, params UpdateParams,
) (*bkmsreg.ImageRegistry, error) {
	ws, err := m.workspaces.Get(ctx, bkmsworkspace.ByWorkspaceID(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s", workspaceID)
	}

	before, err := m.getRegistry(ctx, ws.ID, ws.ImageRegistryType)
	if err != nil {
		return nil, err
	}

	var after *bkmsreg.ImageRegistry
	if params.Type == bkmsreg.ImageRegistryTypeBuiltin {
		after, err = m.switchToBuiltin(ctx, ws)
	} else {
		after, err = m.applyExternal(ctx, ws, params)
	}
	if err != nil {
		return nil, err
	}

	// 生效类型与仓库记录都没变（如平台仓库再切平台仓库、提交相同的外部配置）时不写审计；
	// 只改密码时凭证 ID 会变，记录不相等，仍会记一条
	if ws.ImageRegistryType == after.Type && before != nil && *before == *after {
		return after, nil
	}

	go audit.AddOperationRecordAsync(
		ctx, audit.OperationTypeUpdate, audit.ResourceTypeWorkspace, ws.ID,
		audit.WithAttribute(audit.AttributeImageRegistry),
		audit.WithDataBefore(m.buildAuditData(ws.ImageRegistryType, before)),
		audit.WithDataAfter(m.buildAuditData(after.Type, after)),
		audit.WithWorkspaceID(ws.ID),
	)
	return after, nil
}

// switchToBuiltin 切回平台仓库：平台仓库记录和凭证一直保留，只需改生效类型，无需探测和新建凭证
func (m *Manager) switchToBuiltin(ctx context.Context, ws *bkmsworkspace.Workspace) (*bkmsreg.ImageRegistry, error) {
	builtin, err := m.getRegistry(ctx, ws.ID, bkmsreg.ImageRegistryTypeBuiltin)
	if err != nil {
		return nil, err
	}
	if builtin == nil {
		return nil, errors.Wrapf(ErrBuiltinRegistryNotFound, "workspace %s", ws.ID)
	}

	if ws.ImageRegistryType != bkmsreg.ImageRegistryTypeBuiltin {
		if err = m.setRegistryType(ctx, ws.ID, bkmsreg.ImageRegistryTypeBuiltin); err != nil {
			return nil, err
		}
	}
	return builtin, nil
}

// applyExternal 切换到或修改外部仓库：探测 → 账密变化时创建蓝盾凭证 → 写外部仓库记录 → 切换时更新生效类型
//
// 任一步失败都清理本次新建的凭证并恢复外部仓库记录，空间原配置保持生效
func (m *Manager) applyExternal(
	ctx context.Context, ws *bkmsworkspace.Workspace, params UpdateParams,
) (*bkmsreg.ImageRegistry, error) {
	params.Registry = strings.TrimRight(strings.TrimSpace(params.Registry), "/")
	params.Username = strings.TrimSpace(params.Username)
	if err := m.validateRegistry(params.Registry); err != nil {
		return nil, err
	}

	// 外部仓库记录可能是之前切回平台仓库时留下的，此时覆盖它而不是新建
	existing, err := m.getRegistry(ctx, ws.ID, bkmsreg.ImageRegistryTypeExternal)
	if err != nil {
		return nil, err
	}

	password := params.Password
	if existing != nil {
		password = lo.CoalesceOrEmpty(params.Password, existing.Password)
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}

	if err = m.probe(ctx, params.Registry, params.Username, password); err != nil {
		return nil, err
	}

	// 蓝盾凭证只保存账密，账密不变时复用原凭证；旧凭证不删除，进行中的构建仍在使用它
	credID, newCredID := "", ""
	if existing != nil {
		credID = existing.BkCICredentialID
	}
	if existing == nil || params.Username != existing.Username || password != existing.Password {
		if newCredID, err = m.createCredential(ctx, ws.BkSystems.BkCIProjectID, params, password); err != nil {
			return nil, err
		}
		credID = newCredID
	}

	after := &bkmsreg.ImageRegistry{
		WorkspaceID:      ws.ID,
		Type:             bkmsreg.ImageRegistryTypeExternal,
		Registry:         params.Registry,
		Username:         params.Username,
		Password:         password,
		BkCICredentialID: credID,
	}
	if err = m.saveExternal(ctx, existing, after); err != nil {
		m.deleteCredential(ctx, ws.BkSystems.BkCIProjectID, newCredID)
		return nil, err
	}

	if ws.ImageRegistryType != bkmsreg.ImageRegistryTypeExternal {
		if err = m.setRegistryType(ctx, ws.ID, bkmsreg.ImageRegistryTypeExternal); err != nil {
			m.restoreExternal(ctx, ws.ID, existing)
			m.deleteCredential(ctx, ws.BkSystems.BkCIProjectID, newCredID)
			return nil, err
		}
	}
	return after, nil
}

// validateRegistry 地址需能被解析成「指定仓库主机 + 命名空间」，不能落到 Docker Hub
func (m *Manager) validateRegistry(registry string) error {
	host, namespace, found := strings.Cut(registry, "/")
	if strings.Contains(registry, "://") || !found || host == "" || namespace == "" {
		return errors.Wrapf(ErrInvalidRegistryAddress, "registry %q", registry)
	}

	// 与探测使用同一套解析规则。短主机名（如 harbor/ns）会被当成仓库路径，实际访问 index.docker.io
	repo, err := name.NewRepository(registry + "/" + probeImageName)
	if err != nil {
		return errors.Wrapf(ErrInvalidRegistryAddress, "registry %q: %s", registry, err.Error())
	}
	if repo.RegistryStr() == name.DefaultRegistry {
		return errors.Wrapf(ErrInvalidRegistryAddress, "registry %q resolves to %s", registry, repo.RegistryStr())
	}
	return nil
}

// probe 在时间预算内探测目标仓库，并把错误归为账密错误、命名空间不可访问、仓库不可达三类
func (m *Manager) probe(ctx context.Context, registry, username, password string) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	// insecure 只跳过证书校验，允许自签证书；协议由 CheckPushPermission 决定，公网主机走 HTTPS
	err := infrasreg.New(username, password, true).CheckPushPermission(ctx, registry+"/"+probeImageName)
	switch {
	case err == nil:
		return nil
	case infrasreg.IsAuthRequired(err):
		return errors.Wrap(ErrRegistryAuthFailed, err.Error())
	case errors.Is(err, infrasreg.ErrRegistryEndpointNotFound), infrasreg.IsTagNotFound(err):
		return errors.Wrap(ErrRegistryNamespaceInaccessible, err.Error())
	default:
		return errors.Wrap(ErrRegistryUnreachable, err.Error())
	}
}

// createCredential 以当前操作人身份在空间蓝盾项目下创建镜像凭证，与创建空间时共用同一套 ID 与描述规则
func (m *Manager) createCredential(
	ctx context.Context, bkciProjectID string, params UpdateParams, password string,
) (string, error) {
	credID, err := bkmsworkspace.CreateImageCredential(ctx, bkciProjectID, params.Registry, params.Username, password)
	if err != nil {
		// 包装哨兵供接口层识别，底层错误只能以文本保留
		return "", errors.Wrap(ErrBkCICredentialCreateFailed, err.Error())
	}
	return credID, nil
}

// deleteCredential 补偿删除本次新建的蓝盾凭证（credID 为空时跳过），失败只记日志，不覆盖原始错误
func (m *Manager) deleteCredential(ctx context.Context, bkciProjectID, credID string) {
	if credID == "" {
		return
	}

	ctx = context.WithoutCancel(ctx)
	client, err := bkciapi.New(auth.MustGetUser(ctx))
	if err == nil {
		err = client.DeleteCredential(ctx, bkciProjectID, credID)
	}
	if err != nil {
		log.Errorf(ctx, "compensate: delete bkci credential %s in project %s failed: %v", credID, bkciProjectID, err)
	}
}

// saveExternal 写外部仓库记录：已有记录时覆盖，否则新建
func (m *Manager) saveExternal(ctx context.Context, existing, after *bkmsreg.ImageRegistry) error {
	var err error
	if existing == nil {
		err = m.registries.Create(ctx, after)
	} else {
		err = m.registries.Update(ctx, after, m.mutableFields(), m.byExternal(after.WorkspaceID))
	}
	return errors.Wrapf(err, "save workspace %s external image registry", after.WorkspaceID)
}

// restoreExternal 补偿恢复外部仓库记录：原本没有就删掉新建的，原本有就写回旧值；失败只记日志
func (m *Manager) restoreExternal(ctx context.Context, workspaceID string, existing *bkmsreg.ImageRegistry) {
	ctx = context.WithoutCancel(ctx)

	var err error
	if existing == nil {
		err = m.registries.DeleteIfExists(ctx, m.byExternal(workspaceID))
	} else {
		err = m.registries.Update(ctx, existing, m.mutableFields(), m.byExternal(workspaceID))
	}
	if err != nil {
		log.Errorf(ctx, "compensate: restore workspace %s external image registry failed: %v", workspaceID, err)
	}
}

// setRegistryType 只更新空间的生效类型与更新人，避免覆盖并发修改的其他字段
func (m *Manager) setRegistryType(ctx context.Context, workspaceID string, t bkmsreg.ImageRegistryType) error {
	changes := map[string]any{"imageRegistryType": t, "updater": auth.MustGetUser(ctx).ID}
	err := m.workspaces.Update(ctx, changes, bkmsworkspace.ByWorkspaceID(workspaceID))
	return errors.Wrapf(err, "set workspace %s image registry type to %s", workspaceID, t)
}

// getRegistry 按空间与类型取镜像仓库记录，不存在时返回 nil
func (m *Manager) getRegistry(
	ctx context.Context, workspaceID string, t bkmsreg.ImageRegistryType,
) (*bkmsreg.ImageRegistry, error) {
	reg, err := m.registries.Get(ctx, bkmsreg.ByWorkspaceAndType(workspaceID, t))
	if errors.Is(err, bkmsreg.ErrImageRegistryNotFound) {
		return nil, nil
	}
	return reg, errors.Wrapf(err, "get workspace %s %s image registry", workspaceID, t)
}

func (m *Manager) byExternal(workspaceID string) store.Opt {
	return bkmsreg.ByWorkspaceAndType(workspaceID, bkmsreg.ImageRegistryTypeExternal)
}

func (m *Manager) mutableFields() store.Opt {
	return store.UpdateSelected("registry", "username", "password", "bkCICredentialID")
}

// buildAuditData 审计只记录类型、地址与账号，不写入密码
func (m *Manager) buildAuditData(t bkmsreg.ImageRegistryType, reg *bkmsreg.ImageRegistry) map[string]string {
	data := map[string]string{"type": string(t)}
	if reg != nil {
		data["registry"], data["username"] = reg.Registry, reg.Username
	}
	return data
}
