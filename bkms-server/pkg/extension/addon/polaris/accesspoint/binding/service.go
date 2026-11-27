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

// Package binding 管理集群与北极星接入点的绑定，提供绑定变更、集群绑定视图查询，以及环境切换集群前的校验。
//
// 候选集群为空间 BCS 项目下的全部非共享集群，不按集群状态过滤，包含没有环境的集群；
// 集群上有环境（含特性环境）被北极星配置使用时，集群不能解绑或换绑。
package binding

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Service 管理集群与北极星接入点的绑定。
//
// 涉及的模型：polaris_cluster_bindings、polaris_access_points、environments、applications、polaris_configs
type Service struct {
	db           *mongo.Database
	accessPoints *accesspoint.AccessPointStore
	bindings     *accesspoint.BindingStore
	envs         *envmodel.EnvironmentStore
	resolver     accesspoint.Resolver
}

// NewService 创建 Service
func NewService(db *mongo.Database) *Service {
	return &Service{
		db:           db,
		accessPoints: store.Of(db, accesspoint.AccessPointMeta),
		bindings:     store.Of(db, accesspoint.BindingMeta),
		envs:         store.Of(db, envmodel.EnvMeta),
		resolver:     accesspoint.NewResolver(db),
	}
}

// ClusterView 集群绑定视图中的一个候选集群
type ClusterView struct {
	// Cluster BCS 集群
	Cluster bcs.Cluster
	// Envs 集群上的空间环境（含特性环境），按名称排序
	Envs []envmodel.Environment
	// AccessPoint 集群绑定的接入点，未绑定时为 nil
	AccessPoint *accesspoint.PolarisAccessPoint
	// InUse 集群上是否有环境被北极星配置使用，为 true 时不能解绑或换绑
	InUse bool
}

// ReplaceClusters 将接入点绑定的集群全量替换为 clusterIDs，clusterIDs 为空表示解除全部绑定；
// 返回替换前后绑定的集群 ID，均按集群 ID 升序。接入点不存在或不属于该空间时返回 ErrAccessPointNotFound。
//
// 全部校验通过后才写库：
//   - clusterIDs 必须是空间 BCS 项目下的非共享集群，否则返回 ErrAccessPointInvalid；
//   - 待解绑或从其他接入点换绑的集群上，若有环境正被北极星配置使用，返回 *accesspoint.ClusterInUseError
func (s *Service) ReplaceClusters(
	ctx context.Context, ws *bkmsworkspace.Workspace, accessPointID string, clusterIDs []string,
) (before, after []string, err error) {
	p, err := s.getAccessPoint(ctx, ws.ID, accessPointID)
	if err != nil {
		return nil, nil, err
	}
	after = slices.Compact(slices.Sorted(slices.Values(clusterIDs)))
	if _, err = s.candidateClusters(ctx, ws, after); err != nil {
		return nil, nil, err
	}

	// 绑定保存的是小写 hex，请求中的 id 可能是大写
	current, err := s.bindings.List(ctx, accesspoint.ByAccessPoint(ws.ID, p.ID.Hex()), store.Asc("clusterID"))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "list clusters bound to polaris access point %s", accessPointID)
	}
	before = getClusterIDsOf(current)
	bind, unbind := lo.Without(after, before...), lo.Without(before, after...)

	// bind 已排除本接入点原有的集群；若仍能查到绑定记录，说明原属其他接入点，本次属于换绑
	rebound := []string{}
	if len(bind) > 0 {
		var others []accesspoint.PolarisClusterBinding
		if others, err = s.bindings.List(ctx, accesspoint.ByClusters(ws.ID, bind)); err != nil {
			return nil, nil, errors.Wrapf(err, "list polaris bindings of clusters %v", bind)
		}
		rebound = getClusterIDsOf(others)
	}
	usages, err := s.listClusterUsages(ctx, ws.ID, slices.Concat(unbind, rebound))
	if err != nil {
		return nil, nil, err
	}
	if len(usages) > 0 {
		return nil, nil, &accesspoint.ClusterInUseError{Usages: usages}
	}

	if err = s.ensureBindings(ctx, ws.ID, p.ID.Hex(), bind, unbind); err != nil {
		return nil, nil, err
	}
	return before, after, nil
}

