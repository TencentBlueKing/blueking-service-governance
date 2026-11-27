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

package accesspoint

import (
	"net"
	"strconv"

	"github.com/samber/lo"
)

const (
	// operatorAPIType operator 访问北极星的方式，自建北极星只能使用 HTTP 管理接口
	operatorAPIType = "HTTP_ADMIN"
	// operatorAPIVersion operator 调用的北极星 OpenAPI 版本
	operatorAPIVersion = "v2"
	// operatorAuthMode operator 携带北极星 Token 的方式，Token 本身由 PolarisConfig CR 提供
	operatorAuthMode = "X-Polaris-Token"
)

// GetOperatorValues 返回集群上 bcs-polaris-operator 连接接入点 ap 所需的 values，只包含北极星相关的键与集群标识。
//
// 调用方安装时应以 release 当前的 values 为基础，只覆盖这些键，其余保留；args 按 name 合并。
// 结果不含 Token：Token 只写入各 PolarisConfig CR，不放进 operator 的安装参数。
// 接入点有多个客户端地址时只使用第一个，端口固定为 OpenAPIPort
func GetOperatorValues(ap *AccessPoint, clusterID, bcsProjectID string) map[string]any {
	return map[string]any{
		"args": []any{
			map[string]any{"name": "polaris_api_type", "value": operatorAPIType},
		},
		"env": map[string]any{
			"BK_BCS_clusterId":           clusterID,
			"BK_projectId":               bcsProjectID,
			"BK_BCS_polaris_api_path":    net.JoinHostPort(lo.FirstOrEmpty(ap.ClientHosts), strconv.Itoa(OpenAPIPort)),
			"BK_BCS_polaris_api_version": operatorAPIVersion,
			"BK_BCS_polaris_auth_mode":   operatorAuthMode,
		},
		"authProxy": map[string]any{
			"bcsPolarisPlatformID": "",
		},
	}
}
