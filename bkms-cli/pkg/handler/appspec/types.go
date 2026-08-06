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

package appspec

import "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"

// ResourcesInput 资源规格 YAML 输入结构
type ResourcesInput struct {
	Replicas       *int32  `yaml:"replicas" json:"replicas"`
	CPURequests    *string `yaml:"cpuRequests" json:"cpuRequests"`
	CPULimits      *string `yaml:"cpuLimits" json:"cpuLimits"`
	MemoryRequests *string `yaml:"memoryRequests" json:"memoryRequests"`
	MemoryLimits   *string `yaml:"memoryLimits" json:"memoryLimits"`
}

// UpdateStrategyInput 更新策略 YAML 输入结构
type UpdateStrategyInput struct {
	MaxSurge       *string `yaml:"maxSurge" json:"maxSurge"`
	MaxUnavailable *string `yaml:"maxUnavailable" json:"maxUnavailable"`
}

// LifecycleInput 生命周期 YAML 输入结构
type LifecycleInput struct {
	PostStart                     *LifecycleHandlerInput `yaml:"postStart" json:"postStart"`
	PreStop                       *LifecycleHandlerInput `yaml:"preStop" json:"preStop"`
	TerminationGracePeriodSeconds *int64                 `yaml:"terminationGracePeriodSeconds" json:"terminationGracePeriodSeconds"`
}

// LifecycleHandlerInput 生命周期 handler 输入结构（匹配服务端嵌套格式）
type LifecycleHandlerInput struct {
	Type string                       `yaml:"type" json:"type"`
	Exec *LifecycleExecActionInput    `yaml:"exec,omitempty" json:"exec,omitempty"`
	HTTP *LifecycleHTTPGetActionInput `yaml:"http,omitempty" json:"http,omitempty"`
}

// LifecycleExecActionInput Exec 动作输入
type LifecycleExecActionInput struct {
	Command      []string `yaml:"command,omitempty" json:"command,omitempty"`
	ShCommand    string   `yaml:"shCommand,omitempty" json:"shCommand,omitempty"`
	SleepSeconds *int64   `yaml:"sleepSeconds,omitempty" json:"sleepSeconds,omitempty"`
}

// LifecycleHTTPGetActionInput HTTP 动作输入
type LifecycleHTTPGetActionInput struct {
	URL     string            `yaml:"url,omitempty" json:"url,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

// ProbeInput 探针 YAML 输入结构
type ProbeInput struct {
	Liveness  *ProbeItemInput `yaml:"liveness" json:"liveness"`
	Readiness *ProbeItemInput `yaml:"readiness" json:"readiness"`
	Startup   *ProbeItemInput `yaml:"startup" json:"startup"`
}

// ProbeItemInput 单个探针输入结构
type ProbeItemInput struct {
	Handler             *ProbeHandlerInput `yaml:"handler" json:"probeHandler"`
	InitialDelaySeconds *int32             `yaml:"initialDelaySeconds" json:"initialDelaySeconds"`
	TimeoutSeconds      *int32             `yaml:"timeoutSeconds" json:"timeoutSeconds"`
	PeriodSeconds       *int32             `yaml:"periodSeconds" json:"periodSeconds"`
	SuccessThreshold    *int32             `yaml:"successThreshold" json:"successThreshold"`
	FailureThreshold    *int32             `yaml:"failureThreshold" json:"failureThreshold"`
}

// ProbeHandlerInput 探针 handler 输入结构
type ProbeHandlerInput struct {
	Type      string            `yaml:"type" json:"type"`
	Command   []string          `yaml:"command,omitempty" json:"command,omitempty"`
	ShCommand string            `yaml:"shCommand,omitempty" json:"shCommand,omitempty"`
	URL       string            `yaml:"url,omitempty" json:"url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Port      int32             `yaml:"port,omitempty" json:"port,omitempty"`
}

// LabelsInput 标签 YAML 输入结构
type LabelsInput struct {
	Labels map[string]string `yaml:"labels" json:"labels"`
}

// AnnotationsInput 注解 YAML 输入结构
type AnnotationsInput struct {
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

// --- API 请求体封装 ---

// SetDefaultResourcesRequest 设置默认资源规格的请求体
type SetDefaultResourcesRequest struct {
	AppSpecResources *ResourcesInput `json:"appSpecResources"`
}

// SetDefaultUpdateStrategyRequest 设置默认更新策略的请求体
type SetDefaultUpdateStrategyRequest struct {
	AppSpecUpdateStrategy *UpdateStrategyInput `json:"appSpecUpdateStrategy"`
}

// SetDefaultLifecycleRequest 设置默认生命周期配置的请求体
type SetDefaultLifecycleRequest struct {
	AppSpecLifecycle *LifecycleInput `json:"appSpecLifecycle"`
}

// SetDefaultProbeRequest 设置默认探针配置的请求体
type SetDefaultProbeRequest struct {
	AppSpecProbe *ProbeInput `json:"appSpecProbe"`
}

// SetDefaultLabelsRequest 设置默认标签的请求体
type SetDefaultLabelsRequest struct {
	AppSpecLabels *LabelsInput `json:"appSpecLabels"`
}

// SetDefaultAnnotationsRequest 设置默认注解的请求体
type SetDefaultAnnotationsRequest struct {
	AppSpecAnnotations *AnnotationsInput `json:"appSpecAnnotations"`
}

// --- Start command request types ---

// UpdateStartCommandRequest is the request body for updating start command.
type UpdateStartCommandRequest struct {
	AppModelSpec *StartCommandInput `json:"appModelSpec"`
}

// StartCommandInput holds the command, args and spec for the start command update.
type StartCommandInput struct {
	Command  []string         `json:"command" yaml:"command"`
	Args     []string         `json:"args" yaml:"args"`
	TrpcSpec *client.TrpcSpec `json:"trpcSpec,omitempty" yaml:"trpcSpec,omitempty"`
	TafSpec  *client.TafSpec  `json:"tafSpec,omitempty" yaml:"tafSpec,omitempty"`
}

// ViewAllResult holds the aggregated result of all sections for the view command.
type ViewAllResult struct {
	StartCommand   *StartCommandOutput          `json:"startCommand"`
	Lifecycle      *client.LifecycleConfig      `json:"lifecycle"`
	Probe          *client.ProbeConfig          `json:"probe"`
	Resources      *client.ResourcesConfig      `json:"resources"`
	UpdateStrategy *client.UpdateStrategyConfig `json:"updateStrategy"`
	Labels         *client.LabelsConfig         `json:"labels"`
	Annotations    *client.AnnotationsConfig    `json:"annotations"`
}

func (r *ViewAllResult) setSection(section client.AppSpecSectionName, data any) {
	switch section {
	case client.AppSpecSectionLifecycle:
		r.Lifecycle = data.(*client.LifecycleConfig)
	case client.AppSpecSectionProbe:
		r.Probe = data.(*client.ProbeConfig)
	case client.AppSpecSectionResources:
		r.Resources = data.(*client.ResourcesConfig)
	case client.AppSpecSectionUpdateStrategy:
		r.UpdateStrategy = data.(*client.UpdateStrategyConfig)
	case client.AppSpecSectionLabels:
		r.Labels = data.(*client.LabelsConfig)
	case client.AppSpecSectionAnnotations:
		r.Annotations = data.(*client.AnnotationsConfig)
	}
}
