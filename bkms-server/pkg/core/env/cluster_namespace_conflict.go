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

package env

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// cluster+namespace 占用检查配合 Mongo unique index 使用，调用方分两步：
//
//  1. 写库前：调 findClusterNamespaceOccupant，占用则立刻返回带占用方信息的错误。
//  2. 写库失败后：调 rewriteWriteErrIfClusterOccupied，再查一次占用，
//     确认后把 store 层的笼统错误替换成结构化占用错误。
//
// 第 2 步存在的原因：并发时第 1 步可能漏过，unique index 拦住写入后，store 把所有
// DuplicateKey 都收成「环境名已存在」，丢失了冲突类型信息，需要归因。

// ErrEnvClusterNamespaceOccupied 表示某个 clusterID + namespace 已被其他环境占用，
// 附带参数 clusterID、namespace、occupiedByEnvName、occupiedByWorkspaceID（前端据此展示占用方）
var ErrEnvClusterNamespaceOccupied = bkerrs.Define(
	bkerrs.ErrCodeAlreadyExists, "ENV_CLUSTER_NAMESPACE_OCCUPIED", "env cluster namespace occupied",
)

// findClusterNamespaceOccupant 查询 clusterID+namespace 是否已被其他环境占用
func findClusterNamespaceOccupant(
	ctx context.Context,
	db *mongo.Database,
	cluster model.BizCluster,
	excludeEnvID bson.ObjectID,
) error {
	if cluster.ClusterID == "" || cluster.Namespace == "" {
		return nil
	}

	occupiedEnv, err := model.FindByClusterNamespace(ctx, db, cluster.ClusterID, cluster.Namespace, excludeEnvID)
	if err != nil {
		return errors.Wrap(err, "find env by cluster namespace conflict")
	}
	if occupiedEnv == nil {
		return nil
	}
	return ErrEnvClusterNamespaceOccupied.
		With("clusterID", cluster.ClusterID).
		With("namespace", cluster.Namespace).
		With("occupiedByEnvName", occupiedEnv.Name).
		With("occupiedByWorkspaceID", occupiedEnv.WorkspaceID)
}

// rewriteWriteErrIfClusterOccupied 在 Create/Update 已经失败后调用。
// 查 store 确认是否存在 clusterID+namespace 占用，是则返回结构化冲突错误，否则原样返回 originalErr。
func rewriteWriteErrIfClusterOccupied(
	ctx context.Context,
	db *mongo.Database,
	cluster model.BizCluster,
	excludeEnvID bson.ObjectID,
	originalErr error,
) error {
	if conflictErr := findClusterNamespaceOccupant(ctx, db, cluster, excludeEnvID); conflictErr != nil {
		return conflictErr
	}
	return originalErr
}
