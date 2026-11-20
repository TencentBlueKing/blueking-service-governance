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

// Kind 接入点类型
type Kind string

const (
	// KindPlatform 平台托管：由平台配置生成，不入库、不可编辑
	KindPlatform Kind = "platform"
	// KindWorkspace 空间自建：空间管理员登记，入库
	KindWorkspace Kind = "workspace"
)

const (
	// PlatformID 平台托管接入点的固定 ID
	PlatformID = "platform"
	// PlatformName 平台托管接入点的展示名称
	PlatformName = "平台北极星"

	// OpenAPIPort 自建接入点 OpenAPI 的固定端口
	OpenAPIPort = 8090
	// SDKPort 自建接入点 SDK（服务发现）的固定端口
	SDKPort = 8091
	// MaxClientHosts 自建接入点客户端地址数量上限
	MaxClientHosts = 10

	// OperatorAddonName bcs-polaris-operator 对应的内置集群插件名称，组件的安装与状态查询使用集群插件接口
	OperatorAddonName = "bcs-polaris-operator"
)

// NamespaceSource 北极星配置可选命名空间的来源
type NamespaceSource string

const (
	// NamespaceSourceFixed 固定列表，见 FixedNamespaces
	NamespaceSourceFixed NamespaceSource = "fixed"
	// NamespaceSourceLive 实时读取北极星上的命名空间
	NamespaceSourceLive NamespaceSource = "live"
)

// FixedNamespaces 平台托管接入点可选的命名空间
var FixedNamespaces = []string{"Test", "Production", "Development", "Pre-release"}

// Capabilities 接入点能力。两版北极星的差异都体现为能力差异，调用方不应再判断接入点类型或版本
type Capabilities struct {
	// WeightFactor 是否支持权重因子与动态权重
	WeightFactor bool
	// NamespaceSource 北极星配置可选命名空间的来源
	NamespaceSource NamespaceSource
	// ServiceTokenRequired 引入已有北极星服务时是否需要服务 Token
	ServiceTokenRequired bool
	// InjectClientToken 是否向 Pod 注入 {instanceKey}_polarisToken 环境变量
	InjectClientToken bool
}

// CapabilitiesOf 返回指定类型接入点的能力
func CapabilitiesOf(kind Kind) Capabilities {
	if kind == KindPlatform {
		return Capabilities{
			WeightFactor:         true,
			NamespaceSource:      NamespaceSourceFixed,
			ServiceTokenRequired: true,
			InjectClientToken:    true,
		}
	}
	return Capabilities{
		WeightFactor:         false,
		NamespaceSource:      NamespaceSourceLive,
		ServiceTokenRequired: false,
		InjectClientToken:    false,
	}
}

// AccessPoint 解析得到的接入点，是各调用链路（OpenAPI、SDK、operator、tRPC 配置）唯一的接入信息来源。
//
// Token 为明文，仅供服务端内部使用，不能出现在接口响应、日志与审计记录中
type AccessPoint struct {
	// ID 平台托管为 PlatformID，空间自建为数据库 ObjectID 的 hex
	ID string
	// Kind 接入点类型
	Kind Kind
	// WorkspaceID 所属空间，平台托管为空
	WorkspaceID string
	// Name 展示名称
	Name string
	// ConsoleURL 北极星控制台地址，平台托管为空（前端使用全局配置）
	ConsoleURL string
	// OpenAPIBaseURL 北极星 OpenAPI 地址，形如 http://host:8090；
	// 平台托管取自 depservice polaris 服务 default 方案的 baseUrl，未配置时为空
	OpenAPIBaseURL string
	// JoinPoint SDK 接入点名称，仅平台托管且使用私有 SDK 时有值
	JoinPoint string
	// SDKAddresses SDK 直连地址（host:port）
	SDKAddresses []string
	// ClientHosts 自建接入点登记的客户端地址（IP 或域名），平台托管为空
	ClientHosts []string
	// Token 北极星鉴权 Token（明文），未填写时为空
	Token string
	// Capabilities 接入点能力
	Capabilities Capabilities
}

// IsPlatform 是否为平台托管接入点
func (a *AccessPoint) IsPlatform() bool {
	return a.Kind == KindPlatform
}
