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

// Package appmodel (BKMS Application Model) 是服务治理应用模型的主要模块。
//
// 应用模型是服务治理项目提供的一种抽象概念，它解决了“应用如何以一种简单、灵活以及可扩展的
// 方式部署到 Kubernetes 集群上的问题”。应用模型最重要的设计是 Component（组件）机制，它
// 允许开发者使用一种模块化的方式，对应用的工作负载进行定义和管理。
//
// 当前，并非所有应用都使用了应用模型：
//
// - tRPC 类型：每个应用对应一份应用模型；
// - Helm 类型：暂未使用应用模型；
package appmodel

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

const (
	// LanguageGo 指明 trpc 应用使用 Go 语言
	LanguageGo = "go"
	// LanguageCpp 指明 trpc 应用使用 C++ 语言
	LanguageCpp = "cpp"
)

const (
	// WorkloadTypeTrpc indicates a tRPC workload type.
	WorkloadTypeTrpc = "trpc"
	// WorkloadTypeTaf indicates a TAF workload type.
	WorkloadTypeTaf = "taf"
	// WorkloadTypeStandard indicates a standard workload type.
	// standard 目前仅作为一个内部类型存在，提供通用渲染能力，可以用来测试。
	WorkloadTypeStandard = "standard"
)

const (
	// VariableTypeSystem 系统变量, 由系统自动填充, 用户不可更改
	VariableTypeSystem = "SystemVariable"
	// VariableTypeComponent 组件变量, App关联组件时引入。 (由之前定义先保留)
	VariableTypeComponent = "ComponentVariable"
	// VariableTypeUser 用户变量
	VariableTypeUser = "UserVariable"
)

// PullPolicy 镜像拉取策略，参考 Kubernetes 的定义
type PullPolicy string

const (
	// PullAlways means that kubelet always attempts to pull the latest image. Container will fail If the pull fails.
	PullAlways PullPolicy = "Always"
	// PullNever means that kubelet never pulls an image, but only uses a local image. Container will fail if the image
	// isn't present
	PullNever PullPolicy = "Never"
	// PullIfNotPresent means that kubelet pulls if the image isn't present on disk. Container will fail if the image
	// isn't present and the pull fails.
	PullIfNotPresent PullPolicy = "IfNotPresent"
)

// Probe types
const (
	ProbeTypeHTTP = "HTTP"
	ProbeTypeExec = "EXEC"
	ProbeTypeTCP  = "TCP"
)

// Lifecycle types
const (
	LifecycleTypeHTTP = "HTTP"
	LifecycleTypeExec = "EXEC"
)

// AppModel represents a bkms app model instance owned by a application
//
// 模型中当前主要包含 CD 相关，部分配置比如应用如何被构建（流水线、仓库）、镜像来源等配置
// 信息，主要通过 AppID 反向读取服务治理应用来获取。这有助于产品统一处理不同类型应用的相
// 关功能与逻辑。
//
// TODO: **考虑添加更多字段来扩展应用模型的能力，比如：**
//
// - ports：控制服务端口暴露，比如暴露为 nodePort
// - hostAliases：添加 hostAliases 域名解析支持
// - volumesMounts：添加除了 hostPath 之外的更多类型，比如 configMap、secret 等
// - secrets：快捷创建一些敏感内容作为 Secret 资源，比如 token、密码等
// - files：支持将一些文件直接注入到容器内，提供文件内容或者来源（比如某个 Secret）
type AppModel struct {
	// AppID 为模型所属的应用唯一标识
	AppID string `bson:"appID" validate:"required"`

	// Labels 为应用标签信息，预留给后续管理扩展
	Labels map[string]string `bson:"labels,omitempty"`
	// Annotations 为应用注解信息，预留给后续管理扩展
	Annotations map[string]string `bson:"annotations,omitempty"`

	// Workload workload配置, 包含应用实际运行所需的信息，如启动命令、环境变量等
	Workload Workload `bson:"workload"`

	// Replicas 固定副本数，选填，仅作为默认值使用，实际的副本数可能会在应用部署时被覆盖，
	// 并且可能使用固定副本数以外的其他更动态的模式，比如自动扩缩容。
	Replicas *int32 `bson:"replicas,omitempty"`

	// UpdateStrategy 应用的更新策略，选填
	UpdateStrategy *UpdateStrategy `bson:"updateStrategy,omitempty"`

	// Components 是应用组件，选填
	// 这里是应用维度全局生效的组件配置，应用在任意环境部署时，都会用到这些配置。
	Components []*component.Component `bson:"components,omitempty" validate:"dive"`

	// TkeRouteEni indicates whether to enable TKE Route ENI (VPC-CNI) networking.
	// When true, the builder injects the "tke.cloud.tencent.com/networks: tke-route-eni"
	// annotation into the Pod template. This field is computed at build time and NOT persisted.
	TkeRouteEni bool `bson:"-"`
}

