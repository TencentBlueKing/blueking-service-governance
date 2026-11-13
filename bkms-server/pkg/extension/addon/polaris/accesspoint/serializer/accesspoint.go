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

// Package serializer 北极星接入点相关 API 的请求与响应结构。
//
// 这里只用 binding 校验必填；地址格式、数量上限、名称重复等校验由领域逻辑完成
package serializer

import (
	"time"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/installtask"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
)

// WorkspaceURIInput 空间路径参数
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// AccessPointURIInput 接入点路径参数
type AccessPointURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 空间自建接入点 ID
	AccessPointID string `uri:"accessPointID" binding:"required"`
}

// EnvURIInput 环境路径参数
type EnvURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,mongodb"`
}

// AccessPointCreateInput 创建接入点的请求
type AccessPointCreateInput struct {
	// 接入点名称，空间内唯一
	Name string `json:"name" binding:"required"`
	// 北极星控制台地址，IP:Port 或域名，缺省协议时补 http://
	ConsoleURL string `json:"consoleURL" binding:"required"`
	// 客户端地址，IP 或域名，1 ~ 10 个；OpenAPI 端口固定为 8090，SDK 端口固定为 8091
	ClientHosts []string `json:"clientHosts" binding:"required"`
	// 北极星鉴权 Token，选填
	Token string `json:"token"`
	// 绑定到该接入点的集群 ID；缺省表示不绑定集群
	ClusterIDs []string `json:"clusterIDs"`
}

// AccessPointUpdateInput 更新接入点的请求，字段缺省表示不修改
type AccessPointUpdateInput struct {
	// 接入点名称
	Name *string `json:"name"`
	// 北极星控制台地址
	ConsoleURL *string `json:"consoleURL"`
	// 客户端地址
	ClientHosts *[]string `json:"clientHosts"`
	// 北极星鉴权 Token；显式传空字符串表示清除
	Token *string `json:"token"`
	// 绑定到该接入点的集群 ID，全量替换；显式传空数组表示解除全部绑定
	ClusterIDs *[]string `json:"clusterIDs"`
}

// ConnectivityTestInput 连通测试的请求
type ConnectivityTestInput struct {
	// 已有接入点的 ID
	ID string `json:"id"`
	// 北极星控制台地址
	ConsoleURL string `json:"consoleURL" binding:"required"`
	// 客户端地址
	ClientHosts []string `json:"clientHosts" binding:"required"`
	// 北极星鉴权 Token
	Token *string `json:"token"`
}

// OperatorInstallInput 在接入点绑定的集群上安装或更新 bcs-polaris-operator 的请求
type OperatorInstallInput struct {
	// 目标集群 ID，必须已绑定该接入点
	ClusterIDs []string `json:"clusterIDs" binding:"required,min=1,dive,required"`
	// chart 版本
	ChartVersion string `json:"chartVersion" binding:"required"`
}

// ConnectivityResultOutput 最近一次连通测试结果
type ConnectivityResultOutput struct {
	// 是否通过
	OK bool `json:"ok"`
	// 失败的原因，通过时为空：networkUnreachable 网络不通 / tokenInvalid Token 无效
	Reason accesspoint.FailureReason `json:"reason" enums:"networkUnreachable,tokenInvalid"`
	// 失败原因或提示信息
	Message string `json:"message"`
	// 是否开启控制台鉴权
	ConsoleAuth bool `json:"consoleAuth"`
	// 测试时间
	CheckedAt time.Time `json:"checkedAt"`
}

// NewConnectivityResultOutput 由连通测试结果构造输出，r 为 nil 时返回 nil
func NewConnectivityResultOutput(r *accesspoint.ConnectivityResult) *ConnectivityResultOutput {
	if r == nil {
		return nil
	}
	return &ConnectivityResultOutput{
		OK:          r.OK,
		Reason:      r.Reason,
		Message:     r.Message,
		ConsoleAuth: r.ConsoleAuth,
		CheckedAt:   r.CheckedAt,
	}
}

