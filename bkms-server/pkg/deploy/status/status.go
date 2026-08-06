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

type latestDeployStatus struct {
	Status   string
	ImageTag string
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
	stdEnvs := make([]envmodel.Environment, 0)
	featEnvs := make([]envmodel.Environment, 0)

	envs, err := s.EnvStore.ListBatchAppEnvs(ctx, workspaceID, appIDs)
	if err != nil {
		return nil, errors.Wrap(err, "list batch app environments")
	}

	// 将环境区分为两类：标准和特性
	for i := range envs {
		env := envs[i]
		if env.IsFeatureEnv() {
			featEnvs = append(featEnvs, env)
		} else {
			stdEnvs = append(stdEnvs, env)
		}
	}

	for i := range stdEnvs {
		deployStatuses, err := s.listForEnvironment(ctx, &stdEnvs[i], apps)
		if err != nil {
			return nil, errors.Wrapf(err, "list deploy statuses for standard environment %s", stdEnvs[i].Name)
		}
		for _, deployStatus := range deployStatuses {
			out[deployStatus.AppID] = append(out[deployStatus.AppID], deployStatus)
		}
	}

	for i := range featEnvs {
		ownerApp := appMap[featEnvs[i].OwnerAppID]
		deployStatuses, err := s.listForEnvironment(ctx, &featEnvs[i], []*app.Application{ownerApp})
		if err != nil {
			return nil, errors.Wrapf(err, "list deploy statuses for feature environment %s", featEnvs[i].Name)
		}
		for _, deployStatus := range deployStatuses {
			out[deployStatus.AppID] = append(out[deployStatus.AppID], deployStatus)
		}
	}

	return out, nil
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

// listForEnvironment 查询提供的 apps 在提供环境上的部署状态
func (s *DeployStatusService) listForEnvironment(
	ctx context.Context,
	environment *envmodel.Environment,
	apps []*app.Application,
) ([]AppDeployStatus, error) {
	// 仅环境的 AppIDs 中的应用 ID 才需要进一步查询部署状态，首先通过其过滤给定的 apps 列表。
	// 当查询无任何已部署应用的环境/特性环境时，提前使用这一层过滤可以避免之后再继查询 traffic lane，优化性能。
	envAppIDSet := set.From(environment.AppIDs)
	matchedApps := lo.Filter(apps, func(application *app.Application, _ int) bool {
		return envAppIDSet.Contains(application.ID)
	})
	if len(matchedApps) == 0 {
		return nil, nil
	}

	// 查询泳道
	trafficLaneNames, err := s.listEnvTrafficLaneNames(ctx, environment)
	if err != nil {
		return nil, err
	}

	statuses := make([]AppDeployStatus, 0, len(matchedApps))
	for _, application := range matchedApps {
		deployStatuses, err := s.buildDeployStatuses(
			ctx,
			environment.ID.Hex(),
			environment.Name,
			environment.DisplayName,
			environment.Type,
			string(environment.GetKind()),
			application.ID,
			application.Name,
			application.Type,
			trafficLaneNames,
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
// 如果没有找到部署状态，返回默认的未知状态
func (s *DeployStatusService) buildDeployStatuses(
	ctx context.Context,
	envID, envName, envDisplayName, envType, envKind, appID, appName, appType string,
	laneNames []string,
) ([]AppDeployStatus, error) {
	statuses := make([]AppDeployStatus, 0, len(laneNames))

	// 遍历每个泳道，获取部署状态
	for _, laneName := range laneNames {
		deployStatus, err := s.getLatestDeployStatus(ctx, appID, appType, envName, laneName)
		if err != nil {
			// 如果部署记录不存在，可以忽略错误，继续下一个泳道
			if errors.Is(err, ErrDeployRecordNotFound) {
				continue
			}
			return nil, errors.Wrapf(
				err,
				"get latest deploy status for app %s app type %s env %s lane %s",
				appID,
				appType,
				envName,
				laneName,
			)
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

func (s *DeployStatusService) getLatestDeployStatus(
	ctx context.Context,
	appID, appType, envName, laneName string,
) (*latestDeployStatus, error) {
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
		return &latestDeployStatus{Status: string(record.Status), ImageTag: record.ImageTag}, nil
	default:
		return nil, ErrUnsupportedAppType
	}
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
) *latestDeployStatus {
	if buildRecord == nil && deployRecord == nil {
		return nil
	}
	if deployRecord == nil {
		return &latestDeployStatus{Status: buildRecord.Status, ImageTag: buildRecord.ImageTag}
	}
	if buildRecord == nil {
		return &latestDeployStatus{Status: string(deployRecord.Status), ImageTag: deployRecord.ImageTag}
	}
	// 如果构建部署记录的 DeployID 与应用模型部署记录的 ID 关联，说明该构建部署对应的就是这条应用模型部署，直接返回构建部署状态。
	if buildRecord.DeployID != "" && buildRecord.DeployID == deployRecord.ID.Hex() {
		return &latestDeployStatus{Status: buildRecord.Status, ImageTag: buildRecord.ImageTag}
	}
	// 否则根据创建时间选择最新的状态，避免构建部署记录和应用模型部署记录不一致时返回过时的状态。
	if buildRecord.CreatedAt.Before(deployRecord.CreatedAt) {
		return &latestDeployStatus{Status: string(deployRecord.Status), ImageTag: deployRecord.ImageTag}
	}
	return &latestDeployStatus{Status: buildRecord.Status, ImageTag: buildRecord.ImageTag}
}
