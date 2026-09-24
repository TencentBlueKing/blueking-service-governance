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

package tenant_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

type fakeVerifier struct {
	verifyFunc func(ctx context.Context, bkUsername, tenantID string) error
}

func (f fakeVerifier) Verify(ctx context.Context, bkUsername, tenantID string) error {
	return f.verifyFunc(ctx, bkUsername, tenantID)
}

var _ = Describe("Tenant middleware", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	It("injects default tenant when multi-tenant mode is disabled and header is empty", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(false, fakeVerifier{}))
		router.GET("/", func(c *gin.Context) {
			tenantID, ok := tenant.GetTenantID(c.Request.Context())
			Expect(ok).To(BeTrue())
			Expect(tenantID).To(Equal("default"))
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
	})

	It("accepts default tenant when multi-tenant mode is disabled", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(false, fakeVerifier{}))
		router.GET("/", func(c *gin.Context) {
			tenantID, ok := tenant.GetTenantID(c.Request.Context())
			Expect(ok).To(BeTrue())
			Expect(tenantID).To(Equal("default"))
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "default")
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
	})

	It("rejects non-default tenant when multi-tenant mode is disabled", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(false, fakeVerifier{}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "tenant-a")
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
	})

	It("requires tenant header when multi-tenant mode is enabled", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(true, fakeVerifier{}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "alice"}))
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
	})

	It("verifies authenticated user against the requested tenant when enabled", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(true, fakeVerifier{
			verifyFunc: func(ctx context.Context, bkUsername, tenantID string) error {
				Expect(bkUsername).To(Equal("alice"))
				Expect(tenantID).To(Equal("tenant-a"))
				return nil
			},
		}))
		router.GET("/", func(c *gin.Context) {
			tenantID, ok := tenant.GetTenantID(c.Request.Context())
			Expect(ok).To(BeTrue())
			Expect(tenantID).To(Equal("tenant-a"))
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "tenant-a")
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "alice"}))
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
	})

	It("reuses the tenant id from auth without calling the verifier", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(true, fakeVerifier{
			verifyFunc: func(ctx context.Context, bkUsername, tenantID string) error {
				Fail("verifier should not be called when auth already has tenant id")
				return nil
			},
		}))
		router.GET("/", func(c *gin.Context) {
			tenantID, ok := tenant.GetTenantID(c.Request.Context())
			Expect(ok).To(BeTrue())
			Expect(tenantID).To(Equal("tenant-a"))
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "tenant-a")
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "alice", TenantID: "tenant-a"}))
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusNoContent))
	})

	It("rejects request when auth tenant id does not match the header", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(true, fakeVerifier{
			verifyFunc: func(ctx context.Context, bkUsername, tenantID string) error {
				Fail("verifier should not be called when auth already has tenant id")
				return nil
			},
		}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "tenant-a")
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "alice", TenantID: "tenant-b"}))
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
	})

	It("rejects request when verifier denies the tenant", func() {
		recorder := httptest.NewRecorder()
		router := gin.New()
		router.Use(tenant.Required(true, fakeVerifier{
			verifyFunc: func(ctx context.Context, bkUsername, tenantID string) error {
				return tenant.ErrTenantAccessDenied
			},
		}))
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Bk-Tenant-Id", "tenant-a")
		req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "alice"}))
		router.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
	})
})
