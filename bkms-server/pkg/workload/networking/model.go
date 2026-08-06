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

package networking

import "time"

// Protocol represents a network protocol.
type Protocol string

const (
	// ProtocolTCP is the TCP protocol.
	ProtocolTCP Protocol = "TCP"
	// ProtocolUDP is the UDP protocol.
	ProtocolUDP Protocol = "UDP"
	// ProtocolSCTP is the SCTP protocol.
	ProtocolSCTP Protocol = "SCTP"
)

// Service 表示一个应用在平台中声明的“服务”配置模型。
// 它用于描述如何对外暴露应用的工作负载，其实际对应的 k8s Service 资源由包 deploy/networking 中的 serviceSyncer 负责创建与同步。
//
// 注意：此 Service 是平台层的数据模型，仅用于存储配置，不直接等同于 Kubernetes Service 对象。
//
// 不同类型的应用使用此模型的方式有所不同：
//   - Helm 应用：Helm Chart 本身可能已定义 Service，但平台允许用户在此额外声明独立的 Service，
//     主要用于支持高级平台功能（如流量泳道等），与 Helm Chart 中的 Service 无耦合。
//   - 其他应用：暂时不支持。
type Service struct {
	// AppID 为 Service 所属的应用标识
	AppID string `bson:"appID" validate:"required"`

	// Name 服务名称
	Name string `bson:"name" validate:"required"`
	// Selector 服务选择器
	Selector map[string]string `bson:"selector"`
	// Ports 服务端口配置
	Ports []ServicePort `bson:"ports"`

	// TrafficLaneEnabled 是否设置为泳道服务
	TrafficLaneEnabled bool `bson:"trafficLaneEnabled"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// ServicePort 表示服务端口配置
type ServicePort struct {
	// Name is the name of the port
	Name     string   `bson:"name" json:"name"`
	Protocol Protocol `bson:"protocol" json:"protocol"`
	// Port is the port number of the service
	Port int32 `bson:"port" json:"port"`
	// TargetPort is the port on the pod
	TargetPort string `bson:"targetPort" json:"targetPort"`
}
