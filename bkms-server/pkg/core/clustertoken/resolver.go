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

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Enabled 是否由空间登记集群 Token 并接入集群（bcs.allowWorkspaceClusterTokens）
func Enabled() bool {
	return config.G != nil && config.G.BCS.AllowWorkspaceClusterTokens
}

// Resolver 按空间与集群解析集群接入记录与 Token，是集群 API 获取 Token 的唯一入口
type Resolver struct {
	tokens   *TokenStore
	clusters *WorkspaceClusterStore
}

// NewResolver 创建 Resolver
func NewResolver(db *mongo.Database) *Resolver {
	return &Resolver{tokens: store.Of(db, TokenMeta), clusters: store.Of(db, WorkspaceClusterMeta)}
}

// ResolveToken 返回访问集群 API 使用的 Token：开关关闭时返回平台 Token；
// 开关打开时返回空间为集群分配的 Token，集群未接入时返回 ErrClusterNotAccessed，不回退平台 Token
func (r *Resolver) ResolveToken(ctx context.Context, workspaceID, clusterID string) (string, error) {
	if !Enabled() {
		return config.G.BCS.Token, nil
	}
	wc, err := r.GetCluster(ctx, workspaceID, clusterID)
	if err != nil {
		return "", err
	}
	objID, err := bson.ObjectIDFromHex(wc.TokenID)
	if err != nil {
		return "", errors.Wrapf(err, "invalid token id %q of cluster %s", wc.TokenID, clusterID)
	}
	t, err := r.tokens.Get(ctx, store.Combine(InWorkspace(workspaceID), store.ByID(objID)))
	if err != nil {
		return "", errors.Wrapf(err, "get token of cluster %s in workspace %s", clusterID, workspaceID)
	}
	return t.Token, nil
}

// GetCluster 返回集群在空间内的接入记录，未接入时返回 ErrClusterNotAccessed
func (r *Resolver) GetCluster(ctx context.Context, workspaceID, clusterID string) (*WorkspaceCluster, error) {
	wc, err := r.clusters.Get(ctx, ByCluster(workspaceID, clusterID))
	if errors.Is(err, ErrWorkspaceClusterNotFound) {
		return nil, errors.Wrapf(ErrClusterNotAccessed, "cluster %s, workspace %s", clusterID, workspaceID)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace cluster %s in workspace %s", clusterID, workspaceID)
	}
	return wc, nil
}

// CheckAccessed 开关打开时校验集群已接入空间，未接入返回 ErrClusterNotAccessed；开关关闭时不校验
func (r *Resolver) CheckAccessed(ctx context.Context, workspaceID, clusterID string) error {
	if !Enabled() {
		return nil
	}
	_, err := r.GetCluster(ctx, workspaceID, clusterID)
	return err
}

// AccessedClusters 返回空间已接入的集群，key 为集群 ID
func (r *Resolver) AccessedClusters(ctx context.Context, workspaceID string) (map[string]WorkspaceCluster, error) {
	wcs, err := r.clusters.List(ctx, InWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "list workspace clusters of workspace %s", workspaceID)
	}
	return lo.KeyBy(wcs, func(a WorkspaceCluster) string { return a.ClusterID }), nil
}
