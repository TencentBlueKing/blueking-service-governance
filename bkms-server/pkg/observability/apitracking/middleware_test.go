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

package apitracking_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apitracking"
)

type fakeHandler struct{}

func (fakeHandler) GetApp(c *gin.Context) { c.Status(http.StatusOK) }

var _ = Describe("Middleware", func() {
	const (
		basePath  = "/bkms/v1/bkms-server"
		cliUA     = "bkms-cli/1.0.7 (linux/amd64)"
		browserUA = "Mozilla/5.0"
		userValue = `{"userId":"admin"}`
	)
	deviceID := strings.Repeat("0f", 32)

	var (
		engine   *gin.Engine
		logBuf   *bytes.Buffer
		prevSlog *slog.Logger
	)

	type request struct {
		path       string
		userAgent  string
		authorized string
		credential string
		authed     bool
	}

	BeforeEach(func() {
		prevSlog = slog.Default()
		logBuf = &bytes.Buffer{}
		slog.SetDefault(slog.New(slog.NewJSONHandler(logBuf, nil)))

		gin.SetMode(gin.TestMode)
		engine = gin.New()
		requiredAuth, err := auth.Required(auth.Config{AllowSetUserInHeader: true}, nil)
		Expect(err).NotTo(HaveOccurred())
		names := apitracking.NewHandlerNames()
		v1 := engine.Group(basePath)
		v1.Use(apitracking.Middleware(names), requiredAuth, tenant.Required(false, nil))
		v1.GET("/apps/:appID", fakeHandler{}.GetApp)
		v1.GET("/closure", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		names.Load(context.Background(), engine.Routes(), v1.BasePath())
	})

	AfterEach(func() {
		slog.SetDefault(prevSlog)
	})

	serve := func(r request) int {
		req := httptest.NewRequest(http.MethodGet, r.path, nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("User-Agent", r.userAgent)
		req.Header.Set(apitracking.HeaderDeviceID, deviceID)
		if r.authorized != "" {
			req.Header.Set("Authorization", r.authorized)
		}
		if r.authed {
			req.Header.Set(auth.BKAuthKey, userValue)
		}
		if r.credential != "" {
			req.Header.Set(auth.BKCredentialKey, r.credential)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w.Code
	}

	events := func() []map[string]any {
		var result []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(logBuf.String()), "\n") {
			if line == "" {
				continue
			}
			record := map[string]any{}
			Expect(json.Unmarshal([]byte(line), &record)).To(Succeed())
			if record["msg"] == apitracking.EventName {
				result = append(result, record)
			}
		}
		return result
	}

	It("logs an authenticated cli request", func() {
		code := serve(request{
			path: basePath + "/apps/foo?keyword=x&access_token=secret", userAgent: cliUA, authed: true,
		})
		Expect(code).To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("event", apitracking.EventName),
			HaveKeyWithValue("schema", BeNumerically("==", 1)),
			HaveKeyWithValue("client", apitracking.ClientCLI),
			HaveKeyWithValue("cli_version", "1.0.7"),
			HaveKeyWithValue("os", "linux"),
			HaveKeyWithValue("arch", "amd64"),
			HaveKeyWithValue("device_id", deviceID),
			HaveKeyWithValue("user_agent", cliUA),
			HaveKeyWithValue("username", "admin"),
			HaveKeyWithValue("tenant_id", "default"),
			HaveKeyWithValue("client_ip", "10.0.0.1"),
			HaveKeyWithValue("handler", "GetApp"),
			HaveKeyWithValue("method", http.MethodGet),
			HaveKeyWithValue("route", basePath+"/apps/:appID"),
			HaveKeyWithValue("path", basePath+"/apps/foo"),
			HaveKeyWithValue("query", "access_token=%2A%2A%2A&keyword=x"),
			HaveKeyWithValue("status", BeNumerically("==", http.StatusOK)),
		)))
	})

	It("logs an unauthenticated cli request", func() {
		Expect(serve(request{path: basePath + "/apps/foo", userAgent: cliUA})).To(Equal(http.StatusUnauthorized))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("client", apitracking.ClientCLI),
			HaveKeyWithValue("username", ""),
			HaveKeyWithValue("tenant_id", ""),
			HaveKeyWithValue("handler", "GetApp"),
			HaveKeyWithValue("status", BeNumerically("==", http.StatusUnauthorized)),
		)))
	})

	It("logs a legacy cli request", func() {
		ua := "go-resty/2.17.0 (https://github.com/go-resty/resty)"
		Expect(serve(request{path: basePath + "/apps/foo", userAgent: ua, authorized: "Bearer t", authed: true})).
			To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("client", apitracking.ClientCLI),
			HaveKeyWithValue("cli_version", apitracking.VersionLegacy),
			HaveKeyWithValue("os", ""),
		)))
	})

	It("logs a web request without cli fields", func() {
		code := serve(request{
			path: basePath + "/apps/foo", userAgent: browserUA, authed: true, credential: `{"bkToken":"x"}`,
		})
		Expect(code).To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("client", apitracking.ClientWeb),
			HaveKeyWithValue("cli_version", ""),
			HaveKeyWithValue("device_id", ""),
			HaveKeyWithValue("user_agent", browserUA),
			HaveKeyWithValue("username", "admin"),
		)))
	})

	It("logs a non-cli access token request", func() {
		code := serve(request{
			path: basePath + "/apps/foo", userAgent: "curl/8.0", authed: true, credential: `{"accessToken":"x"}`,
		})
		Expect(code).To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(HaveKeyWithValue("client", apitracking.ClientToken)))
	})

	It("logs an unauthenticated request of unknown client", func() {
		Expect(serve(request{path: basePath + "/apps/foo", userAgent: browserUA})).To(Equal(http.StatusUnauthorized))

		Expect(events()).To(ConsistOf(HaveKeyWithValue("client", apitracking.ClientUnknown)))
	})

	It("falls back to method and route for closure handlers", func() {
		Expect(serve(request{path: basePath + "/closure", userAgent: cliUA, authed: true})).
			To(Equal(http.StatusNoContent))

		Expect(events()).To(ConsistOf(HaveKeyWithValue("handler", "GET "+basePath+"/closure")))
	})

	It("does not log unmatched routes", func() {
		Expect(serve(request{path: basePath + "/not-found", userAgent: cliUA, authed: true})).
			To(Equal(http.StatusNotFound))

		Expect(events()).To(BeEmpty())
	})
})
