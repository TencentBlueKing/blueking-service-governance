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
		bkmsBaseURL = "http://test-bkms.example.com"

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
				Username:    "testuser",
				AccessToken: "test-token",
				Defaults:    Defaults{WorkspaceID: "ws-123"},
				BCS:         BCSConfig{Token: "bcs-token-abc"},
			}
			data, err := yaml.Marshal(cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(cfgFile, data, 0o600)).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://existing.example.com"))
			Expect(conf.Username).To(Equal("testuser"))
			Expect(conf.AccessToken).To(Equal("test-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-123"))
			Expect(conf.BCS.Token).To(Equal("bcs-token-abc"))
			Expect(G).To(Equal(conf))
		})

		It("配置文件不存在时，应自动创建目录、文件并使用默认 bkmsBaseUrl", func() {
			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://test-bkms.example.com"))
			Expect(G).To(Equal(conf))

			// 验证文件已创建且可被再次读取
			_, statErr := os.Stat(cfgFile)
			Expect(statErr).ToNot(HaveOccurred())

			conf2, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf2.BkmsBaseURL).To(Equal("http://test-bkms.example.com"))
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
				Username:    "dumpuser",
				AccessToken: "dump-token",
				Defaults:    Defaults{WorkspaceID: "ws-dump"},
				BCS:         BCSConfig{Token: "bcs-dump"},
			}
			Expect(G.Dump()).To(Succeed())

			conf, err := G.Load()
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.BkmsBaseURL).To(Equal("http://dump.example.com"))
			Expect(conf.Username).To(Equal("dumpuser"))
			Expect(conf.AccessToken).To(Equal("dump-token"))
			Expect(conf.Defaults.WorkspaceID).To(Equal("ws-dump"))
			Expect(conf.BCS.Token).To(Equal("bcs-dump"))
		})
	})

	Describe("String", func() {
		It("应展示普通字段并隐藏敏感字段", func() {
			G = &Config{
				BkmsBaseURL: "http://string-test.example.com",
				Username:    "stringuser",
				AccessToken: "secret-token",
				BCS:         BCSConfig{Token: "secret-bcs-token"},
			}

			s := G.String()
			Expect(s).To(ContainSubstring("http://string-test.example.com"))
			Expect(s).To(ContainSubstring("stringuser"))
			Expect(s).ToNot(ContainSubstring("secret-token"))
			Expect(s).ToNot(ContainSubstring("secret-bcs-token"))
			Expect(s).To(ContainSubstring("[REDACTED]"))
		})
	})
})
