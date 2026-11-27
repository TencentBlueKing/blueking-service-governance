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

package clustertoken

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// TokenRef 引用一个 Token：TokenID 为已有 Token，Token 为新 Token 的明文，必须且只能填一个。
// 新 Token 与空间内已有 Token 相同时复用已有 Token
type TokenRef struct {
	TokenID string
	Token   string
}

// PolarisBindingChecker 返回集群在空间内是否绑定了北极星接入点。
// addon/polaris/accesspoint 收窄候选集群时需依赖本包，由调用方传入北极星绑定的查询，本包不直接引用
type PolarisBindingChecker func(ctx context.Context, workspaceID, clusterID string) (bool, error)

// TokenListResult 空间的集群 Token 列表
type TokenListResult struct {
	// Tokens 按创建时间排序
	Tokens []ClusterToken
	// ClusterCounts Token ID 到使用它的集群数量的映射
	ClusterCounts map[string]int
}

// WorkspaceClusterListResult 空间已接入的集群
type WorkspaceClusterListResult struct {
	// Clusters 按接入时间排序
	Clusters []WorkspaceCluster
	// Tokens Token ID 到 Token 的映射
	Tokens map[string]ClusterToken
	// EnvCounts 集群 ID 到使用它的环境数量（含特性环境）的映射
	EnvCounts map[string]int
}

// Candidate 可接入的集群
type Candidate struct {
	bcs.Cluster
	// Accessed 是否已接入空间
	Accessed bool
}

// AccessResult 接入集群的结果
type AccessResult struct {
	// Clusters 新增的接入记录
	Clusters []WorkspaceCluster
	// Token 接入使用的 Token
	Token *ClusterToken
}

// Manager 空间集群 Token 与集群接入的管理。
//
// 涉及的模型：cluster_tokens、workspace_clusters；移除集群时读取 environments，北极星绑定由调用方传入查询
type Manager struct {
	tokens   *TokenStore
	clusters *WorkspaceClusterStore
	envs     *store.GenericStore[envmodel.Environment]
}

// NewManager 创建 Manager
func NewManager(db *mongo.Database) *Manager {
	return &Manager{
		tokens:   store.Of(db, TokenMeta),
		clusters: store.Of(db, WorkspaceClusterMeta),
		envs:     store.Of(db, envmodel.EnvMeta),
	}
}

// ListTokens 返回空间的集群 Token 以及各 Token 被使用的集群数量
func (m *Manager) ListTokens(ctx context.Context, workspaceID string) (*TokenListResult, error) {
	tokens, err := m.tokens.List(ctx, InWorkspace(workspaceID), store.Asc("createdAt"))
	if err != nil {
		return nil, errors.Wrapf(err, "list cluster tokens of workspace %s", workspaceID)
	}
	wcs, err := m.clusters.List(ctx, InWorkspace(workspaceID), store.SelectFields("tokenID"))
	if err != nil {
		return nil, errors.Wrapf(err, "list workspace clusters of workspace %s", workspaceID)
	}
	return &TokenListResult{
		Tokens:        tokens,
		ClusterCounts: lo.CountValuesBy(wcs, func(a WorkspaceCluster) string { return a.TokenID }),
	}, nil
}

// CreateToken 新建集群 Token，空间内已有相同 Token 时返回 ErrTokenExists；空间内没有默认 Token 时设为默认
func (m *Manager) CreateToken(ctx context.Context, workspaceID, token string) (*ClusterToken, error) {
	token, err := normalizeToken(token)
	if err != nil {
		return nil, err
	}
	exists, err := m.tokens.Exists(ctx, ByFingerprint(workspaceID, fingerprintOf(token)))
	if err != nil {
		return nil, errors.Wrapf(err, "check cluster token duplication in workspace %s", workspaceID)
	}
	if exists {
		return nil, errors.Wrapf(ErrTokenExists, "workspace %s", workspaceID)
	}
	return m.insertToken(ctx, workspaceID, token)
}

// ReplaceToken 更换 Token 的值，使用它的集群一并生效；返回更换前后的 Token。
// 新值与空间内其他 Token 相同时返回 ErrTokenExists
func (m *Manager) ReplaceToken(
	ctx context.Context, workspaceID, id, token string,
) (before, after *ClusterToken, err error) {
	if token, err = normalizeToken(token); err != nil {
		return nil, nil, err
	}
	if before, err = m.getToken(ctx, workspaceID, id); err != nil {
		return nil, nil, err
	}
	after = lo.ToPtr(*before)
	after.Token, after.Masked, after.Fingerprint = token, maskToken(token), fingerprintOf(token)
	after.Updater = auth.MustGetUser(ctx).ID
	// 只写入 fingerprint 而不写 isDefault，唯一键冲突只可能来自 Token 重复
	err = m.tokens.Update(ctx, after,
		store.ByID(before.ID), store.UpdateSelected("token", "masked", "fingerprint", "updater"))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "replace cluster token %s in workspace %s", id, workspaceID)
	}
	return before, after, nil
}

