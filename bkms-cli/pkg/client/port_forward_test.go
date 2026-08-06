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
	"net/url"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildWebSocketURL", func() {
	buildURL := func(baseURL, path string, query url.Values) (string, error) {
		cli := &SvcBasedClient{
			cli: resty.New().SetBaseURL(baseURL),
		}
		return cli.buildWebSocketURL(path, query)
	}

	It("converts http scheme to ws", func() {
		result, err := buildURL("http://localhost:8080", "/api/v1/connect", url.Values{"port": {"8080"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("ws://localhost:8080/api/v1/connect?port=8080"))
	})

	It("converts https scheme to wss", func() {
		result, err := buildURL(
			"https://bkms.example.com",
			"/bkms/v1/apps/myapp/envs/test/instances/pod-1/port-forward/connect",
			url.Values{"remotePort": {"80"}, "localPort": {"8080"}},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(
			result,
		).To(Equal("wss://bkms.example.com/bkms/v1/apps/myapp/envs/test/instances/pod-1/port-forward/connect?localPort=8080&remotePort=80"))
	})

	It("preserves base url path prefix", func() {
		result, err := buildURL("http://gateway.example.com/prefix", "/api/connect", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("ws://gateway.example.com/prefix/api/connect"))
	})

	It("preserves ws scheme", func() {
		result, err := buildURL("ws://localhost:9090", "/tunnel", url.Values{"id": {"abc"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("ws://localhost:9090/tunnel?id=abc"))
	})

	It("preserves wss scheme", func() {
		result, err := buildURL("wss://secure.example.com", "/tunnel", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("wss://secure.example.com/tunnel"))
	})

	It("returns error for empty base url", func() {
		_, err := buildURL("", "/api", nil)
		Expect(err).To(HaveOccurred())
	})

	It("returns error for unsupported scheme", func() {
		_, err := buildURL("ftp://example.com", "/api", nil)
		Expect(err).To(HaveOccurred())
	})

	It("handles path with special characters", func() {
		result, err := buildURL(
			"http://localhost:8080",
			"/apps/my%20app/connect",
			url.Values{"key": {"value with spaces"}},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("ws://localhost:8080/apps/my%20app/connect?key=value+with+spaces"))
	})
})