// BindCluster 将单个集群绑定到接入点 accessPointID，已绑定其他接入点时换绑；accessPointID 为空表示解绑。
// 已是目标状态时不写库。返回变更前后集群绑定的接入点 ID，未绑定时为空。
//
// 校验规则：
//   - 集群必须是空间 BCS 项目下的非共享集群，否则返回 ErrAccessPointInvalid；
//   - 接入点不存在或不属于该空间时返回 ErrAccessPointNotFound；
//   - 换绑或解绑时，若集群上有环境正被北极星配置使用，返回 *accesspoint.ClusterInUseError
func (s *Service) BindCluster(
	ctx context.Context, ws *bkmsworkspace.Workspace, clusterID, accessPointID string,
) (before, after string, err error) {
	if _, err = s.candidateClusters(ctx, ws, []string{clusterID}); err != nil {
		return "", "", err
	}
	// 没有指定 accessPointID 时，表示解绑；否则表示绑定到指定接入点。
	// 绑定保存的是小写 hex，请求中的 id 可能是大写
	if accessPointID != "" {
		var p *accesspoint.PolarisAccessPoint
		if p, err = s.getAccessPoint(ctx, ws.ID, accessPointID); err != nil {
			return "", "", err
		}
		after = p.ID.Hex()
	}
	current, err := s.bindings.List(ctx, accesspoint.ByClusters(ws.ID, []string{clusterID}))
	if err != nil {
		return "", "", errors.Wrapf(err, "list polaris bindings of cluster %s", clusterID)
	}
	if len(current) > 0 {
		before = current[0].AccessPointID
	}
	if before == after {
		return before, after, nil
	}

	if before != "" {
		var usages []accesspoint.ClusterUsage
		if usages, err = s.listClusterUsages(ctx, ws.ID, []string{clusterID}); err != nil {
			return "", "", err
		}
		if len(usages) > 0 {
			return "", "", &accesspoint.ClusterInUseError{Usages: usages}
		}
	}
	if after == "" {
		err = s.ensureBindings(ctx, ws.ID, before, nil, []string{clusterID})
	} else {
		err = s.ensureBindings(ctx, ws.ID, after, []string{clusterID}, nil)
	}
	if err != nil {
		return "", "", err
	}
	return before, after, nil
}

// ListClusters 返回空间的集群绑定视图：候选集群及其环境、绑定的接入点、是否被使用。
// 有环境的集群排在前面、无环境的排在后面，每组内按集群 ID 排序；BCS 查询失败时直接返回错误，不返回部分结果
func (s *Service) ListClusters(ctx context.Context, ws *bkmsworkspace.Workspace) ([]ClusterView, error) {
	clusters, err := clusteraddon.ListClusters(ctx, ws.BkSystems.BkBCSProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list candidate clusters of workspace %s", ws.ID)
	}
	views, err := s.buildViews(ctx, ws.ID, clusters)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(views, func(a, b ClusterView) int {
		if hasEnvA, hasEnvB := len(a.Envs) > 0, len(b.Envs) > 0; hasEnvA != hasEnvB {
			return lo.Ternary(hasEnvA, -1, 1)
		}
		return strings.Compare(a.Cluster.ID, b.Cluster.ID)
	})
	return views, nil
}

// ValidateEnvClusterChange 校验环境切换集群后，所使用的北极星接入点是否保持不变：
// 平台托管接入点可用（不允许空间自建接入点，所有集群与配置都使用它）、集群未变、环境未被北极星配置使用时直接通过；
// 否则要求新集群能解析出接入点（未绑定返回 ErrClusterNotBound），
// 且与各北极星配置锁定的接入点一致（不一致返回 ErrAccessPointMismatch）
func (s *Service) ValidateEnvClusterChange(ctx context.Context, env envmodel.Environment, newClusterID string) error {
	if accesspoint.PlatformAvailable() || newClusterID == env.Cluster.ClusterID {
		return nil
	}
	configs, err := polaris.ListByScopeEnvs(ctx, s.db, env.WorkspaceID, []string{env.Name})
	if err != nil {
		return errors.Wrapf(err, "list polaris configs using env %s", env.Name)
	}
	if len(configs) == 0 {
		return nil
	}

	ap, err := s.resolver.ResolveByCluster(ctx, env.WorkspaceID, newClusterID)
	if err != nil {
		return errors.Wrapf(err, "env %s is used by polaris configs %v", env.Name, getConfigNames(configs))
	}
	mismatched := lo.Filter(configs, func(c polaris.PolarisConfig, _ int) bool {
		return cmp.Or(c.AccessPointID, accesspoint.PlatformID) != ap.ID
	})
	if len(mismatched) > 0 {
		return errors.Wrapf(accesspoint.ErrAccessPointMismatch,
			"env %s is used by polaris configs %v, which are fixed to other access points than %q of cluster %s",
			env.Name, getConfigNames(mismatched), ap.Name, newClusterID,
		)
	}
	return nil
}