// SetDefaultToken 把 Token 设为空间的默认 Token。
// 并发设置时由部分唯一索引保证至多一个默认 Token，冲突方返回 ErrDefaultTokenConflict
func (m *Manager) SetDefaultToken(ctx context.Context, workspaceID, id string) (*ClusterToken, error) {
	target, err := m.getToken(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if target.IsDefault {
		return target, nil
	}
	previous, err := m.tokens.List(ctx, DefaultToken(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get default cluster token of workspace %s", workspaceID)
	}
	_, err = m.tokens.UpdateMany(ctx, map[string]any{"isDefault": false},
		DefaultToken(workspaceID), store.KeepUpdatedAt())
	if err != nil {
		return nil, errors.Wrapf(err, "unset default cluster token of workspace %s", workspaceID)
	}
	err = m.tokens.Update(ctx, map[string]any{"isDefault": true}, store.ByID(target.ID), store.KeepUpdatedAt())
	if errors.Is(err, store.ErrDuplicate) {
		// 并发设置的一方已成为默认 Token，不能再恢复原来的
		return nil, errors.Wrapf(ErrDefaultTokenConflict, "workspace %s", workspaceID)
	}
	if err != nil {
		m.restoreDefaultToken(ctx, workspaceID, previous)
		return nil, errors.Wrapf(err, "set default cluster token %s in workspace %s", id, workspaceID)
	}
	target.IsDefault = true
	return target, nil
}

// restoreDefaultToken 设置默认 Token 失败后，恢复原来的默认 Token；失败只记录日志
func (m *Manager) restoreDefaultToken(ctx context.Context, workspaceID string, previous []ClusterToken) {
	for _, t := range previous {
		err := m.tokens.Update(ctx, map[string]any{"isDefault": true}, store.ByID(t.ID), store.KeepUpdatedAt())
		if err != nil {
			log.Errorf(ctx, "restore default cluster token %s of workspace %s: %v", t.ID.Hex(), workspaceID, err)
		}
	}
}

// DeleteToken 删除 Token，仍有集群使用时返回 TokenInUseError。删除默认 Token 后空间没有默认 Token
func (m *Manager) DeleteToken(ctx context.Context, workspaceID, id string) (*ClusterToken, error) {
	t, err := m.getToken(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	// 接入记录保存小写的 ID；id 可以是大写，按原值查询会漏掉引用
	wcs, err := m.clusters.List(ctx, ByToken(workspaceID, t.ID.Hex()),
		store.Asc("createdAt"), store.SelectFields("clusterID", "clusterName"))
	if err != nil {
		return nil, errors.Wrapf(err, "list clusters using cluster token %s", id)
	}
	if len(wcs) > 0 {
		return nil, &TokenInUseError{Clusters: lo.Map(wcs, func(a WorkspaceCluster, _ int) ClusterRef {
			return ClusterRef{ClusterID: a.ClusterID, ClusterName: a.ClusterName}
		})}
	}
	if err = m.tokens.Delete(ctx, store.ByID(t.ID)); err != nil {
		return nil, errors.Wrapf(err, "delete cluster token %s in workspace %s", id, workspaceID)
	}
	return t, nil
}

// ListCandidates 返回空间关联的 BCS 项目下的全部集群（含共享集群），并标记是否已接入
func (m *Manager) ListCandidates(ctx context.Context, workspaceID, bcsProjectID string) ([]Candidate, error) {
	clusters, err := listProjectClusters(ctx, bcsProjectID)
	if err != nil {
		return nil, err
	}
	wcs, err := m.clusters.List(ctx, InWorkspace(workspaceID), store.SelectFields("clusterID"))
	if err != nil {
		return nil, errors.Wrapf(err, "list workspace clusters of workspace %s", workspaceID)
	}
	accessed := lo.SliceToMap(wcs, func(a WorkspaceCluster) (string, bool) { return a.ClusterID, true })
	return lo.Map(clusters, func(c bcs.Cluster, _ int) Candidate {
		return Candidate{Cluster: c, Accessed: accessed[c.ID]}
	}), nil
}

// ListClusters 返回空间已接入的集群、所用的 Token 以及各集群被环境使用的数量。
// tokenID 非空时只返回使用该 Token 的集群
func (m *Manager) ListClusters(ctx context.Context, workspaceID, tokenID string) (*WorkspaceClusterListResult, error) {
	filter := InWorkspace(workspaceID)
	if tokenID = strings.TrimSpace(tokenID); tokenID != "" {
		if objID, err := bson.ObjectIDFromHex(tokenID); err == nil {
			tokenID = objID.Hex()
		}
		filter = ByToken(workspaceID, tokenID)
	}
	wcs, err := m.clusters.List(ctx, filter, store.Asc("createdAt"))
	if err != nil {
		return nil, errors.Wrapf(err, "list workspace clusters of workspace %s", workspaceID)
	}
	tokens, err := m.tokens.List(ctx, InWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "list cluster tokens of workspace %s", workspaceID)
	}
	envs, err := m.envs.List(ctx, envmodel.InWorkspace(workspaceID), store.SelectFields("cluster"))
	if err != nil {
		return nil, errors.Wrapf(err, "list envs of workspace %s", workspaceID)
	}
	return &WorkspaceClusterListResult{
		Clusters: wcs,
		Tokens:   lo.SliceToMap(tokens, func(t ClusterToken) (string, ClusterToken) { return t.ID.Hex(), t }),
		EnvCounts: lo.CountValuesBy(
			lo.Filter(envs, func(e envmodel.Environment, _ int) bool { return e.Cluster.ClusterID != "" }),
			func(e envmodel.Environment) string { return e.Cluster.ClusterID },
		),
	}, nil
}

// Access 把 BCS 项目下的集群接入空间。集群不属于项目时返回 ErrClusterNotInProject，
// 已接入时返回 ErrWorkspaceClusterExists；ref 为新 Token 时先登记 Token
func (m *Manager) Access(
	ctx context.Context, workspaceID, bcsProjectID string, clusterIDs []string, ref TokenRef,
) (*AccessResult, error) {
	ref, err := normalizeRef(ref)
	if err != nil {
		return nil, err
	}
	clusterIDs = lo.Uniq(
		lo.Compact(lo.Map(clusterIDs, func(id string, _ int) string { return strings.TrimSpace(id) })),
	)
	if len(clusterIDs) == 0 {
		return nil, errors.Wrap(ErrClusterNotInProject, "no cluster to access")
	}
	clusters, err := m.projectClustersByID(ctx, bcsProjectID, clusterIDs)
	if err != nil {
		return nil, err
	}
	existing, err := m.clusters.List(ctx, InWorkspace(workspaceID),
		store.Filter(bson.M{"clusterID": bson.M{"$in": clusterIDs}}), store.SelectFields("clusterID"))
	if err != nil {
		return nil, errors.Wrapf(err, "list workspace clusters of workspace %s", workspaceID)
	}
	if len(existing) > 0 {
		return nil, errors.Wrapf(ErrWorkspaceClusterExists, "clusters %v in workspace %s",
			lo.Map(existing, func(a WorkspaceCluster, _ int) string { return a.ClusterID }), workspaceID)
	}

	token, err := m.resolveRef(ctx, workspaceID, ref)
	if err != nil {
		return nil, err
	}
	username := auth.MustGetUser(ctx).ID
	docs := lo.Map(clusterIDs, func(id string, _ int) *WorkspaceCluster {
		return &WorkspaceCluster{
			WorkspaceID: workspaceID,
			ClusterID:   id,
			ClusterName: clusters[id].Name,
			IsShared:    clusters[id].IsShared,
			TokenID:     token.ID.Hex(),
			Creator:     username,
			Updater:     username,
		}
	})
	if err = m.clusters.CreateMany(ctx, docs); err != nil {
		// 唯一键冲突前的文档已写入，回滚本次写入的记录，避免部分接入
		m.deleteClusters(ctx, docs)
		return nil, errors.Wrapf(err, "create workspace clusters in workspace %s", workspaceID)
	}
	// 与删除 Token 并发时，写入期间 Token 可能已被删除，此时撤销本次接入
	if err = m.ensureTokenExists(ctx, workspaceID, token.ID); err != nil {
		m.deleteClusters(ctx, docs)
		return nil, err
	}
	return &AccessResult{Clusters: lo.FromSlicePtr(docs), Token: token}, nil
}

// SwitchToken 更换集群使用的 Token，返回更换前后的接入记录与新 Token
func (m *Manager) SwitchToken(
	ctx context.Context, workspaceID, clusterID string, ref TokenRef,
) (before, after *WorkspaceCluster, token *ClusterToken, err error) {
	if ref, err = normalizeRef(ref); err != nil {
		return nil, nil, nil, err
	}
	if before, err = m.clusters.Get(ctx, ByCluster(workspaceID, clusterID)); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "get workspace cluster %s in workspace %s", clusterID, workspaceID)
	}
	if token, err = m.resolveRef(ctx, workspaceID, ref); err != nil {
		return nil, nil, nil, err
	}
	after = lo.ToPtr(*before)
	if after.TokenID == token.ID.Hex() {
		return before, after, token, nil
	}
	after.TokenID, after.Updater = token.ID.Hex(), auth.MustGetUser(ctx).ID
	err = m.clusters.Update(ctx, after, store.ByID(before.ID), store.UpdateSelected("tokenID", "updater"))
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "switch token of cluster %s in workspace %s", clusterID, workspaceID)
	}
	if err = m.ensureTokenExists(ctx, workspaceID, token.ID); err != nil {
		if rollbackErr := m.clusters.Update(ctx, before, store.ByID(before.ID),
			store.UpdateSelected("tokenID", "updater")); rollbackErr != nil {
			log.Errorf(ctx, "rollback token of cluster %s in workspace %s: %v", clusterID, workspaceID, rollbackErr)
		}
		return nil, nil, nil, err
	}
	return before, after, token, nil
}

