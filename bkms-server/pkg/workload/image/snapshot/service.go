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

package snapshot

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// 快照陈旧阈值
const (
	// staleTTLDisabled 关闭按时间过期刷新：已有快照即使很久没更新也不刷，只在条数为 0 时初始化
	staleTTLDisabled time.Duration = 0
	// customImageStaleTTL 工作空间自定义镜像超过此时长未刷新则视为过期，查询时顺带异步补刷
	customImageStaleTTL = 5 * time.Minute
)

// Service 镜像快照服务
type Service struct {
	snapshotStore    SnapshotStore
	buildConfigStore build.ConfigStore
	appStore         bkmsapp.ApplicationStore
}

// NewService 创建 Service 实例
func NewService(
	snapshotStore SnapshotStore,
	buildConfigStore build.ConfigStore,
	appStore bkmsapp.ApplicationStore,
) *Service {
	return &Service{
		snapshotStore:    snapshotStore,
		buildConfigStore: buildConfigStore,
		appStore:         appStore,
	}
}

// RepoKeyInfo 仓库 key 解析结果
type RepoKeyInfo struct {
	RepoKey  string
	RepoName string
	Username string
	Password string
}

// ResolveRepoKeyForRepository 根据镜像仓库名称生成无凭据的仓库实例信息。
//
// 运行时镜像当前只保存仓库名称，不保存访问凭据，因此 repoKey 仅由仓库名称决定；
// 后续如果运行时镜像支持独立凭据，可在这里扩展为“仓库名称 + 凭据”的完整仓库实例。
func (s *Service) ResolveRepoKeyForRepository(repoName string) (*RepoKeyInfo, error) {
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}
	return &RepoKeyInfo{
		RepoKey:  GenerateRepoKey(repoName, "", ""),
		RepoName: repoName,
	}, nil
}

// ResolveRepoKeyForApp 解析应用的仓库信息，返回 repoKey、repoName 和认证信息
func (s *Service) ResolveRepoKeyForApp(
	ctx context.Context, appID string,
) (*RepoKeyInfo, error) {
	app, err := s.appStore.GetApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s", appID)
	}

	cfg, err := s.buildConfigStore.Get(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s build config", app.ID)
	}

	info, rErr := build.ResolveImageRepoInfo(ctx, cfg, app.WorkspaceID, app.Name)
	if rErr != nil {
		return nil, errors.Wrap(rErr, "resolve image repo info")
	}

	return &RepoKeyInfo{
		RepoKey:  GenerateRepoKey(info.RepoName, info.Username, info.Password),
		RepoName: info.RepoName,
		Username: info.Username,
		Password: info.Password,
	}, nil
}

// ResolveRepoKeyForWorkspace 解析工作空间自定义镜像的仓库信息，返回 repoKey、repoName 和认证信息。
//
// 自定义镜像既不像运行时镜像那样无凭据（ResolveRepoKeyForRepository），也不像应用镜像
// 那样能从构建配置里取到凭据（ResolveRepoKeyForApp），它必须按 workspaceID 反查
// image_registries 拿到工作空间当前生效镜像源的账密，再按「镜像名 + 账密」生成 repoKey。
//
// 契约约束：涉及自定义镜像的下游实现一律通过本方法取 repoKey，不得各自直接调用
// GenerateRepoKey，否则同一镜像会算出不同的 repoKey 导致快照对不上。
func (s *Service) ResolveRepoKeyForWorkspace(
	ctx context.Context, workspaceID, imageName string,
) (*RepoKeyInfo, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	imageName = strings.TrimSpace(imageName)
	if workspaceID == "" {
		return nil, errors.New("workspaceID is required")
	}
	if imageName == "" {
		return nil, errors.New("image name is required")
	}

	reg, err := workspace.GetWorkspaceImageRegistry(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "get workspace image registry")
	}

	return &RepoKeyInfo{
		RepoKey:  GenerateRepoKey(imageName, reg.Username, reg.Password),
		RepoName: imageName,
		Username: reg.Username,
		Password: reg.Password,
	}, nil
}

// RefreshWorkspaceSnapshots 按工作空间凭证刷新自定义镜像快照
func (s *Service) RefreshWorkspaceSnapshots(
	ctx context.Context, workspaceID, imageName string,
) (*RefreshResult, error) {
	info, err := s.ResolveRepoKeyForWorkspace(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve repo key for workspace %s image %s", workspaceID, imageName)
	}
	return s.refreshSnapshotsByRepoInfo(ctx, info)
}

