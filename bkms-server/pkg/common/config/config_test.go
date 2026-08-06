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

package config_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("Configuration Load", func() {
	var tempDir string
	var configFile string
	var ctx context.Context

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "config-test")
		Expect(err).NotTo(HaveOccurred())
		configFile = filepath.Join(tempDir, "test-config.yaml")

		ctx = context.Background()
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		// Reset global config
		config.G = nil
	})

	Context("Load", func() {
		It("should successfully load a valid config file", func() {
			originalSecret := "my-secret-key-123"
			encodedSecret := base64.StdEncoding.EncodeToString([]byte(originalSecret))

			// Create a valid config file
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
bkPlatUrls:
  bkApiUrlTmpl: http://{api_name}.example.com
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
  authEnvName: test
  backendType: bk_token
bkApiStages:
  bcs: prod
encrypt:
  secret: ` + encodedSecret + `
mongo:
  username: testuser
  password: testpass
  host: localhost
  port: 27017
  database: testdb
asynq:
  redis:
    host: localhost
    port: 6380
    db: 1
    password: asynqpass
# 蓝鲸监控 apm 配置
bkMonitor:
  endpoint: "https://bkmonitorv3.example.com"
  gatewayEndpoint: "https://bk-monitor.example.com"
  # 蓝鲸监控 APM 上报地址
  apmEndpoint: "bk-example.com:4317"
  # 蓝鲸监控 APM http 上报地址
  apmHttpEndpoint: "http://bk-example.com:4318"
taskPoller:
  deployStatus:
    timeout: 1200
    interval: 15
bkci:
  pipelineTmpl:
    baseDir: /app/assets/pipeline_templates
    builderImageCode: tlinux3_custom
    builderImageVersion: 3.*
# Prometheus Metrics Server 配置
metrics:
  # Metrics HTTP Server 监听端口
  port: 8081
httpServer:
  address: 127.0.0.1
  port: 32303
  readHeaderTimeout: 11
  readTimeout: 61
  writeTimeout: 62
  idleTimeout: 121
  shutdownTimeout: 31
logging:
  level: debug
  handlerName: json
  writers:
    - writerName: file
      writerConfig:
        filename: /tmp/bkms-server-test.log
        maxSize: 128
        maxBackups: 7
        maxAge: 14
        compress: true
# Enable development flags for testing
development:
  useStubPerm: true
  allowSetUserInHeader: true
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			// Load the config
			cfg, err := config.Load(ctx, configFile)

			// Verify no error
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			// Verify config values
			Expect(cfg.BkApp.Code).To(Equal("test-app"))
			Expect(cfg.BkApp.Secret).To(Equal("test-secret"))
			Expect(cfg.BkPlatUrls.BkApiUrlTmpl).To(Equal("http://{api_name}.example.com"))
			Expect(cfg.BkApiStages.BCS).To(Equal("prod"))
			Expect(cfg.Account.AuthBaseURL).To(Equal("http://auth.example.com"))
			Expect(cfg.Account.LoginURL).To(Equal("http://login.example.com"))
			Expect(cfg.Account.AuthEnvName).To(Equal("test"))
			Expect(cfg.Account.BackendType).To(Equal("bk_token"))
			Expect(cfg.BkMonitor.Endpoint).To(Equal("https://bkmonitorv3.example.com"))
			Expect(cfg.BkMonitor.GatewayEndpoint).To(Equal("https://bk-monitor.example.com"))
			Expect(config.G.Encrypt.Secret).To(Equal(originalSecret))
			Expect(cfg.Mongo.Username).To(Equal("testuser"))
			Expect(cfg.Mongo.Password).To(Equal("testpass"))
			Expect(cfg.Mongo.Host).To(Equal("localhost"))
			Expect(cfg.Mongo.Port).To(Equal("27017"))
			Expect(cfg.Mongo.Database).To(Equal("testdb"))
			Expect(cfg.Asynq.Redis.Host).To(Equal("localhost"))
			Expect(cfg.Asynq.Redis.Port).To(Equal("6380"))
			Expect(cfg.Asynq.Redis.DB).To(Equal(1))
			Expect(cfg.Asynq.Redis.Password).To(Equal("asynqpass"))
			Expect(cfg.TaskPoller.DeployStatus.Timeout).To(Equal(1200))
			Expect(cfg.TaskPoller.DeployStatus.Interval).To(Equal(15))
			Expect(cfg.BKCI.PipelineTmpl.BaseDir).To(Equal("/app/assets/pipeline_templates"))
			Expect(cfg.BKCI.PipelineTmpl.BuilderImageCode).To(Equal("tlinux3_custom"))
			Expect(cfg.BKCI.PipelineTmpl.BuilderImageVersion).To(Equal("3.*"))
			Expect(cfg.HTTPServer.Address).To(Equal("127.0.0.1"))
			Expect(cfg.HTTPServer.Port).To(Equal(uint(32303)))
			Expect(cfg.HTTPServer.ReadHeaderTimeout).To(Equal(11))
			Expect(cfg.HTTPServer.ReadTimeout).To(Equal(61))
			Expect(cfg.HTTPServer.WriteTimeout).To(Equal(62))
			Expect(cfg.HTTPServer.IdleTimeout).To(Equal(121))
			Expect(cfg.HTTPServer.ShutdownTimeout).To(Equal(31))
			Expect(cfg.Logging.Level).To(Equal("debug"))
			Expect(cfg.Logging.HandlerName).To(Equal("json"))
			Expect(cfg.Logging.Writers).To(HaveLen(1))
			Expect(cfg.Logging.Writers[0].WriterName).To(Equal("file"))
			Expect(cfg.Logging.Writers[0].WriterConfig.Filename).To(Equal("/tmp/bkms-server-test.log"))
			Expect(cfg.Logging.Writers[0].WriterConfig.MaxSize).To(Equal(128))
			Expect(cfg.Logging.Writers[0].WriterConfig.MaxBackups).To(Equal(7))
			Expect(cfg.Logging.Writers[0].WriterConfig.MaxAge).To(Equal(14))
			Expect(cfg.Logging.Writers[0].WriterConfig.Compress).To(BeTrue())

			Expect(cfg.Development.UseStubPerm).To(BeTrue())
			Expect(cfg.Development.AllowSetUserInHeader).To(BeTrue())

			// Verify global config is set
			Expect(config.G).NotTo(BeNil())
			Expect(config.G.BkApp.Code).To(Equal("test-app"))
		})
		It("should successfully load a minimal config file", func() {
			// Create a valid config file
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
# 蓝鲸监控 apm 配置
bkMonitor:
  endpoint: "https://bkmonitorv3.example.com"
  gatewayEndpoint: "https://bk-monitor.example.com"
  # 蓝鲸监控 APM 上报地址
  apmEndpoint: "bk-example.com:4317"
  # 蓝鲸监控 APM http 上报地址
  apmHttpEndpoint: "http://bk-example.com:4318"
metrics:
  port: 8081
httpServer:
  address: 127.0.0.1
  port: 32303
asynq:
  redis:
    host: localhost
    port: 6380
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := config.Load(ctx, configFile)

			// Verify no error
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())

			// Verify config values
			Expect(cfg.BkApp.Code).To(Equal("test-app"))
			Expect(cfg.BkApp.Secret).To(Equal("test-secret"))
			Expect(cfg.Development.UseStubPerm).To(BeFalse())
			Expect(cfg.Development.AllowSetUserInHeader).To(BeFalse())
			Expect(cfg.HTTPServer.Mode).To(Equal("release"))
			Expect(cfg.HTTPServer.EnableSwaggerPath).To(BeFalse())
			Expect(cfg.Metrics.Port).To(Equal(uint(8081)))
			Expect(cfg.HTTPServer.Address).To(Equal("127.0.0.1"))
			Expect(cfg.HTTPServer.Port).To(Equal(uint(32303)))
			Expect(cfg.HTTPServer.ReadHeaderTimeout).To(Equal(config.DefaultHTTPServerReadHeaderTimeout))
			Expect(cfg.HTTPServer.ReadTimeout).To(Equal(config.DefaultHTTPServerReadTimeout))
			Expect(cfg.HTTPServer.WriteTimeout).To(Equal(config.DefaultHTTPServerWriteTimeout))
			Expect(cfg.HTTPServer.IdleTimeout).To(Equal(config.DefaultHTTPServerIdleTimeout))
			Expect(cfg.HTTPServer.ShutdownTimeout).To(Equal(config.DefaultHTTPServerShutdownTimeout))
			Expect(cfg.Account.AuthEnvName).To(Equal("prod"))
			Expect(cfg.Account.BackendType).To(Equal("bk_token"))
			Expect(cfg.BkMonitor.Endpoint).To(Equal("https://bkmonitorv3.example.com"))
			Expect(cfg.BkMonitor.GatewayEndpoint).To(Equal("https://bk-monitor.example.com"))
			Expect(cfg.BKCI.PipelineTmpl.BuilderImageCode).To(BeEmpty())
			Expect(cfg.BKCI.PipelineTmpl.BuilderImageVersion).To(BeEmpty())
		})

		It("should successfully load config without bkMonitor section", func() {
			// bkMonitor 段完全缺省时，config 仍能成功加载，BkMonitor 字段应为零值
			// 该用例保护"apm 相关配置全为可选"这一契约，防止未来误加必填校验
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
metrics:
  port: 8081
httpServer:
  address: 127.0.0.1
  port: 32303
asynq:
  redis:
    host: localhost
    port: 6380
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := config.Load(ctx, configFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.BkMonitor).To(Equal(config.BkMonitorConfig{}))
			Expect(cfg.BkMonitor.APMEndpoint).To(BeEmpty())
			Expect(cfg.BkMonitor.APMHttpEndpoint).To(BeEmpty())
			Expect(cfg.BkMonitor.APMToken).To(BeEmpty())
			Expect(cfg.BkMonitor.APMServiceName).To(BeEmpty())
		})

		It("should fail when http server port is invalid", func() {
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
metrics:
  port: 8081
httpServer:
  address: 127.0.0.1
  port: 70000
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'Port' failed on the 'lte' tag"))
		})

		It("should fail when metrics port is invalid", func() {
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
metrics:
  port: 70000
httpServer:
  address: 127.0.0.1
  port: 32303
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'Port' failed on the 'lte' tag"))
		})

		It("should fail when metrics port is missing", func() {
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
httpServer:
  address: 127.0.0.1
  port: 32303
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validate MetricsConfig"))
			Expect(err.Error()).To(ContainSubstring("'Port' failed on the 'required' tag"))
		})

		It("should fail when http server address is missing", func() {
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
metrics:
  port: 8081
httpServer:
  port: 32303
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validate HTTPServerConfig"))
			Expect(err.Error()).To(ContainSubstring("'Address' failed on the 'required' tag"))
		})

		It("should fail when http server port is missing", func() {
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
metrics:
  port: 8081
httpServer:
  address: 127.0.0.1
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validate HTTPServerConfig"))
			Expect(err.Error()).To(ContainSubstring("'Port' failed on the 'required' tag"))
		})

		It("should validate http server mode fail", func() {
			// Create a valid config file
			configContent := `
bkApp:
  code: test-app
  secret: test-secret
account:
  authBaseURL: http://auth.example.com
  loginURL: http://login.example.com
bkMonitor:
  apmEndpoint: "bk-example.com:4317"
  apmHttpEndpoint: "http://bk-example.com:4318"
metrics:
  port: 8081
httpServer:
  address: 127.0.0.1
  port: 32303
  # 设置为错误的 Mode 来触发验证失败
  mode: invalid-mode
`
			err := os.WriteFile(configFile, []byte(configContent), 0o644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(ctx, configFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'Mode' failed on the 'oneof' tag"))
		})
	})
})
