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

// Package status queries and aggregates application deployment status.
package status

import (
	"context"
	"time"

	"github.com/hashicorp/go-set/v3"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
)

// StatusUnknown 表示无法解析的部署状态
const StatusUnknown = "unknown"

var (
	// ErrDeployRecordNotFound 部署记录不存在错误
	ErrDeployRecordNotFound = errors.New("deploy record not found")
	// ErrUnsupportedAppType 不支持的应用类型错误
	ErrUnsupportedAppType = errors.New("unsupported app type")
)

// AppDeployStatus 应用在某一环境下的某一泳道上的部署状态
type AppDeployStatus struct {
	EnvID           string
	EnvName         string
	EnvDisplayName  string
	EnvType         string
	EnvKind         string
	AppID           string
	AppName         string
	AppType         string
	TrafficLaneName string
	DeployStatus    string
	ImageTag        string
}

// LatestDeployStatus is the latest deploy status for one app+env+lane, including start time.
type LatestDeployStatus struct {
	Status    string
	ImageTag  string
	StartedAt time.Time
}

// DeployStatusService 应用部署状态查询
// - 查询某个环境下所有已部署的应用的部署状态
// - 查询某个工作空间内指定应用在各环境上的部署状态
type DeployStatusService struct {
	AppStore                   app.ApplicationStore
	EnvStore                   envmodel.EnvironmentStore
	BuildAutoDeployRecordStore autodeploy.RecordStore
	AppModelDeployRecordStore  appmodel.RecordStore
	HelmDeployRecordStore      helm.RecordStore
	TrafficManager             trafficmanager.TrafficManager
}

// NewDeployStatusService 构造部署状态服务
func NewDeployStatusService(
	appStore app.ApplicationStore,
	envStore envmodel.EnvironmentStore,
	buildAutoDeployRecordStore autodeploy.RecordStore,
	appModelDeployRecordStore appmodel.RecordStore,
	helmDeployRecordStore helm.RecordStore,
) *DeployStatusService {
	return &DeployStatusService{
		AppStore:                   appStore,
		EnvStore:                   envStore,
		BuildAutoDeployRecordStore: buildAutoDeployRecordStore,
		AppModelDeployRecordStore:  appModelDeployRecordStore,
		HelmDeployRecordStore:      helmDeployRecordStore,
		TrafficManager:             trafficmanager.New(),
	}
}

// ListForEnvironment 获取指定环境下部署了的应用在各泳道上的部署状态
func (s *DeployStatusService) ListForEnvironment(
	ctx context.Context,
	environment *envmodel.Environment,
) ([]AppDeployStatus, error) {
	apps, err := s.AppStore.GetAppsByIDs(ctx, environment.AppIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "list for environment %s", environment.Name)
	}
	return s.listForEnvironment(ctx, environment, apps)
}

// ListForAppsInWorkspace 获取工作空间内指定应用在其所在各环境及各泳道上的部署状态
// 传入 workspaceID 确定环境范围，传入 apps 确定要查询的 app 列表。
// 此处的环境将包括所有标准环境以及应用的特性环境。
// 返回 map，key 为 appID，value 为该应用在各环境及各泳道上的部署状态列表
func (s *DeployStatusService) ListForAppsInWorkspace(
	ctx context.Context, workspaceID string, apps []*app.Application,
) (map[string][]AppDeployStatus, error) {
	out := make(map[string][]AppDeployStatus, len(apps))

	appMap := lo.SliceToMap(apps, func(app *app.Application) (string, *app.Application) { return app.ID, app })
	appIDs := lo.Keys(appMap)

	envs, err := s.EnvStore.ListBatchAppEnvs(ctx, workspaceID, appIDs)
	if err != nil {
		return nil, errors.Wrap(err, "list batch app environments")
	}

	scopes, err := s.resolveEnvStatusScopes(ctx, envs, apps, appMap)
	if err != nil {
		return nil, err
	}

	// 进入环境循环前一次性聚合部署记录，避免按「环境 × 应用 × 泳道」逐条查库
	cache, err := s.prefetchLatestStatuses(ctx, scopes)
	if err != nil {
		return nil, err
	}

	for i := range scopes {
		scope := &scopes[i]
		deployStatuses, buildErr := s.buildEnvDeployStatuses(scope, cache)
		if buildErr != nil {
			return nil, errors.Wrapf(buildErr, "list deploy statuses for environment %s", scope.env.Name)
		}
		for _, deployStatus := range deployStatuses {
			out[deployStatus.AppID] = append(out[deployStatus.AppID], deployStatus)
		}
	}

	return out, nil
}