// GetWorkspaceSnapshotStatus 按工作空间凭证口径查自定义镜像的快照状态。
//
// repoKey 必须经 ResolveRepoKeyForWorkspace 计算，才能与刷新流程写入的状态对上；
// 该镜像尚无任何快照记录时返回 nil，调用方据此判断是否需要初始化刷新
func (s *Service) GetWorkspaceSnapshotStatus(
	ctx context.Context, workspaceID, imageName string,
) (*RepoSnapshotStatus, error) {
	info, err := s.ResolveRepoKeyForWorkspace(ctx, workspaceID, imageName)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve repo key for workspace %s image %s", workspaceID, imageName)
	}

	status, err := s.snapshotStore.GetStatus(ctx, info.RepoKey)
	if err != nil {
		return nil, errors.Wrapf(err, "get snapshot status of %s", info.RepoKey)
	}
	return status, nil
}

// ListRepositorySnapshots 从本地快照查询指定镜像仓库的标签列表。
//
// 如果本地还没有该仓库的快照记录，会复用刷新流程异步触发一次初始化同步；
// 本次请求仍返回当前本地结果，避免前端接口被远端镜像仓库请求阻塞。
func (s *Service) ListRepositorySnapshots(
	ctx context.Context,
	repoName, keyword string,
	page, pageSize int,
) ([]Image, int64, *RepoSnapshotStatus, error) {
	info, err := s.ResolveRepoKeyForRepository(repoName)
	if err != nil {
		return nil, 0, nil, err
	}

	// 官方镜像不启用 TTL 懒刷新，其快照陈旧属存量问题，另行评估
	return s.listSnapshotsByRepoInfo(ctx, info, keyword, page, pageSize, staleTTLDisabled)
}

// ListAppSnapshots 从本地快照查询镜像列表
func (s *Service) ListAppSnapshots(
	ctx context.Context,
	appID, keyword string,
	page, pageSize int,
) ([]Image, int64, *RepoSnapshotStatus, error) {
	info, err := s.ResolveRepoKeyForApp(ctx, appID)
	if err != nil {
		return nil, 0, nil, errors.Wrapf(err, "resolve repo key for app %s", appID)
	}

	// 应用产物镜像已有构建成功等 4 个事件驱动的刷新触发点，不需要 TTL 懒刷新
	return s.listSnapshotsByRepoInfo(ctx, info, keyword, page, pageSize, staleTTLDisabled)
}

// ListWorkspaceSnapshots 从本地快照查询工作空间自定义镜像的标签列表。
//
// 与官方镜像、应用产物镜像不同，本路径启用 TTL 懒刷新：快照超过 customImageStaleTTL
// 未成功刷新时异步补一次，本次请求仍立即返回当前快照内容
func (s *Service) ListWorkspaceSnapshots(
	ctx context.Context,
	workspaceID, imageName, keyword string,
	page, pageSize int,
) ([]Image, int64, *RepoSnapshotStatus, error) {
	info, err := s.ResolveRepoKeyForWorkspace(ctx, workspaceID, imageName)
	if err != nil {
		return nil, 0, nil, errors.Wrapf(err, "resolve repo key for workspace %s image %s", workspaceID, imageName)
	}

	return s.listSnapshotsByRepoInfo(ctx, info, keyword, page, pageSize, customImageStaleTTL)
}

// ListWorkspaceSnapshotsByRepoInfo 在 repoKey 已解析好时读自定义镜像快照，启用 TTL 懒刷新
func (s *Service) ListWorkspaceSnapshotsByRepoInfo(
	ctx context.Context, info *RepoKeyInfo, keyword string, page, pageSize int,
) ([]Image, int64, *RepoSnapshotStatus, error) {
	return s.listSnapshotsByRepoInfo(ctx, info, keyword, page, pageSize, customImageStaleTTL)
}

// RefreshSnapshotsByRepoInfo 在 repoKey 已解析好时同步刷新快照
func (s *Service) RefreshSnapshotsByRepoInfo(ctx context.Context, info *RepoKeyInfo) (*RefreshResult, error) {
	return s.refreshSnapshotsByRepoInfo(ctx, info)
}

