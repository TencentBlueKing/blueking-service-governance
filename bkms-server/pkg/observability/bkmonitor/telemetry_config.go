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
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// serviceNameKey 服务名称
	// 蓝鲸监控规范，定义的服务名称标识符
	serviceNameKey = "service_name"
)

// TelemetryType 上报类型
type TelemetryType string

const (
	// TelemetryTypeOpenTelemetry OpenTelemetry
	TelemetryTypeOpenTelemetry TelemetryType = "opentelemetry"

	// TelemetryTypeGalileo Galileo
	TelemetryTypeGalileo TelemetryType = "galileo"

	// TelemetryTypeUnknown 未知
	TelemetryTypeUnknown TelemetryType = "unknown"
)

// TrpcServerConfig trpc 服务配置
type TrpcServerConfig struct {
	Server struct {
		App string `yaml:"app"`

		Server string `yaml:"server"`
	} `yaml:"server"`
}

// PluginsConfig 插件配置
type PluginsConfig struct {
	Plugins Plugins `yaml:"plugins"`
}

// Plugins 插件列表
type Plugins struct {
	Telemetry Telemetry `yaml:"telemetry"`
}

// Telemetry 遥测配置
// opentelemetry 和 galileo 二选一
type Telemetry struct {
	OpenTelemetry *OpenTelemetryConfig `yaml:"opentelemetry,omitempty"`

	Galileo *GalileoConfig `yaml:"galileo,omitempty"`
}

// GetTelemetryType 获取遥测上报类型
func (t *Telemetry) GetTelemetryType() TelemetryType {
	// 优先判断 galileo（因为 galileo 配置更完整时优先使用）
	if t.Galileo != nil && t.Galileo.OCPAddr != "" {
		return TelemetryTypeGalileo
	}
	// 判断 opentelemetry
	if t.OpenTelemetry != nil && len(t.OpenTelemetry.Attributes) > 0 && t.OpenTelemetry.Addr != "" &&
		t.OpenTelemetry.TenantID != "" {
		return TelemetryTypeOpenTelemetry
	}
	// 如果 galileo 有配置但 ocp_addr 为空，仍然认为是 galileo 类型
	if t.Galileo != nil {
		return TelemetryTypeGalileo
	}
	// 如果 opentelemetry 有配置
	if t.OpenTelemetry != nil {
		return TelemetryTypeOpenTelemetry
	}
	return TelemetryTypeUnknown
}

// IsOpenTelemetry 是否为 OpenTelemetry 类型
func (t *Telemetry) IsOpenTelemetry() bool {
	return t.GetTelemetryType() == TelemetryTypeOpenTelemetry
}

// IsGalileo 是否为 Galileo 类型
func (t *Telemetry) IsGalileo() bool {
	return t.GetTelemetryType() == TelemetryTypeGalileo
}

// ========== OpenTelemetry 配置 ==========

// OpenTelemetryConfig OpenTelemetry 配置
type OpenTelemetryConfig struct {
	Addr       string      `yaml:"addr"`
	TenantID   string      `yaml:"tenant_id"`
	Attributes []Attribute `yaml:"attributes,omitempty"`
}

// Attribute 属性键值对
type Attribute struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// ========== Galileo 配置 ==========

// GalileoConfig Galileo 配置
type GalileoConfig struct {
	OCPAddr  string          `yaml:"ocp_addr"`
	Config   GalileoDetail   `yaml:"config"`
	Resource GalileoResource `yaml:"resource"`
}