// resolveEnvStatusScopes 按「先标准环境、后特性环境」解析每个环境的查询范围。
// 标准环境面向传入的全部应用，特性环境只面向其归属应用；没有待查应用的环境不产出 scope。
func (s *DeployStatusService) resolveEnvStatusScopes(
	ctx context.Context,
	envs []envmodel.Environment,
	apps []*app.Application,
	appMap map[string]*app.Application,
) ([]envStatusScope, error) {
	// 将环境区分为两类：标准和特性
	stdEnvs := make([]*envmodel.Environment, 0, len(envs))
	featEnvs := make([]*envmodel.Environment, 0, len(envs))
	for i := range envs {
		if envs[i].IsFeatureEnv() {
			featEnvs = append(featEnvs, &envs[i])
		} else {
			stdEnvs = append(stdEnvs, &envs[i])
		}
	}

	scopes := make([]envStatusScope, 0, len(envs))
	for _, env := range stdEnvs {
		scope, err := s.resolveEnvStatusScope(ctx, env, apps)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve deploy status scope for standard environment %s", env.Name)
		}
		if scope != nil {
			scopes = append(scopes, *scope)
		}
	}
	for _, env := range featEnvs {
		ownerApp, ok := appMap[env.OwnerAppID]
		if !ok {
			continue
		}
		scope, err := s.resolveEnvStatusScope(ctx, env, []*app.Application{ownerApp})
		if err != nil {
			return nil, errors.Wrapf(err, "resolve deploy status scope for feature environment %s", env.Name)
		}
		if scope != nil {
			scopes = append(scopes, *scope)
		}
	}
	return scopes, nil
}

// ListFeatureEnvsForApp 获取单个应用在一批特性环境下的汇总部署状态。
// 传入的环境必须全部是特性环境且必须属于当前应用。
//
// Returns:
//   - map[envName][]value，value 每条泳道部署状态，如果多泳道有部署则返回多个值，空 slice
//     表示当前特性环境尚无部署记录。
func (s *DeployStatusService) ListFeatureEnvsForApp(
	ctx context.Context,
	application *app.Application,
	featureEnvs []envmodel.Environment,
) (map[string][]AppDeployStatus, error) {
	// 先整体校验传入环境的合法性
	for i := range featureEnvs {
		featureEnv := &featureEnvs[i]
		if !featureEnv.IsFeatureEnv() {
			return nil, errors.Errorf("env %s is not a feature environment", featureEnv.Name)
		}
		if featureEnv.OwnerAppID != application.ID {
			return nil, errors.Errorf("feature env %s does not belong to app %s", featureEnv.Name, application.ID)
		}
	}

	out := make(map[string][]AppDeployStatus, len(featureEnvs))
	for i := range featureEnvs {
		featureEnv := &featureEnvs[i]
		deployStatuses, err := s.listForEnvironment(ctx, featureEnv, []*app.Application{application})
		if err != nil {
			return nil, errors.Wrapf(err, "list deploy statuses for feature environment %s", featureEnv.Name)
		}
		out[featureEnv.Name] = deployStatuses
	}
	return out, nil
}

