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

package clusteraddon

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// ErrClusterNotInWorkspace 集群不是空间绑定的 BCS 项目下的非共享集群
var ErrClusterNotInWorkspace = errors.New("cluster is not a dedicated cluster of the workspace bcs project")

// ListWorkspaceClusters 返回空间下的集群列表
//
// 共享集群上的集群级组件由平台维护，空间不能安装或更新,从返回列表中筛掉。空间未绑定 BCS 项目时返回空列表
func ListWorkspaceClusters(ctx context.Context, bcsProjectID string) ([]bcs.Cluster, error) {
	if bcsProjectID == "" {
		return nil, nil
	}
	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get user for bcs client")
	}
	client, err := bcs.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "init bcs client")
	}
	clusters, err := client.ListClustersByProject(ctx, bcsProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list clusters of bcs project %s", bcsProjectID)
	}
	return lo.Filter(clusters, func(c bcs.Cluster, _ int) bool { return !c.IsShared }), nil
}

// GetWorkspaceCluster 返回空间下的指定集群
// 不在范围内时返回 ErrClusterNotInWorkspace
func GetWorkspaceCluster(ctx context.Context, bcsProjectID, clusterID string) (*bcs.Cluster, error) {
	clusters, err := ListWorkspaceClusters(ctx, bcsProjectID)
	if err != nil {
		return nil, err
	}
	found, ok := lo.Find(clusters, func(c bcs.Cluster) bool { return c.ID == clusterID })
	if !ok {
		return nil, errors.Wrapf(
			ErrClusterNotInWorkspace,
			"find cluster %s in bcs project %s",
			clusterID,
			bcsProjectID,
		)
	}
	return &found, nil
}
