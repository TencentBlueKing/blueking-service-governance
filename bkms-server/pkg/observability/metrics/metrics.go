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
// 该包包含三类指标：Gin 入站请求指标、外部系统客户端调用指标以及少量低基数业务指标。
// 业务侧只通过本包导出的便捷函数上报，避免在各业务包散落 Prometheus collector、标签约束和状态映射逻辑。
package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/samber/lo"
)

const (
	// StatusOK 调用成功
	StatusOK = "ok"
	// StatusErr 调用失败
	StatusErr = "err"
	// StatusFail 二元结果中的失败（与 StatusErr 区分，用于 result=ok|fail）
	StatusFail = "fail"
	// StatusTimeout 调用超时/轮询超时
	StatusTimeout = "timeout"
	// StatusCancelled 调用被取消
	StatusCancelled = "cancelled"

	// DeployKindAppModel AppModel 应用部署
	DeployKindAppModel = "appmodel"
	// DeployKindHelm Helm 应用部署
	DeployKindHelm = "helm"
	// DeployKindUnknown 未知部署类型
	DeployKindUnknown = "unknown"

	// ClusterAddonOperationDeploy 集群插件部署/升级
	ClusterAddonOperationDeploy = "deploy"
	// ClusterAddonOperationUninstall 集群插件卸载
	ClusterAddonOperationUninstall = "uninstall"
	// ClusterAddonOperationUnknown 未知集群插件操作
	ClusterAddonOperationUnknown = "unknown"

	// ScaleDirectionUp 扩容
	ScaleDirectionUp = "up"
	// ScaleDirectionDown 缩容
	ScaleDirectionDown = "down"
	// ScaleDirectionSame 副本数不变
	ScaleDirectionSame = "same"

	// WatchEventTypeUnknown 无法识别的 Watch 事件类型，防止标签基数膨胀
	WatchEventTypeUnknown = "unknown"
)

// 异步任务类耗时指标共用的 Bucket（单位：秒）
var durationBuckets = []float64{5, 15, 30, 60, 120, 300, 600, 1800}

// 使用 promauto 自动注册指标，无需手动调用 prometheus.MustRegister。
var (
	// clientRequestTotal 记录 bkms-server 调用外部系统的结果总数。
	clientRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "client",
			Name:      "request_total",
			Help:      "Total number of requests from bkms-server to external systems.",
		},
		[]string{"system", "handler", "status"},
	)

	// clientRequestLatency 记录 bkms-server 调用外部系统的耗时分布。
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

	// serverRequestTotal 记录 Gin 入站请求的响应总数。
	serverRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "server",
			Name:      "request_total",
			Help:      "Total number of inbound requests to bkms-server.",
		},
		[]string{"handler", "method", "status_code"},
	)

	// serverRequestLatency 记录 Gin 入站请求的耗时分布。
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

	// createEnvFailure 创建环境失败计数
	createEnvFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "environment",
			Name:      "create_failure",
			Help:      "Create environment failure count.",
		},
		[]string{"workspace_id", "env_name"},
	)

	// createApmFailure APM 应用创建/获取失败计数
	createApmFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "apm",
			Name:      "create_failure",
			Help:      "Create APM app failure count.",
		},
		[]string{"workspace_id", "env_name", "detail"},
	)

	// bindApmFailure APM 与环境绑定失败计数
	bindApmFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "apm",
			Name:      "bind_failure",
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

	// deployDuration 部署链路终态耗时（其 _count 子指标等价于部署次数计数）。
	deployDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "deploy",
			Name:      "duration_seconds",
			Help:      "Duration in seconds of deploy operations by kind and terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"kind", "status"},
	)

	// buildDuration 记录构建终态耗时（其 _count 子指标等价于构建次数计数）。
	buildDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "build",
			Name:      "duration_seconds",
			Help:      "Duration in seconds of image builds by terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"status"},
	)

	// workspaceInitDuration 记录工作空间初始化终态耗时（其 _count 子指标等价于初始化次数计数）。
	workspaceInitDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "workspace",
			Name:      "init_duration_seconds",
			Help:      "Duration in seconds of workspace init operations by terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"status"},
	)

	// deployUninstallDuration 记录应用卸载终态耗时（其 _count 子指标等价于卸载次数计数）。
	deployUninstallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "deploy",
			Name:      "uninstall_duration_seconds",
			Help:      "Duration in seconds of deploy uninstall operations by kind and terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"kind", "status"},
	)

	// clusterAddonDuration 记录集群插件部署/卸载终态耗时。
	clusterAddonDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "cluster_addon",
			Name:      "duration_seconds",
			Help:      "Duration in seconds of cluster addon operations by operation type and terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"operation", "status"},
	)

	// featureEnvNsInitFailure 特性环境 namespace 初始化失败计数。
	featureEnvNsInitFailure = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "feature_env",
			Name:      "ns_init_failure_total",
			Help:      "Total number of feature environment namespace init failures.",
		},
	)

	// imageSnapshotRefreshDuration 记录镜像快照刷新终态耗时。
	imageSnapshotRefreshDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "image",
			Name:      "snapshot_refresh_duration_seconds",
			Help:      "Duration in seconds of image snapshot refresh operations by terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"status"},
	)

	// imageTagDeleteDuration 记录镜像 Tag 删除终态耗时。
	imageTagDeleteDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "image",
			Name:      "tag_delete_duration_seconds",
			Help:      "Duration in seconds of image tag delete operations by terminal status.",
			Buckets:   durationBuckets,
		},
		[]string{"status"},
	)

	// instanceScaleTotal 记录实例扩缩容成功次数。
	instanceScaleTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "instance",
			Name:      "scale_total",
			Help:      "Total number of successful instance scale operations by direction.",
		},
		[]string{"direction"},
	)

	// instanceWatchActiveConnections 当前活跃的实例 Watch SSE 连接数。
	instanceWatchActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "bkms",
			Subsystem: "instance",
			Name:      "watch_active_connections",
			Help:      "Current number of active instance watch SSE connections.",
		},
	)

	// instanceWatchEventsTotal 实例 Watch 成功写出的 SSE 事件数。
	instanceWatchEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "instance",
			Name:      "watch_events_total",
			Help:      "Total number of instance watch SSE events pushed by type.",
		},
		[]string{"type"},
	)

	// instanceWatchPluginFetchTotal 实例 Watch 附属数据插件的拉取次数。
	instanceWatchPluginFetchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "instance",
			Name:      "watch_plugin_fetch_total",
			Help:      "Total number of instance watch plugin fetches by plugin and result.",
		},
		[]string{"plugin", "result"},
	)

	// depservicePolarisFailure 北极星依赖服务操作失败计数。
	depservicePolarisFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "depservice_polaris",
			Name:      "failure_total",
			Help:      "Total number of Polaris dep-service operation failures by operation type.",
		},
		[]string{"operation"},
	)

	// depserviceRedisFailure Redis 依赖服务生命周期终态失败计数。
	depserviceRedisFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "depservice_redis",
			Name:      "failure_total",
			Help:      "Total number of Redis dep-service terminal failures by operation type.",
		},
		[]string{"operation"},
	)

	// bscpcfgFailure BSCP 配置管理关键步骤失败计数。
	bscpcfgFailure = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "bscpcfg",
			Name:      "operation_failure_total",
			Help:      "Total number of BSCP config management operation failures by operation type.",
		},
		[]string{"operation"},
	)
)