// envStatusScope 是单个环境上「要去查部署状态」的范围，不是已经发生的部署。
//
// 格子是 apps × laneNames：应用来自「已挂到该环境且在本次查询列表中」，
// 泳道来自 TrafficManager（并始终包含基线 ""）。某条 (app, env, lane) 没有记录是正常的，
// 真正有没有部署要等 prefetch 查库之后才知道。
type envStatusScope struct {
	env       *envmodel.Environment
	apps      []*app.Application // 已挂到该环境、且属于本次查询列表的应用
	laneNames []string           // 该环境下可能存在的泳道（含基线 ""），不代表都有部署
}

// resolveEnvStatusScope 解析单个环境的查询范围。
// 环境上没有任何待查应用时返回 nil，表示无需为该环境查部署记录（也不用问泳道）。
func (s *DeployStatusService) resolveEnvStatusScope(
	ctx context.Context,
	environment *envmodel.Environment,
	apps []*app.Application,
) (*envStatusScope, error) {
	// 仅环境的 AppIDs 中的应用才需要查部署状态。
	// 环境上没有任何待查应用时提前返回，避免无意义的泳道查询。
	envAppIDSet := set.From(environment.AppIDs)
	matchedApps := lo.Filter(apps, func(application *app.Application, _ int) bool {
		return envAppIDSet.Contains(application.ID)
	})
	if len(matchedApps) == 0 {
		return nil, nil
	}

	trafficLaneNames, err := s.listEnvTrafficLaneNames(ctx, environment)
	if err != nil {
		return nil, err
	}

	return &envStatusScope{env: environment, apps: matchedApps, laneNames: trafficLaneNames}, nil
}

// listForEnvironment 查询提供的 apps 在提供环境上的部署状态
func (s *DeployStatusService) listForEnvironment(
	ctx context.Context,
	environment *envmodel.Environment,
	apps []*app.Application,
) ([]AppDeployStatus, error) {
	scope, err := s.resolveEnvStatusScope(ctx, environment, apps)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return nil, nil
	}

	cache, err := s.prefetchLatestStatuses(ctx, []envStatusScope{*scope})
	if err != nil {
		return nil, err
	}
	return s.buildEnvDeployStatuses(scope, cache)
}

// buildEnvDeployStatuses 依据预取的部署状态缓存，构建某个环境上各应用、各泳道的部署状态
func (s *DeployStatusService) buildEnvDeployStatuses(
	scope *envStatusScope,
	cache latestStatusCache,
) ([]AppDeployStatus, error) {
	environment := scope.env
	statuses := make([]AppDeployStatus, 0, len(scope.apps))
	for _, application := range scope.apps {
		deployStatuses, err := buildDeployStatuses(
			environment.ID.Hex(),
			environment.Name,
			environment.DisplayName,
			environment.Type,
			string(environment.GetKind()),
			application.ID,
			application.Name,
			application.Type,
			scope.laneNames,
			cache,
		)
		if err != nil {
			return nil, errors.Wrap(err, "build deploy statuses")
		}
		statuses = append(statuses, deployStatuses...)
	}
	return statuses, nil
}

// listEnvTrafficLaneNames 列出环境下的泳道
func (s *DeployStatusService) listEnvTrafficLaneNames(
	ctx context.Context,
	environment *envmodel.Environment,
) ([]string, error) {
	// 获取环境下的泳道列表
	trafficLanes, err := s.TrafficManager.ListTrafficLanes(ctx, environment.WorkspaceID, environment.Name)
	if err != nil {
		return nil, err
	}

	// 默认包含空泳道，用于表示未指定泳道
	trafficLanes = append(trafficLanes, &trafficmanager.TrafficLane{LaneName: ""})
	return lo.UniqMap(trafficLanes, func(trafficLane *trafficmanager.TrafficLane, _ int) string {
		if trafficLane != nil {
			return trafficLane.LaneName
		}
		return ""
	}), nil
}