// listSnapshotsByRepoInfo 从本地快照查询仓库实例的标签列表。
//
// staleTTL 控制是否启用 TTL 懒刷新，传 staleTTLDisabled 则只保留「快照为空则初始化」的原有行为
func (s *Service) listSnapshotsByRepoInfo(
	ctx context.Context,
	info *RepoKeyInfo,
	keyword string,
	page, pageSize int,
	staleTTL time.Duration,
) ([]Image, int64, *RepoSnapshotStatus, error) {
	// 查询快照
	snapshots, total, err := s.snapshotStore.ListByRepoKey(ctx, info.RepoKey, keyword, page, pageSize)
	if err != nil {
		return nil, 0, nil, errors.Wrap(err, "list snapshots")
	}

	// 获取仓库状态
	status, err := s.snapshotStore.GetStatus(ctx, info.RepoKey)
	if err != nil {
		return nil, 0, nil, errors.Wrap(err, "get snapshot status")
	}

	// 刷新只能异步：本次请求必须立即返回当前快照，不被远端镜像仓库请求阻塞
	if s.shouldTriggerRefresh(total, status, keyword, staleTTL) {
		log.Infof(ctx, "snapshot of %s is empty or stale, triggering refresh", info.RepoName)
		go func() {
			rCtx := context.WithoutCancel(ctx)
			if _, rErr := s.refreshSnapshotsByRepoInfo(rCtx, info); rErr != nil {
				log.Errorf(rCtx, "trigger refresh for %s failed: %v", info.RepoName, rErr)
			}
		}()
	}

	return snapshots, total, status, nil
}

// shouldTriggerRefresh 判断查询标签列表时是否顺带触发一次异步刷新
func (s *Service) shouldTriggerRefresh(
	total int64, status *RepoSnapshotStatus, keyword string, staleTTL time.Duration,
) bool {
	// 已在刷新中再起 goroutine 也只会撞上 TrySetRefreshing，直接跳过
	if status != nil && status.RefreshStatus == RefreshStatusRefreshing {
		return false
	}
	// 未过滤时 total==0 才是从未初始化；带关键字的 0 只说明当前搜索没命中
	if total == 0 && strings.TrimSpace(keyword) == "" {
		return true
	}
	// 存量路径传 staleTTLDisabled，不做陈旧判断
	if staleTTL <= staleTTLDisabled {
		return false
	}
	// 状态记录缺失或从未成功刷新过，都视为已过期，否则这批快照将永远不会被刷新
	if status == nil || status.LastRefreshedAt == nil {
		return true
	}
	return time.Since(*status.LastRefreshedAt) > staleTTL
}

// RefreshAppSnapshots 刷新应用镜像快照。
//
// forceDetailSyncTags 中的标签会无条件重新拉取详情，即使本地快照已有详情；
// 不传该参数时，默认同步详情尚未补全（builtAt 为空）、被标记需重拉，以及 latest 的标签
func (s *Service) RefreshAppSnapshots(
	ctx context.Context, appID string, forceDetailSyncTags ...string,
) (*RefreshResult, error) {
	info, err := s.ResolveRepoKeyForApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	return s.refreshSnapshotsByRepoInfo(ctx, info, forceDetailSyncTags...)
}

// RefreshRepositorySnapshots 刷新指定镜像仓库的快照。
//
// 该方法用于平台运行时镜像等不依赖应用构建配置的场景，当前只按仓库名称访问远端镜像仓库。
func (s *Service) RefreshRepositorySnapshots(ctx context.Context, repoName string) (*RefreshResult, error) {
	info, err := s.ResolveRepoKeyForRepository(repoName)
	if err != nil {
		return nil, err
	}
	return s.refreshSnapshotsByRepoInfo(ctx, info)
}

func (s *Service) refreshSnapshotsByRepoInfo(
	ctx context.Context, info *RepoKeyInfo, forceDetailSyncTags ...string,
) (*RefreshResult, error) {
	// 先落库标记，确保即使抢不到刷新权，这些标签也会在后续任意一次刷新中被重新拉取详情
	if len(forceDetailSyncTags) > 0 {
		if err := s.snapshotStore.MarkDetailSyncPending(ctx, info.RepoKey, forceDetailSyncTags); err != nil {
			return nil, errors.Wrap(err, "mark detail sync pending")
		}
	}

	// 幂等性检查
	acquired, err := s.snapshotStore.TrySetRefreshing(ctx, info.RepoKey)
	if err != nil {
		return nil, errors.Wrap(err, "try set refreshing")
	}
	if !acquired {
		log.Infof(ctx, "snapshot refresh for %s is already in progress", info.RepoKey)
		return &RefreshResult{
			Status:  RefreshResultRefreshing,
			Message: "Snapshot refresh for this repository is already in progress",
		}, nil
	}

	result, err := s.doRefresh(ctx, info)
	if err != nil {
		// 回置要用未取消的 ctx：失败常源于 ctx 超时，复用同一个 ctx 会连状态也写不进去，
		// 而 TrySetRefreshing 无超时恢复，状态将永久停在 refreshing 挡死后续所有刷新
		rCtx := context.WithoutCancel(ctx)
		if statusErr := s.snapshotStore.UpsertStatus(rCtx, &RepoSnapshotStatus{
			RepoKey:       info.RepoKey,
			RepoName:      info.RepoName,
			RefreshStatus: RefreshStatusIdle,
			LastError:     err.Error(),
		}); statusErr != nil {
			log.Errorf(rCtx, "record refresh failure status for %s failed: %v", info.RepoKey, statusErr)
		}
		return nil, errors.Wrapf(err, "refresh snapshots for %s", info.RepoKey)
	}
	return result, nil
}

