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
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("ApiClient", func() {
	newClientWithServer := func(handler http.HandlerFunc) Client {
		server := httptest.NewServer(handler)
		DeferCleanup(server.Close)

		original := config.G
		config.G = &config.Config{}
		config.G.BKRepo.BaseUrl = server.URL
		DeferCleanup(func() { config.G = original })

		client, err := New("operator")
		Expect(err).NotTo(HaveOccurred())
		return client
	}

	It("includes status code and raw body when the response is not successful", func() {
		client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":250103,"message":"platform account invalid"}`))
		})

		err := client.CreateProject(context.Background(), "demo")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("401"))
		Expect(err.Error()).To(ContainSubstring("platform account invalid"))
	})

	It("reports an empty non-json body without decode error", func() {
		client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})

		err := client.CreateProject(context.Background(), "demo")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("status code: 401"))
		Expect(err.Error()).NotTo(ContainSubstring("decode"))
	})

	It("treats project already exists as success", func() {
		client := newClientWithServer(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":251005,"message":"project exists"}`))
		})

		Expect(client.CreateProject(context.Background(), "demo")).To(Succeed())
	})
})
