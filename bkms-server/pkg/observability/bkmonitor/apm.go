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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"fmt"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
)

var (
	// ErrAPMConfigMissing 表示 APM 相关配置缺失（如未配置 telemetry、缺少 service_name 等）
	ErrAPMConfigMissing = errors.New("apm config missing")

	// ErrAPMConfigParse 表示 APM 配置解析或渲染失败
	ErrAPMConfigParse = errors.New("apm config parse error")
)

const (
	// ApmTypeCustom 自定义 APM
	// 其他 APM 类型，在 env 中(pkg/core/env/constants.go) ，与环境类型相同，因此，不再重复定义。
	ApmTypeCustom = "custom"
)

// GetApmServiceName 根据应用类型获取 APM 服务名称（统一入口）
func GetApmServiceName(appType, content string, appEnvVars map[string]string) (string, error) {
	switch appType {
	case bkmsapp.AppTypeTRPC:
		return GetTrpcApmServiceName(content, appEnvVars)
	case bkmsapp.AppTypeTAF:
		return GetTafApmServiceName(content, appEnvVars)
	default:
		return "", errors.Errorf("unsupported app type %q for apm service name", appType)
	}
}

// GetTrpcApmServiceName 获取 trpc 应用的 APM 服务名称
// trpc 配置使用 YAML 格式，变量渲染使用 ExpandShellVars（Shell 变量格式 ${VAR}）
// 1、解析 telemetry 配置
// 2、如果是 OpenTelemetry 类型, 直接获取服务名称，并返回给调用者
// 3、如果是 伽利略 类型, 解析 trpc 服务配置, 获取服务名称，并返回给调用者
func GetTrpcApmServiceName(content string, appEnvVars map[string]string) (string, error) {
	contentByte := []byte(render.RenderShellVars(content, appEnvVars))

	telemetryConfig, err := ParseTelemetryConfig(contentByte)
	if err != nil {
		return "", errors.Wrapf(ErrAPMConfigParse, "parse telemetry config: %s", err)
	}
	if telemetryConfig.GetTelemetryType() == TelemetryTypeUnknown {
		return "", errors.Wrap(ErrAPMConfigMissing, "telemetry config type unknown")
	}
	if telemetryConfig.IsOpenTelemetry() {
		return telemetryConfig.OpenTelemetry.GetServiceName()
	}

	trpcServerConfig, err := ParseTrpcServerConfig(contentByte)
	if err != nil {
		return "", errors.Wrapf(ErrAPMConfigParse, "parse trpc server config: %s", err)
	}
	if trpcServerConfig.Server.App == "" {
		return "", errors.Wrap(ErrAPMConfigMissing, "trpc server config app empty")
	}
	if trpcServerConfig.Server.Server == "" {
		return "", errors.Wrap(ErrAPMConfigMissing, "trpc server config server empty")
	}

	return fmt.Sprintf("%s.%s", trpcServerConfig.Server.App, trpcServerConfig.Server.Server), nil
}

// GetTafApmServiceName 获取 taf 应用的 APM 服务名称
// taf 配置使用 XML 格式，变量渲染使用 Renderer（支持 ${{env.KEY}}）
func GetTafApmServiceName(content string, appEnvVars map[string]string) (string, error) {
	rendered, err := render.New(render.SetEnvContext(appEnvVars)).Render(content)
	if err != nil {
		return "", errors.Wrapf(ErrAPMConfigParse, "render taf config: %s", err)
	}

	tafConfig, err := ParseTafConfig([]byte(rendered))
	if err != nil {
		return "", errors.Wrapf(ErrAPMConfigParse, "parse taf config: %s", err)
	}
	return tafConfig.GetServiceName()
}
