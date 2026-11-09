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

package client

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/deviceid"
)

var _ = Describe("Request telemetry", func() {
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"

	var (
		server   *httptest.Server
		received http.Header
		status   int
	)

	BeforeEach(func() {
		received = nil
		status = http.StatusOK
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = r.Header.Clone()
			w.Header().Set(headerTraceID, traceID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if status == http.StatusOK {
				_, _ = w.Write([]byte(`{"username": "blueking", "data": []}`))
				return
			}
			_, _ = w.Write([]byte(`{"error": {"message": "bad request"}}`))
		}))
		config.G = &config.Config{BkmsBaseURL: server.URL, AccessToken: "token"}
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("device id header", func() {
		BeforeEach(func() {
			if deviceid.Get() == "" {
				Skip("no trusted machine id on the current machine")
			}
		})

		It("is sent with business requests", func() {
			_, err := New().ListPublicEnvVars(context.Background(), "ws")
			Expect(err).NotTo(HaveOccurred())
			Expect(received.Get(headerDeviceID)).To(Equal(deviceid.Get()))
		})

		It("is sent with auth requests", func() {
			_, err := New().ValidateAccessToken("token")
			Expect(err).NotTo(HaveOccurred())
			Expect(received.Get(headerDeviceID)).To(Equal(deviceid.Get()))
		})
	})

	Describe("trace id in errors", func() {
		BeforeEach(func() {
			// 4xx 不触发重试
			status = http.StatusBadRequest
		})

		It("is included in env var errors", func() {
			_, err := New().ListPublicEnvVars(context.Background(), "ws")
			Expect(err).To(MatchError(ContainSubstring("(trace_id: " + traceID + ")")))
		})

		It("is included in dashboard errors", func() {
			_, err := New().ListDashboardDirectoryTree(context.Background(), "ws")
			Expect(err).To(MatchError(ContainSubstring("(trace_id: " + traceID + ")")))
		})

		It("is included in auth errors", func() {
			_, err := New().ValidateAccessToken("token")
			Expect(err).To(MatchError(SatisfyAll(
				ContainSubstring("validate access token failed"),
				ContainSubstring("(trace_id: "+traceID+")"),
			)))
		})

		It("is included in port-forward permission errors", func() {
			status = http.StatusForbidden
			err := New().CheckPortForwardPermission(context.Background(), "app", "env", "pod", 80, 8080)
			Expect(err).To(MatchError(SatisfyAll(
				ContainSubstring("port-forward permission denied"),
				ContainSubstring("(trace_id: "+traceID+")"),
			)))
		})
	})

	Describe("debug log", func() {
		var (
			logBuf   *bytes.Buffer
			prevSlog *slog.Logger
		)

		BeforeEach(func() {
			prevSlog = slog.Default()
			logBuf = &bytes.Buffer{}
			slog.SetDefault(slog.New(slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})))
		})

		AfterEach(func() {
			slog.SetDefault(prevSlog)
		})

		It("logs method, path, status and trace id without query parameters", func() {
			_, err := New().ValidateAccessToken("secret-token")
			Expect(err).NotTo(HaveOccurred())

			Expect(logBuf.String()).To(SatisfyAll(
				ContainSubstring("method=GET"),
				ContainSubstring("path=/user_token/validate"),
				ContainSubstring("status=200"),
				ContainSubstring("trace_id="+traceID),
				Not(ContainSubstring("secret-token")),
			))
		})
	})
})
