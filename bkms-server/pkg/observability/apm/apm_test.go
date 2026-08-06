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

package apm

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

const (
	testAPMEndpoint     = "bk-example.com:4317"
	testAPMHTTPEndpoint = "http://bk-example.com:4318"
	testAPMToken        = "test-token"
	testAPMServiceName  = "bkms.custom-server"
	testServerRole      = "webserver"
	testTraceParent     = "00-11111111111111111111111111111111-2222222222222222-01"
	expectedSpanCount   = 1
)

type fakeSpanExporter struct {
	shutdownCalled bool
}

var _ sdktrace.SpanExporter = &fakeSpanExporter{}

func (e *fakeSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *fakeSpanExporter) Shutdown(context.Context) error {
	e.shutdownCalled = true
	return nil
}

var _ = Describe("APM", func() {
	Describe("ServiceName", func() {
		It("uses explicit service name first", func() {
			cfg := config.BkMonitorConfig{APMServiceName: testAPMServiceName}
			Expect(ServiceName(cfg, testServerRole)).To(Equal(testAPMServiceName))
		})

		It("uses bkms.${serverRole} when APMServiceName is empty", func() {
			Expect(ServiceName(config.BkMonitorConfig{}, testServerRole)).To(Equal("bkms." + testServerRole))
		})

		It("falls back to default server name when serverRole is empty", func() {
			Expect(ServiceName(config.BkMonitorConfig{}, "")).To(Equal("bkms." + defaultServerName))
		})
	})

	Describe("resolveEndpoint", func() {
		It("uses APMHttpEndpoint as HTTP OTLP endpoint first", func() {
			endpoint, httpEnabled := resolveEndpoint(config.BkMonitorConfig{
				APMEndpoint:     testAPMEndpoint,
				APMHttpEndpoint: testAPMHTTPEndpoint,
			})
			Expect(endpoint).To(Equal(testAPMHTTPEndpoint))
			Expect(httpEnabled).To(BeTrue())
		})

		It("uses APMEndpoint as gRPC OTLP endpoint when APMHttpEndpoint is empty", func() {
			endpoint, httpEnabled := resolveEndpoint(config.BkMonitorConfig{APMEndpoint: testAPMEndpoint})
			Expect(endpoint).To(Equal(testAPMEndpoint))
			Expect(httpEnabled).To(BeFalse())
		})

		It("treats http scheme APMEndpoint as HTTP OTLP endpoint when APMHttpEndpoint is empty", func() {
			endpoint, httpEnabled := resolveEndpoint(config.BkMonitorConfig{APMEndpoint: testAPMHTTPEndpoint})
			Expect(endpoint).To(Equal(testAPMHTTPEndpoint))
			Expect(httpEnabled).To(BeTrue())
		})

		It("uses APMHttpEndpoint when APMEndpoint is empty", func() {
			endpoint, httpEnabled := resolveEndpoint(config.BkMonitorConfig{APMHttpEndpoint: testAPMHTTPEndpoint})
			Expect(endpoint).To(Equal(testAPMHTTPEndpoint))
			Expect(httpEnabled).To(BeTrue())
		})
	})

	Describe("Setup", func() {
		AfterEach(func() {
			otel.SetTracerProvider(oteltracenoop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
		})

		It("returns noop shutdown when endpoint is empty", func() {
			shutdown := Setup(context.Background(), config.BkMonitorConfig{APMToken: testAPMToken}, testServerRole)
			Expect(shutdown).NotTo(BeNil())
			Expect(shutdown(context.Background())).To(Succeed())
		})

		It("falls back to noop shutdown when exporter creation fails", func() {
			mockey.PatchConvey("setup failure", GinkgoT(), func() {
				mockey.Mock(newTraceExporter).Return(nil, errors.New("create exporter failed")).Build()

				shutdown := Setup(context.Background(), config.BkMonitorConfig{
					APMEndpoint: testAPMHTTPEndpoint,
					APMToken:    testAPMToken,
				}, testServerRole)

				Expect(shutdown).NotTo(BeNil())
				Expect(shutdown(context.Background())).To(Succeed())
			})
		})

		It("installs tracer provider and shuts it down", func() {
			mockey.PatchConvey("setup success", GinkgoT(), func() {
				exporter := &fakeSpanExporter{}
				mockey.Mock(newTraceExporter).Return(exporter, nil).Build()

				shutdown := Setup(context.Background(), config.BkMonitorConfig{
					APMEndpoint:    testAPMHTTPEndpoint,
					APMToken:       testAPMToken,
					APMServiceName: testAPMServiceName,
				}, testServerRole)

				Expect(shutdown).NotTo(BeNil())
				_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
				Expect(ok).To(BeTrue())
				Expect(shutdown(context.Background())).To(Succeed())
				Expect(exporter.shutdownCalled).To(BeTrue())
			})
		})
	})

	Describe("resolveSetupConfig", func() {
		It("keeps endpoint, transport, tenant, and service name", func() {
			setupCfg := resolveSetupConfig(context.Background(), config.BkMonitorConfig{
				APMEndpoint:    testAPMHTTPEndpoint,
				APMToken:       testAPMToken,
				APMServiceName: testAPMServiceName,
			}, testServerRole)

			Expect(setupCfg.Endpoint).To(Equal(testAPMHTTPEndpoint))
			Expect(setupCfg.HTTPEnabled).To(BeTrue())
			Expect(setupCfg.TenantID).To(Equal(testAPMToken))
			Expect(setupCfg.ServiceName).To(Equal(testAPMServiceName))
		})

		It("uses default tenant when token is empty", func() {
			setupCfg := resolveSetupConfig(
				context.Background(),
				config.BkMonitorConfig{APMEndpoint: testAPMHTTPEndpoint},
				testServerRole,
			)
			Expect(setupCfg.TenantID).To(Equal(defaultTenantID))
		})

		It("uses bkms.${serverRole} when APMServiceName is empty", func() {
			setupCfg := resolveSetupConfig(
				context.Background(),
				config.BkMonitorConfig{APMEndpoint: testAPMHTTPEndpoint},
				testServerRole,
			)
			Expect(setupCfg.ServiceName).To(Equal("bkms." + testServerRole))
		})
	})

	Describe("newResource", func() {
		It("keeps legacy resource attributes", func() {
			res, err := newResource(context.TODO(), setupConfig{
				TenantID:    testAPMToken,
				ServiceName: testAPMServiceName,
			})
			Expect(err).NotTo(HaveOccurred())

			attrs := resourceAttributes(res.Attributes())
			Expect(attrs[legacyTenantIDAttributeKey]).To(Equal(testAPMToken))
			Expect(attrs[legacyServiceNameAttributeKey]).To(Equal(testAPMServiceName))
			Expect(attrs["service.name"]).To(Equal(testAPMServiceName))
			Expect(attrs["telemetry.sdk.name"]).To(Equal(legacyTelemetrySDKName))
		})
	})

	Describe("Middleware", func() {
		var recorder *tracetest.SpanRecorder

		BeforeEach(func() {
			gin.SetMode(gin.TestMode)
			recorder = tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
		})

		AfterEach(func() {
			otel.SetTracerProvider(oteltracenoop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())
		})

		It("continues incoming trace context", func() {
			r := gin.New()
			r.Use(Middleware(config.BkMonitorConfig{APMServiceName: testAPMServiceName}, testServerRole))
			r.GET("/healthz", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set("traceparent", testTraceParent)
			r.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			spans := recorder.Ended()
			Expect(spans).To(HaveLen(expectedSpanCount))
			Expect(spans[0].SpanContext().TraceID().String()).To(Equal("11111111111111111111111111111111"))
			Expect(spans[0].Name()).To(Equal("GET /healthz"))
		})

		It("marks server error status on span", func() {
			r := gin.New()
			r.Use(
				Middleware(config.BkMonitorConfig{APMServiceName: testAPMServiceName}, testServerRole),
				ErrorStatusMiddleware(),
			)
			r.GET("/failed", func(c *gin.Context) {
				c.Status(http.StatusInternalServerError)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/failed", nil)
			r.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			spans := recorder.Ended()
			Expect(spans).To(HaveLen(expectedSpanCount))
			Expect(spans[0].Status().Code).To(Equal(codes.Error))
		})
	})
})

func resourceAttributes(attrs []attribute.KeyValue) map[string]string {
	values := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		values[string(attr.Key)] = attr.Value.AsString()
	}
	return values
}
