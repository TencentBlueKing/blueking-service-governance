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

package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// EnsureBkSystems 保证依赖的蓝鲸项目存在
func EnsureBkSystems(ctx context.Context, workspaceID, bkciProjectID string, bizID int64) (*BkSystems, error) {
	cmdbInfo, err := fetchCMDBInfo(ctx, bkciProjectID, bizID)
	if err != nil {
		return nil, errors.Wrap(err, "fetch cmdb cmdbInfo")
	}

	bkciProjMgr := bkci.NewProjectManager(workspaceID)
	// 创建/绑定蓝盾项目
	createProjResp, err := bkciProjMgr.Initialize(ctx, bkciProjectID, cmdbInfo.ObsProductID, cmdbInfo.ObsProductName)
	if err != nil {
		return nil, err
	}

	isBoundExistedBKCIProject := true
	// 如果 bkCiProjectID 为空, 说明是创建新项目
	if bkciProjectID == "" {
		isBoundExistedBKCIProject = false
		// 新建项目时会创建一条默认的镜像构建流水线
		// ps: 这里目前产品上没有对应说明, 用户可能不知晓有这个操作
		_, err = bkci.NewPipelineManager(workspaceID).Initialize(ctx, string(bkci.PipelineTypeDockerfile))
		if err != nil {
			return nil, errors.Wrap(err, "create builtin dockerfile pipeline")
		}
		// 同时初始化 Helm Git Build 流水线
		_, err = bkci.NewPipelineManager(workspaceID).Initialize(ctx, string(bkci.PipelineTypeHelmGitBuild))
		if err != nil {
			return nil, errors.Wrap(err, "create builtin helm-git-build pipeline")
		}

		bkciProjectID = createProjResp.Code
	}

	// 初始化蓝盾制品库仓库
	if err = bkrepo.NewProjectManager(workspaceID, auth.MustGetUser(ctx).ID).Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "init bkrepo project")
	}

	return &BkSystems{
		// 蓝盾项目 Code, 可读唯一字符串，如：bkce
		BkCIProjectID: bkciProjectID,
		// 蓝盾项目 UID (32 位字符串)
		BkCIProjectUID: createProjResp.ID,
		// BkRepo 项目 ID 使用蓝盾项目可读 code (如 bkce)
		BkRepoProjectID: createProjResp.Code,
		// BCS 项目 ID 使用蓝盾项目 UID (32 位字符串)
		BkBCSProjectID: createProjResp.ID,
		// BCS 项目 Code, 使用蓝盾项目可读 code (如 bkce)
		BkBCSProjectCode: createProjResp.Code,
		// 表明用户创建项目时是否绑定了已有的蓝盾项目
		IsBoundExistedBKCIProject: isBoundExistedBKCIProject,
		// 运营产品 ID
		ObsProductID: cmdbInfo.ObsProductID,
		// 运营产品名称
		ObsProductName: cmdbInfo.ObsProductName,
		// bkcc ID
		BkCCBizID: cmdbInfo.BizID,
		// 二级业务 ID
		Level2BizID: cmdbInfo.Level2BizID,
	}, nil
}

// CreateExternalImageRegistry 创建外部镜像仓库信息
// 注：内置的镜像仓库在初始化 bkrepo 项目的时候，由 bkrepo.ProjectManager 创建
func CreateExternalImageRegistry(
	ctx context.Context,
	workspaceID string,
	bkciProjectID string,
	store bkmsreg.ImageRegistryStore,
	registry, username, password string,
) error {
	if registry == "" || username == "" || password == "" {
		return errors.Errorf("registry, username, password are required")
	}

	credID := strings.ReplaceAll(uuid.NewString(), "-", "")
	description := fmt.Sprintf("bkms image credential for registry %s", registry)

	client, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "create bkci client")
	}
	// 预先调用蓝盾 API 创建凭证（用于推送镜像）
	if err = client.CreateCredential(ctx, bkciProjectID, credID, description, username, password); err != nil {
		return errors.Wrap(err, "create bkci credential")
	}

	imageReg := &bkmsreg.ImageRegistry{
		WorkspaceID:      workspaceID,
		Type:             bkmsreg.ImageRegistryTypeExternal,
		Registry:         registry,
		Username:         username,
		Password:         password,
		BkCICredentialID: credID,
	}
	if _, err = store.Create(ctx, imageReg); err != nil {
		return errors.Wrap(err, "create image registry")
	}
	return nil
}

// InitWorkspaceUser 初始化 workspace 默认设计的四类用户：管理员、开发者、SRE、运营
func InitWorkspaceUser(ctx context.Context, id, displayName string, managers []string, bkSystem BkSystems) error {
	permMgr := perm.NewManager()

	// 创建管理员角色
	if err := permMgr.CreateWorkspaceAdmin(
		ctx, id, displayName, managers, bkSystem.BkCIProjectID, bkSystem.BkBCSProjectID, bkSystem.BkRepoProjectID,
	); err != nil {
		return err
	}

	// 创建内置角色（developer, sre, operator）
	if err := permMgr.CreateWorkspaceScopeBuiltinRoles(
		ctx, id, displayName, bkSystem.BkCIProjectID, bkSystem.BkBCSProjectID, bkSystem.BkRepoProjectID,
	); err != nil {
		return err
	}

	return nil
}

// fetchCMDBInfo 查询 CMDB 相关字段（二级业务 ID、运营产品 ID/名称）
func fetchCMDBInfo(ctx context.Context, bkciProjectID string, bizID int64) (*cmdb.BusinessDetail, error) {
	user := auth.MustGetUser(ctx)

	// 当 bizID 未传入时，通过 bkciProjectID 从 BCS 获取项目关联的业务 ID
	if bizID <= 0 && bkciProjectID != "" {
		bcsClient, err := bcs.New(user)
		if err != nil {
			return nil, errors.Wrap(err, "initial bcs client")
		}
		project, err := bcsClient.GetProject(ctx, bkciProjectID)
		if err != nil {
			return nil, errors.Wrapf(err, "get bcs project by bkciProjectID: %s", bkciProjectID)
		}
		bizID = cast.ToInt64(project.BizID)
		if bizID <= 0 {
			return nil, errors.Errorf("bcs project(%s) has no associated bizID", bkciProjectID)
		}
	}

	cmdbSvc, err := cmdb.NewService(user)
	if err != nil {
		return nil, errors.Wrap(err, "initial cmdb service")
	}

	return cmdbSvc.GetCMDBInfo(ctx, bizID)
}
