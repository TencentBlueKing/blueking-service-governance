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

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Update source configuration", func() {
	var internal UpdateSource

	BeforeEach(func() {
		oldPath, oldConfig := cfgFilePath, G
		DeferCleanup(func() { cfgFilePath, G = oldPath, oldConfig })
		cfgFilePath = filepath.Join(GinkgoT().TempDir(), "config.yaml")
		G = &Config{}
		internal = UpdateSource{
			LatestVersionURL:    "https://artifacts.example.com/bkms-cli/releases/latest.txt",
			DownloadURLTemplate: "https://artifacts.example.com/bkms-cli/releases/v{version}/{archive}",
		}
	})

	It("uses official defaults for an unconfigured source", func() {
		source, err := G.UpdateSource()
		Expect(err).NotTo(HaveOccurred())
		Expect(source).To(Equal(UpdateSource{
			LatestVersionURL:    DefaultUpdateLatestURL,
			DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
		}))
	})

	It("preserves an existing source when ifUnset is explicitly requested", func() {
		G.Update = internal
		updated, err := G.SetEndpoints("", UpdateSource{
			LatestVersionURL:    DefaultUpdateLatestURL,
			DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
		}, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeFalse())
		Expect(G.Update).To(Equal(internal))
	})

	It("never fills a partly configured source from another channel", func() {
		G.Update.LatestVersionURL = "https://old.internal/latest.txt"
		updated, err := G.SetEndpoints("https://bkms.example.com", internal, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).To(BeTrue())
		Expect(G.Update.LatestVersionURL).To(Equal("https://old.internal/latest.txt"))
		Expect(G.Update.DownloadURLTemplate).To(BeEmpty())
		_, err = G.UpdateSource()
		Expect(err).To(HaveOccurred())
	})

	It("does not save the service URL when the update pair is invalid", func() {
		G.BkmsBaseURL = "https://old.example.com"
		Expect(G.Dump()).To(Succeed())
		before, err := os.ReadFile(cfgFilePath)
		Expect(err).NotTo(HaveOccurred())
		_, err = G.SetEndpoints(
			"https://new.example.com",
			UpdateSource{LatestVersionURL: internal.LatestVersionURL},
			false,
		)
		Expect(err).To(HaveOccurred())
		after, err := os.ReadFile(cfgFilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(after).To(Equal(before))
		Expect(G.BkmsBaseURL).To(Equal("https://old.example.com"))
	})

	DescribeTable("persists the replacement source and changes only supplied settings",
		func(baseURL, expectedBaseURL string) {
			G.BkmsBaseURL = "https://old.example.com"
			G.Update = internal
			G.Username, G.AccessToken = "user", "token"
			official := UpdateSource{
				LatestVersionURL:    DefaultUpdateLatestURL,
				DownloadURLTemplate: DefaultUpdateDownloadURLTemplate,
			}
			_, err := G.SetEndpoints(baseURL, official, false)
			Expect(err).NotTo(HaveOccurred())
			loaded, err := (&Config{}).Load()
			Expect(err).NotTo(HaveOccurred())
			Expect(loaded.Update).To(Equal(official))
			Expect(loaded.BkmsBaseURL).To(Equal(expectedBaseURL))
			Expect(loaded.Username).To(Equal("user"))
			Expect(loaded.AccessToken).To(Equal("token"))
		},
		Entry("omitted base URL", "", "https://old.example.com"),
		Entry("supplied base URL", "https://new.example.com/", "https://new.example.com"),
	)

	It("does not change in-memory config when persistence fails", func() {
		cfgFilePath = filepath.Join(GinkgoT().TempDir(), "missing", "config.yaml")
		_, err := G.SetEndpoints("https://bkms.example.com", internal, false)
		Expect(err).To(HaveOccurred())
		Expect(G.Update.IsZero()).To(BeTrue())
		Expect(G.BkmsBaseURL).To(BeEmpty())
	})

	DescribeTable(
		"validates template syntax and HTTP endpoints",
		func(latest, template string, valid bool) {
			source := UpdateSource{LatestVersionURL: latest, DownloadURLTemplate: template}
			if valid {
				Expect(source.Validate()).To(Succeed())
			} else {
				Expect(source.Validate()).NotTo(Succeed())
			}
		},
		Entry("internal HTTP", "http://mirror/latest.txt", "http://mirror/{archive}", true),
		Entry(
			"query parameters",
			"https://example.com/latest",
			"https://example.com/download?version={version}&file={archive}",
			true,
		),
		Entry("file URL", "file:///latest.txt", "https://example.com/{archive}", false),
		Entry("missing archive", "https://example.com/latest", "https://example.com/{version}/binary", false),
		Entry("unknown placeholder", "https://example.com/latest", "https://example.com/{os}/{archive}", false),
	)
})
