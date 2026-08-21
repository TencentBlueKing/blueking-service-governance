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

package overview

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	workloadappmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// defaultTrafficLaneName 默认（基线）泳道名称；空字符串表示基线泳道。
// 部署总览只统计该泳道，不展示非基线泳道的部署状态与实例。
const defaultTrafficLaneName = ""

// maxConcurrentK8sRequests 单次总览请求内在途 K8s 请求数上限，防止对集群 API Server
// 与 BCS 网关造成过大压力。
const maxConcurrentK8sRequests = 10

// Service 组装应用部署总览。
type Service struct {
	envStore            envmodel.EnvironmentStore
	gpaConfigStore      gpa.GPAConfigStore
	gpaService          *gpa.GPAService
	deployStatusService *deploystatus.DeployStatusService
}

// NewService 创建部署总览 Service。
func NewService(
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	appModelStore workloadappmodel.AppModelStore,
	buildAutoDeployRecordStore autodeploy.RecordStore,
	appModelDeployRecordStore appmodel.RecordStore,
	// helmDeployRecordStore 总览只处理 AppModel 类型应用，用不到 helm 记录，
	// 但它是 deploystatus.NewDeployStatusService 的必需依赖，需原样透传。
	helmDeployRecordStore helm.RecordStore,
	gpaConfigStore gpa.GPAConfigStore,
) *Service {
	return &Service{
		envStore:       envStore,
		gpaConfigStore: gpaConfigStore,
		gpaService:     gpa.NewGPAService(appModelStore),
		deployStatusService: deploystatus.NewDeployStatusService(
			appStore,
			envStore,
			buildAutoDeployRecordStore,
			appModelDeployRecordStore,
			helmDeployRecordStore,
		),
	}
}

// envRowSources 组装表格行所需的全部数据源，均由批量读库一次性取齐。
type envRowSources struct {
	// trackedEnvs 应出现在表格中的环境
	trackedEnvs []envmodel.Environment
	// autoscalingByEnv envName -> GPA 配置摘要；无配置的环境不出现
	autoscalingByEnv map[string]*AutoscalingInfo
	// statusesByEnv envName -> 最新部署状态；无部署记录的环境不出现
	statusesByEnv map[string]*deploystatus.LatestDeployStatus
	// deployByEnv envName -> 最新 AppModel 部署记录；无记录的环境不出现
	deployByEnv map[string]*appmodel.Record
}

// GetOverview 查询并组装 trpc/taf 应用的部署总览。
//
// 行集合与 env.AppIDs 对齐；仅统计默认（基线）泳道。
//
// Args:
//   - application 目标应用，须为 AppModel 类型（trpc/taf）
//
// Returns:
//   - 部署总览结果
//   - error
func (s *Service) GetOverview(ctx context.Context, application *bkmsapp.Application) (*Result, error) {
	if !bkmsapp.IsAppModelType(application.Type) {
		return nil, errors.Errorf("unsupported app type: %s", application.Type)
	}

	sources, err := s.loadEnvRowSources(ctx, application)
	if err != nil {
		return nil, err
	}

	rows, recordsForInstances := assembleEnvRows(sources)

	// 实例数/资源规格、GPA 状态、BCS 集群名互不依赖，并行回查；任一侧失败只降级对应字段。
	// 各侧只读 rows、各自产出 map，待 Wait 后再单线程合并，避免并发写同一批行。
	// K8s 两侧共用一个闸门，使在途 K8s 请求总数受 maxConcurrentK8sRequests 约束。
	sem := semaphore.NewWeighted(maxConcurrentK8sRequests)
	var (
		clusterData         envClusterDataByEnv
		autoscalingStatuses autoscalingStatusByEnv
		clusterNames        clusterNameByID
		wg                  sync.WaitGroup
	)
	wg.Go(func() {
		clusterData = queryEnvClusterData(ctx, sem, recordsForInstances)
	})
	wg.Go(func() {
		autoscalingStatuses = s.queryAutoscalingStatuses(ctx, sem, sources.trackedEnvs, rows)
	})
	wg.Go(func() {
		clusterNames = queryClusterNames(ctx, rows)
	})
	wg.Wait()

	for i := range rows {
		if data, ok := clusterData[rows[i].EnvName]; ok {
			rows[i].Instances = data.Instances
			rows[i].Resources = data.Resources
		}
		if status, ok := autoscalingStatuses[rows[i].EnvName]; ok && rows[i].Autoscaling != nil {
			rows[i].Autoscaling.Status = status
		}
		if name, ok := clusterNames[rows[i].Cluster.ClusterID]; ok {
			rows[i].Cluster.Name = name
		}
	}

	return &Result{Envs: rows}, nil
}

