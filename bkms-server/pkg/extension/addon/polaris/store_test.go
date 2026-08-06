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

package polaris_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

var _ = Describe("PolarisConfigStore", func() {
	var store polaris.PolarisConfigStore
	var ctx context.Context
	var diApp *fxtest.App
	var testAppID string

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			polaris.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		// 清理测试数据
		_ = store.DeleteByApp(ctx, testAppID)
		diApp.RequireStop()
	})

	Describe("Create", func() {
		Context("when creating a valid polaris config", func() {
			It("should create successfully with default values", func() {
				config := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:       "custom-test",
						PolarisName:       "custom-polaris",
						PolarisNamespace:  "custom-ns",
						PolarisToken:      "token",
						ServicePort:       9090,
						Direct:            true,
						KeepNotReadyPod:   false,
						EnableHealthCheck: true,
						Weight:            20,
						ServiceLabels: map[string]string{
							"env": "test",
							"app": "demo",
						},
					},
				}

				err := store.Create(ctx, config)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Name).NotTo(BeEmpty())

				// Verify stored config
				storedConfig, err := store.Get(ctx, testAppID, config.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(storedConfig.KeepNotReadyPod).To(BeFalse())
				Expect(storedConfig.EnableHealthCheck).To(BeTrue())
				Expect(storedConfig.ServiceLabels).To(Equal(map[string]string{
					"env": "test",
					"app": "demo",
				}))
				Expect(storedConfig.Weight).To(Equal(int32(20)))
				Expect(storedConfig.Direct).To(BeTrue())
			})
		})

		Context("when creating duplicate config name", func() {
			It("should return ErrConfigNameExists", func() {
				config1 := &polaris.PolarisConfig{
					Name:  "duplicate-name",
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "instance-1",
						PolarisName:      "polaris-1",
						PolarisNamespace: "ns",
						PolarisToken:     "token",
						ServicePort:      8080,
					},
				}
				err := store.Create(ctx, config1)
				Expect(err).NotTo(HaveOccurred())

				config2 := &polaris.PolarisConfig{
					Name:  "duplicate-name",
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "instance-2",
						PolarisName:      "polaris-2",
						PolarisNamespace: "ns",
						PolarisToken:     "token",
						ServicePort:      8081,
					},
				}
				err = store.Create(ctx, config2)
				Expect(err).To(MatchError(polaris.ErrConfigNameExists))
			})
		})
	})

	Describe("Update", func() {
		var configName string

		BeforeEach(func() {
			config := &polaris.PolarisConfig{
				AppID: testAppID,
				Properties: polaris.Properties{
					InstanceKey:      "update-test",
					PolarisName:      "original-name",
					PolarisNamespace: "original-ns",
					PolarisToken:     "original-token",
					ServicePort:      8080,
				},
			}
			err := store.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			configName = config.Name
		})

		Context("when updating an existing config", func() {
			It("should update config successfully", func() {
				newPort := int32(9090)
				err := store.Update(ctx, testAppID, configName, &polaris.ConfigUpdateData{
					ServicePort: &newPort,
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				updatedConfig, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.ServicePort).To(Equal(newPort))
			})
		})

		Context("when updating polarisToken", func() {
			It("should update polarisToken successfully", func() {
				newToken := "new-updated-token"
				err := store.Update(ctx, testAppID, configName, &polaris.ConfigUpdateData{
					PolarisToken: &newToken,
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				updatedConfig, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.PolarisToken).To(Equal(newToken))
				// Verify other fields unchanged
				Expect(updatedConfig.PolarisName).To(Equal("original-name"))
				Expect(updatedConfig.ServicePort).To(Equal(int32(8080)))
			})
		})

		Context("when config does not exist", func() {
			It("should return ErrConfigNotFound", func() {
				newPort := int32(9090)
				err := store.Update(ctx, testAppID, "non-existent-config", &polaris.ConfigUpdateData{
					ServicePort: &newPort,
				})
				Expect(err).To(MatchError(polaris.ErrConfigNotFound))
			})
		})
	})

	Describe("Delete", func() {
		var configName string

		BeforeEach(func() {
			config := &polaris.PolarisConfig{
				AppID: testAppID,
				Properties: polaris.Properties{
					InstanceKey:      "remove-test",
					PolarisName:      "to-remove",
					PolarisNamespace: "ns",
					PolarisToken:     "token",
					ServicePort:      8080,
				},
			}
			err := store.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			configName = config.Name
		})

		Context("when deleting an existing config", func() {
			It("should delete config successfully", func() {
				err := store.Delete(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.Get(ctx, testAppID, configName)
				Expect(err).To(MatchError(polaris.ErrConfigNotFound))
			})
		})

		Context("when config does not exist", func() {
			It("should return ErrConfigNotFound", func() {
				err := store.Delete(ctx, testAppID, "non-existent-config")
				Expect(err).To(MatchError(polaris.ErrConfigNotFound))
			})
		})

		Context("when deleting one of multiple configs", func() {
			It("should only delete the specified config", func() {
				// Add another config
				config2 := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "another-instance",
						PolarisName:      "another-polaris",
						PolarisNamespace: "ns",
						PolarisToken:     "token",
						ServicePort:      9090,
					},
				}
				err := store.Create(ctx, config2)
				Expect(err).NotTo(HaveOccurred())

				// Delete the first config
				err = store.Delete(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())

				// Verify only the second config remains
				configs, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(1))
				Expect(configs[0].Name).To(Equal(config2.Name))
			})
		})
	})

	Describe("ListByApp", func() {
		Context("when no configs exist", func() {
			It("should return empty list", func() {
				configs, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(BeEmpty())
			})
		})

		Context("when multiple configs exist", func() {
			It("should return all configs", func() {
				// Add multiple configs
				config1 := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "instance-1",
						PolarisName:      "polaris-1",
						PolarisNamespace: "ns",
						PolarisToken:     "token",
						ServicePort:      8081,
					},
				}
				err := store.Create(ctx, config1)
				Expect(err).NotTo(HaveOccurred())

				config2 := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "instance-2",
						PolarisName:      "polaris-2",
						PolarisNamespace: "ns",
						PolarisToken:     "token",
						ServicePort:      8082,
					},
				}
				err = store.Create(ctx, config2)
				Expect(err).NotTo(HaveOccurred())

				configs, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(2))
			})
		})
	})

	Describe("Get", func() {
		var configName string

		BeforeEach(func() {
			config := &polaris.PolarisConfig{
				AppID: testAppID,
				Properties: polaris.Properties{
					InstanceKey:      "get-test",
					PolarisName:      "get-polaris",
					PolarisNamespace: "get-ns",
					PolarisToken:     "get-token-123456789",
					ServicePort:      8080,
					Weight:           20,
				},
			}
			err := store.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			configName = config.Name
		})

		Context("when config exists", func() {
			It("should return the config", func() {
				config, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(config).NotTo(BeNil())
				Expect(config.Name).To(Equal(configName))
				Expect(config.InstanceKey).To(Equal("get-test"))
				Expect(config.PolarisName).To(Equal("get-polaris"))
				Expect(config.Weight).To(Equal(int32(20)))
			})
		})

		Context("when config does not exist", func() {
			It("should return ErrConfigNotFound", func() {
				_, err := store.Get(ctx, testAppID, "non-existent-config")
				Expect(err).To(MatchError(polaris.ErrConfigNotFound))
			})
		})
	})

	Describe("ListByEnv", func() {
		Context("when configs exist with environment scope", func() {
			BeforeEach(func() {
				// Create config available only in specific environments
				config := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						InstanceKey:      "env-instance",
						PolarisName:      "env-polaris",
						PolarisNamespace: "env-ns",
						PolarisToken:     "env-token",
						ServicePort:      8080,
					},
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{"dev", "staging"},
				}
				err := store.Create(ctx, config)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return config for matching environment", func() {
				configs, err := store.ListByEnv(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(1))
			})

			It("should not return config for non-matching environment", func() {
				configs, err := store.ListByEnv(ctx, testAppID, "production")
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(BeEmpty())
			})
		})
	})

	Describe("EnvState operations", func() {
		var configName string

		BeforeEach(func() {
			// 落库一条带生效范围的配置，供 envState 操作使用
			config := &polaris.PolarisConfig{
				AppID: testAppID,
				Properties: polaris.Properties{
					InstanceKey:      "k1",
					PolarisName:      "polaris-env-state",
					PolarisNamespace: "ns",
					PolarisToken:     "t1",
					ServicePort:      8080,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{"dev", "staging"},
			}
			Expect(store.Create(ctx, config)).NotTo(HaveOccurred())
			configName = config.Name
		})

		DescribeTable("should reject environment names that are unsafe as MongoDB field keys",
			func(envName string) {
				Expect(store.UpsertEnvState(
					ctx, testAppID, configName, envName, polaris.PolarisEnvStateUpdate{},
				)).To(MatchError(ContainSubstring("invalid env name")))
				Expect(store.RemoveEnvStates(
					ctx, testAppID, configName, []string{envName},
				)).To(MatchError(ContainSubstring("invalid env name")))
			},
			Entry("empty name", ""),
			Entry("name containing a dot", "dev.test"),
			Entry("name containing a dollar sign", "$dev"),
		)

		Describe("UpsertEnvState", func() {
			It("should add an environment idempotently", func() {
				Expect(store.UpsertEnvState(
					ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())
				Expect(store.UpsertEnvState(
					ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveLen(1))
				Expect(stored.GetEnvState("dev").AppliedFields).To(BeNil())
				Expect(stored.GetEnvState("dev").UpdatedAt).NotTo(BeZero())
			})
			It("should update only the provided fields", func() {
				fields := &polaris.RedeployRequiredFields{InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080}
				errMessage := "apply cr timed out"
				Expect(store.UpsertEnvState(ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: fields,
					LastError:     &errMessage,
				})).To(Succeed())

				updatedFields := &polaris.RedeployRequiredFields{
					InstanceKey: "k2", PolarisToken: "t2", ServicePort: 9090,
				}
				Expect(store.UpsertEnvState(ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: updatedFields,
				})).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				state := stored.GetEnvState("dev")
				Expect(state.AppliedFields).To(Equal(updatedFields))
				Expect(state.LastError).To(Equal(errMessage))
				Expect(state.UpdatedAt).NotTo(BeZero())
			})

			It("should clear an error without changing other fields", func() {
				fields := &polaris.RedeployRequiredFields{InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080}
				errMessage := "failed"
				Expect(store.UpsertEnvState(ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: fields, LastError: &errMessage,
				})).To(Succeed())

				errMessage = ""
				Expect(store.UpsertEnvState(ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					LastError: &errMessage,
				})).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.GetEnvState("dev").LastError).To(BeEmpty())
				Expect(stored.GetEnvState("dev").AppliedFields).To(Equal(fields))
			})

			It("should create a missing environment with the provided fields", func() {
				errMessage := "apply error"
				Expect(store.UpsertEnvState(ctx, testAppID, configName, "staging", polaris.PolarisEnvStateUpdate{
					LastError: &errMessage,
				})).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.GetEnvState("staging").LastError).To(Equal(errMessage))
				Expect(stored.GetEnvState("staging").UpdatedAt).NotTo(BeZero())
			})
		})

		Describe("RemoveEnvStates", func() {
			It("should remove multiple environments and preserve the others", func() {
				for _, envName := range []string{"dev", "staging", "prod"} {
					Expect(store.UpsertEnvState(
						ctx, testAppID, configName, envName, polaris.PolarisEnvStateUpdate{},
					)).To(Succeed())
				}

				Expect(store.RemoveEnvStates(
					ctx, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())
				Expect(store.RemoveEnvStates(
					ctx, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveLen(1))
				Expect(stored.EnvStates).To(HaveKey("prod"))
			})

			It("should do nothing for an empty environment list", func() {
				Expect(store.UpsertEnvState(
					ctx, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())

				Expect(store.RemoveEnvStates(ctx, testAppID, configName, nil)).To(Succeed())

				stored, err := store.Get(ctx, testAppID, configName)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveKey("dev"))
			})
		})
	})
})
