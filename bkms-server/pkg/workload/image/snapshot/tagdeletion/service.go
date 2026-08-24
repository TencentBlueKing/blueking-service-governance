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

package tagdeletion

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	helmrelease "helm.sh/helm/v3/pkg/release"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/credentials"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	trafficMgr "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// ImageUsage 表示某个镜像标签当前仍被哪些工作负载引用。
// 该结构仅用于独立的镜像占用检查接口，帮助前端在删除前提示风险；
// 删除接口本身不会依据这些结果阻止删除
type ImageUsage struct {
	AppID        string // 应用 ID
	AppName      string // 应用名称
	EnvName      string // 环境名称（如 production、staging）
	LaneName     string // 流量泳道名称，基线泳道为空字符串
	WorkloadName string // 工作负载名称（如 Deployment、StatefulSet 等）
	Status       string // 命中的部署记录原始状态值
}

var ErrImageRepoAuthRequired = errors.New("image repository authentication required")

// Service 提供镜像占用检查与镜像删除能力
type Service struct {
	snapshotStore         snapshot.SnapshotStore
	promotionStore        promotion.PromotionStore
	buildConfigStore      build.ConfigStore
	appStore              bkmsapp.ApplicationStore
	envStore              envmodel.EnvironmentStore
	appModelDeployStore   appmodeldeploy.RecordStore
	helmDeployRecordStore helmdeploy.RecordStore
}

// NewService 创建删除服务。
func NewService(
	snapshotStore snapshot.SnapshotStore,
	promotionStore promotion.PromotionStore,
	buildConfigStore build.ConfigStore,
	appStore bkmsapp.ApplicationStore,
	envStore envmodel.EnvironmentStore,
	appModelDeployStore appmodeldeploy.RecordStore,
	helmDeployRecordStore helmdeploy.RecordStore,
) *Service {
	return &Service{
		snapshotStore:         snapshotStore,
		promotionStore:        promotionStore,
		buildConfigStore:      buildConfigStore,
		appStore:              appStore,
		envStore:              envStore,
		appModelDeployStore:   appModelDeployStore,
		helmDeployRecordStore: helmDeployRecordStore,
	}
}

// DeleteImageTag 删除指定应用下的镜像标签，并同步清理本地记录
// 该接口不再额外校验镜像标签是否仍被当前工作负载引用，直接删除，如遇到问题直接抛出错误
func (s *Service) DeleteImageTag(ctx context.Context, appID, tag string) error {
	repoResolver := snapshot.NewService(s.snapshotStore, s.buildConfigStore, s.appStore)
	info, err := repoResolver.ResolveRepoKeyForApp(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "resolve repo key for app %s", appID)
	}
	if !credentials.HasUserPass(info.Username, info.Password) {
		return errors.Wrapf(
			ErrImageRepoAuthRequired,
			"image repository credential is required for deleting remote image tag",
		)
	}

	client := registry.New(info.Username, info.Password, true)
	if err = client.DeleteTag(ctx, info.RepoName, tag); err != nil {
		if registry.IsAuthRequired(err) {
			return errors.Wrapf(ErrImageRepoAuthRequired, "registry authentication required: %v", err)
		}
		return errors.Wrapf(err, "delete remote image %s:%s", info.RepoName, tag)
	}
	if _, err = s.snapshotStore.DeleteByRepoKeyAndTag(ctx, info.RepoKey, tag); err != nil {
		return errors.Wrapf(err, "delete local snapshot %s:%s", info.RepoKey, tag)
	}
	if err = s.promotionStore.DeleteTag(ctx, appID, info.RepoKey, tag); err != nil {
		return errors.Wrapf(err, "delete promotion %s:%s", info.RepoKey, tag)
	}
	return nil
}

