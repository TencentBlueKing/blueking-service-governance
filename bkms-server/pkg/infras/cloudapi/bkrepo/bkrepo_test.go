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

package bkrepo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var _ = Describe("ApiClient", func() {
	type capturedRequest struct {
		auth string
		body []byte
	}

	var (
		originalCfg *config.Config
		server      *httptest.Server
		requests    chan capturedRequest
	)

	// lastRequest 取出服务端收到的请求，返回认证头与解码后的请求体
	lastRequest := func() (string, map[string]any) {
		var req capturedRequest
		Eventually(requests).Should(Receive(&req))
		var body map[string]any
		Expect(json.Unmarshal(req.body, &body)).To(Succeed())
		return req.auth, body
	}

	BeforeEach(func() {
		originalCfg = config.G
		requests = make(chan capturedRequest, 1)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			requests <- capturedRequest{auth: r.Header.Get("Authorization"), body: body}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code": 0, "data": "ok"}`))
		}))
		config.G = &config.Config{}
		config.G.BKRepo.BaseUrl = server.URL
		config.G.BKRepo.Username = "admin"
		config.G.BKRepo.Password = "secret"
	})

	AfterEach(func() {
		server.Close()
		config.G = originalCfg
	})

	expectedCredential := base64.StdEncoding.EncodeToString([]byte("admin:secret"))

	It("should use platform auth by default", func() {
		client, err := New("alice")
		Expect(err).NotTo(HaveOccurred())

		Expect(client.CreateProject(context.Background(), "demo")).To(Succeed())
		auth, _ := lastRequest()
		Expect(auth).To(Equal("Platform " + expectedCredential))
	})

	It("should use basic auth for independent bkrepo", func() {
		config.G.FeatureIntegrations.EnableIndependentBkRepo = true
		client, err := New("alice")
		Expect(err).NotTo(HaveOccurred())

		Expect(client.CreateProject(context.Background(), "demo")).To(Succeed())
		auth, _ := lastRequest()
		Expect(auth).To(Equal("Basic " + expectedCredential))
	})

	It("should keep project name as is when multi-tenant mode is disabled", func() {
		client, err := New("alice")
		Expect(err).NotTo(HaveOccurred())

		ctx := tenant.WithTenantID(context.Background(), "system")
		Expect(client.CreateProject(ctx, "system.demo")).To(Succeed())
		_, body := lastRequest()
		Expect(body["name"]).To(Equal("system.demo"))
	})

	It("should trim tenant prefix from project name in multi-tenant mode", func() {
		config.G.Tenant.EnableMultiTenantMode = true
		client, err := New("alice")
		Expect(err).NotTo(HaveOccurred())

		ctx := tenant.WithTenantID(context.Background(), "system")
		Expect(client.CreateProject(ctx, "system.demo")).To(Succeed())
		_, body := lastRequest()
		Expect(body["name"]).To(Equal("demo"))
		Expect(body["displayName"]).To(Equal("demo"))
	})

	It("should not trim prefix of another tenant", func() {
		config.G.Tenant.EnableMultiTenantMode = true
		client, err := New("alice")
		Expect(err).NotTo(HaveOccurred())

		ctx := tenant.WithTenantID(context.Background(), "tenant-a")
		Expect(client.CreateProject(ctx, "system.demo")).To(Succeed())
		_, body := lastRequest()
		Expect(body["name"]).To(Equal("system.demo"))
	})
})
