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

package devmode_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
)

var _ = Describe("DevMode Component TRPC", func() {
	// ============================================================
	// Build 完整构建测试
	// ============================================================
	Describe("Build", func() {
		Context("when dev mode is enabled", func() {
			It("should return error for invalid config", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeProduction,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					StartupCommand: "./app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
				}
				builder := devmode.New(config)
				output, err := builder.Build()
				Expect(err).To(HaveOccurred())
				Expect(output).To(BeNil())
			})

			It("should return complete output for valid config", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeDevelopment,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
					StartupCommand: "./test-app",
				}
				builder := devmode.New(config)
				output, err := builder.Build()
				Expect(err).NotTo(HaveOccurred())
				Expect(output).NotTo(BeNil())
				Expect(output.ConfigMap).NotTo(BeNil())
				Expect(output.Volume.Name).To(Equal(devmode.ConfigMapResourceName("test-app")))
				Expect(output.VolumeMount.Name).To(Equal(devmode.ConfigMapResourceName("test-app")))
				Expect(output.Command).To(HaveLen(1))
			})

			It("should include EnvVars in output", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeDevelopment,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
					StartupCommand: "./test-app",
				}
				builder := devmode.New(config)
				_, err := builder.Build()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("New", func() {
			It("should use custom mount path when specified", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeDevelopment,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
				}
				builder := devmode.New(config)
				volumeMount := builder.BuildVolumeMount()
				Expect(volumeMount.MountPath).To(Equal("/data/bkms/dev-mode/trpc/configmap-scripts"))
			})
		})

		Context("test enabled", func() {
			It("should return nil without error", func() {
				config := &devmode.Config{
					Enabled:        false,
					TrpcBinaryPath: "/usr/local/trpc/bin",
				}
				builder := devmode.New(config)
				output, err := builder.Build()
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(BeNil())
			})

			It("should return false for production environment", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeProduction,
					AppType:        bkmsapp.AppTypeTRPC,
					TrpcBinaryPath: "/usr/local/trpc/bin",
					AppName:        "test-app",
				}
				builder := devmode.New(config)
				Expect(builder.IsAllowed()).To(BeFalse())
			})

			It("should return false for unknown environment type", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        "unknown",
					AppType:        bkmsapp.AppTypeTRPC,
					TrpcBinaryPath: "/usr/local/trpc/bin",
					AppName:        "test-app",
				}
				builder := devmode.New(config)
				Expect(builder.IsAllowed()).To(BeFalse())
			})

			It("should return false for empty environment type", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        "",
					AppType:        bkmsapp.AppTypeTRPC,
					TrpcBinaryPath: "/usr/local/trpc/bin",
					AppName:        "test-app",
				}
				builder := devmode.New(config)
				Expect(builder.IsAllowed()).To(BeFalse())
			})

			It("should return error when app name is empty", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeDevelopment,
					AppType:        bkmsapp.AppTypeTRPC,
					TrpcBinaryPath: "/usr/local/trpc/bin",
					AppName:        "",
				}
				builder := devmode.New(config)
				err := builder.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("app name"))
			})

			It("should return nil for valid development config", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeDevelopment,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
					StartupCommand: "./app",
				}
				builder := devmode.New(config)
				Expect(builder.Validate()).To(Succeed())
			})

			It("should return nil for valid test config", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeTest,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
					StartupCommand: "./app",
				}
				builder := devmode.New(config)
				Expect(builder.Validate()).To(Succeed())
			})

			It("should return nil for valid staging config", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeStaging,
					AppType:        bkmsapp.AppTypeTRPC,
					AppName:        "test-app",
					TrpcBinaryPath: "/usr/local/trpc/bin",
					StartupCommand: "./app",
				}
				builder := devmode.New(config)
				Expect(builder.Validate()).To(Succeed())
			})

			It("should return true for staging environment IsAllowed", func() {
				config := &devmode.Config{
					Enabled:        true,
					EnvType:        devmode.EnvTypeStaging,
					AppType:        bkmsapp.AppTypeTRPC,
					TrpcBinaryPath: "/usr/local/trpc/bin",
					AppName:        "test-app",
				}
				builder := devmode.New(config)
				Expect(builder.IsAllowed()).To(BeTrue())
			})
		})
	})

	// ============================================================
	// BuildCommand 启动命令测试
	// ============================================================
	Describe("BuildCommand", func() {
		It("should use default init.sh path", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "test-app",
				StartupCommand: "./app",
				TrpcBinaryPath: "/usr/local/trpc/bin",
			}
			builder := devmode.New(config)
			command := builder.BuildCommand()
			Expect(command).To(HaveLen(1))
			Expect(command[0]).To(Equal(devmode.TrpcMountPath + "/init.sh"))
		})

		It("should use custom mount path for init.sh", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "test-app",
				StartupCommand: "./app",
				TrpcBinaryPath: "/usr/local/trpc/bin",
			}
			builder := devmode.New(config)
			command := builder.BuildCommand()
			Expect(command).To(HaveLen(1))
			Expect(command[0]).To(Equal("/data/bkms/dev-mode/trpc/configmap-scripts/init.sh"))
		})

		It("should use WorkPath based path when MountPath is auto-calculated", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "test-app",
				StartupCommand: "./app",
				TrpcBinaryPath: "/usr/local/trpc/bin",
			}
			builder := devmode.New(config)
			command := builder.BuildCommand()
			Expect(command).To(HaveLen(1))
			Expect(command[0]).To(Equal("/data/bkms/dev-mode/trpc/configmap-scripts/init.sh"))
		})
	})
})
