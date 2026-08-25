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

package customruntime

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// tag 查询与刷新的同步时间预算
const (
	// realtimeTagFetchTimeout 手动输入镜像实时拉取 tag 的同步等待上限，给慢网络留余量
	realtimeTagFetchTimeout = 30 * time.Second
	// tagRefreshTimeout 显式刷新的同步等待上限
	tagRefreshTimeout = 10 * time.Second
)

// 仓库侧失败的对外提示，逐条可读且不回显仓库返回的原始内容
const (
	registryAuthFailedMessage  = "registry authentication failed, please check the workspace image registry credential"
	registryImageMissedMessage = "image not found in the registry, please check whether the image has been pushed"
	registryUnreachableMessage = "failed to access the registry, please try again later"
)

// TagQueryResult tag 查询结果。
//
// 快照来源与实时来源共用该结构，调用方无从区分来源，也不应据此分支处理
type TagQueryResult struct {
	// Tags 当前页的 tag，实时来源只填 Tag 字段
	Tags []snapshot.Image
	// Total 关键字过滤后的总数，不是当前页长度
	Total int64
	// Status 快照状态，实时来源无快照记录故为 nil，序列化后表现为 idle
	Status *snapshot.RepoSnapshotStatus
}

// TagQueryManager 查询与刷新工作空间自定义镜像的 tag
type TagQueryManager struct {
	store       Store
	checker     *ExistenceChecker
	snapshotSvc *snapshot.Service
}

// NewTagQueryManager 创建 TagQueryManager
func NewTagQueryManager(store Store, snapshotSvc *snapshot.Service) *TagQueryManager {
	return &TagQueryManager{
		store:       store,
		checker:     NewExistenceChecker(snapshotSvc),
		snapshotSvc: snapshotSvc,
	}
}

// ListTags 查询某镜像可选的 tag 列表。
//
// 已落库镜像读本地快照，用户手动输入、平台尚无记录的镜像则用工作空间凭证实时远程拉取。
// 两条来源的分页与关键字过滤口径一致，来源判定在服务端完成，返回值中不含来源标识
func (m *TagQueryManager) ListTags(
	ctx context.Context, workspaceID, imageName, keyword string, page, pageSize int,
) (*TagQueryResult, error) {
	// 归属校验必须排在取凭证之前，理由见 ensureBelongsToWorkspace
	reg, err := m.ensureBelongsToWorkspace(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrap(err, "list custom runtime image tags")
	}

	persisted, err := m.store.Exists(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "decide tag source of image %s", imageName)
	}
	if persisted {
		return m.listFromSnapshot(ctx, m.repoInfoFromRegistry(imageName, reg), keyword, page, pageSize)
	}
	return m.listFromRegistry(ctx, reg, imageName, keyword, page, pageSize)
}

// RefreshTags 同步刷新某镜像的 tag 快照。
//
// 不受快照陈旧阈值限制，服务「刚推完想立刻看到」的场景。仓库侧失败折成 failed 结果正常返回，
// 超时与平台自身故障一律上抛，避免把平台故障伪装成刷新失败而掩盖问题
func (m *TagQueryManager) RefreshTags(
	ctx context.Context, workspaceID, imageName string,
) (*snapshot.RefreshResult, error) {
	reg, err := m.ensureBelongsToWorkspace(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrap(err, "refresh custom runtime image tags")
	}

	persisted, err := m.store.Exists(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "check if image %s is persisted in workspace %s", imageName, workspaceID)
	}
	if !persisted {
		return nil, errors.Wrapf(
			ErrCustomRuntimeImageNotFound, "image %s is not persisted in workspace %s", imageName, workspaceID,
		)
	}

	ctx, cancel := context.WithTimeout(ctx, tagRefreshTimeout)
	defer cancel()

	result, err := m.snapshotSvc.RefreshSnapshotsByRepoInfo(ctx, m.repoInfoFromRegistry(imageName, reg))
	if err == nil {
		return result, nil
	}

	// 超时判定要排在仓库侧判定之前：超时同样会被底层包成网络错误，先判才不会被误归为仓库侧失败
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, errors.Wrapf(ErrRegistryAccessTimeout, "refresh tags of %s", imageName)
	}
	if isRegistryFailure(err) {
		// 错误在此被吞掉换成 failed 结果，必须留日志，否则仓库侧故障不可见
		log.Errorf(ctx, "refresh tags of %s failed: %v", imageName, err)
		return &snapshot.RefreshResult{
			Status:  snapshot.RefreshResultFailed,
			Message: registryFailureMessage(err),
		}, nil
	}
	return nil, errors.Wrapf(err, "refresh tag snapshots of image %s", imageName)
}

// ensureBelongsToWorkspace 确认镜像名落在工作空间生效镜像源路径下。
//
// 该校验必须在任何取凭证的动作之前完成：镜像名来自请求参数，而后续会拿工作空间镜像源账密
// 去访问它指向的仓库，放行任意地址等于把凭证发往第三方主机
func (m *TagQueryManager) ensureBelongsToWorkspace(
	ctx context.Context, workspaceID, imageName string,
) (*bkmsreg.ImageRegistry, error) {
	reg, err := m.checker.lookupRegistry(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup workspace %s image registry", workspaceID)
	}
	if reg == nil {
		return nil, errors.Wrapf(ErrWorkspaceRegistryUnbound, "workspace %s", workspaceID)
	}
	if !nameBelongsToRegistry(imageName, reg.Registry) {
		return nil, errors.Wrapf(
			ErrImageNotInWorkspaceRegistry, "image %s must be prefixed with %s", imageName, reg.Registry,
		)
	}
	return reg, nil
}