// ensureBindings 先写入新增与换绑的集群，再删除被移出集群的绑定。
// 未使用多文档事务：中途写入失败时直接返回错误；请求携带全量状态，重新提交即可最终一致
func (s *Service) ensureBindings(ctx context.Context, workspaceID, accessPointID string, bind, unbind []string) error {
	if len(bind) > 0 {
		username := auth.MustGetUser(ctx).ID
		objs := lo.Map(bind, func(clusterID string, _ int) *accesspoint.PolarisClusterBinding {
			return &accesspoint.PolarisClusterBinding{
				WorkspaceID:   workspaceID,
				ClusterID:     clusterID,
				AccessPointID: accessPointID,
				Creator:       username,
				Updater:       username,
			}
		})
		_, err := s.bindings.UpsertMany(ctx, objs,
			func(b *accesspoint.PolarisClusterBinding) store.Opt {
				return accesspoint.ByCluster(b.WorkspaceID, b.ClusterID)
			},
			store.UpdateSelected("accessPointID", "updater"),
		)
		if err != nil {
			return errors.Wrapf(err, "bind clusters %v to polaris access point %s", bind, accessPointID)
		}
	}
	if len(unbind) > 0 {
		// 只删除仍绑定在本接入点上的记录，不误删校验后被其他请求换绑走的集群
		_, err := s.bindings.DeleteMany(ctx,
			accesspoint.ByAccessPoint(workspaceID, accessPointID),
			store.Where("clusterID", bson.M{"$in": unbind}),
		)
		if err != nil {
			return errors.Wrapf(err, "unbind clusters %v from polaris access point %s", unbind, accessPointID)
		}
	}
	return nil
}

// buildViews 为候选集群组装集群绑定视图，顺序与 clusters 一致
func (s *Service) buildViews(ctx context.Context, workspaceID string, clusters []bcs.Cluster) ([]ClusterView, error) {
	if len(clusters) == 0 {
		return []ClusterView{}, nil
	}
	clusterIDs := lo.Map(clusters, func(c bcs.Cluster, _ int) string { return c.ID })
	envs, err := s.listEnvsOnClusters(ctx, workspaceID, clusterIDs)
	if err != nil {
		return nil, err
	}
	bindings, err := s.bindings.List(ctx, accesspoint.ByClusters(workspaceID, clusterIDs))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris bindings of workspace %s", workspaceID)
	}
	aps, err := s.accessPoints.List(ctx, accesspoint.InWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris access points of workspace %s", workspaceID)
	}
	usages, err := s.listClusterUsagesOfEnvs(ctx, workspaceID, envs)
	if err != nil {
		return nil, err
	}

	envsByCluster := lo.GroupBy(envs, func(e envmodel.Environment) string { return e.Cluster.ClusterID })
	apByCluster := lo.SliceToMap(bindings, func(b accesspoint.PolarisClusterBinding) (string, string) {
		return b.ClusterID, b.AccessPointID
	})
	apByID := lo.SliceToMap(aps, func(p accesspoint.PolarisAccessPoint) (string, *accesspoint.PolarisAccessPoint) {
		return p.ID.Hex(), &p
	})
	inUse := lo.Keyify(lo.Map(usages, func(u accesspoint.ClusterUsage, _ int) string { return u.ClusterID }))

	return lo.Map(clusters, func(c bcs.Cluster, _ int) ClusterView {
		return ClusterView{
			Cluster:     c,
			Envs:        lo.CoalesceSliceOrEmpty(envsByCluster[c.ID]),
			AccessPoint: apByID[apByCluster[c.ID]],
			InUse:       lo.HasKey(inUse, c.ID),
		}
	}), nil
}

// getAccessPoint 获取空间内的自建接入点，不存在或不属于该空间时返回 ErrAccessPointNotFound
func (s *Service) getAccessPoint(
	ctx context.Context, workspaceID, accessPointID string,
) (*accesspoint.PolarisAccessPoint, error) {
	opt, err := accesspoint.ByWorkspaceAndID(workspaceID, accessPointID)
	if err != nil {
		return nil, err
	}
	p, err := s.accessPoints.Get(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "get polaris access point %s in workspace %s", accessPointID, workspaceID)
	}
	return p, nil
}