// Remove 移除空间接入的集群。集群仍被环境（含特性环境）使用或绑定了北极星接入点时返回 ClusterInUseError
func (m *Manager) Remove(
	ctx context.Context, workspaceID, clusterID string, polarisBound PolarisBindingChecker,
) (*WorkspaceCluster, error) {
	wc, err := m.clusters.Get(ctx, ByCluster(workspaceID, clusterID))
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace cluster %s in workspace %s", clusterID, workspaceID)
	}
	envs, err := m.envs.List(ctx, envsOnCluster(workspaceID, clusterID),
		store.Asc("createdAt"), store.SelectFields("name", "displayName"))
	if err != nil {
		return nil, errors.Wrapf(err, "list envs on cluster %s in workspace %s", clusterID, workspaceID)
	}
	bound := false
	if polarisBound != nil {
		if bound, err = polarisBound(ctx, workspaceID, clusterID); err != nil {
			return nil, errors.Wrapf(err, "check polaris binding of cluster %s", clusterID)
		}
	}
	if len(envs) > 0 || bound {
		return nil, &ClusterInUseError{
			Envs: lo.Map(envs, func(e envmodel.Environment, _ int) EnvRef {
				return EnvRef{Name: e.Name, DisplayName: e.DisplayName}
			}),
			PolarisBound: bound,
		}
	}
	if err = m.clusters.Delete(ctx, store.ByID(wc.ID)); err != nil {
		return nil, errors.Wrapf(err, "delete workspace cluster %s in workspace %s", clusterID, workspaceID)
	}
	return wc, nil
}