// buildDeployStatuses 根据提供的环境、应用、应用类型和泳道列表，构建部署状态
// 部署状态取自 cache，由调用方在进入循环前批量预取
// 如果没有找到部署状态，返回默认的未知状态
func buildDeployStatuses(
	envID, envName, envDisplayName, envType, envKind, appID, appName, appType string,
	laneNames []string,
	cache latestStatusCache,
) ([]AppDeployStatus, error) {
	statuses := make([]AppDeployStatus, 0, len(laneNames))

	// 遍历每个泳道，获取部署状态
	for _, laneName := range laneNames {
		deployStatus, ok := cache[latestStatusKey{laneName: laneName, appID: appID, envName: envName}]
		// 部署记录不存在，跳过该泳道
		if !ok {
			continue
		}
		statuses = append(statuses, AppDeployStatus{
			EnvID:           envID,
			EnvName:         envName,
			EnvDisplayName:  envDisplayName,
			EnvType:         envType,
			EnvKind:         envKind,
			AppName:         appName,
			AppID:           appID,
			AppType:         appType,
			TrafficLaneName: laneName,
			DeployStatus:    deployStatus.Status,
			ImageTag:        deployStatus.ImageTag,
		})
	}

	// 如果存在部署状态，则返回部署状态
	if len(statuses) > 0 {
		return statuses, nil
	}

	// 否则返回未知状态
	return []AppDeployStatus{{
		EnvID:           envID,
		EnvName:         envName,
		EnvDisplayName:  envDisplayName,
		EnvType:         envType,
		EnvKind:         envKind,
		AppName:         appName,
		AppID:           appID,
		AppType:         appType,
		TrafficLaneName: "",
		DeployStatus:    StatusUnknown,
	}}, nil
}

// latestStatusKey 定位一条最新部署状态：同一应用在不同环境、不同泳道上互相独立。
type latestStatusKey struct {
	laneName string
	appID    string
	envName  string
}

// latestStatusCache 缓存一批应用在各泳道、各环境下的最新部署状态。
// 查不到的键表示该应用在该环境、该泳道上没有部署记录。
type latestStatusCache map[latestStatusKey]*LatestDeployStatus

// prefetchLatestStatuses 一次性聚合各 scope 中应用在各泳道、各环境上的最新部署状态。
//
// 按应用类型分流：AppModel 类应用查构建部署与 AppModel 部署两张表，Helm 类应用查 Helm 部署表。
// 每个泳道最多三次聚合查询，往返次数与环境数、应用数无关。
func (s *DeployStatusService) prefetchLatestStatuses(
	ctx context.Context, scopes []envStatusScope,
) (latestStatusCache, error) {
	// 同一应用可能出现在多个环境上，按泳道汇总待查应用，交由存储层去重后批量聚合
	appModelIDsByLane := make(map[string][]string)
	helmIDsByLane := make(map[string][]string)
	for i := range scopes {
		for _, laneName := range scopes[i].laneNames {
			for _, application := range scopes[i].apps {
				switch {
				case app.IsAppModelType(application.Type):
					appModelIDsByLane[laneName] = append(appModelIDsByLane[laneName], application.ID)
				case app.IsHelmBasedType(application.Type):
					helmIDsByLane[laneName] = append(helmIDsByLane[laneName], application.ID)
				default:
					return nil, errors.Wrapf(
						ErrUnsupportedAppType, "app %s app type %s", application.ID, application.Type,
					)
				}
			}
		}
	}

	cache := make(latestStatusCache)

	for laneName, appIDs := range appModelIDsByLane {
		// AppModel 应用可能存在一键构建部署记录；优先根据记录关联关系和创建时间选择真正最新的状态。
		buildByApp, err := s.BuildAutoDeployRecordStore.ListLatestByApps(ctx, appIDs, laneName)
		if err != nil {
			return nil, errors.Wrapf(err, "list latest build auto deploy records for lane %s", laneName)
		}
		deployByApp, err := s.AppModelDeployRecordStore.ListLatestByApps(ctx, appIDs, laneName)
		if err != nil {
			return nil, errors.Wrapf(err, "list latest appmodel deploy records for lane %s", laneName)
		}
		for _, appID := range appIDs {
			for envName, status := range mergeAppModelStatuses(buildByApp[appID], deployByApp[appID]) {
				cache[latestStatusKey{laneName: laneName, appID: appID, envName: envName}] = status
			}
		}
	}

	for laneName, appIDs := range helmIDsByLane {
		// Helm 应用没有一键构建部署记录，直接读取 Helm 部署记录即可。
		recordsByApp, err := s.HelmDeployRecordStore.ListLatestByApps(ctx, appIDs, laneName)
		if err != nil {
			return nil, errors.Wrapf(err, "list latest helm deploy records for lane %s", laneName)
		}
		for appID, byEnv := range recordsByApp {
			for envName, record := range byEnv {
				cache[latestStatusKey{laneName: laneName, appID: appID, envName: envName}] = &LatestDeployStatus{
					Status:    string(record.Status),
					ImageTag:  record.ImageTag,
					StartedAt: record.StartedAt,
				}
			}
		}
	}

	return cache, nil
}

