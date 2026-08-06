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

package topology

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8smanifest "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/manifest"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Service 拓扑服务，对外提供拓扑图查询能力
type Service struct {
	store             ResourceSnapshotStore
	builder           *Builder
	appModelStore     appmodel.AppModelStore
	scopedEnvVarStore envvars.ScopedEnvVarStore
	appDepsVarReader  *depenvvars.Reader
	polarisVarReader  *polarisenvvars.Reader
}

// NewService 创建 Service 实例
func NewService(
	store ResourceSnapshotStore,
	builder *Builder,
	appModelStore appmodel.AppModelStore,
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appDepsVarReader *depenvvars.Reader,
	polarisVarReader *polarisenvvars.Reader,
) *Service {
	return &Service{
		store:             store,
		builder:           builder,
		appModelStore:     appModelStore,
		scopedEnvVarStore: scopedEnvVarStore,
		appDepsVarReader:  appDepsVarReader,
		polarisVarReader:  polarisVarReader,
	}
}

// GetTopology 获取指定应用环境的资源拓扑图
// 流程：从 Store 读取 ResourceSnapshot → 调用 Builder 构建拓扑 → 处理无数据场景
func (s *Service) GetTopology(ctx context.Context, appID, envName, trafficLaneName string) (*Graph, error) {
	snapshot, err := s.store.Get(ctx, appID, envName, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "get resource snapshot for %s/%s/%s", appID, envName, trafficLaneName)
	}

	// 无数据场景 — 返回空拓扑 + partial=true
	if snapshot == nil {
		log.Infof(
			ctx, "topology service: no resource snapshot found for %s/%s/%s, returning empty topology",
			appID, envName, trafficLaneName,
		)
		return &Graph{
			Metadata: Metadata{
				AppID:           appID,
				EnvName:         envName,
				TrafficLaneName: trafficLaneName,
			},
			Nodes:       []Node{},
			Edges:       []Edge{},
			RootID:      "",
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			IsPartial:   true,
			Warnings:    []string{"no resource snapshot data found, please deploy first"},
			DataVersion: 0,
		}, nil
	}

	// 如果刷新状态为 failed，在 warnings 中提示
	graph, err := s.builder.Build(ctx, snapshot)
	if err != nil {
		return nil, errors.Wrap(err, "build topology graph")
	}

	if snapshot.RefreshStatus == RefreshStatusFailed {
		graph.IsPartial = true
		graph.Warnings = append(graph.Warnings, fmt.Sprintf(
			"resource snapshot data may be stale (last refresh failed: %s)", snapshot.WarningSummary,
		))
	}

	return graph, nil
}

// GetNodeDetail 获取指定节点的结构化详情
// 流程：获取 scope → 解码 nodeID → 范围校验 → 集群 Get → 构建 NodeDetail（extras + conditions + createdAt）
func (s *Service) GetNodeDetail(
	ctx context.Context, appID, envName, trafficLaneName, nodeID string,
) (*NodeDetail, error) {
	_, clusterCfg, kind, ns, name, err := s.resolveAndValidateNode(ctx, appID, envName, trafficLaneName, nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve node %s", nodeID)
	}

	// 从集群获取资源对象
	obj, err := s.getClusterObject(ctx, clusterCfg, kind, ns, name)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s/%s/%s from cluster", kind, ns, name)
	}

	// 构建 NodeDetail
	detail := &NodeDetail{
		ID:        nodeID,
		Kind:      kind,
		Namespace: ns,
		Name:      name,
		CreatedAt: obj.GetCreationTimestamp().UTC().Format(time.RFC3339),
	}

	// 提取类型专属 extras
	if provider, ok := kindExtrasProviders[kind]; ok {
		detail.Extras = provider(obj)
	}

	// 提取 conditions
	detail.Conditions = ExtractConditions(obj)

	return detail, nil
}

// ListNodeEvents 获取指定节点的事件列表
// 流程：获取 snapshot → 解码 nodeID → 范围校验 → 调用 clusterresources API 获取事件
func (s *Service) ListNodeEvents(
	ctx context.Context,
	appID, envName, trafficLaneName string,
	projectCode, nodeID, level string,
	startedAt, endedAt int64,
	page, pageSize int64,
) (*clusterresources.PaginatedEvents, error) {
	snapshot, _, kind, ns, name, err := s.resolveAndValidateNode(ctx, appID, envName, trafficLaneName, nodeID)
	if err != nil {
		return nil, err
	}

	// 创建 clusterresources 用户态客户端
	client, err := clusterresources.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create cluster resources client")
	}

	// 调用 clusterresources API 获取事件（服务端分页）
	paginatedEvents, err := client.ListEvents(
		ctx,
		projectCode,
		snapshot.ClusterID,
		clusterresources.ListEventParams{
			Namespace:     ns,
			ResourceKinds: []string{kind},
			ResourceNames: []string{name},
			Level:         level,
			StartedAt:     startedAt,
			EndedAt:       endedAt,
			Page:          int(page),
			PageSize:      int(pageSize),
		},
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list events for %s/%s/%s", kind, ns, name)
	}

	return paginatedEvents, nil
}