// GalileoDetail Galileo 详细配置
type GalileoDetail struct {
	AccessPoint    int            `yaml:"access_point"`
	MetricsConfig  MetricsConfig  `yaml:"metrics_config"`
	TracesConfig   TracesConfig   `yaml:"traces_config"`
	LogsConfig     LogsConfig     `yaml:"logs_config"`
	ProfilesConfig ProfilesConfig `yaml:"profiles_config"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enable   bool            `yaml:"enable"`
	Exporter MetricsExporter `yaml:"exporter,omitempty"`
}

// MetricsExporter 指标导出器
type MetricsExporter struct {
	Collector CollectorConfig `yaml:"collector"`
}

// TracesConfig 链路追踪配置
type TracesConfig struct {
	Enable   bool           `yaml:"enable"`
	Exporter TracesExporter `yaml:"exporter,omitempty"`
}

// TracesExporter 链路导出器
type TracesExporter struct {
	Collector CollectorConfig `yaml:"collector"`
}

// LogsConfig 日志配置
type LogsConfig struct {
	Enable bool `yaml:"enable"`
}

// ProfilesConfig 性能分析配置
type ProfilesConfig struct {
	Enable   bool             `yaml:"enable"`
	Exporter ProfilesExporter `yaml:"exporter,omitempty"`
}

// ProfilesExporter 性能分析导出器
type ProfilesExporter struct {
	Collector CollectorConfig `yaml:"collector"`
}

// CollectorConfig 收集器配置
type CollectorConfig struct {
	Addr string `yaml:"addr"`
}

// GalileoResource Galileo 资源配置
type GalileoResource struct {
	Platform string `yaml:"platform"`
	TenantID string `yaml:"tenant_id"`
}

// ========== 解析方法 ==========

// ParseTelemetryConfig 从 YAML 字节解析配置
func ParseTelemetryConfig(data []byte) (*Telemetry, error) {
	var config PluginsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshal telemetry config")
	}
	return &config.Plugins.Telemetry, nil
}

// ParseTrpcServerConfig 从 YAML 字节解析配置
func ParseTrpcServerConfig(data []byte) (*TrpcServerConfig, error) {
	var config TrpcServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrapf(err, "unmarshal trpc server config")
	}
	return &config, nil
}

// GetServiceName 获取服务名称
func (s *OpenTelemetryConfig) GetServiceName() (string, error) {
	for _, attribute := range s.Attributes {
		if attribute.Key == serviceNameKey {
			return attribute.Value, nil
		}
	}

	return "", errors.Wrapf(ErrAPMConfigMissing, "%s service name not found", TelemetryTypeOpenTelemetry)
}

// ========== TAF 配置 ==========

// TafConfig TAF 配置文件根结构（XML 格式）
// TAF 配置使用非标准 XML 格式，在 XML 标签内使用 key=value 格式存储配置项
type TafConfig struct {
	XMLName     xml.Name       `xml:"taf"`
	Application TafApplication `xml:"application"`
}

// TafApplication TAF application 节点
type TafApplication struct {
	// CharData 包含 setdivision 等 key=value 格式的配置项
	CharData string    `xml:",chardata"`
	Server   TafServer `xml:"server"`
}

// TafServer TAF server 节点
type TafServer struct {
	// CharData 包含 app、server 等 key=value 格式的配置项
	CharData string `xml:",chardata"`
}

// ParseTafConfig 从 XML 字节解析 TAF 配置
func ParseTafConfig(data []byte) (*TafConfig, error) {
	var config TafConfig
	if err := xml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshal taf config")
	}
	return &config, nil
}

// GetServiceName 从 TAF 配置中获取 APM 服务名称
// 拼接规则：{app}.{server}.{setdivision去除点号保留星号}
// setdivision 为可选字段，为空时拼接结果为 {app}.{server}
func (c *TafConfig) GetServiceName() (string, error) {
	serverKV := parseTafKeyValues(c.Application.Server.CharData)
	app := serverKV["app"]
	server := serverKV["server"]

	if app == "" || server == "" {
		return "", errors.Wrap(ErrAPMConfigMissing, "taf config missing app or server field")
	}

	appKV := parseTafKeyValues(c.Application.CharData)
	setDivision := appKV["setdivision"]

	if setDivision == "" {
		return fmt.Sprintf("%s.%s", app, server), nil
	}

	// 去除点号，保留星号
	setDivision = strings.ReplaceAll(setDivision, ".", "")
	return fmt.Sprintf("%s.%s.%s", app, server, setDivision), nil
}

// parseTafKeyValues 解析 TAF 配置中 key=value 格式的文本内容
func parseTafKeyValues(text string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
