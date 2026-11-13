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

package cliusage_test

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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/cliusage"
)

type fakeHandler struct{}

func (fakeHandler) GetApp(c *gin.Context) { c.Status(http.StatusOK) }

var _ = Describe("Middleware", func() {
	const (
		basePath = "/bkms/v1/bkms-server"
		cliUA    = "bkms-cli/1.0.7 (linux/amd64)"
	)
	deviceID := strings.Repeat("0f", 32)

	var (
		engine    *gin.Engine
		logBuf    *bytes.Buffer
		prevSlog  *slog.Logger
		userValue = `{"userId":"admin"}`
	)

	BeforeEach(func() {
		prevSlog = slog.Default()
		logBuf = &bytes.Buffer{}
		slog.SetDefault(slog.New(slog.NewJSONHandler(logBuf, nil)))

		gin.SetMode(gin.TestMode)
		engine = gin.New()
		requiredAuth, err := auth.Required(auth.Config{AllowSetUserInHeader: true}, nil)
		Expect(err).NotTo(HaveOccurred())
		names := cliusage.NewHandlerNames()
		v1 := engine.Group(basePath)
		v1.Use(cliusage.Middleware(names), requiredAuth, tenant.Required(false, nil))
		v1.GET("/apps/:appID", fakeHandler{}.GetApp)
		v1.GET("/closure", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		names.Load(context.Background(), engine.Routes(), v1.BasePath())
	})

	AfterEach(func() {
		slog.SetDefault(prevSlog)
	})

	serve := func(path, userAgent, authorization string, authed bool) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set(cliusage.HeaderDeviceID, deviceID)
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		if authed {
			req.Header.Set(auth.BKAuthKey, userValue)
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
			if record["msg"] == cliusage.EventName {
				result = append(result, record)
			}
		}
		return result
	}

	It("logs an event for an authenticated cli request", func() {
		Expect(serve(basePath+"/apps/foo", cliUA, "", true)).To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("event", cliusage.EventName),
			HaveKeyWithValue("schema", BeNumerically("==", 1)),
			HaveKeyWithValue("cli_version", "1.0.7"),
			HaveKeyWithValue("os", "linux"),
			HaveKeyWithValue("arch", "amd64"),
			HaveKeyWithValue("device_id", deviceID),
			HaveKeyWithValue("username", "admin"),
			HaveKeyWithValue("tenant_id", "default"),
			HaveKeyWithValue("handler", "GetApp"),
			HaveKeyWithValue("method", http.MethodGet),
			HaveKeyWithValue("route", basePath+"/apps/:appID"),
			HaveKeyWithValue("status", BeNumerically("==", http.StatusOK)),
		)))
	})

	It("logs an event for an unauthenticated cli request", func() {
		Expect(serve(basePath+"/apps/foo", cliUA, "", false)).To(Equal(http.StatusUnauthorized))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("username", ""),
			HaveKeyWithValue("tenant_id", ""),
			HaveKeyWithValue("handler", "GetApp"),
			HaveKeyWithValue("status", BeNumerically("==", http.StatusUnauthorized)),
		)))
	})

	It("logs a legacy cli request", func() {
		ua := "go-resty/2.17.0 (https://github.com/go-resty/resty)"
		Expect(serve(basePath+"/apps/foo", ua, "Bearer token", true)).To(Equal(http.StatusOK))

		Expect(events()).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("cli_version", cliusage.VersionLegacy),
			HaveKeyWithValue("os", ""),
			HaveKeyWithValue("arch", ""),
		)))
	})

	It("falls back to method and route for closure handlers", func() {
		Expect(serve(basePath+"/closure", cliUA, "", true)).To(Equal(http.StatusNoContent))

		Expect(events()).To(ConsistOf(HaveKeyWithValue("handler", "GET "+basePath+"/closure")))
	})

	It("does not log requests from other clients", func() {
		Expect(serve(basePath+"/apps/foo", "Mozilla/5.0", "", true)).To(Equal(http.StatusOK))

		Expect(events()).To(BeEmpty())
	})

	It("does not log unmatched routes", func() {
		Expect(serve(basePath+"/not-found", cliUA, "", true)).To(Equal(http.StatusNotFound))

		Expect(events()).To(BeEmpty())
	})
})
