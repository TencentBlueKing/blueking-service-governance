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
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// Middleware 返回 Gin OpenTelemetry middleware，用于替代 oteltrpc server filter 的 HTTP 请求观测入口
//
// serverRole 用于区分不同进程角色（如 "webserver" / "worker"），当配置未显式指定 APMServiceName 时，
// 作为 bkms.${serverRole} 中的进程角色部分参与服务名拼接
func Middleware(cfg config.BkMonitorConfig, serverRole string) gin.HandlerFunc {
	return otelgin.Middleware(ServiceName(cfg, serverRole))
}

// ServiceName 返回 APM 中使用的服务名称
//
// 优先使用显式配置的 APMServiceName；未配置时按 bkms.${serverRole} 拼接，
// serverRole 为空时退化为历史默认值 bkmsserver，保证蓝鲸监控侧历史告警/查询兼容
func ServiceName(cfg config.BkMonitorConfig, serverRole string) string {
	if cfg.APMServiceName != "" {
		return cfg.APMServiceName
	}
	return defaultAppName + "." + cmp.Or(serverRole, defaultServerName)
}

// ErrorStatusMiddleware 兜底标记 5xx 响应的 Span 错误状态
//
// otelgin 会创建 HTTP server span，但部分异常链路只设置响应码，没有向 span 写入错误状态；该中间件在请求结束后
// 统一检查最终状态码，确保蓝鲸监控 APM 能按错误请求聚合 5xx 响应
func ErrorStatusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		statusCode := c.Writer.Status()
		if statusCode < serverErrorStatusCodeStart {
			return
		}
		span := trace.SpanFromContext(c.Request.Context())
		if span == nil || !span.SpanContext().IsValid() {
			return
		}
		span.SetStatus(codes.Error, httpStatusErrorMessage(statusCode))
	}
}

func httpStatusErrorMessage(statusCode int) string {
	return "HTTP " + cmp.Or(http.StatusText(statusCode), "server error")
}

// TraceIDResponseMiddleware 将 TraceID 写入响应 Header（X-Trace-Id），供前端关联后端链路。
// 需注册在 otelgin.Middleware 之后；APM 未启用时随机生成兜底。
func TraceIDResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 必须在 c.Next() 之前设置，否则 c.JSON() 写响应后再设置 header 无效
		c.Header("X-Trace-Id", resolveTraceID(c))
		c.Next()
	}
}

// resolveTraceID 按优先级确定本次请求的 TraceID：
//  1. 从 otelgin span 获取 traceID（APM 已启用）
//  2. 随机生成兜底（APM 未启用时）
func resolveTraceID(c *gin.Context) string {
	if span := trace.SpanFromContext(c.Request.Context()); span != nil && span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}

	// APM 未启用时兜底随机生成
	return generateTraceID()
}

// fallbackCounter 用于 crypto/rand 失败时的兜底计数，保证同一进程内每次生成的 TraceID 唯一
var fallbackCounter atomic.Uint64

// generateTraceID 生成一个随机的 16 字节（32 位小写十六进制）TraceID，符合 OpenTelemetry TraceID 格式。
func generateTraceID() string {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		// 退化：高 8 字节用纳秒时间戳，低 8 字节用自增计数器，保证格式合法且不重复
		binary.BigEndian.PutUint64(b[:8], uint64(time.Now().UnixNano())) // nolint:gosec
		binary.BigEndian.PutUint64(b[8:], fallbackCounter.Add(1))
	}

	return hex.EncodeToString(b)
}
