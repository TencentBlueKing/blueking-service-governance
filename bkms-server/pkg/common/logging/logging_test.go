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
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

const (
	testLogLevelDebug        = "debug"
	testLogLevelInfo         = "info"
	testLogLevelWarn         = "warn"
	testLogLevelError        = "error"
	testTraceID              = "00000000000000000000000000000001"
	testSpanID               = "0000000000000001"
	expectedSingleOccurrence = 1
	testLoggingFileName      = "logging_test.go"
	shimLoggingFileName      = "shim.go"
	testLogFileNameOne       = "logging-one.log"
	testLogFileNameTwo       = "logging-two.log"
	testMultiWriterMsg       = "multi writer message"
)

var _ = Describe("Logging", func() {
	Describe("toSlogLevel", func() {
		DescribeTable("converts supported levels",
			func(level string, expected slog.Level) {
				actual, err := toSlogLevel(level)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(Equal(expected))
			},
			Entry("debug", testLogLevelDebug, slog.LevelDebug),
			Entry("info", testLogLevelInfo, slog.LevelInfo),
			Entry("warn", testLogLevelWarn, slog.LevelWarn),
			Entry("warning", "warning", slog.LevelWarn),
			Entry("error", testLogLevelError, slog.LevelError),
		)

		It("returns error for unsupported level", func() {
			_, err := toSlogLevel("verbose")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("normalizeConfig", func() {
		It("fills defaults for empty config", func() {
			cfg := normalizeConfig(config.LoggingConfig{})
			Expect(cfg.Level).To(Equal(defaultLevel))
			Expect(cfg.HandlerName).To(Equal(defaultHandlerName))
			Expect(cfg.Writers).To(Equal([]config.LoggingWriterConfig{{WriterName: defaultWriterName}}))
		})

		It("keeps explicit writers", func() {
			cfg := normalizeConfig(config.LoggingConfig{
				Writers: []config.LoggingWriterConfig{
					{WriterName: WriterStdout},
					{
						WriterName:   WriterFile,
						WriterConfig: config.LoggingWriterFileConfig{Filename: testLogFileNameOne},
					},
				},
			})
			Expect(cfg.Writers).To(HaveLen(2))
			Expect(cfg.Writers[0].WriterName).To(Equal(WriterStdout))
			Expect(cfg.Writers[1].WriterName).To(Equal(WriterFile))
		})
	})

	Describe("newLogger", func() {
		It("rejects invalid handler name before opening writers", func() {
			tempDir := GinkgoT().TempDir()
			logPath := filepath.Join(tempDir, testLogFileNameOne)
			_, err := newLogger(config.LoggingConfig{
				Level:       testLogLevelInfo,
				HandlerName: "yaml",
				Writers: []config.LoggingWriterConfig{
					{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: logPath}},
				},
			})
			Expect(err).To(HaveOccurred())
			// 校验必须发生在 writer 构造之前，因此不应产生日志文件。
			_, statErr := os.Stat(logPath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
	})

	Describe("newWriter", func() {
		It("creates stdout writer", func() {
			writer, err := newWriter(WriterStdout, config.LoggingWriterFileConfig{})
			Expect(err).NotTo(HaveOccurred())
			Expect(writer).NotTo(BeNil())
		})

		It("rejects unsupported writer", func() {
			_, err := newWriter("unsupported", config.LoggingWriterFileConfig{})
			Expect(err).To(HaveOccurred())
		})

		It("writes logs to multiple writers", func() {
			tempDir := GinkgoT().TempDir()
			firstLogPath := filepath.Join(tempDir, testLogFileNameOne)
			secondLogPath := filepath.Join(tempDir, testLogFileNameTwo)
			logger, err := newLogger(config.LoggingConfig{
				Level:       testLogLevelInfo,
				HandlerName: HandlerJSON,
				Writers: []config.LoggingWriterConfig{
					{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: firstLogPath}},
					{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: secondLogPath}},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			logger.Info(testMultiWriterMsg)

			firstContent, err := os.ReadFile(firstLogPath)
			Expect(err).NotTo(HaveOccurred())
			secondContent, err := os.ReadFile(secondLogPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(firstContent)).To(ContainSubstring(testMultiWriterMsg))
			Expect(string(secondContent)).To(ContainSubstring(testMultiWriterMsg))
		})
	})

	Describe("newRotateFileWriter", func() {
		It("creates missing log dir automatically", func() {
			tempDir := GinkgoT().TempDir()
			// 目标目录尚不存在，writer 应主动 MkdirAll。
			logDir := filepath.Join(tempDir, "nested", "logs")
			logPath := filepath.Join(logDir, testLogFileNameOne)

			writer, err := newRotateFileWriter(config.LoggingWriterFileConfig{Filename: logPath})
			Expect(err).NotTo(HaveOccurred())
			Expect(writer).NotTo(BeNil())

			_, statErr := os.Stat(logDir)
			Expect(statErr).NotTo(HaveOccurred())

			_, err = writer.Write([]byte("hello world\n"))
			Expect(err).NotTo(HaveOccurred())
			content, err := os.ReadFile(logPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("hello world"))
		})

		It("rejects negative max age", func() {
			tempDir := GinkgoT().TempDir()
			logPath := filepath.Join(tempDir, testLogFileNameOne)
			_, err := newRotateFileWriter(config.LoggingWriterFileConfig{Filename: logPath, MaxAge: -1})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("newMultiWriter file uniqueness", func() {
		It("rejects duplicate filenames", func() {
			tempDir := GinkgoT().TempDir()
			logPath := filepath.Join(tempDir, testLogFileNameOne)
			_, err := newMultiWriter([]config.LoggingWriterConfig{
				{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: logPath}},
				{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: logPath}},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate file writer filename"))
		})

		It("treats relative and cleaned paths as duplicates", func() {
			_, err := newMultiWriter([]config.LoggingWriterConfig{
				{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: "./logs/app.log"}},
				{WriterName: WriterFile, WriterConfig: config.LoggingWriterFileConfig{Filename: "logs/app.log"}},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate file writer filename"))
		})
	})

	Describe("Log", func() {
		var buf bytes.Buffer

		BeforeEach(func() {
			buf.Reset()
			slog.SetDefault(
				slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})),
			)
		})

		It("writes structured attributes via InfoAttrs", func() {
			InfoAttrs(context.Background(), "hello", slog.String("component", "logging"), slog.Int("count", 1))
			output := buf.String()
			Expect(output).To(ContainSubstring("hello"))
			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("logging"))
			Expect(output).To(ContainSubstring("count"))
		})

		It("uses explicit trace fields before context fields", func() {
			traceID, err := trace.TraceIDFromHex(testTraceID)
			Expect(err).NotTo(HaveOccurred())
			spanID, err := trace.SpanIDFromHex(testSpanID)
			Expect(err).NotTo(HaveOccurred())
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
			ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

			Log(ctx, slog.LevelInfo, "trace override", slog.String(FieldTraceID, "manual-trace"))
			output := buf.String()
			Expect(strings.Count(output, FieldTraceID)).To(Equal(expectedSingleOccurrence))
			Expect(output).To(ContainSubstring("manual-trace"))
			Expect(output).To(ContainSubstring(testSpanID))
		})

		It("writes context attrs with trace fields", func() {
			traceID, err := trace.TraceIDFromHex(testTraceID)
			Expect(err).NotTo(HaveOccurred())
			spanID, err := trace.SpanIDFromHex(testSpanID)
			Expect(err).NotTo(HaveOccurred())
			spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
			ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

			InfoAttrs(ctx, "attrs with trace", slog.String("component", "logging"))
			output := buf.String()
			Expect(output).To(ContainSubstring("attrs with trace"))
			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring(testTraceID))
			Expect(output).To(ContainSubstring(testSpanID))
		})

		It("writes no-context text and formatted logs without trace fields", func() {
			InfoNoContext("startup ready")
			WarnNoContextf("cleanup %s", "timeout")
			output := buf.String()
			Expect(output).To(ContainSubstring("startup ready"))
			Expect(output).To(ContainSubstring("cleanup timeout"))
			Expect(output).NotTo(ContainSubstring(FieldTraceID))
			Expect(output).NotTo(ContainSubstring(FieldSpanID))
		})

		It("writes no-context slog attributes and groups", func() {
			ErrorNoContextAttrs(
				"attrs without context",
				slog.String("component", "logging"),
				slog.Group("payload", slog.String("id", "app-1")),
			)
			output := buf.String()
			Expect(output).To(ContainSubstring("attrs without context"))
			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("logging"))
			Expect(output).To(ContainSubstring("payload"))
			Expect(output).To(ContainSubstring("app-1"))
			Expect(output).NotTo(ContainSubstring(FieldTraceID))
			Expect(output).NotTo(ContainSubstring(FieldSpanID))
		})

		It("writes no-context logs by level", func() {
			LogNoContext(slog.LevelInfo, "log no context", slog.String("component", "logging"))
			output := buf.String()
			Expect(output).To(ContainSubstring("log no context"))
			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("logging"))
			Expect(output).NotTo(ContainSubstring(FieldTraceID))
			Expect(output).NotTo(ContainSubstring(FieldSpanID))
		})

		It("reports info caller source", func() {
			Info(context.Background(), "structured source")
			output := buf.String()
			Expect(output).To(ContainSubstring(testLoggingFileName))
			Expect(output).NotTo(ContainSubstring(shimLoggingFileName))
		})

		It("reports no-context caller source", func() {
			InfoNoContext("no context source")
			output := buf.String()
			Expect(output).To(ContainSubstring(testLoggingFileName))
			Expect(output).NotTo(ContainSubstring(shimLoggingFileName))
		})
	})
})
