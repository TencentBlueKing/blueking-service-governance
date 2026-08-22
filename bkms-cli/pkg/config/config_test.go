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
	"gopkg.in/yaml.v3"
)

var _ = Describe("Config", func() {
	var (
		tmpDir     string
		cfgFile    string
		origConfig *Config
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "bkms-cli-config-test-*")
		Expect(err).ToNot(HaveOccurred())

		// 使用测试专属配置文件名，避免覆盖已有配置
		cfgFile = filepath.Join(tmpDir, ".bkms", "test-config.yaml")
		cfgFilePath = cfgFile

		origConfig = G
		G = &Config{}
	})

	AfterEach(func() {
		G = origConfig
		os.RemoveAll(tmpDir)
	})

	Describe("Load", func() {
		It("配置文件存在时，应正确读取并去除尾斜杠", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())

			cfg := &Config{
				BkmsBaseURL: "http://existing.example.com/",
				BcsAPIHost:  "https://bcs.example.com/",
				Username:    "testuser",
				AccessToken: "test-token",
				Defaults:    Defaults{WorkspaceID: "ws-123"},
			}
			data, err := yaml.Marshal(cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(cfgFile, data, 0o600)).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://existing.example.com"))
			Expect(conf.BcsAPIHost).To(Equal("https://bcs.example.com"))
			Expect(conf.Username).To(Equal("testuser"))
			Expect(conf.AccessToken).To(Equal("test-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-123"))
			Expect(G).To(Equal(conf))
		})

		It("配置文件不存在时，应自动创建空配置", func() {
			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal(""))
			Expect(conf.BcsAPIHost).To(Equal(""))
			Expect(G).To(Equal(conf))

			_, statErr := os.Stat(cfgFile)
			Expect(statErr).ToNot(HaveOccurred())
		})

		It("配置文件内容无效时，应返回错误", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())
			Expect(os.WriteFile(cfgFile, []byte("invalid: [yaml: content"), 0o600)).To(Succeed())

			_, err := G.Load()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Dump", func() {
		It("Dump 后 Load 应读取到一致的配置", func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())

			G = &Config{
				BkmsBaseURL: "http://dump.example.com",
				BcsAPIHost:  "https://bcs-dump.example.com",
				Username:    "dumpuser",
				AccessToken: "dump-token",
				Defaults:    Defaults{WorkspaceID: "ws-dump"},
			}
			Expect(G.Dump()).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://dump.example.com"))
			Expect(conf.BcsAPIHost).To(Equal("https://bcs-dump.example.com"))
			Expect(conf.Username).To(Equal("dumpuser"))
			Expect(conf.AccessToken).To(Equal("dump-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-dump"))
		})
	})

	Describe("String", func() {
		It("应展示普通字段并隐藏敏感字段", func() {
			G = &Config{
				BkmsBaseURL: "http://string-test.example.com",
				BcsAPIHost:  "https://bcs-string.example.com",
				Username:    "stringuser",
				AccessToken: "secret-token",
			}

			s := G.String()
			Expect(s).To(ContainSubstring("http://string-test.example.com"))
			Expect(s).To(ContainSubstring("https://bcs-string.example.com"))
			Expect(s).To(ContainSubstring("stringuser"))
			Expect(s).ToNot(ContainSubstring("secret-token"))
			Expect(s).To(ContainSubstring("[REDACTED]"))
		})
	})

	Describe("SetEndpoints", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Dir(cfgFile), 0o755)).To(Succeed())
			G = &Config{}
		})

		It("应写入地址并持久化", func() {
			updated, err := G.SetEndpoints("http://bkms.example.com/", "https://bcs.example.com/", false)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Changed()).To(BeTrue())
			Expect(updated.BkmsBaseURLUpdated).To(BeTrue())
			Expect(updated.BcsAPIHostUpdated).To(BeTrue())
			Expect(G.BkmsBaseURL).To(Equal("http://bkms.example.com"))
			Expect(G.BcsAPIHost).To(Equal("https://bcs.example.com"))

			conf, err := (&Config{}).Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://bkms.example.com"))
			Expect(conf.BcsAPIHost).To(Equal("https://bcs.example.com"))
		})

		It("ifUnset 时不应覆盖已有值", func() {
			G.BkmsBaseURL = "http://existing.example.com"
			G.BcsAPIHost = "https://existing-bcs.example.com"
			Expect(G.Dump()).To(Succeed())

			updated, err := G.SetEndpoints("http://new.example.com", "https://new-bcs.example.com", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Changed()).To(BeFalse())
			Expect(G.BkmsBaseURL).To(Equal("http://existing.example.com"))
			Expect(G.BcsAPIHost).To(Equal("https://existing-bcs.example.com"))
		})

		It("ifUnset 时应仅填充空字段", func() {
			G.BkmsBaseURL = "http://existing.example.com"
			Expect(G.Dump()).To(Succeed())

			updated, err := G.SetEndpoints("http://new.example.com", "https://new-bcs.example.com", true)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.BkmsBaseURLUpdated).To(BeFalse())
			Expect(updated.BcsAPIHostUpdated).To(BeTrue())
			Expect(G.BkmsBaseURL).To(Equal("http://existing.example.com"))
			Expect(G.BcsAPIHost).To(Equal("https://new-bcs.example.com"))
		})

		It("两个地址都为空时应返回错误", func() {
			_, err := G.SetEndpoints("  ", "", false)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RequireBkmsBaseURL", func() {
		It("地址为空时应返回引导错误", func() {
			G = &Config{}
			err := G.RequireBkmsBaseURL()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bkms-cli config set"))
		})

		It("地址已配置时应通过", func() {
			G = &Config{BkmsBaseURL: "https://bkms.example.com"}
			Expect(G.RequireBkmsBaseURL()).To(Succeed())
			Expect(G.HasBkmsBaseURL()).To(BeTrue())
		})
	})
})