// CountEnvsOnCluster 返回空间内使用集群的环境数量（含特性环境）
func (m *Manager) CountEnvsOnCluster(ctx context.Context, workspaceID, clusterID string) (int, error) {
	n, err := m.envs.Count(ctx, envsOnCluster(workspaceID, clusterID))
	if err != nil {
		return 0, errors.Wrapf(err, "count envs on cluster %s in workspace %s", clusterID, workspaceID)
	}
	return int(n), nil
}

func envsOnCluster(workspaceID, clusterID string) store.Opt {
	return store.Combine(envmodel.InWorkspace(workspaceID), store.Where("cluster.clusterID", clusterID))
}

func (m *Manager) getToken(ctx context.Context, workspaceID, id string) (*ClusterToken, error) {
	opt, err := TokenByID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	t, err := m.tokens.Get(ctx, opt)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster token %s in workspace %s", id, workspaceID)
	}
	return t, nil
}

// insertToken 写入新 Token，空间内没有默认 Token 时设为默认
func (m *Manager) insertToken(ctx context.Context, workspaceID, token string) (*ClusterToken, error) {
	hasDefault, err := m.tokens.Exists(ctx, DefaultToken(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "check default cluster token of workspace %s", workspaceID)
	}
	username := auth.MustGetUser(ctx).ID
	t := &ClusterToken{
		WorkspaceID: workspaceID,
		Token:       token,
		Masked:      maskToken(token),
		Fingerprint: fingerprintOf(token),
		IsDefault:   !hasDefault,
		Creator:     username,
		Updater:     username,
	}
	err = m.tokens.Create(ctx, t)
	if t.IsDefault && errors.Is(err, store.ErrDuplicate) {
		// 冲突可能来自并发创建的默认 Token，不设默认再试一次；仍冲突则是 Token 重复
		t.ID, t.IsDefault = bson.NilObjectID, false
		err = m.tokens.Create(ctx, t)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "create cluster token in workspace %s", workspaceID)
	}
	return t, nil
}

