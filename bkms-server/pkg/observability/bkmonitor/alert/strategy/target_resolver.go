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

package strategy

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var managedWorkloadKinds = map[string]struct{}{
	k8skind.Deploy:     {},
	k8skind.STS:        {},
	k8skind.DS:         {},
	k8skind.GameDeploy: {},
}

type remoteTargetContext struct {
	Env             envmodel.Environment
	TrafficLaneName string
	// Workloads 表示该 env/lane 下命中的 workload 名称集合。
	// Helm 场景下可能同时命中多个受管 workload；非 Helm 场景通常只有一个默认 workload 名称。
	Workloads []string
}

// resolveEffectiveEnvs 根据策略生效范围解析出实际需要同步的环境列表。
func (s *Service) resolveEffectiveEnvs(ctx context.Context, strategy *AlertStrategy) ([]envmodel.Environment, error) {
	// 1. 按策略生效范围类型分别查询环境集合。
	switch strategy.EffectiveScope.Type {
	case EffectiveScopeAll:
		allEnvs, err := s.envStore.ListAppEnvs(ctx, strategy.WorkspaceID, strategy.AppID)
		if err != nil {
			return nil, errors.Wrap(err, "list envs")
		}
		// 2. 全量范围下仍需过滤出当前应用真实部署过的环境，避免把无关环境带入远端同步。
		allEnvs = lo.Filter(allEnvs, func(e envmodel.Environment, _ int) bool {
			return isEnvDeployedForApp(e, strategy.AppID)
		})
		return allEnvs, nil
	case EffectiveScopeEnvType:
		envs, err := s.envStore.ListAppEnvsByTypes(
			ctx,
			strategy.WorkspaceID,
			strategy.AppID,
			strategy.EffectiveScope.EnvTypes,
		)
		if err != nil {
			return nil, errors.Wrap(err, "list envs by types")
		}
		return envs, nil
	case EffectiveScopeSpecificEnvs:
		envs, err := s.envStore.ListAppEnvsByIDs(
			ctx,
			strategy.WorkspaceID,
			strategy.AppID,
			strategy.EffectiveScope.EnvIDs,
		)
		if err != nil {
			return nil, errors.Wrap(err, "list envs by ids")
		}
		return envs, nil
	default:
		return nil, errors.Errorf("unknown effective scope type: %s", strategy.EffectiveScope.Type)
	}
}

// isEnvDeployedForApp 判断应用是否真实部署在指定环境中。
func isEnvDeployedForApp(env envmodel.Environment, appID string) bool {
	// 1. 特性环境只认所属应用，避免把共享信息误判为已部署。
	if env.IsFeatureEnv() {
		return env.OwnerAppID == appID
	}
	// 2. 常规环境通过环境记录上的应用列表判断是否已部署。
	return lo.Contains(env.AppIDs, appID)
}

// scopeMatchesEnv 判断某个环境是否命中策略的生效范围配置。
func scopeMatchesEnv(scope EffectiveScope, env envmodel.Environment) bool {
	// 1. 根据 scope 类型分别做命中判断；未知类型统一视为不匹配。
	switch scope.Type {
	case EffectiveScopeAll:
		return true
	case EffectiveScopeEnvType:
		return lo.Contains(scope.EnvTypes, env.Type)
	case EffectiveScopeSpecificEnvs:
		return lo.Contains(scope.EnvIDs, env.ID)
	default:
		return false
	}
}

// buildRemoteTargetContext 为指定环境或泳道构造远端告警同步所需的目标上下文。
func (s *Service) buildRemoteTargetContext(
	ctx context.Context,
	strategy *AlertStrategy,
	env envmodel.Environment,
	trafficLaneName string,
) (remoteTargetContext, error) {
	// 1. 先校验目标环境是否具备监控定位所需的最小信息：clusterID + namespace。
	if env.Cluster.ClusterID == "" || env.Cluster.Namespace == "" {
		log.Errorf(
			ctx,
			"skip syncing alert strategy because env is missing cluster info, "+
				"strategyID=%s env=%s lane=%s clusterID=%s namespace=%s",
			strategy.ID.Hex(),
			env.Name,
			trafficLaneName,
			env.Cluster.ClusterID,
			env.Cluster.Namespace,
		)
		return remoteTargetContext{}, errors.Errorf("env %s missing clusterID or namespace", env.Name)
	}
	// 2. 再解析该 env/lane 下真正要纳入告警的 workloads；依赖异常时直接失败，避免误同步到错误目标。
	workloads, err := s.resolveStrategyWorkloads(ctx, strategy, env, trafficLaneName)
	if err != nil {
		return remoteTargetContext{}, err
	}
	if len(workloads) == 0 {
		log.Errorf(
			ctx,
			"skip syncing alert strategy because workloads resolved empty, strategyID=%s code=%s env=%s lane=%s",
			strategy.ID.Hex(),
			strategy.StrategyCode,
			env.Name,
			trafficLaneName,
		)
		return remoteTargetContext{}, errors.Errorf(
			"env %s has no resolved workloads for app %s",
			env.Name,
			strategy.AppID,
		)
	}
	return remoteTargetContext{
		Env:             env,
		TrafficLaneName: trafficLaneName,
		Workloads:       workloads,
	}, nil
}