// loadEnvRowSources 批量读取组装表格行所需的各数据源。
//
// 三组查询互不依赖，并发发起以省去逐个叠加的 DB 往返；任一失败即整体失败，
// 因为缺任何一组都无法给出完整的总览。
func (s *Service) loadEnvRowSources(
	ctx context.Context,
	application *bkmsapp.Application,
) (*envRowSources, error) {
	var (
		trackedEnvs      []envmodel.Environment
		autoscalingByEnv map[string]*AutoscalingInfo
		statusesByEnv    map[string]*deploystatus.LatestDeployStatus
		deployByEnv      map[string]*appmodel.Record
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		trackedEnvs, err = s.listTrackedEnvs(gctx, application)
		return err
	})
	g.Go(func() error {
		var err error
		autoscalingByEnv, err = s.listAutoscalingConfigsByEnv(gctx, application.ID)
		return err
	})
	g.Go(func() error {
		// 批量结果可能含已不在 AppIDs 中的历史环境，assembleEnvRows 只按 trackedEnvs 取用。
		var err error
		statusesByEnv, deployByEnv, err = s.deployStatusService.ListLatestByAppLane(
			gctx, application.ID, application.Type, defaultTrafficLaneName,
		)
		if err != nil {
			return errors.Wrap(err, "list latest deploy statuses")
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &envRowSources{
		trackedEnvs:      trackedEnvs,
		autoscalingByEnv: autoscalingByEnv,
		statusesByEnv:    statusesByEnv,
		deployByEnv:      deployByEnv,
	}, nil
}

// listTrackedEnvs 返回应出现在总览表格中的环境（AppIDs 含该应用）。
func (s *Service) listTrackedEnvs(
	ctx context.Context,
	application *bkmsapp.Application,
) ([]envmodel.Environment, error) {
	envs, err := s.envStore.ListBatchAppEnvs(ctx, application.WorkspaceID, []string{application.ID})
	if err != nil {
		return nil, errors.Wrap(err, "list app environments")
	}
	return lo.Filter(envs, func(env envmodel.Environment, _ int) bool {
		return lo.Contains(env.AppIDs, application.ID)
	}), nil
}

// assembleEnvRows 在内存中组装表格行，并收集有 AppModel 部署记录的环境供后续查 K8s。
//
// Resources / Instances 稍后从集群 GameDeployment 回填；此处保持零值。
// 部署状态 / 部署记录 map 可能含非 tracked 环境，此处只按 trackedEnvs 取用。
func assembleEnvRows(sources *envRowSources) ([]EnvRow, []deployRecordForEnv) {
	rows := make([]EnvRow, 0, len(sources.trackedEnvs))
	records := make([]deployRecordForEnv, 0, len(sources.trackedEnvs))

	for i := range sources.trackedEnvs {
		env := &sources.trackedEnvs[i]
		row := EnvRow{
			EnvID:          env.ID.Hex(),
			EnvName:        env.Name,
			EnvDisplayName: env.DisplayName,
			EnvType:        env.Type,
			EnvKind:        string(env.GetKind()),
			DeployStatus:   deploystatus.StatusUnknown,
			Cluster: ClusterInfo{
				ProjectCode: env.Cluster.ProjectCode,
				ClusterID:   env.Cluster.ClusterID,
				ClusterType: env.Cluster.ClusterType,
				Namespace:   env.Cluster.Namespace,
			},
			Autoscaling: sources.autoscalingByEnv[env.Name],
		}
		if latest := sources.statusesByEnv[env.Name]; latest != nil {
			row.DeployStatus = latest.Status
			row.ImageTag = latest.ImageTag
			if !latest.StartedAt.IsZero() {
				row.LastDeployStartedAt = lo.ToPtr(latest.StartedAt)
			}
		}
		if rec := sources.deployByEnv[env.Name]; rec != nil {
			records = append(records, deployRecordForEnv{EnvName: env.Name, Record: rec})
		}
		rows = append(rows, row)
	}
	return rows, records
}