// AccessPointOutput 空间自建接入点，不返回 Token 明文。
// 接口只在允许空间自建接入点时注册，此时平台托管接入点不可用，因此不会输出平台托管接入点
type AccessPointOutput struct {
	// 接入点 ID
	ID string `json:"id"`
	// 接入点名称
	Name string `json:"name"`
	// 北极星控制台地址
	ConsoleURL string `json:"consoleURL"`
	// 客户端地址
	ClientHosts []string `json:"clientHosts"`
	// 是否填写了 Token
	HasToken bool `json:"hasToken"`
	// 绑定的集群数量
	BoundClusterCount int `json:"boundClusterCount"`
	// 最近一次连通测试结果，从未测试时为 null
	LastConnectivity *ConnectivityResultOutput `json:"lastConnectivity"`
	// 最近修改人
	Updater string `json:"updater"`
	// 最近修改时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewAccessPointOutput 由空间自建接入点构造输出
func NewAccessPointOutput(p *accesspoint.PolarisAccessPoint, boundClusterCount int) *AccessPointOutput {
	return &AccessPointOutput{
		ID:                p.ID.Hex(),
		Name:              p.Name,
		ConsoleURL:        p.ConsoleURL,
		ClientHosts:       lo.Ternary(p.ClientHosts == nil, []string{}, p.ClientHosts),
		HasToken:          p.Token != "",
		BoundClusterCount: boundClusterCount,
		LastConnectivity:  NewConnectivityResultOutput(p.LastConnectivity),
		Updater:           p.Updater,
		UpdatedAt:         p.UpdatedAt,
	}
}

// ConnectivityTestOutput 连通测试结果
type ConnectivityTestOutput struct {
	// 是否通过
	OK bool `json:"ok"`
	// 失败的原因，通过时为空：networkUnreachable 网络不通 / tokenInvalid Token 无效
	Reason accesspoint.FailureReason `json:"reason" enums:"networkUnreachable,tokenInvalid"`
	// 失败原因或提示信息
	Message string `json:"message"`
	// 是否开启控制台鉴权
	ConsoleAuth bool `json:"consoleAuth"`
	// 北极星上的命名空间（已过滤系统命名空间 Polaris），网络不通时为空
	Namespaces []string `json:"namespaces"`
}

// NewConnectivityTestOutput 由连通测试结果构造输出
func NewConnectivityTestOutput(r *accesspoint.ConnectivityReport) *ConnectivityTestOutput {
	return &ConnectivityTestOutput{
		OK:          r.OK,
		Reason:      r.Reason,
		Message:     r.Message,
		ConsoleAuth: r.ConsoleAuth,
		Namespaces:  lo.Ternary(r.Namespaces == nil, []string{}, r.Namespaces),
	}
}

// NetworkInfoOutput 平台访问北极星的网络信息
type NetworkInfoOutput struct {
	// 平台访问北极星的出口 IP，需加入北极星白名单
	EgressIPs []string `json:"egressIPs"`
}

// PolarisClusterEnvOutput 集群上的环境
type PolarisClusterEnvOutput struct {
	// 环境 ID
	ID string `json:"id"`
	// 环境名称
	Name string `json:"name"`
	// 是否为特性环境
	IsFeature bool `json:"isFeature"`
}

// PolarisClusterOutput 集群绑定视图中的一个候选集群
type PolarisClusterOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 集群名称
	ClusterName string `json:"clusterName"`
	// 集群状态
	ClusterStatus string `json:"clusterStatus"`
	// 集群上的空间环境（含特性环境）
	Envs []PolarisClusterEnvOutput `json:"envs"`
	// 绑定的自建接入点 ID；未绑定为空，此时集群不能使用北极星
	AccessPointID string `json:"accessPointID"`
	// 绑定的自建接入点名称，未绑定为空
	AccessPointName string `json:"accessPointName"`
	// 集群上是否有环境被北极星配置使用；为 true 时不能解绑或换绑
	InUse bool `json:"inUse"`
}

