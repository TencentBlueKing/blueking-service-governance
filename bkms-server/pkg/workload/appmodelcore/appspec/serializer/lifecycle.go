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
	"strconv"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	lifecyclesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/lifecycle"
)

// AppSpecLifecycleOutput is the JSON representation of the lifecycle section.
type AppSpecLifecycleOutput struct {
	// 容器启动后执行的钩子
	PostStart *LifecycleHandlerOutput `json:"postStart"`
	// 容器终止前执行的钩子
	PreStop *LifecycleHandlerOutput `json:"preStop"`
	// Pod 优雅终止超时时间（秒）
	TerminationGracePeriodSeconds *string `json:"terminationGracePeriodSeconds"`
}

// FromModel fills output fields from an AppSpec lifecycle section.
func (o *AppSpecLifecycleOutput) FromModel(spec *appspec.LifecycleSpec) *AppSpecLifecycleOutput {
	if spec == nil {
		return nil
	}
	*o = AppSpecLifecycleOutput{
		PostStart: new(LifecycleHandlerOutput).FromModel(spec.PostStart),
		PreStop:   new(LifecycleHandlerOutput).FromModel(spec.PreStop),
	}
	if spec.TerminationGracePeriodSeconds != nil {
		// 非标准行为：保持 proto JSON 兼容，int64 字段在响应里输出为字符串。
		o.TerminationGracePeriodSeconds = formatInt64Ptr(spec.TerminationGracePeriodSeconds)
	}
	return o
}

// LifecycleHandlerOutput is the JSON representation of a lifecycle handler.
type LifecycleHandlerOutput struct {
	// 处理器类型：EXEC 或 HTTP
	Type string `json:"type"`
	// Exec 命令配置，type 为 "EXEC" 时存在
	Exec *LifecycleExecActionOutput `json:"exec"`
	// HTTP 请求配置，type 为 "HTTP" 时存在
	HTTP *LifecycleHTTPGetActionOutput `json:"http"`
}

// FromModel fills output fields from a lifecycle handler.
func (o *LifecycleHandlerOutput) FromModel(h *lifecyclesection.Handler) *LifecycleHandlerOutput {
	if h == nil {
		return nil
	}
	*o = LifecycleHandlerOutput{Type: h.Type}
	switch h.Type {
	case "EXEC":
		o.Exec = &LifecycleExecActionOutput{
			Command:   h.Command,
			ShCommand: h.ShCommand,
			// 非标准行为：保持 proto JSON 兼容，int64 字段在响应里输出为字符串。
			SleepSeconds: formatInt64Ptr(h.SleepSeconds),
		}
	case "HTTP":
		o.HTTP = &LifecycleHTTPGetActionOutput{
			URL:     h.URL,
			Headers: h.Headers,
		}
	}
	return o
}

// LifecycleExecActionOutput is the JSON representation of an exec lifecycle action.
type LifecycleExecActionOutput struct {
	// 在容器内执行的命令，与 shCommand 二选一，表示命令使用 exec 模式调用
	Command []string `json:"command"`
	// shCommand 与 command 二选一的脚本正文，以 /bin/sh -c 方式执行
	ShCommand string `json:"shCommand"`
	// 睡眠等待时间（秒），可选。应用到 workload 时转换为 sleep 命令
	SleepSeconds *string `json:"sleepSeconds"`
}

// LifecycleHTTPGetActionOutput is the JSON representation of an HTTP lifecycle action.
type LifecycleHTTPGetActionOutput struct {
	// 发送 HTTP GET 请求的完整 URL
	URL string `json:"url"`
	// 自定义请求头
	Headers map[string]string `json:"headers"`
}

// AppSpecLifecycleInput is the input structure of the lifecycle section.
type AppSpecLifecycleInput struct {
	// 容器启动后执行的钩子
	PostStart *LifecycleHandlerInput `json:"postStart"`
	// 容器终止前执行的钩子
	PreStop *LifecycleHandlerInput `json:"preStop"`
	// Pod 优雅终止超时时间（秒），必须 >= 0
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds" binding:"omitempty,gte=0"`
}