// GetNodeManifest 获取指定节点的 YAML Manifest
// 流程：获取 scope → 解码 nodeID → 范围校验 → 集群 Get → 敏感字段脱敏 → BuildNodeManifest
func (s *Service) GetNodeManifest(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName, nodeID string,
) (*NodeManifest, error) {
	_, clusterCfg, kind, ns, name, err := s.resolveAndValidateNode(ctx, app.ID, env.Name, trafficLaneName, nodeID)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve node %s", nodeID)
	}

	// 从集群获取资源对象
	obj, err := s.getClusterObject(ctx, clusterCfg, kind, ns, name)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s/%s/%s from cluster", kind, ns, name)
	}

	manifestObj := obj.DeepCopy()
	if err = s.maskManifestSensitiveValues(ctx, app, env, manifestObj); err != nil {
		return nil, errors.Wrap(err, "mask sensitive values in node manifest")
	}

	manifest, err := BuildNodeManifest(manifestObj)
	if err != nil {
		return nil, errors.Wrapf(err, "build node manifest for %s/%s/%s", kind, ns, name)
	}
	return manifest, nil
}

// ======================== 公共辅助方法 ========================

// maskManifestSensitiveValues 对 manifest 中的敏感字段进行替换
func (s *Service) maskManifestSensitiveValues(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	obj *unstructured.Unstructured,
) error {
	// 非 AppModel 类型的应用没有应用环境变量上下文，但仍需执行 Secret 等资源级脱敏
	if !bkmsapp.IsAppModelType(app.Type) {
		k8smanifest.NewMasker(nil, envvartypes.SensitiveValueMask).Mask(obj)
		return nil
	}
	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return errors.Wrap(err, "get app model")
	}

	appEnvVars, err := envvars.BuildAppEnvVars(
		ctx, app, appModel, env,
		envvars.NewUnifiedEnvVarsReader(s.scopedEnvVarStore, s.appDepsVarReader, s.polarisVarReader),
	)
	if err != nil {
		return errors.Wrap(err, "build app env vars")
	}

	// 统计敏感环境变量
	sensitiveEnvVarValues := make(map[string]string)
	for _, item := range appEnvVars {
		if !item.IsSensitive {
			continue
		}
		sensitiveEnvVarValues[item.Key] = item.Value
	}

	// 执行敏感字段替换
	k8smanifest.NewMasker(sensitiveEnvVarValues, envvartypes.SensitiveValueMask).Mask(obj)
	return nil
}

// resolveAndValidateNode 公共流程：获取 snapshot → 解码 nodeID → 创建集群配置 → 范围校验
// 返回 snapshot、集群配置、kind/ns/name（已解码并校验通过）
func (s *Service) resolveAndValidateNode(
	ctx context.Context, appID, envName, trafficLaneName, nodeID string,
) (*ResourceSnapshot, *cluster.Config, string, string, string, error) {
	snapshot, err := s.store.Get(ctx, appID, envName, trafficLaneName)
	if err != nil {
		return nil, nil, "", "", "", errors.Wrapf(
			err,
			"get resource snapshot for %s/%s/%s",
			appID,
			envName,
			trafficLaneName,
		)
	}
	if snapshot == nil {
		return nil, nil, "", "", "", errors.Errorf(
			"no resource snapshot found for %s/%s/%s",
			appID,
			envName,
			trafficLaneName,
		)
	}

	// 解码 nodeID
	kind, ns, name, err := DecodeNodeID(nodeID)
	if err != nil {
		return nil, nil, "", "", "", errors.Wrap(err, "invalid node ID")
	}

	// 创建集群配置并执行范围校验
	clusterCfg := cluster.NewConfig(snapshot.ClusterID)
	validator := NewScopeValidator(snapshot, clusterCfg)
	if err = validator.Validate(ctx, kind, ns, name); err != nil {
		return nil, nil, "", "", "", err
	}

	return snapshot, clusterCfg, kind, ns, name, nil
}

// getClusterObject 从集群获取指定资源的非结构化对象
func (s *Service) getClusterObject(
	ctx context.Context, clusterCfg *cluster.Config, kind, namespace, name string,
) (*unstructured.Unstructured, error) {
	resGVR, err := discovery.GetGroupVersionResource(clusterCfg, kind, "")
	if err != nil {
		return nil, errors.Wrapf(err, "resolve GVR for %s", kind)
	}
	cli := k8sclient.NewWithGVR(clusterCfg, *resGVR)
	obj, err := cli.Get(ctx, namespace, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "get %s/%s/%s from cluster", kind, namespace, name)
	}
	return obj, nil
}