// OperatorTaskOutput 组件安装任务
type OperatorTaskOutput struct {
	// 任务 ID
	ID string `json:"id"`
	// 目标集群 ID
	ClusterID string `json:"clusterID"`
	// chart 版本
	ChartVersion string `json:"chartVersion"`
	// 任务状态
	Status installtask.Status `json:"status" enums:"running,succeeded,failed"`
	// 失败原因等补充信息
	Message string `json:"message"`
	// 发起人
	Operator string `json:"operator"`
	// 开始时间
	StartedAt time.Time `json:"startedAt"`
	// 结束时间，运行中为 null
	FinishedAt *time.Time `json:"finishedAt"`
}

// NewOperatorTaskOutput 由组件安装任务构造输出，t 为 nil 时返回 nil
func NewOperatorTaskOutput(t *installtask.InstallTask) *OperatorTaskOutput {
	if t == nil {
		return nil
	}
	return &OperatorTaskOutput{
		ID:           t.ID.Hex(),
		ClusterID:    t.ClusterID,
		ChartVersion: t.ChartVersion,
		Status:       t.Status,
		Message:      t.Message,
		Operator:     t.Operator,
		StartedAt:    t.StartedAt,
		FinishedAt:   lo.EmptyableToPtr(t.FinishedAt),
	}
}

// PolarisOperatorOutput 集群上 bcs-polaris-operator 的组件状态
type PolarisOperatorOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 集群绑定的自建接入点 ID，未绑定为空
	AccessPointID string `json:"accessPointID"`
	// 是否已安装
	Installed bool `json:"installed"`
	// helm release 名称，未安装为空
	ReleaseName string `json:"releaseName"`
	// helm release 所在命名空间，未安装为空
	Namespace string `json:"namespace"`
	// 已安装的 chart 版本，未安装为空
	ChartVersion string `json:"chartVersion"`
	// 组件状态
	Status accesspoint.OperatorStatus `json:"status" enums:"ready,installing,failed,outdated,not_installed"`
	// 与接入点期望值不一致的 values 键，仅 outdated 时非空
	DriftFields []string `json:"driftFields"`
	// 最近一次安装任务，没有任务时为 null
	LastTask *OperatorTaskOutput `json:"lastTask"`
}

// PolarisOperatorListOutput 组件状态视图
type PolarisOperatorListOutput struct {
	// 候选集群的组件状态
	Operators []PolarisOperatorOutput `json:"operators"`
	// 可选的 chart 版本，从新到旧
	ChartVersions []string `json:"chartVersions"`
}

// AccessPointListResponse 接入点列表的响应
type AccessPointListResponse struct {
	Data []*AccessPointOutput `json:"data"`
}

// AccessPointResponse 单个接入点的响应
type AccessPointResponse struct {
	Data *AccessPointOutput `json:"data"`
}

// ConnectivityTestResponse 连通测试的响应
type ConnectivityTestResponse struct {
	Data *ConnectivityTestOutput `json:"data"`
}

// NetworkInfoResponse 网络信息的响应
type NetworkInfoResponse struct {
	Data *NetworkInfoOutput `json:"data"`
}

// NamespacesResponse 命名空间列表的响应
type NamespacesResponse struct {
	Data []string `json:"data"`
}

// PolarisClusterListResponse 集群绑定视图的响应
type PolarisClusterListResponse struct {
	Data []*PolarisClusterOutput `json:"data"`
}

// PolarisOperatorListResponse 组件状态视图的响应
type PolarisOperatorListResponse struct {
	Data *PolarisOperatorListOutput `json:"data"`
}

// OperatorTaskListResponse 组件安装任务列表的响应
type OperatorTaskListResponse struct {
	Data []*OperatorTaskOutput `json:"data"`
}

// PolarisOperatorResponse 单个集群组件状态的响应
type PolarisOperatorResponse struct {
	Data *PolarisOperatorOutput `json:"data"`
}

// EmptyResponse 无数据的响应
type EmptyResponse struct{}