// ListImageUsages 查询指定镜像标签在当前生效工作负载中的引用关系，不作为删除接口的前置约束
func (s *Service) ListImageUsages(ctx context.Context, appID, tag string) ([]ImageUsage, error) {
	app, err := s.appStore.GetApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s", appID)
	}
	if app == nil || app.WorkspaceID == "" {
		return nil, errors.Errorf("app %s has no workspace", appID)
	}
	// Include the app's feature environments so image tags deployed there are visible.
	envs, err := s.envStore.ListAppEnvs(ctx, app.WorkspaceID, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "list environments for app %s", appID)
	}

	// laneUsageCollectorFunc 按 env/lane 收集仍在使用该 tag 的 workload 与部署状态；
	// 返回空切片表示当前 lane 未发现占用
	type laneUsageCollectorFunc = func(ctx context.Context, appID, envName, laneName, tag string) ([]ImageUsage, error)

	var collectLaneUsageNamesFunc laneUsageCollectorFunc
	switch {
	case bkmsapp.IsAppModelType(app.Type):
		collectLaneUsageNamesFunc = s.collectAppModelLaneUsageNames
	case bkmsapp.IsHelmBasedType(app.Type):
		collectLaneUsageNamesFunc = s.collectHelmLaneUsageNames
	default:
		return nil, errors.Errorf("unsupported app type %s", app.Type)
	}

	tm := trafficMgr.New()
	var usages []ImageUsage
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for _, env := range envs {
		if !lo.Contains(env.AppIDs, appID) {
			continue
		}
		// 避免循环变量问题
		envVal := env
		g.Go(func() error {
			laneNames, laneErr := listLaneNames(gCtx, tm, app.WorkspaceID, envVal.Name)
			if laneErr != nil {
				return errors.Wrapf(laneErr, "list lanes for envVal %s", envVal.Name)
			}
			for _, laneName := range laneNames {
				laneUsages, collectErr := collectLaneUsageNamesFunc(gCtx, appID, envVal.Name, laneName, tag)
				if collectErr != nil {
					return collectErr
				}
				// lane 中没有工作负载引用此镜像 tag
				if len(laneUsages) == 0 {
					continue
				}
				// 补充 env/lane/app 上下文信息；WorkloadName 和 Status 已由 collector 填充。
				for idx := range laneUsages {
					laneUsages[idx].AppID = app.ID
					laneUsages[idx].AppName = app.Name
					laneUsages[idx].EnvName = envVal.Name
					laneUsages[idx].LaneName = laneName
				}
				mu.Lock()
				usages = append(usages, laneUsages...)
				mu.Unlock()
			}
			return nil
		})
	}
	if err = g.Wait(); err != nil {
		return nil, err
	}
	// 环境层使用并发收集 usages，排序可避免返回顺序受 goroutine 调度影响，
	// 让前端展示和测试断言都保持稳定
	slices.SortFunc(usages, func(a, b ImageUsage) int {
		if result := cmp.Compare(a.EnvName, b.EnvName); result != 0 {
			return result
		}
		if result := cmp.Compare(a.LaneName, b.LaneName); result != 0 {
			return result
		}
		return cmp.Compare(a.WorkloadName, b.WorkloadName)
	})

	return usages, nil
}

// collectAppModelLaneUsageNames 先基于当前 env/lane 的最新部署记录判断指定 tag
// 是否仍可能被使用；若最新记录不足以直接得出结论，则回退到最近一次成功部署记录继续判断
func (s *Service) collectAppModelLaneUsageNames(
	ctx context.Context, appID, envName, laneName, tag string,
) ([]ImageUsage, error) {
	scope := fmt.Sprintf("app %s env %s lane %s", appID, envName, laneName)

	// 1. 获取最新部署记录
	record, err := s.appModelDeployStore.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get latest appmodel deploy record for %s", scope)
	}
	// 2. 若最新记录已能证明目标 tag 仍在使用，直接返回对应 workload 名称
	if usages := collectLatestAppModelLaneUsages(record, tag); len(usages) > 0 {
		return usages, nil
	}
	// 3. 若最新记录属于允许回退的状态，则继续检查最近一次成功部署记录
	if !shouldFallbackAppModel(record) {
		return nil, nil
	}

	// 4. 获取最近一次成功部署记录
	lastSuccessfulRecord, err := s.appModelDeployStore.GetLatestByStatuses(
		ctx,
		appID,
		envName,
		laneName,
		[]appmodeldeploy.Status{appmodeldeploy.StatusDeployed},
	)
	if err != nil {
		if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "find last successful appmodel deploy record for %s", scope)
	}
	// 5. 若最近一次成功部署记录仍使用目标 tag，则返回对应 workload 名称
	return collectLatestAppModelLaneUsages(lastSuccessfulRecord, tag), nil
}

// collectHelmLaneUsageNames 基于当前 env/lane 的 Helm 部署记录，
// 返回仍可能使用指定 tag 的用户可见 workload 名称
// 优先看最新记录，仅当最新记录是非卸载类的非活跃状态时，回退到最近一次 deployed 记录继续判断
func (s *Service) collectHelmLaneUsageNames(
	ctx context.Context, appID, envName, laneName, tag string,
) ([]ImageUsage, error) {
	scope := fmt.Sprintf("app %s env %s lane %s", appID, envName, laneName)

	// 1. 获取最新部署记录
	record, err := s.helmDeployRecordStore.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		if errors.Is(err, helmdeploy.ErrLatestDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get latest helm deploy record for %s", scope)
	}
	// 2. 若最新记录已能证明目标 tag 仍在使用，直接返回对应 workload 名称
	if usages := collectLatestHelmLaneUsages(record, tag); len(usages) > 0 {
		return usages, nil
	}
	// 3. 若最新记录属于允许回退的状态，则继续检查最近一次成功部署记录
	if !shouldFallbackHelm(record) {
		return nil, nil
	}

	// 4. 获取最近一次成功部署记录
	lastSuccessfulRecord, err := s.helmDeployRecordStore.GetLatestByStatuses(
		ctx,
		appID,
		envName,
		laneName,
		[]helmrelease.Status{helmrelease.StatusDeployed},
	)
	if err != nil {
		if errors.Is(err, helmdeploy.ErrLatestDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "find last successful helm deploy record for %s", scope)
	}
	// 5. 若最近一次成功部署记录仍使用目标 tag，则返回对应 workload 名称
	return collectLatestHelmLaneUsages(lastSuccessfulRecord, tag), nil
}
