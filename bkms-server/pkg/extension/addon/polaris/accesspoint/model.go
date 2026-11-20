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

// Package accesspoint 定义北极星接入点：空间管理员登记的自建北极星，以及由平台配置生成的平台托管北极星。
//
// 接入点按集群绑定，环境、北极星配置、bcs-polaris-operator 都由集群绑定推导出所用接入点。
// 内部版与社区版的差异收敛为接入点的能力字段（Capabilities），调用方只看能力，不判断版本。
// 设计说明见 design_notes/polaris_access_point.md。
package accesspoint

import (
	"net"
	"strconv"
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// PolarisAccessPoint 空间自建北极星接入点
type PolarisAccessPoint struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 所属工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Name 接入点名称，空间内唯一
	Name string `bson:"name"`
	// ConsoleURL 北极星控制台地址（IP:Port 或域名），缺省协议时补 http://
	ConsoleURL string `bson:"consoleURL"`
	// ClientHosts 客户端地址（IP 或域名），1 ~ MaxClientHosts 个；
	// OpenAPI 端口固定为 OpenAPIPort，SDK 端口固定为 SDKPort
	ClientHosts []string `bson:"clientHosts"`
	// Token 北极星鉴权 Token，选填，在库中对称加密存储
	Token string `bson:"token"`
	// LastConnectivity 最近一次连通测试结果，从未测试时为 nil
	LastConnectivity *ConnectivityResult `bson:"lastConnectivity,omitempty"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	Updater   string    `bson:"updater"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// ToAccessPoint 转换为解析结果
func (p *PolarisAccessPoint) ToAccessPoint() *AccessPoint {
	ap := &AccessPoint{
		ID:          p.ID.Hex(),
		Kind:        KindWorkspace,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		ConsoleURL:  p.ConsoleURL,
		SDKAddresses: lo.Map(p.ClientHosts, func(host string, _ int) string {
			return net.JoinHostPort(host, strconv.Itoa(SDKPort))
		}),
		ClientHosts:  p.ClientHosts,
		Token:        p.Token,
		Capabilities: CapabilitiesOf(KindWorkspace),
	}
	if len(p.ClientHosts) > 0 {
		ap.OpenAPIBaseURL = OpenAPIBaseURL(p.ClientHosts[0])
	}
	return ap
}

// OpenAPIBaseURL 返回自建接入点的 OpenAPI 地址：只使用第一个客户端地址，端口固定为 OpenAPIPort
func OpenAPIBaseURL(host string) string {
	return "http://" + net.JoinHostPort(host, strconv.Itoa(OpenAPIPort))
}

// FailureReason 连通测试失败的原因，供前端选择提示
type FailureReason string

const (
	// FailureNetworkUnreachable 网络不通：GET /naming/v1/namespaces 失败
	FailureNetworkUnreachable FailureReason = "networkUnreachable"
	// FailureTokenInvalid Token 无效
	FailureTokenInvalid FailureReason = "tokenInvalid"
)

// ConnectivityResult 连通测试结果
type ConnectivityResult struct {
	// OK 是否通过
	OK bool `bson:"ok"`
	// Reason 失败的原因，通过时为空
	Reason FailureReason `bson:"reason"`
	// Message 失败原因或提示信息，不含 Token
	Message string `bson:"message"`
	// ConsoleAuth 北极星是否开启控制台鉴权（consoleOpen）
	ConsoleAuth bool `bson:"consoleAuth"`
	// CheckedAt 测试时间
	CheckedAt time.Time `bson:"checkedAt"`
}

// PolarisClusterBinding 集群与北极星接入点的绑定，一个空间内每个集群至多绑定一个接入点
type PolarisClusterBinding struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 所属工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// ClusterID BCS 集群 ID
	ClusterID string `bson:"clusterID"`
	// AccessPointID 绑定的空间自建接入点 ID。允许绑定时平台托管接入点不可用，因此这里不会是 PlatformID
	AccessPointID string `bson:"accessPointID"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	Updater   string    `bson:"updater"`
	UpdatedAt time.Time `bson:"updatedAt"`
}
