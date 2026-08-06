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

// Package apm 提供 bkms-server 运行时 APM 初始化和 Gin 请求链路追踪中间件
//
// 该包只负责 HTTP/Gin 链路的 OpenTelemetry 接入，保留部分历史 tRPC APM 命名规则是为了保证迁移期间
// 蓝鲸监控侧的服务名、标签和已有告警查询不发生变化
package apm

import (
	"cmp"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/version"
)

const (
	// defaultAppName 与历史 tRPC server.app 配置保持一致
	defaultAppName = "bkms"
	// defaultServerName 与历史 tRPC server.server 配置保持一致，当调用方未传入 serverRole 时使用
	defaultServerName = "bkmsserver"
	// defaultTenantID 与原内部 SDK 的默认租户保持一致
	defaultTenantID = "default"
	// tenantHeaderKey 与原内部 SDK 发送给蓝鲸监控采集端的租户 Header 保持一致
	tenantHeaderKey = "x-bk-token"
	// legacyTenantIDAttributeKey 保留原内部 SDK 上报的租户资源属性
	legacyTenantIDAttributeKey = "tps.tenant.id"
	// legacyTelemetrySDKName 保留原内部 SDK 上报的 telemetry SDK 名称
	legacyTelemetrySDKName = "opentelemetry"
	// legacyServiceNameAttributeKey 保留历史 oteltrpc attributes 中的 service_name 属性
	legacyServiceNameAttributeKey = "service_name"
	// grpcCompressorName 与原内部 SDK 的 gRPC gzip 压缩配置保持一致
	grpcCompressorName = "gzip"
	// maxSendMessageSize 与原内部 SDK 的 gRPC 单次发送大小限制保持一致
	maxSendMessageSize = 4194304
	// serverErrorStatusCodeStart 表示 HTTP 5xx 区间起点，5xx 响应需要显式标记 span 为错误
	serverErrorStatusCodeStart = 500
	// exporterRetryInitialInterval 与官方 exporter 默认重试初始间隔一致
	exporterRetryInitialInterval = 5 * time.Second
	// exporterRetryMaxInterval 与官方 exporter 默认重试最大间隔一致
	exporterRetryMaxInterval = 30 * time.Second
	// exporterRetryMaxElapsedTime 与官方 exporter 默认重试总耗时一致
	exporterRetryMaxElapsedTime = time.Minute
)

type setupConfig struct {
	Endpoint    string
	HTTPEnabled bool
	TenantID    string
	ServiceName string
}

// Setup 初始化全局 OpenTelemetry Provider，初始化失败或配置为空时仅记录日志，不阻断服务启动
//
// serverRole 表示当前进程角色（如 "webserver" / "worker"），当 cfg.APMServiceName 为空时会拼接进服务名
// 返回值统一为 shutdown 函数，调用方无需区分 APM 是否真的完成初始化；当配置不可用或初始化失败时，返回空实现
func Setup(ctx context.Context, cfg config.BkMonitorConfig, serverRole string) func(context.Context) error {
	setupCfg := resolveSetupConfig(ctx, cfg, serverRole)
	if setupCfg.Endpoint == "" {
		log.Warn(ctx, "bk monitor APM endpoint is empty, skip APM setup")
		return noopShutdown
	}

	// exporter 内部会长期持有传入的 ctx（建连、维护连接池），而外部 ctx 通常来自 cmd.Context()，进程收到 SIGTERM
	// 时会先被 cancel，导致 shutdown 阶段最后一批 span 无法完成 export；此处用 WithoutCancel 剥离取消传播，
	// 只保留 endpoint / 超时相关的建连语义
	exporterCtx := context.WithoutCancel(ctx)
	shutdown, err := setupTraceProvider(exporterCtx, setupCfg)
	if err != nil {
		log.Warnf(ctx, "failed to setup bk monitor APM, skip APM setup: %v", err)
		return noopShutdown
	}

	log.Infof(ctx, "bk monitor APM setup completed, serviceName=%s", setupCfg.ServiceName)
	return shutdown
}