// mergeAppModelStatuses 按环境合并 AppModel 应用的一键构建部署记录与 AppModel 部署记录，
// 取两者中真正最新的状态。两类记录覆盖的环境取并集，某环境可能只有其中一类记录。
func mergeAppModelStatuses(
	buildByEnv map[string]*autodeploy.Record,
	deployByEnv map[string]*appmodel.Record,
) map[string]*LatestDeployStatus {
	envNames := make(map[string]struct{}, len(buildByEnv)+len(deployByEnv))
	for envName := range buildByEnv {
		envNames[envName] = struct{}{}
	}
	for envName := range deployByEnv {
		envNames[envName] = struct{}{}
	}

	statuses := make(map[string]*LatestDeployStatus, len(envNames))
	for envName := range envNames {
		if status := selectLatestStatus(buildByEnv[envName], deployByEnv[envName]); status != nil {
			statuses[envName] = status
		}
	}
	return statuses
}

// GetLatestDeployStatus 获取应用在指定环境、泳道下的最新部署状态。
//
// Args:
//   - appID 应用 ID
//   - appType 应用类型
//   - envName 环境名称
//   - laneName 泳道名称，空字符串表示默认（基线）泳道
//
// Returns:
//   - 最新部署状态；无记录时返回 ErrDeployRecordNotFound
//   - error
func (s *DeployStatusService) GetLatestDeployStatus(
	ctx context.Context,
	appID, appType, envName, laneName string,
) (*LatestDeployStatus, error) {
	switch {
	case app.IsAppModelType(appType):
		// AppModel 应用可能存在一键构建部署记录；优先根据记录关联关系和创建时间选择真正最新的状态。
		buildRecord, err := s.getLatestBuildAutoDeployRecord(ctx, appID, envName, laneName)
		if err != nil {
			return nil, err
		}
		deployRecord, err := s.getLatestAppModelDeployRecord(ctx, appID, envName, laneName)
		if err != nil {
			return nil, err
		}
		status := selectLatestStatus(buildRecord, deployRecord)
		if status == nil {
			return nil, ErrDeployRecordNotFound
		}
		return status, nil
	case app.IsHelmBasedType(appType):
		// Helm 应用没有一键构建部署记录，直接读取 Helm 部署记录即可。
		record, err := s.getLatestHelmDeployRecord(ctx, appID, envName, laneName)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, ErrDeployRecordNotFound
		}
		return &LatestDeployStatus{
			Status:    string(record.Status),
			ImageTag:  record.ImageTag,
			StartedAt: record.StartedAt,
		}, nil
	default:
		return nil, ErrUnsupportedAppType
	}
}

