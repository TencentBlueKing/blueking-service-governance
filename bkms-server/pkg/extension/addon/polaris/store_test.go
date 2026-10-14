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
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("PolarisConfigStore", func() {
	var (
		configStore *polaris.ConfigStore
		db          *mongo.Database
		ctx         context.Context
		testAppID   string
	)

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		configStore = store.Of(db, polaris.ConfigMeta)
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		// 清理测试数据
		_, _ = configStore.DeleteMany(ctx, polaris.ByApp(testAppID))
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
						ServiceLabels: map[string]string{
							"env": "test",
							"app": "demo",
						},
					},
				}

				err := configStore.Create(ctx, config)
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Name).NotTo(BeEmpty())

				// Verify stored config
				storedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, config.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(storedConfig.KeepNotReadyPod).To(BeFalse())
				Expect(storedConfig.EnableHealthCheck).To(BeTrue())
				Expect(storedConfig.ServiceLabels).To(Equal(map[string]string{
					"env": "test",
					"app": "demo",
				}))
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
				err := configStore.Create(ctx, config1)
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
				err = configStore.Create(ctx, config2)
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
			err := configStore.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			configName = config.Name
		})

		Context("when updating an existing config", func() {
			It("should update config successfully", func() {
				newPort := int32(9090)
				err := polaris.UpdateConfig(ctx, db, testAppID, configName, &polaris.ConfigUpdateData{
					ServicePort: &newPort,
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.ServicePort).To(Equal(newPort))
			})
		})

		Context("when upserting an environment weight", func() {
			It("should set the weight of a single environment without touching others", func() {
				Expect(polaris.UpsertEnvWeight(ctx, db, testAppID, configName, "dev", 0, nil)).To(Succeed())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.EnvWeights).To(HaveKeyWithValue("dev", BeZero()))
				Expect(updatedConfig.EnvWeights).NotTo(HaveKey("staging"))
			})

			It("should write the weight and the dynamic weight switch in one update", func() {
				Expect(polaris.UpsertEnvWeight(
					ctx, db, testAppID, configName, "dev", 80, lo.ToPtr(true),
				)).To(Succeed())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.EnvWeights["dev"]).To(Equal(int32(80)))
				Expect(updatedConfig.EnvDynamicWeights["dev"]).To(BeTrue())
			})

			It("should keep the dynamic weight switch untouched when it is not given", func() {
				Expect(polaris.UpsertEnvWeight(
					ctx, db, testAppID, configName, "dev", 80, lo.ToPtr(true),
				)).To(Succeed())
				Expect(polaris.UpsertEnvWeight(ctx, db, testAppID, configName, "dev", 20, nil)).To(Succeed())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.EnvWeights["dev"]).To(Equal(int32(20)))
				Expect(updatedConfig.EnvDynamicWeights["dev"]).To(BeTrue())
			})

			It("should return not found when the config does not exist", func() {
				err := polaris.UpsertEnvWeight(ctx, db, testAppID, "missing-config", "dev", 10, nil)
				Expect(errors.Is(err, polaris.ErrConfigNotFound)).To(BeTrue())
			})
		})

		Context("when removing environment settings", func() {
			BeforeEach(func() {
				Expect(polaris.UpsertEnvWeight(
					ctx, db, testAppID, configName, "dev", 10, lo.ToPtr(true),
				)).To(Succeed())
				Expect(polaris.UpsertEnvWeight(ctx, db, testAppID, configName, "staging", 20, nil)).To(Succeed())
				Expect(polaris.UpsertEnvWeight(
					ctx, db, testAppID, configName, "prod", 30, lo.ToPtr(true),
				)).To(Succeed())
			})

			It("should remove every env setting of the selected environments in one go", func() {
				Expect(polaris.RemoveEnvSettings(
					ctx, db, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())
				Expect(polaris.RemoveEnvSettings(
					ctx, db, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.EnvWeights).To(Equal(map[string]int32{"prod": 30}))
				Expect(updatedConfig.EnvDynamicWeights).To(Equal(map[string]bool{"prod": true}))
			})

			It("should do nothing for an empty environment list", func() {
				Expect(polaris.RemoveEnvSettings(ctx, db, testAppID, configName, nil)).To(Succeed())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.EnvWeights["dev"]).To(Equal(int32(10)))
				Expect(updatedConfig.EnvDynamicWeights["dev"]).To(BeTrue())
			})

			It("should reject unsafe environment names", func() {
				Expect(polaris.RemoveEnvSettings(
					ctx, db, testAppID, configName, []string{"dev.test"},
				)).To(MatchError(ContainSubstring("invalid env name")))
				Expect(polaris.RemoveEnvSettings(
					ctx, db, testAppID, configName, []string{"dev$test"},
				)).To(MatchError(ContainSubstring("invalid env name")))
			})
		})

		Context("when updating polarisToken", func() {
			It("should update polarisToken successfully", func() {
				newToken := "new-updated-token"
				err := polaris.UpdateConfig(ctx, db, testAppID, configName, &polaris.ConfigUpdateData{
					PolarisToken: &newToken,
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.PolarisToken).To(Equal(newToken))
				// Verify other fields unchanged
				Expect(updatedConfig.PolarisName).To(Equal("original-name"))
				Expect(updatedConfig.ServicePort).To(Equal(int32(8080)))
			})
		})

		Context("when updating operator", func() {
			It("should update operator successfully", func() {
				operator := "lisi,wangwu"
				err := polaris.UpdateConfig(ctx, db, testAppID, configName, &polaris.ConfigUpdateData{
					Operator: &operator,
				})
				Expect(err).NotTo(HaveOccurred())

				updatedConfig, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedConfig.Operator).To(Equal(operator))
				Expect(updatedConfig.PolarisName).To(Equal("original-name"))
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
			err := configStore.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			configName = config.Name
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
				err := configStore.Create(ctx, config2)
				Expect(err).NotTo(HaveOccurred())

				// Delete the first config
				err = configStore.Delete(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())

				// Verify only the second config remains
				configs, err := configStore.List(ctx, polaris.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(1))
				Expect(configs[0].Name).To(Equal(config2.Name))
			})
		})
	})

	Describe("ListByApp", func() {
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
				err := configStore.Create(ctx, config1)
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
				err = configStore.Create(ctx, config2)
				Expect(err).NotTo(HaveOccurred())

				configs, err := configStore.List(ctx, polaris.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(2))
			})
		})
	})

	Describe("Get", func() {
		Context("when config does not exist", func() {
			It("should return ErrConfigNotFound", func() {
				_, err := configStore.Get(ctx, polaris.ByName(testAppID, "non-existent-config"))
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
					ScopeEnvNames: []string{"dev", "staging"},
				}
				err := configStore.Create(ctx, config)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return config for matching environment", func() {
				configs, err := polaris.ListByEnv(ctx, db, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(1))
			})

			It("should not return config for non-matching environment", func() {
				configs, err := polaris.ListByEnv(ctx, db, testAppID, "production")
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
				ScopeEnvNames: []string{"dev", "staging"},
			}
			Expect(configStore.Create(ctx, config)).NotTo(HaveOccurred())
			configName = config.Name
		})

		DescribeTable("should reject environment names that are unsafe as MongoDB field keys",
			func(envName string) {
				Expect(polaris.UpsertEnvState(
					ctx, db, testAppID, configName, envName, polaris.PolarisEnvStateUpdate{},
				)).To(MatchError(ContainSubstring("invalid env name")))
				Expect(polaris.RemoveEnvStates(
					ctx, db, testAppID, configName, []string{envName},
				)).To(MatchError(ContainSubstring("invalid env name")))
			},
			Entry("empty name", ""),
			Entry("name containing a dot", "dev.test"),
			Entry("name containing a dollar sign", "$dev"),
		)

		Describe("UpsertEnvState", func() {
			It("should add an environment idempotently", func() {
				Expect(polaris.UpsertEnvState(
					ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())
				Expect(polaris.UpsertEnvState(
					ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveLen(1))
				Expect(stored.GetEnvState("dev").AppliedFields).To(BeNil())
				Expect(stored.GetEnvState("dev").UpdatedAt).NotTo(BeZero())
			})
			It("should update only the provided fields", func() {
				fields := &polaris.RedeployRequiredFields{InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080}
				errMessage := "apply cr timed out"
				Expect(polaris.UpsertEnvState(ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: fields,
					LastError:     &errMessage,
				})).To(Succeed())

				updatedFields := &polaris.RedeployRequiredFields{
					InstanceKey: "k2", PolarisToken: "t2", ServicePort: 9090,
				}
				Expect(polaris.UpsertEnvState(ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: updatedFields,
				})).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				state := stored.GetEnvState("dev")
				Expect(state.AppliedFields).To(Equal(updatedFields))
				Expect(state.LastError).To(Equal(errMessage))
				Expect(state.UpdatedAt).NotTo(BeZero())
			})

			It("should clear an error without changing other fields", func() {
				fields := &polaris.RedeployRequiredFields{InstanceKey: "k1", PolarisToken: "t1", ServicePort: 8080}
				errMessage := "failed"
				Expect(polaris.UpsertEnvState(ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					AppliedFields: fields, LastError: &errMessage,
				})).To(Succeed())

				errMessage = ""
				Expect(polaris.UpsertEnvState(ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{
					LastError: &errMessage,
				})).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.GetEnvState("dev").LastError).To(BeEmpty())
				Expect(stored.GetEnvState("dev").AppliedFields).To(Equal(fields))
			})

			It("should create a missing environment with the provided fields", func() {
				errMessage := "apply error"
				Expect(polaris.UpsertEnvState(ctx, db, testAppID, configName, "staging", polaris.PolarisEnvStateUpdate{
					LastError: &errMessage,
				})).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.GetEnvState("staging").LastError).To(Equal(errMessage))
				Expect(stored.GetEnvState("staging").UpdatedAt).NotTo(BeZero())
			})
		})

		Describe("RemoveEnvStates", func() {
			It("should remove multiple environments and preserve the others", func() {
				for _, envName := range []string{"dev", "staging", "prod"} {
					Expect(polaris.UpsertEnvState(
						ctx, db, testAppID, configName, envName, polaris.PolarisEnvStateUpdate{},
					)).To(Succeed())
				}

				Expect(polaris.RemoveEnvStates(
					ctx, db, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())
				Expect(polaris.RemoveEnvStates(
					ctx, db, testAppID, configName, []string{"dev", "staging"},
				)).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveLen(1))
				Expect(stored.EnvStates).To(HaveKey("prod"))
			})

			It("should do nothing for an empty environment list", func() {
				Expect(polaris.UpsertEnvState(
					ctx, db, testAppID, configName, "dev", polaris.PolarisEnvStateUpdate{},
				)).To(Succeed())

				Expect(polaris.RemoveEnvStates(ctx, db, testAppID, configName, nil)).To(Succeed())

				stored, err := configStore.Get(ctx, polaris.ByName(testAppID, configName))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvStates).To(HaveKey("dev"))
			})
		})
	})
})
