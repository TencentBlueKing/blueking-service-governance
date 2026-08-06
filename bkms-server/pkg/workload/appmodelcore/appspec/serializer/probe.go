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

package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
)

// AppSpecProbeOutput is the JSON representation of the probe section.
type AppSpecProbeOutput struct {
	// 存活探针配置
	Liveness *ProbeOutput `json:"liveness"`
	// 就绪探针配置
	Readiness *ProbeOutput `json:"readiness"`
	// 启动探针配置
	Startup *ProbeOutput `json:"startup"`
}

// FromModel fills output fields from an AppSpec probe section.
func (o *AppSpecProbeOutput) FromModel(spec *appspec.ProbeSpec) *AppSpecProbeOutput {
	if spec == nil {
		return nil
	}
	*o = AppSpecProbeOutput{
		Liveness:  new(ProbeOutput).FromModel(spec.Liveness),
		Readiness: new(ProbeOutput).FromModel(spec.Readiness),
		Startup:   new(ProbeOutput).FromModel(spec.Startup),
	}
	return o
}

// ProbeOutput is the JSON representation of one probe.
type ProbeOutput struct {
	// 探针处理器
	ProbeHandler *ProbeHandlerOutput `json:"probeHandler"`
	// 容器启动后延迟探测的秒数
	InitialDelaySeconds int32 `json:"initialDelaySeconds"`
	// 探针超时秒数
	TimeoutSeconds int32 `json:"timeoutSeconds"`
	// 探测频率（秒）
	PeriodSeconds int32 `json:"periodSeconds"`
	// 连续成功次数阈值
	SuccessThreshold int32 `json:"successThreshold"`
	// 连续失败次数阈值
	FailureThreshold int32 `json:"failureThreshold"`
}

// FromModel fills output fields from one probe.
func (o *ProbeOutput) FromModel(p *probesection.Probe) *ProbeOutput {
	if p == nil {
		return nil
	}
	*o = ProbeOutput{
		ProbeHandler: new(ProbeHandlerOutput).FromModel(p.Handler),
	}
	if p.InitialDelaySeconds != nil {
		o.InitialDelaySeconds = *p.InitialDelaySeconds
	}
	if p.TimeoutSeconds != nil {
		o.TimeoutSeconds = *p.TimeoutSeconds
	}
	if p.PeriodSeconds != nil {
		o.PeriodSeconds = *p.PeriodSeconds
	}
	if p.SuccessThreshold != nil {
		o.SuccessThreshold = *p.SuccessThreshold
	}
	if p.FailureThreshold != nil {
		o.FailureThreshold = *p.FailureThreshold
	}
	return o
}

// ProbeHandlerOutput is the JSON representation of a probe handler.
type ProbeHandlerOutput struct {
	// 处理器类型：EXEC、HTTP 或 TCP
	Type string `json:"type"`
	// 执行命令（仅当 type=EXEC 时有效；与 shCommand 二选一，表示命令使用 exec 模式调用）
	Command []string `json:"command"`
	// HTTP 请求 URL（仅当 type=HTTP 时有效）
	URL string `json:"url"`
	// HTTP 请求头（仅当 type=HTTP 时有效）
	Headers map[string]string `json:"headers"`
	// 检查端口（type=HTTP 或 type=TCP 时有效，取值 1~65535）
	Port int32 `json:"port"`
	// shCommand 与 command 二选一的脚本正文（仅当 type=EXEC 时有效；以 /bin/sh -c 方式执行）
	ShCommand string `json:"shCommand"`
}

// FromModel fills output fields from a probe handler.
func (o *ProbeHandlerOutput) FromModel(h *probesection.Handler) *ProbeHandlerOutput {
	if h == nil {
		return nil
	}
	*o = ProbeHandlerOutput{
		Type:      h.Type,
		Command:   h.Command,
		ShCommand: h.ShCommand,
		URL:       h.URL,
		Headers:   h.Headers,
		Port:      h.Port,
	}
	return o
}

// AppSpecProbeInput is the input structure of the probe section.
type AppSpecProbeInput struct {
	// 存活探针配置
	Liveness *ProbeInput `json:"liveness"`
	// 就绪探针配置
	Readiness *ProbeInput `json:"readiness"`
	// 启动探针配置
	Startup *ProbeInput `json:"startup"`
}