// Workload represents a workload definition
type Workload struct {
	// Type 工作负载类型，为空时默认为 tRPC
	Type string `bson:"type"`
	// Name 工作负载名称, 通常与 App 保持一致
	Name string `bson:"name"`
	// Version 版本， 暂未使用
	Version string `bson:"version,omitempty"`
	// Image 容器镜像
	Image string `bson:"image,omitempty"`

	// ImagePullPolicy 镜像拉取策略
	ImagePullPolicy PullPolicy `bson:"imagePullPolicy,omitempty"`
	// ImagePullSecrets 镜像拉取密钥，默认情况下，平台在部署应用时可能会注入并引用一份默认密钥
	ImagePullSecrets []string `bson:"imagePullSecrets,omitempty"`

	// Command 容器启动命令
	Command []string `bson:"command,omitempty"`
	// Args 容器启动参数
	Args []string `bson:"args,omitempty"`
	// EnvVars 容器环境变量
	EnvVars []Variable `bson:"envVars,omitempty"`

	// Resources 容器资源配置，key 为资源类型（cpu、memory 等），value 为具体数值。
	// value 支持使用 "-" 来分割 requests 和 limits，例如 "cpu": "100m-200m"
	// 未使用分隔符时，表示同时设置 requests 和 limits 为相同数值。
	//
	// 示例值： {"cpu": "100m-200m", "memory": "256Mi-512Mi"}
	Resources map[string]string `bson:"resources,omitempty"`

	// VolumeMounts 容器卷挂载配置，目前仅支持 hostPath 类型，更多待添加
	VolumeMounts VolumeMounts `bson:"volumeMounts,omitempty"`

	// LivenessProbe 存活探针配置
	LivenessProbe *Probe `bson:"livenessProbe,omitempty"`
	// ReadinessProbe 就绪探针配置
	ReadinessProbe *Probe `bson:"readinessProbe,omitempty"`
	// StartupProbe 启动探针配置
	StartupProbe *Probe `bson:"startupProbe,omitempty"`
	// Lifecycle 容器生命周期钩子配置
	Lifecycle *Lifecycle `bson:"lifecycle,omitempty"`

	// TerminationGracePeriodSeconds Pod 优雅终止超时时间（秒）
	TerminationGracePeriodSeconds *int64 `bson:"terminationGracePeriodSeconds,omitempty"`

	// TrpcConfig tRPC 配置, 仅当 Type 为 tRPC 时有效
	TrpcConfig TrpcConfig `bson:"trpcConfig,omitempty"`
	// TafConfig TAF 配置, 仅当 Type 为 TAF 时有效
	TafConfig TafConfig `bson:"tafConfig,omitempty"`
}

// UpdateStrategy 应用更新策略配置，更多详情查看 GameDeploymentUpdateStrategy
type UpdateStrategy struct {
	Type           string  `bson:"type"`
	MaxUnavailable *string `bson:"maxUnavailable,omitempty"`
	MaxSurge       *string `bson:"maxSurge,omitempty"`
}

// VolumeMounts 挂载卷配置
type VolumeMounts struct {
	// HostPath 存放 hostPath 类型卷挂载配置
	HostPath []VolumeMountHostPath `bson:"hostPath,omitempty"`
}

// VolumeMountHostPath 表示一个 hostPath 类型的卷挂载配置
type VolumeMountHostPath struct {
	// MountPath 容器内挂载路径
	MountPath string `bson:"mountPath"`
	// HostPath 宿主机路径
	HostPath string `bson:"hostPath"`
	// Type 是挂载类型，默认值为 "DirectoryOrCreate"
	Type string `bson:"type,omitempty"`
}

// Probe 是容器探针配置。
// 探针的结构体定义参考 Kubernetes 的 Probe 定义。
type Probe struct {
	// The action taken to determine the health of a container.
	ProbeHandler *ProbeHandler `bson:"probeHandler"`
	// Number of seconds after the container has started before liveness probes are initiated.
	InitialDelaySeconds int32 `bson:"initialDelaySeconds,omitempty"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `bson:"timeoutSeconds,omitempty"`
	// How often (in seconds) to perform the probe.
	PeriodSeconds int32 `bson:"periodSeconds,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	SuccessThreshold int32 `bson:"successThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	FailureThreshold int32 `bson:"failureThreshold,omitempty"`
}

// Lifecycle describes actions that the management system should take in response
// to container lifecycle events.
type Lifecycle struct {
	// PreStop is called immediately before a container is terminated due to an
	// API request or management event such as liveness/startup probe failure,
	// preemption, resource contention, etc.
	PreStop *LifecycleHandler `bson:"preStop,omitempty"`
	// PostStart is called immediately after a container is created.
	PostStart *LifecycleHandler `bson:"postStart,omitempty"`
}

