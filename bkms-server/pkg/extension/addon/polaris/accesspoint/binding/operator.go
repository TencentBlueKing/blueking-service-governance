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

package binding

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// OperatorValuesView 已绑定接入点的集群上 bcs-polaris-operator 应使用的北极星 values
type OperatorValuesView struct {
	// ClusterID BCS 集群 ID
	ClusterID string
	// AccessPoint 集群绑定的自建接入点
	AccessPoint *accesspoint.PolarisAccessPoint
	// Values 由接入点推导出的北极星相关 values，见 accesspoint.OperatorValues
	Values map[string]any
}

// ListOperatorValues 返回空间内已绑定接入点的集群上组件应使用的北极星 values，按集群 ID 排序；
// accessPointID 非空时只返回绑定到该接入点的集群，接入点不存在或不属于该空间时返回 ErrAccessPointNotFound。
//
// 只读取绑定与接入点，不查询 BCS 与集群内的 release：组件的安装状态与当前 values 由集群插件接口提供
func (s *Service) ListOperatorValues(
	ctx context.Context, ws *bkmsworkspace.Workspace, accessPointID string,
) ([]OperatorValuesView, error) {
	opt := accesspoint.InWorkspace(ws.ID)
	if accessPointID != "" {
		p, err := s.getAccessPoint(ctx, ws.ID, accessPointID)
		if err != nil {
			return nil, err
		}
		opt = accesspoint.ByAccessPoint(ws.ID, p.ID.Hex())
	}
	bindings, err := s.bindings.List(ctx, opt, store.Asc("clusterID"))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris bindings of workspace %s", ws.ID)
	}
	aps, err := s.accessPoints.List(ctx, accesspoint.InWorkspace(ws.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris access points of workspace %s", ws.ID)
	}
	apByID := lo.SliceToMap(aps, func(p accesspoint.PolarisAccessPoint) (string, *accesspoint.PolarisAccessPoint) {
		return p.ID.Hex(), &p
	})

	// 仍绑定集群的接入点不允许删除，找不到接入点的绑定只可能来自并发删除，跳过即可
	return lo.FilterMap(bindings, func(b accesspoint.PolarisClusterBinding, _ int) (OperatorValuesView, bool) {
		p, ok := apByID[b.AccessPointID]
		if !ok {
			return OperatorValuesView{}, false
		}
		return OperatorValuesView{
			ClusterID:   b.ClusterID,
			AccessPoint: p,
			Values:      accesspoint.GetOperatorValues(p.ToAccessPoint(), b.ClusterID, ws.BkSystems.BkBCSProjectID),
		}, true
	}), nil
}