// ListLatestByAppLane 批量获取应用在指定泳道下各环境的最新部署状态与 AppModel 部署记录。
// 仅支持 AppModel 类型应用（trpc/taf）；通过两次批量查询完成，复杂度与环境数无关。
//
// Args:
//   - appID 应用 ID
//   - appType 应用类型
//   - laneName 泳道名称，空字符串表示默认（基线）泳道
//
// Returns:
//   - map[envName]*LatestDeployStatus，无部署记录的环境不出现在 map 中
//   - map[envName]*appmodel.Record，各环境最新 AppModel 部署记录（无记录的环境不出现）
//   - error
func (s *DeployStatusService) ListLatestByAppLane(
	ctx context.Context,
	appID, appType, laneName string,
) (map[string]*LatestDeployStatus, map[string]*appmodel.Record, error) {
	if !app.IsAppModelType(appType) {
		return nil, nil, ErrUnsupportedAppType
	}

	// 一次聚合拉齐该泳道下各环境最新的一键构建部署记录与 AppModel 部署记录。
	buildByEnv, err := s.BuildAutoDeployRecordStore.ListLatestByApp(ctx, appID, laneName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "list latest build auto deploy records")
	}
	deployByEnv, err := s.AppModelDeployRecordStore.ListLatestByApp(ctx, appID, laneName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "list latest appmodel deploy records")
	}

	// 按环境比较两类记录的关联关系与时间，选出真正最新的状态。
	return mergeAppModelStatuses(buildByEnv, deployByEnv), deployByEnv, nil
}

func (s *DeployStatusService) getLatestBuildAutoDeployRecord(
	ctx context.Context,
	appID, envName, laneName string,
) (*autodeploy.Record, error) {
	record, err := s.BuildAutoDeployRecordStore.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		if errors.Is(err, autodeploy.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest build auto deploy record")
	}
	return record, nil
}

func (s *DeployStatusService) getLatestAppModelDeployRecord(
	ctx context.Context,
	appID, envName, laneName string,
) (*appmodel.Record, error) {
	record, err := s.AppModelDeployRecordStore.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		if errors.Is(err, appmodel.ErrDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest appmodel deploy record")
	}
	return record, nil
}

func (s *DeployStatusService) getLatestHelmDeployRecord(
	ctx context.Context,
	appID, envName, laneName string,
) (*helm.Record, error) {
	record, err := s.HelmDeployRecordStore.GetLatest(ctx, appID, envName, laneName)
	if err != nil {
		if errors.Is(err, helm.ErrLatestDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest helm deploy record")
	}
	return record, nil
}

// selectLatestStatus 根据构建部署记录和应用模型部署记录，选择最新的部署状态
func selectLatestStatus(
	buildRecord *autodeploy.Record,
	deployRecord *appmodel.Record,
) *LatestDeployStatus {
	if buildRecord == nil && deployRecord == nil {
		return nil
	}
	if deployRecord == nil {
		return &LatestDeployStatus{
			Status:    buildRecord.Status,
			ImageTag:  buildRecord.ImageTag,
			StartedAt: buildRecord.StartedAt,
		}
	}
	if buildRecord == nil {
		return &LatestDeployStatus{
			Status:    string(deployRecord.Status),
			ImageTag:  deployRecord.ImageTag,
			StartedAt: deployRecord.StartedAt,
		}
	}
	// 如果构建部署记录的 DeployID 与应用模型部署记录的 ID 关联，说明该构建部署对应的就是这条应用模型部署，直接返回构建部署状态。
	if buildRecord.DeployID != "" && buildRecord.DeployID == deployRecord.ID.Hex() {
		return &LatestDeployStatus{
			Status:    buildRecord.Status,
			ImageTag:  buildRecord.ImageTag,
			StartedAt: buildRecord.StartedAt,
		}
	}
	// 否则根据创建时间选择最新的状态，避免构建部署记录和应用模型部署记录不一致时返回过时的状态。
	if buildRecord.CreatedAt.Before(deployRecord.CreatedAt) {
		return &LatestDeployStatus{
			Status:    string(deployRecord.Status),
			ImageTag:  deployRecord.ImageTag,
			StartedAt: deployRecord.StartedAt,
		}
	}
	return &LatestDeployStatus{
		Status:    buildRecord.Status,
		ImageTag:  buildRecord.ImageTag,
		StartedAt: buildRecord.StartedAt,
	}
}