// LifecycleHandler defines a specific action that should be taken in a lifecycle
// hook. One and only one of the fields, except TCPSocket must be specified.
type LifecycleHandler struct {
	// Type of action to be taken.
	TypeWrapper `bson:"_type"`
	// Exec specifies the action to take.
	// +optional
	*ExecAction `bson:",inline"`
	// HTTPGet specifies the http request to perform.
	// +optional
	*HTTPGetAction `bson:",inline"`
}

// TypeWrapper embeds the action type discriminator used by lifecycle handlers.
type TypeWrapper struct {
	// Type of action to be taken.
	Type string `bson:"_type"`
}

// ExecAction describes an action that executes a command inside the container.
type ExecAction struct {
	// Command is the command line to execute inside the container, the working directory for the
	// command  is root ('/') in the container's filesystem.
	// Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
	Command []string `bson:"command,omitempty"`
	// ShCommand is set for exec actions when using script mode (mutually exclusive with Command).
	ShCommand string `bson:"shCommand,omitempty"`
	// SleepSeconds indicates sleeping seconds after command execution.
	// When applied to workload, it is converted to a "sleep <n>" shell command.
	SleepSeconds *int64 `bson:"sleepSeconds,omitempty"`
}

// HTTPGetAction describes an action based on HTTP Get requests.
type HTTPGetAction struct {
	// URL is the full qualified url location to send HTTP requests.
	URL string `bson:"url,omitempty"`
	// Port is the port number to check (1~65535).
	Port int32 `bson:"httpPort,omitempty"`
	// Custom headers to set in the request. HTTP allows repeated headers.
	Headers map[string]string `bson:"headers,omitempty"`
}

// TCPSocketAction describes an action based on opening a socket.
type TCPSocketAction struct {
	// Port is the port number to check (1~65535).
	Port int32 `bson:"tcpPort,omitempty"`
}

// ProbeHandler defines a specific action that should be taken in a probe.
// One and only one of the fields must be specified.
type ProbeHandler struct {
	// Type of action to be taken.
	TypeWrapper `bson:"_type"`
	// Exec specifies the action to take.
	// +optional
	*ExecAction `bson:",inline"`
	// HTTPGet specifies the http request to perform.
	// +optional
	*HTTPGetAction `bson:",inline"`
	// TCPSocket specifies an action involving a TCP port.
	// +optional
	*TCPSocketAction `bson:",inline"`
}

// Variable represents a variable
type Variable struct {
	// Key 变量名
	Key string `bson:"key"`
	// Value 变量值
	Value string `bson:"value"`
	// Description 变量描述
	Description string `bson:"description,omitempty"`
	// IsSensitive 是否敏感
	IsSensitive bool `bson:"isSensitive,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// TrpcConfig represents tRPC config
// NOTE: 配置文件实际内容存储于 AppConfigFile 表, 通过 appID 字段关联, appID-envName 唯一确定一个配置文件,
//
//	当 AppConfigFile.EnvName = "" 时, 表示应用级别的默认配置;
//	当 AppConfigFile.EnvName != "" 时, 表示环境级别的配置, 默认为覆盖层配置，通过与默认配置合并得到最终的配置文件;
type TrpcConfig struct {
	// FileName tRPC 配置文件名
	FileName string `bson:"fileName"`
	// FilePath tRPC 配置文件在 workload 容器内的路径
	FilePath string `bson:"filePath"`
	// FileContent tRPC 配置文件内容
	// 配置内容已迁移到 AppConfigFile 存储，该字段仅用于计算过程中临时存储结果
	FileContent string `bson:"fileContent"`
	// Language tRPC 框架语言类型, 目前仅影响渲染出的配置文件后缀, 例如：trpc_go.yaml, trpc_cpp.yaml 等
	Language string `bson:"language"`
}

// TafConfig represents TAF config
// NOTE: 配置文件实际内容存储于 AppConfigFile 表, 通过 appID 字段关联
type TafConfig struct {
	// FileName TAF 配置文件名
	FileName string `bson:"fileName"`
	// FilePath TAF 配置文件在 workload 容器内的路径
	FilePath string `bson:"filePath"`
	// FileContent TAF 配置文件内容
	FileContent string `bson:"fileContent"`
}

// NormalizeComponents normalize the components properties to go native types
func (am *AppModel) NormalizeComponents() {
	for _, comp := range am.Components {
		comp.NormalizeProperties()
	}
}

// EnsureComponentsName ensure the components have names
func (am *AppModel) EnsureComponentsName() {
	for _, comp := range am.Components {
		comp.EnsureName()
	}
}