func setupTraceProvider(ctx context.Context, cfg setupConfig) (func(context.Context) error, error) {
	exporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "build otel resource")
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		// TraceContext 支持 W3C traceparent/tracestate Header，otelgin 会自动从请求中提取上游 TraceID 并注入 span
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Errorf(ctx, "[otel] error: %v", err)
	}))

	return provider.Shutdown, nil
}

func newTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	if cfg.HTTPEnabled {
		return newHTTPTraceExporter(ctx, cfg)
	}
	return newGRPCTraceExporter(ctx, cfg)
}

func newHTTPTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	// 使用 url.Parse 解析 endpoint，将 Host 与 Path 分开传给 exporter，避免把带 path 的 URL
	// 整段作为 host 传入导致上报地址异常
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "parse APM HTTP endpoint %q", cfg.Endpoint)
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(u.Host),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	}
	// 仅当 endpoint 显式携带 path 时才透传，避免覆盖 otlptracehttp 默认的 /v1/traces
	if u.Path != "" && u.Path != "/" {
		options = append(options, otlptracehttp.WithURLPath(u.Path))
	}
	// 非 https scheme 视为明文上报，走 WithInsecure
	if u.Scheme != "https" {
		options = append(options, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.New(ctx, options...)
}

func newGRPCTraceExporter(ctx context.Context, cfg setupConfig) (sdktrace.SpanExporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()),
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithCompressor(grpcCompressorName),
		otlptracegrpc.WithHeaders(map[string]string{tenantHeaderKey: cfg.TenantID}),
		otlptracegrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxSendMessageSize))),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: exporterRetryInitialInterval,
			MaxInterval:     exporterRetryMaxInterval,
			MaxElapsedTime:  exporterRetryMaxElapsedTime,
		}),
	)
}

func newResource(ctx context.Context, cfg setupConfig) (*resource.Resource, error) {
	extraRes, err := resource.New(
		ctx,
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String(legacyTenantIDAttributeKey, cfg.TenantID),
			semconv.TelemetrySDKLanguageGo,
			semconv.TelemetrySDKNameKey.String(legacyTelemetrySDKName),
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(version.Version),
			attribute.String(legacyServiceNameAttributeKey, cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return resource.Merge(resource.Default(), extraRes)
}

func resolveSetupConfig(ctx context.Context, cfg config.BkMonitorConfig, serverRole string) setupConfig {
	endpoint, httpEnabled := resolveEndpoint(cfg)
	serviceName := ServiceName(cfg, serverRole)
	// APMToken 在原内部 SDK 中对应 tenant id；为空时使用 SDK 默认租户，保证本地和缺省环境仍可启动
	tenantID := cmp.Or(strings.TrimSpace(cfg.APMToken), defaultTenantID)
	if cfg.APMToken == "" {
		log.Warn(ctx, "bk monitor APM token is empty, use default tenant id")
	}

	return setupConfig{
		Endpoint:    endpoint,
		HTTPEnabled: httpEnabled,
		TenantID:    tenantID,
		ServiceName: serviceName,
	}
}

// resolveEndpoint 选择蓝鲸监控 APM 上报地址，并返回该地址是否应该按 HTTP exporter 处理
//
// APMHttpEndpoint 是 bkms-server 自身 trace 上报的首选地址，固定走 otlptracehttp；当它为空时，
// 兼容旧配置 APMEndpoint：带 http/https scheme 时走 otlptracehttp，否则走 otlptracegrpc
func resolveEndpoint(cfg config.BkMonitorConfig) (string, bool) {
	httpEndpoint := strings.TrimSpace(cfg.APMHttpEndpoint)
	if httpEndpoint != "" {
		return httpEndpoint, true
	}

	endpoint := strings.TrimSpace(cfg.APMEndpoint)
	if endpoint == "" {
		return "", false
	}
	return endpoint, strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}

func noopShutdown(context.Context) error {
	return nil
}