// doRefresh 执行标签刷新核心逻辑
func (s *Service) doRefresh(ctx context.Context, info *RepoKeyInfo) (*RefreshResult, error) {
	// 获取远程所有标签
	client := registry.New(info.Username, info.Password, true)
	remoteTags, err := client.ListAllTags(ctx, info.RepoName)
	if err != nil {
		return nil, errors.Wrapf(err, "list all tags for %s", info.RepoName)
	}

	// 远端已返回，后续落库不得再受请求取消影响，否则 tag 已拉到却写不进状态
	persistCtx := context.WithoutCancel(ctx)

	// 获取本地已有标签
	localTags, err := s.snapshotStore.ListAllTags(persistCtx, info.RepoKey)
	if err != nil {
		return nil, errors.Wrap(err, "list all tags")
	}

	// 计算 diff：找出新增标签
	localTagSet := set.NewStringSetWithValues(localTags)
	newTags := lo.Filter(remoteTags, func(tag string, _ int) bool {
		return !localTagSet.Has(tag)
	})

	// 只对新增标签进行插入
	var addedCount int64
	if len(newTags) > 0 {
		snapshots := make([]Image, 0, len(newTags))
		for _, tag := range newTags {
			snapshots = append(snapshots, Image{
				RepoKey: info.RepoKey,
				Tag:     tag,
			})
		}
		if err = s.snapshotStore.UpsertSnapshots(persistCtx, info.RepoKey, snapshots); err != nil {
			return nil, errors.Wrap(err, "upsert new snapshots")
		}
		addedCount = int64(len(newTags))
	}

	// 删除远程已消失的标签
	removedCount, err := s.snapshotStore.DeleteByRepoKeyExcludeTags(persistCtx, info.RepoKey, remoteTags)
	if err != nil {
		return nil, errors.Wrap(err, "delete excluded tags")
	}

	// 更新状态为 idle + lastRefreshedAt（必须在提交异步任务之前，确保状态已重置为 idle，
	// 否则异步任务消费时 TrySetDetailSyncing 会因状态非 idle 而失败）
	refreshAt := time.Now()
	if err = s.snapshotStore.UpsertStatus(persistCtx, &RepoSnapshotStatus{
		RepoKey:         info.RepoKey,
		RepoName:        info.RepoName,
		RefreshStatus:   RefreshStatusIdle,
		LastRefreshedAt: &refreshAt,
		LastError:       "",
	}); err != nil {
		log.Errorf(persistCtx, "update status after refresh for %s failed: %v", info.RepoKey, err)
		return nil, errors.Wrap(err, "update status after refresh")
	}

	// 状态重置为 idle 后，判断是否需要异步触发详情同步
	resultMsg := "Snapshot refresh completed, no tags need detail sync"
	unsyncedDetailTags, err := s.snapshotStore.ListUnsyncedDetailTags(persistCtx, info.RepoKey)
	if err != nil {
		return nil, errors.Wrap(err, "list unsynced detail tags")
	}

	if len(unsyncedDetailTags) > 0 {
		// 通过任务队列异步触发详情同步（构造时即完成凭据加密）
		detailSyncArgs, encryptErr := NewImageDetailSyncArgs(
			info.RepoKey, info.RepoName, info.Username, info.Password,
		)
		if encryptErr != nil {
			log.Errorf(persistCtx, "encrypt credentials for detail sync task %s failed: %v", info.RepoKey, encryptErr)
			resultMsg = "Snapshot refresh completed, but the detail sync task was not submitted"
		} else if enqueueErr := taskq.Enqueue(persistCtx, DetailSyncTask.NewTask(*detailSyncArgs)); enqueueErr != nil {
			log.Errorf(persistCtx, "enqueue image detail sync task for %s failed: %v", info.RepoKey, enqueueErr)
			return nil, errors.Wrap(enqueueErr, "enqueue image detail sync task")
		} else {
			resultMsg = "Snapshot refresh completed, and the detail sync task has started asynchronously"
		}
	}

	return &RefreshResult{
		Status:        RefreshResultSuccess,
		Message:       resultMsg,
		AddedTagCnt:   addedCount,
		RemovedTagCnt: removedCount,
	}, nil
}
