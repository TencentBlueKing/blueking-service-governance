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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/platformaccount"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// ErrClusterNotFound 集群不存在于 BCS 项目下，或是共享集群
var ErrClusterNotFound = errors.New("cluster not found in bcs project or is shared")

// ListClusters 返回 BCS 项目下可安装集群插件的集群，即全部非共享集群
//
// 共享集群上的集群级组件由平台维护，空间不能安装或更新，因此筛掉。bcsProjectID 为空（空间未绑定 BCS 项目）时返回空列表
func ListClusters(ctx context.Context, bcsProjectID string) ([]bcs.Cluster, error) {
	if bcsProjectID == "" {
		return nil, nil
	}
	// 调用方已检查空间权限，项目取自空间记录，用平台级虚拟账号查询
	caller, err := platformaccount.GetCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get caller for bcs client")
	}
	client, err := bcs.New(caller)
	if err != nil {
		return nil, errors.Wrap(err, "init bcs client")
	}
	clusters, err := client.ListClustersByProject(ctx, bcsProjectID)
	if err != nil {
		return nil, errors.Wrapf(err, "list clusters of bcs project %s", bcsProjectID)
	}
	return lo.Filter(clusters, func(c bcs.Cluster, _ int) bool { return !c.IsShared }), nil
}

// GetCluster 返回 BCS 项目下指定的可安装集群插件的集群
// 集群不存在或是共享集群时返回 ErrClusterNotFound
func GetCluster(ctx context.Context, bcsProjectID, clusterID string) (*bcs.Cluster, error) {
	clusters, err := ListClusters(ctx, bcsProjectID)
	if err != nil {
		return nil, err
	}
	found, ok := lo.Find(clusters, func(c bcs.Cluster) bool { return c.ID == clusterID })
	if !ok {
		return nil, errors.Wrapf(
			ErrClusterNotFound,
			"find cluster %s in bcs project %s",
			clusterID,
			bcsProjectID,
		)
	}
	return &found, nil
}