// resolveRef 解析 Token 引用，新 Token 与已有 Token 相同时复用已有 Token；ref 须已经过 normalizeRef
func (m *Manager) resolveRef(ctx context.Context, workspaceID string, ref TokenRef) (*ClusterToken, error) {
	if ref.TokenID != "" {
		return m.getToken(ctx, workspaceID, ref.TokenID)
	}
	byFingerprint := ByFingerprint(workspaceID, fingerprintOf(ref.Token))
	t, err := m.tokens.Get(ctx, byFingerprint)
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, ErrTokenNotFound) {
		return nil, errors.Wrapf(err, "get cluster token by fingerprint in workspace %s", workspaceID)
	}
	t, err = m.insertToken(ctx, workspaceID, ref.Token)
	if errors.Is(err, ErrTokenExists) {
		// 并发登记了相同的 Token
		if t, err = m.tokens.Get(ctx, byFingerprint); err != nil {
			return nil, errors.Wrapf(err, "get cluster token by fingerprint in workspace %s", workspaceID)
		}
	}
	return t, err
}

func (m *Manager) ensureTokenExists(ctx context.Context, workspaceID string, id bson.ObjectID) error {
	exists, err := m.tokens.Exists(ctx, store.Combine(InWorkspace(workspaceID), store.ByID(id)))
	if err != nil {
		return errors.Wrapf(err, "check cluster token %s", id.Hex())
	}
	if !exists {
		return errors.Wrapf(ErrTokenNotFound, "cluster token %s was deleted concurrently", id.Hex())
	}
	return nil
}

// deleteClusters 删除本次接入记录，失败只记录日志
func (m *Manager) deleteClusters(ctx context.Context, docs []*WorkspaceCluster) {
	ids := lo.FilterMap(docs, func(d *WorkspaceCluster, _ int) (bson.ObjectID, bool) { return d.ID, !d.ID.IsZero() })
	if len(ids) == 0 {
		return
	}
	if _, err := m.clusters.DeleteMany(ctx, store.Filter(bson.M{"_id": bson.M{"$in": ids}})); err != nil {
		log.Errorf(ctx, "rollback workspace clusters %v: %v", ids, err)
	}
}

// projectClustersByID 返回 BCS 项目下指定的集群，任一集群不属于项目时返回 ErrClusterNotInProject
func (m *Manager) projectClustersByID(
	ctx context.Context, bcsProjectID string, clusterIDs []string,
) (map[string]bcs.Cluster, error) {
	clusters, err := listProjectClusters(ctx, bcsProjectID)
	if err != nil {
		return nil, err
	}
	byID := lo.KeyBy(clusters, func(c bcs.Cluster) string { return c.ID })
	missing := lo.Reject(clusterIDs, func(id string, _ int) bool { _, ok := byID[id]; return ok })
	if len(missing) > 0 {
		return nil, errors.Wrapf(ErrClusterNotInProject, "clusters %v, bcs project %q", missing, bcsProjectID)
	}
	return byID, nil
}

// listProjectClusters 通过 BCS 云 API 列出项目下的全部 k8s 集群（含共享集群），项目为空时返回空列表
func listProjectClusters(ctx context.Context, bcsProjectID string) ([]bcs.Cluster, error) {
	if bcsProjectID == "" {
		return nil, nil
	}
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "init bcs client")
	}
	clusters, err := client.ListClustersByProject(ctx, bcsProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list clusters of bcs project %s", bcsProjectID)
	}
	return clusters, nil
}

func normalizeRef(ref TokenRef) (TokenRef, error) {
	ref.TokenID = strings.TrimSpace(ref.TokenID)
	switch {
	case ref.TokenID != "" && strings.TrimSpace(ref.Token) == "":
		return TokenRef{TokenID: ref.TokenID}, nil
	case ref.TokenID == "" && ref.Token != "":
		token, err := normalizeToken(ref.Token)
		return TokenRef{Token: token}, err
	default:
		return TokenRef{}, ErrTokenRefInvalid
	}
}