// resolveStrategyWorkloads 解析策略在目标环境下应匹配的 workload 名称列表。
// Helm 应用可能返回多个 workload；非 Helm 应用通常只返回一个基于应用名或泳道名推导出的 workload。
func (s *Service) resolveStrategyWorkloads(
	ctx context.Context,
	strategy *AlertStrategy,
	env envmodel.Environment,
	trafficLaneName string,
) ([]string, error) {
	// 1. 如果应用存储未注入，则视为依赖异常并直接失败，避免误用 fallback 规则同步错误目标。
	if s.appStore == nil {
		log.Errorf(
			ctx,
			"app store is nil when resolving workloads, strategyID=%s appID=%s appName=%s lane=%s",
			strategy.ID.Hex(),
			strategy.AppID,
			strategy.AppName,
			trafficLaneName,
		)
		return nil, errors.New("app store is nil")
	}
	// 2. 优先读取应用信息；读取失败时直接返回错误，避免误同步到错误 workload。
	app, err := s.appStore.GetApp(ctx, strategy.AppID)
	if err != nil || app == nil {
		log.Errorf(
			ctx,
			"get app failed when resolving workloads, strategyID=%s appID=%s appName=%s lane=%s err=%v appNil=%v",
			strategy.ID.Hex(),
			strategy.AppID,
			strategy.AppName,
			trafficLaneName,
			err,
			app == nil,
		)
		if err != nil {
			return nil, errors.Wrap(err, "get app")
		}
		return nil, errors.New("app is nil")
	}
	// 3. Helm 应用优先从资源快照中解析真实受管 workload，解析不到时再回退到应用名。
	if bkmsapp.IsHelmBasedType(app.Type) {
		if workloads := s.resolveHelmWorkloads(ctx, app.ID, env.Name, trafficLaneName); len(workloads) > 0 {
			log.Infof(
				ctx,
				"resolved helm workloads for alert strategy, strategyID=%s appID=%s env=%s lane=%s workloads=%v",
				strategy.ID.Hex(),
				app.ID,
				env.Name,
				trafficLaneName,
				workloads,
			)
			return workloads, nil
		}
		log.Warnf(
			ctx,
			"helm workloads not found from snapshot, fallback to app name, strategyID=%s appID=%s appName=%s env=%s lane=%s",
			strategy.ID.Hex(),
			app.ID,
			app.Name,
			env.Name,
			trafficLaneName,
		)
		return []string{app.Name}, nil
	}
	// 4. 非 Helm 应用沿用名称规则生成 workload 名称列表。
	workloads := fallbackWorkloads(app.Name, trafficLaneName)
	log.Infof(
		ctx,
		"resolved non-helm fallback workloads for alert strategy, strategyID=%s appID=%s env=%s lane=%s workloads=%v",
		strategy.ID.Hex(),
		app.ID,
		env.Name,
		trafficLaneName,
		workloads,
	)
	return workloads, nil
}

// resolveHelmWorkloads 从资源快照中提取当前 Helm 应用受管的 workload 名称列表。
func (s *Service) resolveHelmWorkloads(ctx context.Context, appID, envName, trafficLaneName string) []string {
	// 1. 没有快照存储时无法解析真实 workload，直接返回空让上层决定是否回退。
	if s.snapshotStore == nil {
		log.Warnf(
			ctx,
			"snapshot store is nil when resolving helm workloads, appID=%s env=%s lane=%s",
			appID,
			envName,
			trafficLaneName,
		)
		return nil
	}
	// 2. 读取指定环境与泳道的资源快照，读取失败或快照缺失时返回空。
	snapshot, err := s.snapshotStore.Get(ctx, appID, envName, trafficLaneName)
	if err != nil || snapshot == nil {
		log.Warnf(
			ctx,
			"get resource snapshot failed when resolving helm workloads, appID=%s env=%s lane=%s err=%v snapshot_nil=%v",
			appID,
			envName,
			trafficLaneName,
			err,
			snapshot == nil,
		)
		return nil
	}
	// 3. 只保留受平台托管且属于工作负载类型的资源名称。
	workloads := make([]string, 0)
	for _, resource := range snapshot.Resources {
		if !resource.IsManaged {
			continue
		}
		if _, ok := managedWorkloadKinds[resource.Kind]; !ok {
			continue
		}
		workloads = append(workloads, resource.Name)
	}
	// 4. 对 workload 名称去重排序，保证下游同步结果稳定可复现。
	workloads = lo.Uniq(workloads)
	sort.Strings(workloads)
	log.Infof(
		ctx,
		"resolved managed helm workloads from snapshot, appID=%s env=%s lane=%s workloadCount=%d workloads=%v",
		appID,
		envName,
		trafficLaneName,
		len(workloads),
		workloads,
	)
	return workloads
}

// fallbackWorkloads 在无法解析真实 workload 时回退到基于应用名/泳道名的匹配规则。
func fallbackWorkloads(appName, trafficLaneName string) []string {
	// 1. 泳道场景下优先拼接泳道前缀，保持与实际工作负载命名约定一致。
	if trafficLaneName != "" {
		return []string{trafficLaneName + "-" + appName}
	}
	// 2. 非泳道场景直接使用应用名作为默认 workload 名称。
	return []string{appName}
}