// repoInfoFromRegistry 用已查出的镜像源组装 repoKey，避免同一次请求再反查一次凭证
func (m *TagQueryManager) repoInfoFromRegistry(imageName string, reg *bkmsreg.ImageRegistry) *snapshot.RepoKeyInfo {
	imageName = strings.TrimSpace(imageName)
	return &snapshot.RepoKeyInfo{
		RepoKey:  snapshot.GenerateRepoKey(imageName, reg.Username, reg.Password),
		RepoName: imageName,
		Username: reg.Username,
		Password: reg.Password,
	}
}

// listFromSnapshot 读本地快照，并按陈旧阈值顺带触发异步刷新
func (m *TagQueryManager) listFromSnapshot(
	ctx context.Context, info *snapshot.RepoKeyInfo, keyword string, page, pageSize int,
) (*TagQueryResult, error) {
	tags, total, status, err := m.snapshotSvc.ListWorkspaceSnapshotsByRepoInfo(
		ctx, info, keyword, page, pageSize,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list tag snapshots of image %s", info.RepoName)
	}
	return &TagQueryResult{Tags: tags, Total: total, Status: status}, nil
}

// listFromRegistry 实时向仓库拉取全量 tag 后在内存里过滤分页。
//
// 结果既不写快照集合也不建镜像记录，避免用户尚未保存的临时输入污染数据；不做结果缓存，
// 因此翻页与改关键字都会各触发一次新的全量拉取
func (m *TagQueryManager) listFromRegistry(
	ctx context.Context, reg *bkmsreg.ImageRegistry, imageName, keyword string, page, pageSize int,
) (*TagQueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, realtimeTagFetchTimeout)
	defer cancel()

	client := infrasreg.New(reg.Username, reg.Password, true)
	allTags, err := client.ListAllTags(ctx, imageName)
	if err != nil {
		classified := classifyRealtimeTagError(err, imageName)
		if errors.Is(classified, ErrRegistryAccessFailed) {
			// 对外只回固定文案，原始仓库响应留在这里，避免进到 API message
			log.Errorf(ctx, "list tags of %s from registry failed: %v", imageName, err)
		}
		return nil, classified
	}

	matched := filterTagsByKeyword(allTags, keyword)
	return &TagQueryResult{Tags: pageTags(matched, page, pageSize), Total: int64(len(matched))}, nil
}

// classifyRealtimeTagError 把实时拉取的仓库错误归类，供上层映射为对外错误码。
//
// 超时必须最先判定，否则会被底层包成网络错误而落到通用分支；
// ListAllTags 的 404 直接表示镜像名不存在，无需像 tag 校验那样再回退一次列表
func classifyRealtimeTagError(err error, imageName string) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errors.Wrapf(ErrRegistryAccessTimeout, "list tags of %s", imageName)
	case infrasreg.IsAuthRequired(err):
		return errors.Wrapf(ErrRegistryAccessDenied, "list tags of %s", imageName)
	case infrasreg.IsTagNotFound(err):
		return errors.Wrapf(ErrImageNameNotFound, "image %s", imageName)
	default:
		return errors.Wrapf(ErrRegistryAccessFailed, "list tags of %s", imageName)
	}
}

// isRegistryFailure 判断刷新失败是否来自镜像仓库侧。
//
// 仓库侧失败对用户是可解释、可自行处理的，折成 failed 结果返回；
// 平台自身故障（如数据库不可用）必须上抛为服务端错误，不能伪装成刷新失败
func isRegistryFailure(err error) bool {
	if infrasreg.IsAuthRequired(err) || infrasreg.IsTagNotFound(err) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// registryFailureMessage 仓库侧失败的对外提示，不回显仓库原始响应，避免带出无关内部信息
func registryFailureMessage(err error) string {
	switch {
	case infrasreg.IsAuthRequired(err):
		return registryAuthFailedMessage
	case infrasreg.IsTagNotFound(err):
		return registryImageMissedMessage
	default:
		return registryUnreachableMessage
	}
}

// filterTagsByKeyword 按关键字过滤 tag。
//
// 大小写不敏感，与快照来源的 Mongo 正则过滤口径保持一致；registry 客户端自带的 ListTags
// 用的是大小写敏感匹配，两条来源的过滤语义会出现漂移，故不复用它
func filterTagsByKeyword(tags []string, keyword string) []string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return tags
	}
	lowered := strings.ToLower(keyword)
	return lo.Filter(tags, func(tag string, _ int) bool {
		return strings.Contains(strings.ToLower(tag), lowered)
	})
}

// pageTags 对过滤后的结果做内存分页切片，页码越界返回空列表而不报错。
//
// 只填 Tag：补齐 digest / size / builtAt 需为当前页每个 tag 各发一次详情请求，
// 会突破实时拉取的同步预算；快照来源在详情同步完成前同样只有 tag 名
func pageTags(tags []string, page, pageSize int) []snapshot.Image {
	start := (page - 1) * pageSize
	if start < 0 || start >= len(tags) {
		return []snapshot.Image{}
	}
	end := min(start+pageSize, len(tags))
	return lo.Map(tags[start:end], func(tag string, _ int) snapshot.Image {
		return snapshot.Image{Tag: tag}
	})
}
