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

// Package metrics 提供基于 Prometheus 的指标定义、注册和暴露能力
//
// 该包包含两类指标：Gin 入站请求指标和外部系统客户端调用指标；业务侧只通过 Report* 或便捷函数上报，
// 不直接接触 Prometheus collector，避免标签维度和注册行为散落在各业务包中
package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// StatusOK 调用成功
	StatusOK = "ok"
	// StatusErr 调用失败
	StatusErr = "err"
)

// 使用 promauto 自动注册指标，无需手动调用 prometheus.MustRegister
//
// 指标定义集中在包级变量中，进程启动后由 Prometheus 默认 registry 持有；不要在请求路径中动态创建 collector，
// 避免重复注册和高基数标签导致 metrics endpoint 不稳定
var (
	// clientRequestTotal 记录 bkms-server 调用外部系统的结果总数
	// 标签 system 表示外部系统名称，handler 表示外部接口或操作名称，status 仅允许 ok/err 两类
	clientRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "client",
			Name:      "request_total",
			Help:      "Total number of requests from bkms-server to external systems.",
		},
		[]string{"system", "handler", "status"},
	)

	// clientRequestLatency 记录 bkms-server 调用外部系统的耗时分布
	// 标签必须与 clientRequestTotal 的 system、handler 语义保持一致，便于按系统和接口聚合延迟
	clientRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "client",
			Name:      "request_latency_seconds",
			Help:      "Request latency in seconds for bkms-server calling external systems.",
			Buckets:   []float64{0.01, 0.1, 0.5, 0.75, 1.0, 2.0, 3.0, 5.0, 10.0},
		},
		[]string{"system", "handler"},
	)

	// serverRequestTotal 记录 Gin 入站请求的响应总数
	// handler 使用 Gin 路由模板，method 使用 HTTP method，status_code 使用最终 HTTP 状态码字符串
	serverRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "request_total",
			Help:      "Total number of inbound requests to bkms-server.",
		},
		[]string{"handler", "method", "status_code"},
	)

	// serverRequestLatency 记录 Gin 入站请求的耗时分布
	// handler 与 method 标签应和 serverRequestTotal 保持一致，状态码只放在 total 指标中以控制延迟指标基数
	serverRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "request_latency_seconds",
			Help:      "Inbound request latency in seconds for bkms-server.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"handler", "method"},
	)

	// CreateEnvFailure 创建环境失败计数
	CreateEnvFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "create_env_failure",
			Help:      "Create environment failure count.",
		},
		[]string{"workspace_id", "env_name"},
	)

	// CreateApmFailure APM 应用创建/获取失败计数
	CreateApmFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "create_apm_failure",
			Help:      "Create APM app failure count.",
		},
		[]string{"workspace_id", "env_name", "detail"},
	)

	// BindApmFailure APM 与环境绑定失败计数
	BindApmFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "bind_apm_failure",
			Help:      "Bind APM to environment failure count.",
		},
		[]string{"workspace_id", "env_name", "detail"},
	)

	// portForwardSessionTotal 记录端口转发会话结果总数。
	portForwardSessionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "port_forward",
			Name:      "session_total",
			Help:      "Total number of port-forward sessions, labeled by outcome (success/failure).",
		},
		[]string{"outcome"},
	)

	// portForwardSessionDuration 记录端口转发会话耗时。
	portForwardSessionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "port_forward",
			Name:      "session_duration_seconds",
			Help:      "Port-forward session duration in seconds.",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"outcome"},
	)

	// portForwardActiveSessions 记录当前活跃端口转发会话数。
	portForwardActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "bkms",
			Subsystem: "port_forward",
			Name:      "active_sessions",
			Help:      "Current number of active port-forward sessions.",
		},
	)
)

// ReportClientRequestMetric 用于 defer 场景下上报 infras 层 API 调用指标
// errp 使用指针以支持延迟求值，确保 defer 时能读取到函数返回时的最终错误值
func ReportClientRequestMetric(system, handler string, started time.Time, errp *error) {
	status := StatusOK
	if *errp != nil {
		status = StatusErr
	}

	clientRequestTotal.WithLabelValues(system, handler, status).Inc()
	clientRequestLatency.WithLabelValues(system, handler).Observe(time.Since(started).Seconds())
}

// ReportServerRequestMetric 记录入站 API 请求指标
func ReportServerRequestMetric(handler, method string, statusCode int, started time.Time) {
	code := strconv.Itoa(statusCode)
	serverRequestTotal.WithLabelValues(handler, method, code).Inc()
	serverRequestLatency.WithLabelValues(handler, method).Observe(time.Since(started).Seconds())
}

// CreateEnvFailed 记录创建环境失败
func CreateEnvFailed(workspaceID, envName string) {
	CreateEnvFailure.WithLabelValues(workspaceID, envName).Inc()
}

// CreateEnvApmFailed 记录创建 APM 应用失败
func CreateEnvApmFailed(workspaceID, envName, detail string) {
	CreateApmFailure.WithLabelValues(workspaceID, envName, detail).Inc()
}

// BindEnvApmFailed 记录 APM 与环境绑定失败
func BindEnvApmFailed(workspaceID, envName, detail string) {
	BindApmFailure.WithLabelValues(workspaceID, envName, detail).Inc()
}

// PortForwardSessionStarted 记录端口转发会话开始。
func PortForwardSessionStarted() {
	portForwardActiveSessions.Inc()
}

// PortForwardSessionFinished 记录端口转发会话结束。
func PortForwardSessionFinished(started time.Time, outcome string) {
	if outcome == "" {
		outcome = StatusErr
	}
	portForwardActiveSessions.Dec()
	portForwardSessionTotal.WithLabelValues(outcome).Inc()
	portForwardSessionDuration.WithLabelValues(outcome).Observe(time.Since(started).Seconds())
}
