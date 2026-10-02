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

package update

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/version"
)

var _ = Describe("Update", func() {
	var requests []string

	BeforeEach(func() {
		oldVersion, oldConfig := version.Version, config.G
		DeferCleanup(func() { version.Version, config.G = oldVersion, oldConfig })
		version.Version = "1.2.0"
		GinkgoT().Setenv("BKMS_CLI_INSTALL_SOURCE", "npm")
		requests = nil
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.URL.Path)
			if r.URL.Path == "/latest.txt" {
				_, _ = w.Write([]byte("1.3.0"))
				return
			}
			http.NotFound(w, r)
		}))
		DeferCleanup(server.Close)
		config.G = &config.Config{Update: config.UpdateSource{
			LatestVersionURL:    server.URL + "/latest.txt",
			DownloadURLTemplate: server.URL + "/v{version}/{archive}",
		}}
	})

	It("keeps check-only read-only even with force", func() {
		Expect(Run(context.Background(), true, true)).To(Succeed())
		Expect(requests).To(Equal([]string{"/latest.txt"}))
	})
})
