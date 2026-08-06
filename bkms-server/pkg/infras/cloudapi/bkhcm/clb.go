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

package bkhcm

import (
	"context"
	"fmt"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
)

// CreateBizApplicationForCreateLoadBalancer 创建负载均衡申请
//
// 业务下创建负载均衡申请，提交创建 CLB 的业务申请单（固定厂商 tcloud-ziyan）。
func (c *ApiClient) CreateBizApplicationForCreateLoadBalancer(
	ctx context.Context, req *CreateLoadBalancerReq,
) (string, error) {
	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_biz_application_for_create_load_balancer",
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/cloud/vendors/%s/applications/types/create_load_balancer", VendorTCloudZiYan),
		},
		bkapi.OptSetRequestBody(req),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return "", err
	}

	return mapx.GetStr(result, "data.id"), nil
}