// ToModel converts input to an AppSpec probe section.
func (i *AppSpecProbeInput) ToModel() *appspec.ProbeSpec {
	if i == nil {
		return nil
	}
	return &appspec.ProbeSpec{
		Liveness:  i.Liveness.ToModel(),
		Readiness: i.Readiness.ToModel(),
		Startup:   i.Startup.ToModel(),
	}
}

// EnvAppSpecProbeInput is the env-scoped input structure of the probe section.
type EnvAppSpecProbeInput = AppSpecProbeInput

// ProbeInput is the input structure of one probe.
type ProbeInput struct {
	// 探针处理器
	ProbeHandler *ProbeHandlerInput `json:"probeHandler" binding:"required"`
	// 容器启动后延迟探测的秒数
	InitialDelaySeconds *int32 `json:"initialDelaySeconds" binding:"omitempty,gte=0"`
	// 探针超时秒数
	TimeoutSeconds *int32 `json:"timeoutSeconds" binding:"omitempty,gte=0"`
	// 探测频率（秒）
	PeriodSeconds *int32 `json:"periodSeconds" binding:"omitempty,gte=0"`
	// 连续成功次数阈值
	SuccessThreshold *int32 `json:"successThreshold" binding:"omitempty,gte=0"`
	// 连续失败次数阈值
	FailureThreshold *int32 `json:"failureThreshold" binding:"omitempty,gte=0"`
}

// ToModel converts input to one probe.
func (i *ProbeInput) ToModel() *probesection.Probe {
	if i == nil {
		return nil
	}
	return &probesection.Probe{
		Handler:             i.ProbeHandler.ToModel(),
		InitialDelaySeconds: i.InitialDelaySeconds,
		TimeoutSeconds:      i.TimeoutSeconds,
		PeriodSeconds:       i.PeriodSeconds,
		SuccessThreshold:    i.SuccessThreshold,
		FailureThreshold:    i.FailureThreshold,
	}
}

// ProbeHandlerInput is the input structure of a probe handler.
type ProbeHandlerInput struct {
	// 处理器类型：EXEC、HTTP 或 TCP
	Type string `json:"type" binding:"required,oneof=EXEC HTTP TCP"`
	// 执行命令（仅当 type=EXEC 时有效；与 shCommand 二选一，表示命令使用 exec 模式调用）
	Command []string `json:"command"`
	// HTTP 请求 URL（仅当 type=HTTP 时有效）
	URL string `json:"url"`
	// HTTP 请求头（仅当 type=HTTP 时有效）
	Headers map[string]string `json:"headers"`
	// 检查端口（type=HTTP 或 type=TCP 时有效，取值 1~65535）
	Port int32 `json:"port"`
	// shCommand 与 command 二选一的脚本正文（仅当 type=EXEC 时有效；以 /bin/sh -c 方式执行）
	ShCommand string `json:"shCommand"`
}

// ToModel converts input to a probe handler.
func (i *ProbeHandlerInput) ToModel() *probesection.Handler {
	if i == nil {
		return nil
	}
	return &probesection.Handler{
		Type:      i.Type,
		Command:   i.Command,
		ShCommand: i.ShCommand,
		URL:       i.URL,
		Headers:   i.Headers,
		Port:      i.Port,
	}
}

// SetAppDefaultAppSpecProbeInput is the JSON body for setting default probes.
type SetAppDefaultAppSpecProbeInput struct {
	// 待设置的 probe section 值
	AppSpecProbe *AppSpecProbeInput `json:"appSpecProbe" binding:"required"`
}

// SetEnvAppSpecProbeInput is the JSON body for setting env probes.
type SetEnvAppSpecProbeInput struct {
	// 待设置的 probe section 值
	AppSpecProbe *EnvAppSpecProbeInput `json:"appSpecProbe" binding:"required"`
}

// AppSpecProbeSectionOutput is the JSON response for querying probes.
type AppSpecProbeSectionOutput struct {
	Data *AppSpecProbeOutput `json:"data"`
}