// ClientRequest 用于 defer 场景下上报 infras 层 API 调用指标。
func ClientRequest(system, handler string, started time.Time, errPtr *error) {
	status := StatusOK
	if lo.FromPtr(errPtr) != nil {
		status = StatusErr
	}

	clientRequestTotal.WithLabelValues(system, handler, status).Inc()
	clientRequestLatency.WithLabelValues(system, handler).Observe(time.Since(started).Seconds())
}

// ServerRequest 记录入站 API 请求指标。
func ServerRequest(handler, method string, statusCode int, started time.Time) {
	code := strconv.Itoa(statusCode)
	serverRequestTotal.WithLabelValues(handler, method, code).Inc()
	serverRequestLatency.WithLabelValues(handler, method).Observe(time.Since(started).Seconds())
}

// CreateEnvFailed 记录创建环境失败
func CreateEnvFailed(workspaceID, envName string) {
	createEnvFailure.WithLabelValues(workspaceID, envName).Inc()
}

// CreateEnvApmFailed 记录创建 APM 应用失败
func CreateEnvApmFailed(workspaceID, envName, detail string) {
	createApmFailure.WithLabelValues(workspaceID, envName, detail).Inc()
}

// BindEnvApmFailed 记录 APM 与环境绑定失败
func BindEnvApmFailed(workspaceID, envName, detail string) {
	bindApmFailure.WithLabelValues(workspaceID, envName, detail).Inc()
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

// DeployFinished 记录部署终态结果与耗时。
func DeployFinished(kind, rawStatus string, startedAt, endedAt time.Time) {
	kind = normalizeDeployKind(kind)
	status := normalizeResultStatus(rawStatus)
	deployDuration.WithLabelValues(kind, status).Observe(durationSeconds(startedAt, endedAt))
}

// BuildFinished 记录构建终态结果与耗时。
func BuildFinished(rawStatus string, startedAt, endedAt time.Time) {
	status := normalizeResultStatus(rawStatus)
	buildDuration.WithLabelValues(status).Observe(durationSeconds(startedAt, endedAt))
}

// WorkspaceInitFinished 记录工作空间初始化终态结果与耗时。
func WorkspaceInitFinished(rawStatus string, startedAt time.Time) {
	status := normalizeResultStatus(rawStatus)
	workspaceInitDuration.WithLabelValues(status).Observe(time.Since(startedAt).Seconds())
}

// DeployUninstallFinished 记录应用卸载终态结果与耗时。
func DeployUninstallFinished(kind, rawStatus string, startedAt time.Time) {
	kind = normalizeDeployKind(kind)
	status := normalizeResultStatus(rawStatus)
	deployUninstallDuration.WithLabelValues(kind, status).Observe(time.Since(startedAt).Seconds())
}

// ClusterAddonOperationFinished 记录集群插件操作终态结果与耗时
func ClusterAddonOperationFinished(operation string, startedAt time.Time, errPtr *error) {
	status := StatusOK
	if lo.FromPtr(errPtr) != nil {
		status = StatusErr
	}
	operation = normalizeClusterAddonOperation(operation)
	clusterAddonDuration.WithLabelValues(operation, status).Observe(time.Since(startedAt).Seconds())
}

// FeatureEnvNamespaceInitFailed 记录特性环境 namespace 初始化失败。
func FeatureEnvNamespaceInitFailed() {
	featureEnvNsInitFailure.Inc()
}

// ImageSnapshotRefreshFinished 记录镜像快照刷新终态结果与耗时。
func ImageSnapshotRefreshFinished(rawStatus string, startedAt time.Time) {
	status := normalizeResultStatus(rawStatus)
	imageSnapshotRefreshDuration.WithLabelValues(status).Observe(time.Since(startedAt).Seconds())
}

// ImageTagDeleteFinished 记录镜像 Tag 删除终态结果与耗时。
func ImageTagDeleteFinished(rawStatus string, startedAt time.Time) {
	status := normalizeResultStatus(rawStatus)
	imageTagDeleteDuration.WithLabelValues(status).Observe(time.Since(startedAt).Seconds())
}

// InstanceScaled 记录实例扩缩容成功次数。
func InstanceScaled(oldReplicas, targetReplicas int32) {
	instanceScaleTotal.WithLabelValues(normalizeScaleDirection(oldReplicas, targetReplicas)).Inc()
}

// InstanceWatchStarted 记录一条实例 Watch SSE 连接开始。
func InstanceWatchStarted() {
	instanceWatchActiveConnections.Inc()
}

// InstanceWatchFinished 记录一条实例 Watch SSE 连接结束。
func InstanceWatchFinished() {
	instanceWatchActiveConnections.Dec()
}

// InstanceWatchEventPushed 记录一条成功写出的实例 Watch SSE 事件。
func InstanceWatchEventPushed(eventType string) {
	instanceWatchEventsTotal.WithLabelValues(normalizeWatchEventType(eventType)).Inc()
}

// InstanceWatchPluginFetch 记录一轮实例 Watch 附属数据插件的拉取结果。
func InstanceWatchPluginFetch(plugin string, ok bool) {
	result := StatusOK
	if !ok {
		result = StatusFail
	}

	instanceWatchPluginFetchTotal.WithLabelValues(plugin, result).Inc()
}

// DepservicePolarisFailed 记录北极星依赖服务操作失败。
func DepservicePolarisFailed(operation string) {
	depservicePolarisFailure.WithLabelValues(operation).Inc()
}

// DepserviceRedisFailed 记录 Redis 依赖服务生命周期终态失败。
func DepserviceRedisFailed(operation string) {
	depserviceRedisFailure.WithLabelValues(operation).Inc()
}

// BscpcfgStepFailed 记录 BSCP 配置管理关键步骤失败。
func BscpcfgStepFailed(operation string) {
	bscpcfgFailure.WithLabelValues(operation).Inc()
}

func normalizeDeployKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case DeployKindHelm:
		return DeployKindHelm
	case DeployKindAppModel, "trpc", "taf":
		return DeployKindAppModel
	default:
		return DeployKindUnknown
	}
}