// candidateClusters 返回 clusterIDs 对应的候选集群（顺序与 BCS 返回的一致）。
// 任一集群不是空间 BCS 项目下的非共享集群时返回 ErrAccessPointInvalid；clusterIDs 为空时不查询 BCS
func (s *Service) candidateClusters(
	ctx context.Context, ws *bkmsworkspace.Workspace, clusterIDs []string,
) ([]bcs.Cluster, error) {
	if len(clusterIDs) == 0 {
		return nil, nil
	}
	clusters, err := clusteraddon.ListClusters(ctx, ws.BkSystems.BkBCSProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list candidate clusters of workspace %s", ws.ID)
	}
	invalid := lo.Without(clusterIDs, lo.Map(clusters, func(c bcs.Cluster, _ int) string { return c.ID })...)
	if len(invalid) > 0 {
		return nil, errors.Wrapf(accesspoint.ErrAccessPointInvalid,
			"clusters %v are not non-shared clusters of the workspace bcs project", invalid)
	}
	return lo.Filter(clusters, func(c bcs.Cluster, _ int) bool { return slices.Contains(clusterIDs, c.ID) }), nil
}

// listClusterUsages 返回集群上的环境（含特性环境）被北极星配置使用的情况
func (s *Service) listClusterUsages(
	ctx context.Context, workspaceID string, clusterIDs []string,
) ([]accesspoint.ClusterUsage, error) {
	if len(clusterIDs) == 0 {
		return nil, nil
	}
	envs, err := s.listEnvsOnClusters(ctx, workspaceID, clusterIDs)
	if err != nil {
		return nil, err
	}
	return s.listClusterUsagesOfEnvs(ctx, workspaceID, envs)
}

// listClusterUsagesOfEnvs 返回 envs 被北极星配置使用的情况，按集群、环境、应用、配置排序；
// 已查出集群上环境的调用方直接使用它，避免重复查询环境
func (s *Service) listClusterUsagesOfEnvs(
	ctx context.Context, workspaceID string, envs []envmodel.Environment,
) ([]accesspoint.ClusterUsage, error) {
	envByName := lo.KeyBy(envs, func(e envmodel.Environment) string { return e.Name })
	configs, err := polaris.ListByScopeEnvs(ctx, s.db, workspaceID, lo.Keys(envByName))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris configs of workspace %s", workspaceID)
	}
	usages := []accesspoint.ClusterUsage{}
	for _, cfg := range configs {
		for _, envName := range lo.Uniq(cfg.ScopeEnvNames) {
			env, ok := envByName[envName]
			if !ok {
				continue
			}
			usages = append(usages, accesspoint.ClusterUsage{
				ClusterID:  env.Cluster.ClusterID,
				EnvName:    envName,
				AppID:      cfg.AppID,
				ConfigName: cfg.Name,
			})
		}
	}
	slices.SortFunc(usages, func(a, b accesspoint.ClusterUsage) int {
		return cmp.Or(
			strings.Compare(a.ClusterID, b.ClusterID),
			strings.Compare(a.EnvName, b.EnvName),
			strings.Compare(a.AppID, b.AppID),
			strings.Compare(a.ConfigName, b.ConfigName),
		)
	})
	return usages, nil
}

// listEnvsOnClusters 列出空间内位于这些集群上的环境（含特性环境），按名称排序
func (s *Service) listEnvsOnClusters(
	ctx context.Context, workspaceID string, clusterIDs []string,
) ([]envmodel.Environment, error) {
	envs, err := s.envs.List(ctx,
		envmodel.InWorkspace(workspaceID),
		store.Where("cluster.clusterID", bson.M{"$in": clusterIDs}),
		store.Asc("name"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list envs on clusters %v in workspace %s", clusterIDs, workspaceID)
	}
	return envs, nil
}

func getClusterIDsOf(bindings []accesspoint.PolarisClusterBinding) []string {
	return lo.Map(bindings, func(b accesspoint.PolarisClusterBinding, _ int) string { return b.ClusterID })
}

// getConfigNames 以 appID/configName 的形式列出北极星配置
func getConfigNames(configs []polaris.PolarisConfig) []string {
	return lo.Map(configs, func(c polaris.PolarisConfig, _ int) string { return c.AppID + "/" + c.Name })
}
