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

package client

// --- AppSpec types ---

// AppSpecSectionName defines the section name type for AppSpec.
type AppSpecSectionName string

const (
	AppSpecSectionResources      AppSpecSectionName = "resources"
	AppSpecSectionUpdateStrategy AppSpecSectionName = "update-strategy"
	AppSpecSectionLifecycle      AppSpecSectionName = "lifecycle"
	AppSpecSectionProbe          AppSpecSectionName = "probe"
	AppSpecSectionLabels         AppSpecSectionName = "labels"
	AppSpecSectionAnnotations    AppSpecSectionName = "annotations"
)

// AppDetail represents the app detail response from GET /apps/:appID.
type AppDetail struct {
	Type         string        `json:"type"`
	AppModelSpec *AppModelSpec `json:"appModelSpec"`
}

// AppModelSpec represents the appModelSpec field in app detail.
type AppModelSpec struct {
	Command  []string  `json:"command"`
	Args     []string  `json:"args"`
	TrpcSpec *TrpcSpec `json:"trpcSpec,omitempty"`
	TafSpec  *TafSpec  `json:"tafSpec,omitempty"`
}

// TrpcSpec tRPC 框架配置
type TrpcSpec struct {
	Language    string `json:"language"`
	FileName    string `json:"fileName"`
	FilePath    string `json:"filePath"`
	FileContent string `json:"fileContent,omitempty"`
}

// TafSpec TAF 框架配置
type TafSpec struct {
	FileName    string `json:"fileName"`
	FilePath    string `json:"filePath"`
	FileContent string `json:"fileContent,omitempty"`
}

// GetAppDetailRespData is the response wrapper for GetAppDetail.
type GetAppDetailRespData struct {
	Data *AppDetail `json:"data"`
}

// --- Section output types (确定类型，不再使用 json.RawMessage) ---

// ResourcesConfig 资源规格配置
type ResourcesConfig struct {
	Replicas       *int32  `json:"replicas"`
	CPURequests    *string `json:"cpuRequests"`
	CPULimits      *string `json:"cpuLimits"`
	MemoryRequests *string `json:"memoryRequests"`
	MemoryLimits   *string `json:"memoryLimits"`
}

// UpdateStrategyConfig 更新策略配置
type UpdateStrategyConfig struct {
	MaxSurge       *string `json:"maxSurge"`
	MaxUnavailable *string `json:"maxUnavailable"`
}

// LifecycleConfig 生命周期配置
type LifecycleConfig struct {
	PostStart                     *LifecycleHandlerConfig `json:"postStart,omitempty"`
	PreStop                       *LifecycleHandlerConfig `json:"preStop,omitempty"`
	TerminationGracePeriodSeconds *string                 `json:"terminationGracePeriodSeconds,omitempty"`
}

// Sanitize 清理空值字段，将无效配置置为 nil。
func (c *LifecycleConfig) Sanitize() {
	if c.PostStart != nil && c.PostStart.Type == "" {
		c.PostStart = nil
	}
	if c.PreStop != nil && c.PreStop.Type == "" {
		c.PreStop = nil
	}
}

// LifecycleHandlerConfig 生命周期 handler 配置
type LifecycleHandlerConfig struct {
	Type         string            `json:"_type"`
	Command      []string          `json:"command,omitempty"`
	ShCommand    string            `json:"shCommand,omitempty"`
	SleepSeconds *string           `json:"sleepSeconds,omitempty"`
	URL          string            `json:"url,omitempty"`
	Port         int32             `json:"port,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ProbeConfig 探针配置
type ProbeConfig struct {
	Liveness  *ProbeItemConfig `json:"liveness,omitempty"`
	Readiness *ProbeItemConfig `json:"readiness,omitempty"`
	Startup   *ProbeItemConfig `json:"startup,omitempty"`
}

// ProbeItemConfig 单个探针配置
type ProbeItemConfig struct {
	Handler             *ProbeHandlerConfig `json:"probeHandler"`
	InitialDelaySeconds *int32              `json:"initialDelaySeconds"`
	TimeoutSeconds      *int32              `json:"timeoutSeconds"`
	PeriodSeconds       *int32              `json:"periodSeconds"`
	SuccessThreshold    *int32              `json:"successThreshold"`
	FailureThreshold    *int32              `json:"failureThreshold"`
}

// ProbeHandlerConfig 探针 handler 配置
type ProbeHandlerConfig struct {
	Type      string            `json:"_type"`
	Command   []string          `json:"command,omitempty"`
	ShCommand string            `json:"shCommand,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Port      int32             `json:"port,omitempty"`
}

// LabelsConfig 标签配置
type LabelsConfig struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// IsEmpty 判断是否为空配置。
func (c *LabelsConfig) IsEmpty() bool {
	return len(c.Labels) == 0
}

// AnnotationsConfig 注解配置
type AnnotationsConfig struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

// IsEmpty 判断是否为空配置。
func (c *AnnotationsConfig) IsEmpty() bool {
	return len(c.Annotations) == 0
}