func normalizeClusterAddonOperation(operation string) string {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case ClusterAddonOperationDeploy, "install", "upgrade", "upsert":
		return ClusterAddonOperationDeploy
	case ClusterAddonOperationUninstall, "delete":
		return ClusterAddonOperationUninstall
	default:
		return ClusterAddonOperationUnknown
	}
}

func normalizeResultStatus(rawStatus string) string {
	switch strings.ToLower(strings.TrimSpace(rawStatus)) {
	case StatusOK, "success", "succeeded", "deployed":
		return StatusOK
	case StatusTimeout, "polling-timeout", "pollingtimeout":
		return StatusTimeout
	case StatusCancelled, "canceled":
		return StatusCancelled
	default:
		return StatusErr
	}
}

func normalizeScaleDirection(oldReplicas, targetReplicas int32) string {
	switch {
	case targetReplicas > oldReplicas:
		return ScaleDirectionUp
	case targetReplicas < oldReplicas:
		return ScaleDirectionDown
	default:
		return ScaleDirectionSame
	}
}

func normalizeWatchEventType(eventType string) string {
	switch eventType {
	case "ADDED", "MODIFIED", "DELETED", "ENDED", "PLUGIN":
		return eventType
	default:
		return WatchEventTypeUnknown
	}
}

func durationSeconds(startedAt, endedAt time.Time) float64 {
	if startedAt.IsZero() {
		return 0
	}
	if !endedAt.IsZero() && endedAt.After(startedAt) {
		return endedAt.Sub(startedAt).Seconds()
	}
	return time.Since(startedAt).Seconds()
}
