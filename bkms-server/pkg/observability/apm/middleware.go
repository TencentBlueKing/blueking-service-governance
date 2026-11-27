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
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm/apitracking"
)

// fallbackCounter 用于 crypto/rand 失败时的兜底计数，保证同一进程内每次生成的 TraceID 唯一
var fallbackCounter atomic.Uint64

// OTelMiddleware 创建 span、传播 trace context、标记 5xx error。
func OTelMiddleware(cfg config.BkMonitorConfig, serverRole string) gin.HandlerFunc {
	return otelgin.Middleware(ServiceName(cfg, serverRole))
}

// HTTPLogMiddleware 访问日志中间件，附加 API 追踪字段并脱敏 query、referer，须挂在认证之外
func HTTPLogMiddleware(names *apitracking.HandlerNames) gin.HandlerFunc {
	logger := slog.New(maskHandler{slog.Default().Handler()})
	return sloggin.NewWithConfig(logger, sloggin.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
		WithUserAgent:    true,
		WithClientIP:     true,
		// 以下内容重复了，需要显示关闭
		WithTraceID:   false,
		WithSpanID:    false,
		WithRequestID: false,
		WithCustomMessage: func(c *gin.Context) string {
			return fmt.Sprintf("%s %s %d", c.Request.Method, c.FullPath(), c.Writer.Status())
		},
		Filters: []sloggin.Filter{
			sloggin.IgnorePath("/healthz", "/readyz", "/metrics"),
			trackingFilter(names),
		},
	})
}

// trackingFilter 附加 API 追踪字段，恒返回 true。
// sloggin 在 c.Next() 之后、输出日志之前执行 Filters，此时认证结果与状态码均已确定。
func trackingFilter(names *apitracking.HandlerNames) sloggin.Filter {
	return func(c *gin.Context) bool {
		if attrs := apitracking.Attrs(c, names); attrs != nil {
			sloggin.AddCustomAttributes(c, attrs...)
		}
		return true
	}
}

// ResponseTraceMiddleware 将 TraceID 写入响应 Header（X-Trace-Id），供前端关联后端链路。
// 需注册在 otelgin.Middleware 之后；APM 未启用时随机生成兜底。
func ResponseTraceMiddleware() gin.HandlerFunc {
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
