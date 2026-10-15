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
	"log/slog"
	"maps"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

// autoscalingStatusByEnv envName -> GPA 运行状态；缺 key 表示未启用 GPA 或回查失败。
type autoscalingStatusByEnv map[string]*AutoscalingStatus

// listAutoscalingConfigsByEnv 返回各环境 GPA 配置摘要；无配置的环境不出现在 map 中。
func (s *Service) listAutoscalingConfigsByEnv(
	ctx context.Context,
	appID string,
) (map[string]*AutoscalingInfo, error) {
	out := map[string]*AutoscalingInfo{}
	configs, err := s.gpaConfigs.List(ctx, gpa.ByApp(appID))
	if err != nil {
		return nil, errors.Wrap(err, "list gpa configs")
	}
	for _, cfg := range configs {
		out[cfg.EnvName] = &AutoscalingInfo{
			Enabled:     cfg.Enabled,
			CRName:      cfg.Name,
			MinReplicas: cfg.MinReplicas,
			MaxReplicas: cfg.MaxReplicas,
			Metrics: lo.Map(cfg.Metrics, func(m gpa.GPAMetric, _ int) AutoscalingMetric {
				return AutoscalingMetric{
					Resource:           string(m.Resource),
					AverageUtilization: m.AverageUtilization,
				}
			}),
			ComputeByLimits: cfg.ComputeByLimits,
		}
	}
	return out, nil
}

// autoscalingTarget 单个环境待回查的 GPA CR 位置。
type autoscalingTarget struct {
	envName   string
	namespace string
	crName    string
}

// autoscalingTargetsByCluster clusterID -> 该集群上需要一并回查的 GPA CR。
type autoscalingTargetsByCluster map[string][]autoscalingTarget

// queryAutoscalingStatuses 为已启用 GPA 的环境按集群并发回查集群 CR 状态。
//
// rows 只用于读取待查环境与 CR 名，不写回；回查失败的环境不出现在结果中，
// 不使整次总览失败（与 instances 降级策略一致）。
//
// Args:
//   - sem 本次请求内所有集群回查共享的在途请求闸门
//
// Returns:
//   - envName -> GPA 运行状态
func (s *Service) queryAutoscalingStatuses(
	ctx context.Context,
	sem *semaphore.Weighted,
	trackedEnvs []envmodel.Environment,
	rows []EnvRow,
) autoscalingStatusByEnv {
	out := make(autoscalingStatusByEnv, len(rows))

	byCluster := groupAutoscalingTargetsByCluster(trackedEnvs, rows)
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for clusterID, targets := range byCluster {
		wg.Go(func() {
			statuses := s.queryAutoscalingStatusesForCluster(ctx, sem, clusterID, targets)
			mu.Lock()
			defer mu.Unlock()
			maps.Copy(out, statuses)
		})
	}
	wg.Wait()
	return out
}

// groupAutoscalingTargetsByCluster 按集群归拢待回查的 GPA CR。
// 未启用 GPA、缺 CR 名、或环境缺集群信息的行直接跳过。
func groupAutoscalingTargetsByCluster(
	trackedEnvs []envmodel.Environment,
	rows []EnvRow,
) autoscalingTargetsByCluster {
	envByName := lo.KeyBy(trackedEnvs, func(env envmodel.Environment) string { return env.Name })

	byCluster := autoscalingTargetsByCluster{}
	for i := range rows {
		info := rows[i].Autoscaling
		if info == nil || !info.Enabled || info.CRName == "" {
			continue
		}
		env, ok := envByName[rows[i].EnvName]
		if !ok || env.Cluster.ClusterID == "" {
			continue
		}
		byCluster[env.Cluster.ClusterID] = append(byCluster[env.Cluster.ClusterID], autoscalingTarget{
			envName:   rows[i].EnvName,
			namespace: env.Cluster.Namespace,
			crName:    info.CRName,
		})
	}
	return byCluster
}

// queryAutoscalingStatusesForCluster 回查单集群上各环境的 GPA CR 状态。
//
// 客户端按集群创建一次（含一次 GVR 解析与 Redis 缓存查询），供该集群下各环境复用；
// 创建失败时该集群整批降级为「状态不可用」，不影响其它集群。
//
// Returns:
//   - envName -> GPA 运行状态；失败环境不写入
func (s *Service) queryAutoscalingStatusesForCluster(
	ctx context.Context,
	sem *semaphore.Weighted,
	clusterID string,
	targets []autoscalingTarget,
) autoscalingStatusByEnv {
	out := make(autoscalingStatusByEnv, len(targets))
	clusterClient, err := s.gpaService.NewClusterClient(clusterID)
	if err != nil {
		// 集群未安装 GPA 组件属预期状态（配置可能建于组件卸载之前），不计入错误日志
		if errors.Is(err, gpa.ErrComponentNotInstalled) {
			log.WarnAttrs(ctx, "deploy overview skips gpa status, component not installed in cluster",
				slog.String("cluster_id", clusterID),
			)
			return out
		}
		log.ErrorAttrs(ctx, "create deploy overview gpa cluster client failed",
			slog.String("cluster_id", clusterID),
			slog.Any("error", err),
		)
		return out
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, target := range targets {
		wg.Go(func() {
			status := queryAutoscalingStatusForTarget(ctx, sem, clusterClient, clusterID, target)
			if status == nil {
				return
			}
			mu.Lock()
			out[target.envName] = status
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

// queryAutoscalingStatusForTarget 回查单个环境的 GPA CR 状态。
// CR 不存在、集群不可达时返回 nil，由调用方降级为「状态不可用」。
func queryAutoscalingStatusForTarget(
	ctx context.Context,
	sem *semaphore.Weighted,
	client *gpa.ClusterClient,
	clusterID string,
	target autoscalingTarget,
) *AutoscalingStatus {
	// 仅在 ctx 结束（客户端断连 / 请求超时）时失败，此时响应已无人接收，记日志便于与 CR 缺失区分
	if err := sem.Acquire(ctx, 1); err != nil {
		log.WarnAttrs(ctx, "deploy overview gives up gpa status query",
			slog.String("env_name", target.envName),
			slog.String("cluster_id", clusterID),
			slog.Any("error", err),
		)
		return nil
	}
	defer sem.Release(1)

	status, err := client.GetStatus(ctx, target.namespace, target.crName)
	if err != nil {
		if !errors.Is(err, gpa.ErrCRNotFound) {
			log.ErrorAttrs(ctx, "query deploy overview gpa status failed",
				slog.String("env_name", target.envName),
				slog.String("cluster_id", clusterID),
				slog.String("gpa_name", target.crName),
				slog.Any("error", err),
			)
		}
		return nil
	}
	return &AutoscalingStatus{
		CurrentReplicas: status.CurrentReplicas,
		DesiredReplicas: status.DesiredReplicas,
		LastScaleTime:   status.LastScaleTime,
		Phase:           status.Phase,
		StatusMessage:   status.StatusMessage,
	}
}
