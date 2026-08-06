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

package logging

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"runtime"
	"time"

	traceutil "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/trace"
)

// callerSkip 用于所有直接封装 logWithAttrs 的导出日志入口。
// runtime.Callers 的 skip 语义：0 表示 runtime.Callers 自身
// 1 表示 logWithAttrs，2 表示日志入口函数（Info/Infof/…），因此 3 才是业务调用方。
const callerSkip = 3

// Debug 打印带 context 的 debug 日志。
func Debug(ctx context.Context, msg string) {
	logWithAttrs(ctx, true, slog.LevelDebug, msg)
}

// Debugf 打印带 context 的 debug 格式化日志。
func Debugf(ctx context.Context, format string, args ...any) {
	logWithAttrs(ctx, true, slog.LevelDebug, fmt.Sprintf(format, args...))
}

// DebugAttrs 打印带 context 的 debug 结构化日志。
func DebugAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	logWithAttrs(ctx, true, slog.LevelDebug, msg, attrs...)
}

// DebugNoContext 打印不携带 context 的 debug 日志。
// 适用于进程启动、退出清理、测试辅助等确实没有可用 context 的场景；请求链路应优先使用 Debug。
func DebugNoContext(msg string) {
	logWithAttrs(context.Background(), false, slog.LevelDebug, msg)
}

// DebugNoContextf 打印不携带 context 的 debug 格式化日志。
func DebugNoContextf(format string, args ...any) {
	logWithAttrs(context.Background(), false, slog.LevelDebug, fmt.Sprintf(format, args...))
}

// DebugNoContextAttrs 打印不携带 context 的 debug 结构化日志。
func DebugNoContextAttrs(msg string, attrs ...slog.Attr) {
	logWithAttrs(context.Background(), false, slog.LevelDebug, msg, attrs...)
}

// Info 打印带 context 的 info 日志。
func Info(ctx context.Context, msg string) {
	logWithAttrs(ctx, true, slog.LevelInfo, msg)
}

// Infof 打印带 context 的 info 格式化日志。
func Infof(ctx context.Context, format string, args ...any) {
	logWithAttrs(ctx, true, slog.LevelInfo, fmt.Sprintf(format, args...))
}

// InfoAttrs 打印带 context 的 info 结构化日志。
func InfoAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	logWithAttrs(ctx, true, slog.LevelInfo, msg, attrs...)
}

// InfoNoContext 打印不携带 context 的 info 日志。
// 适用于进程启动、退出清理、测试辅助等确实没有可用 context 的场景；请求链路应优先使用 Info。
func InfoNoContext(msg string) {
	logWithAttrs(context.Background(), false, slog.LevelInfo, msg)
}

// InfoNoContextf 打印不携带 context 的 info 格式化日志。
func InfoNoContextf(format string, args ...any) {
	logWithAttrs(context.Background(), false, slog.LevelInfo, fmt.Sprintf(format, args...))
}

// InfoNoContextAttrs 打印不携带 context 的 info 结构化日志。
func InfoNoContextAttrs(msg string, attrs ...slog.Attr) {
	logWithAttrs(context.Background(), false, slog.LevelInfo, msg, attrs...)
}

// Warn 打印带 context 的 warn 日志。
func Warn(ctx context.Context, msg string) {
	logWithAttrs(ctx, true, slog.LevelWarn, msg)
}

// Warnf 打印带 context 的 warn 格式化日志。
func Warnf(ctx context.Context, format string, args ...any) {
	logWithAttrs(ctx, true, slog.LevelWarn, fmt.Sprintf(format, args...))
}

// WarnAttrs 打印带 context 的 warn 结构化日志。
func WarnAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	logWithAttrs(ctx, true, slog.LevelWarn, msg, attrs...)
}

// WarnNoContext 打印不携带 context 的 warn 日志。
// 适用于进程启动、退出清理、测试辅助等确实没有可用 context 的场景；请求链路应优先使用 Warn。
func WarnNoContext(msg string) {
	logWithAttrs(context.Background(), false, slog.LevelWarn, msg)
}

// WarnNoContextf 打印不携带 context 的 warn 格式化日志。
func WarnNoContextf(format string, args ...any) {
	logWithAttrs(context.Background(), false, slog.LevelWarn, fmt.Sprintf(format, args...))
}

// WarnNoContextAttrs 打印不携带 context 的 warn 结构化日志。
func WarnNoContextAttrs(msg string, attrs ...slog.Attr) {
	logWithAttrs(context.Background(), false, slog.LevelWarn, msg, attrs...)
}

// Error 打印带 context 的 error 日志。
func Error(ctx context.Context, msg string) {
	logWithAttrs(ctx, true, slog.LevelError, msg)
}

// Errorf 打印带 context 的 error 格式化日志。
func Errorf(ctx context.Context, format string, args ...any) {
	logWithAttrs(ctx, true, slog.LevelError, fmt.Sprintf(format, args...))
}

// ErrorAttrs 打印带 context 的 error 结构化日志。
func ErrorAttrs(ctx context.Context, msg string, attrs ...slog.Attr) {
	logWithAttrs(ctx, true, slog.LevelError, msg, attrs...)
}

// ErrorNoContext 打印不携带 context 的 error 日志。
// 适用于进程启动、退出清理、测试辅助等确实没有可用 context 的场景；请求链路应优先使用 Error。
func ErrorNoContext(msg string) {
	logWithAttrs(context.Background(), false, slog.LevelError, msg)
}

// ErrorNoContextf 打印不携带 context 的 error 格式化日志。
func ErrorNoContextf(format string, args ...any) {
	logWithAttrs(context.Background(), false, slog.LevelError, fmt.Sprintf(format, args...))
}

// ErrorNoContextAttrs 打印不携带 context 的 error 结构化日志。
func ErrorNoContextAttrs(msg string, attrs ...slog.Attr) {
	logWithAttrs(context.Background(), false, slog.LevelError, msg, attrs...)
}

// Fatal 打印 fatal 日志到标准错误并退出程序。
func Fatal(msg string) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	logger.Fatal(msg)
}

// Fatalf 打印 fatal 格式化日志到标准错误并退出程序。
func Fatalf(format string, args ...any) {
	Fatal(fmt.Sprintf(format, args...))
}

// Log 按指定调用栈深度打印日志，供第三方 SDK 桥接场景（如 Helm SDK Debug 回调）使用。
func Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logWithAttrs(ctx, true, level, msg, attrs...)
}

// LogNoContext 按指定级别打印不携带 context 的结构化日志。
func LogNoContext(level slog.Level, msg string, attrs ...slog.Attr) {
	logWithAttrs(context.Background(), false, level, msg, attrs...)
}

func logWithAttrs(ctx context.Context, addContextFields bool, level slog.Level, msg string, attrs ...slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := slog.Default()
	if !logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(callerSkip, pcs[:])
	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	if addContextFields {
		addContextAttrs(ctx, &record, attrs)
	}
	record.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, record)
}

func addContextAttrs(ctx context.Context, record *slog.Record, attrs []slog.Attr) {
	existingKeys := make(map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		existingKeys[attr.Key] = struct{}{}
	}

	if traceID := traceutil.GetTraceID(ctx); traceID != "" {
		if _, ok := existingKeys[FieldTraceID]; !ok {
			record.AddAttrs(slog.String(FieldTraceID, traceID))
		}
	}
	if spanID := traceutil.GetSpanID(ctx); spanID != "" {
		if _, ok := existingKeys[FieldSpanID]; !ok {
			record.AddAttrs(slog.String(FieldSpanID, spanID))
		}
	}
}