// ToModel converts input to an AppSpec lifecycle section.
func (i *AppSpecLifecycleInput) ToModel() *appspec.LifecycleSpec {
	if i == nil {
		return nil
	}
	return &appspec.LifecycleSpec{
		PostStart:                     i.PostStart.ToModel(),
		PreStop:                       i.PreStop.ToModel(),
		TerminationGracePeriodSeconds: i.TerminationGracePeriodSeconds,
	}
}

// EnvAppSpecLifecycleInput is the env-scoped input structure of the lifecycle section.
type EnvAppSpecLifecycleInput = AppSpecLifecycleInput

// LifecycleHandlerInput is the input structure of a lifecycle handler.
type LifecycleHandlerInput struct {
	// 处理器类型：EXEC 或 HTTP
	Type string `json:"type" binding:"required,oneof=EXEC HTTP"`
	// Exec 命令配置，type 为 "EXEC" 时需要提供
	Exec *LifecycleExecActionInput `json:"exec"`
	// HTTP 请求配置，type 为 "HTTP" 时需要提供
	HTTP *LifecycleHTTPGetActionInput `json:"http"`
}

// ToModel converts input to a lifecycle handler.
func (i *LifecycleHandlerInput) ToModel() *lifecyclesection.Handler {
	if i == nil {
		return nil
	}
	handler := &lifecyclesection.Handler{Type: i.Type}
	switch i.Type {
	case "EXEC":
		if i.Exec != nil {
			handler.Command = i.Exec.Command
			handler.ShCommand = i.Exec.ShCommand
			handler.SleepSeconds = i.Exec.SleepSeconds
		}
	case "HTTP":
		if i.HTTP != nil {
			handler.URL = i.HTTP.URL
			handler.Headers = i.HTTP.Headers
		}
	}
	return handler
}

// LifecycleExecActionInput is the input structure of an exec lifecycle action.
type LifecycleExecActionInput struct {
	// 在容器内执行的命令，可选；与 shCommand 二选一，表示命令使用 exec 模式调用
	Command []string `json:"command"`
	// shCommand 与 command 二选一的脚本正文，以 /bin/sh -c 方式执行
	ShCommand string `json:"shCommand"`
	// 睡眠等待时间（秒），可选，必须 >= 0。应用到 workload 时转换为 sleep 命令
	SleepSeconds *int64 `json:"sleepSeconds" binding:"omitempty,gte=0"`
}

// LifecycleHTTPGetActionInput is the input structure of an HTTP lifecycle action.
type LifecycleHTTPGetActionInput struct {
	// 发送 HTTP GET 请求的完整 URL
	URL string `json:"url"`
	// 自定义请求头，可选
	Headers map[string]string `json:"headers"`
}

// SetAppDefaultAppSpecLifecycleInput is the JSON body for setting default lifecycle.
type SetAppDefaultAppSpecLifecycleInput struct {
	// 待设置的 lifecycle section 值
	AppSpecLifecycle *AppSpecLifecycleInput `json:"appSpecLifecycle" binding:"required"`
}

// SetEnvAppSpecLifecycleInput is the JSON body for setting env lifecycle.
type SetEnvAppSpecLifecycleInput struct {
	// 待设置的 lifecycle section 值
	AppSpecLifecycle *EnvAppSpecLifecycleInput `json:"appSpecLifecycle" binding:"required"`
}

// AppSpecLifecycleSectionOutput is the JSON response for querying lifecycle.
type AppSpecLifecycleSectionOutput struct {
	Data *AppSpecLifecycleOutput `json:"data"`
}

func formatInt64Ptr(value *int64) *string {
	if value == nil {
		return nil
	}
	formatted := strconv.FormatInt(*value, 10)
	return &formatted
}
