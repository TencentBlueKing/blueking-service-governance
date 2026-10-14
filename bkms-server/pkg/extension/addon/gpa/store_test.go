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

package gpa_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
)

// newValidConfig 构造一个合法的 GPA 配置（min=2/max=10/CPU 60%），默认环境 dev
func newValidConfig(appID string) *gpa.GPAConfig {
	return &gpa.GPAConfig{
		AppID:       appID,
		EnvName:     "dev",
		MinReplicas: 2,
		MaxReplicas: 10,
		Metrics: []gpa.GPAMetric{
			{Resource: gpa.ResourceCPU, AverageUtilization: 60},
		},
	}
}

var _ = Describe("GPAConfig store", func() {
	var configs *gpa.ConfigStore
	var ctx context.Context
	var testAppID string

	BeforeEach(func() {
		configs = testutil.NewStore(GinkgoT(), gpa.ConfigMeta)
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_, err := configs.DeleteMany(ctx, gpa.ByApp(testAppID))
		Expect(err).NotTo(HaveOccurred())
	})

	get := func(envName string) (*gpa.GPAConfig, error) {
		return configs.Get(ctx, gpa.ByAppAndEnv(testAppID, envName))
	}

	Describe("CreateConfig", func() {
		Context("when creating an invalid gpa config", func() {
			It("should reject with validation error", func() {
				config := newValidConfig(testAppID)
				config.MaxReplicas = 1
				Expect(gpa.CreateConfig(ctx, configs, config)).To(MatchError(ContainSubstring("maxReplicas")))
			})
		})

		Context("when creating a valid gpa config", func() {
			It("should create successfully and be retrievable by (appID, envName)", func() {
				config := newValidConfig(testAppID)

				err := gpa.CreateConfig(ctx, configs, config)
				Expect(err).NotTo(HaveOccurred())
				// Name 由 GenerateName 固定为 gpa-{appID}
				Expect(config.Name).To(Equal("gpa-" + testAppID))

				// 记录成功写入且可通过 (appID, envName) 查询到
				stored, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.MinReplicas).To(Equal(int32(2)))
				Expect(stored.MaxReplicas).To(Equal(int32(10)))
				Expect(stored.Metrics).To(HaveLen(1))
				Expect(stored.Metrics[0].Resource).To(Equal(gpa.ResourceCPU))
				Expect(stored.Metrics[0].AverageUtilization).To(Equal(int32(60)))
				// 新建即生效：Enabled 默认置为 true
				Expect(stored.Enabled).To(BeTrue())
				// 默认以 requests 为基准，ComputeByLimits 为 false
				Expect(stored.ComputeByLimits).To(BeFalse())
			})
		})

		Context("when creating a config with ComputeByLimits enabled", func() {
			It("should persist ComputeByLimits=true and read it back", func() {
				config := newValidConfig(testAppID)
				config.ComputeByLimits = true

				Expect(gpa.CreateConfig(ctx, configs, config)).To(Succeed())

				stored, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.ComputeByLimits).To(BeTrue())
			})
		})

		Context("when creating a second config for the same app and env", func() {
			It("should reject with ErrConfigEnvExists", func() {
				Expect(gpa.CreateConfig(ctx, configs, newValidConfig(testAppID))).To(Succeed())
				err := gpa.CreateConfig(ctx, configs, newValidConfig(testAppID))
				Expect(err).To(MatchError(gpa.ErrConfigEnvExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when config does not exist", func() {
			It("should return ErrConfigNotFound", func() {
				_, err := get("non-existent")
				Expect(err).To(MatchError(gpa.ErrConfigNotFound))
			})
		})
	})

	Describe("UpdateConfig", func() {
		BeforeEach(func() {
			Expect(gpa.CreateConfig(ctx, configs, newValidConfig(testAppID))).To(Succeed())
		})

		Context("when updating maxReplicas", func() {
			It("should update successfully", func() {
				newMax := int32(20)
				err := gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					MaxReplicas: &newMax,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.MaxReplicas).To(Equal(int32(20)))
				Expect(updated.MinReplicas).To(Equal(int32(2)))
			})
		})

		Context("when updating metrics", func() {
			It("should replace the metrics entirely", func() {
				err := gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					Metrics: []gpa.GPAMetric{
						{Resource: gpa.ResourceCPU, AverageUtilization: 50},
						{Resource: gpa.ResourceMemory, AverageUtilization: 80},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Metrics).To(HaveLen(2))
			})
		})

		Context("when toggling enabled", func() {
			It("should update enabled to false and back to true", func() {
				disable := false
				err := gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					Enabled: &disable,
				})
				Expect(err).NotTo(HaveOccurred())
				disabled, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(disabled.Enabled).To(BeFalse())

				enable := true
				err = gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					Enabled: &enable,
				})
				Expect(err).NotTo(HaveOccurred())
				enabled, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(enabled.Enabled).To(BeTrue())
			})
		})

		Context("when toggling computeByLimits", func() {
			It("should update computeByLimits to true and back to false", func() {
				enable := true
				err := gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					ComputeByLimits: &enable,
				})
				Expect(err).NotTo(HaveOccurred())
				byLimits, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(byLimits.ComputeByLimits).To(BeTrue())

				disable := false
				err = gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					ComputeByLimits: &disable,
				})
				Expect(err).NotTo(HaveOccurred())
				byRequests, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(byRequests.ComputeByLimits).To(BeFalse())
			})
		})

		Context("when the update would make maxReplicas less than minReplicas", func() {
			It("should reject with validation error and keep the original value", func() {
				newMax := int32(1)
				err := gpa.UpdateConfig(ctx, configs, testAppID, "dev", &gpa.ConfigUpdateData{
					MaxReplicas: &newMax,
				})
				Expect(err).To(MatchError(ContainSubstring("maxReplicas")))

				unchanged, err := get("dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(unchanged.MaxReplicas).To(Equal(int32(10)))
			})
		})
	})

	Describe("ByApp", func() {
		Context("when configs exist for multiple envs", func() {
			It("should return all configs", func() {
				c1 := newValidConfig(testAppID)
				c1.EnvName = "dev"
				Expect(gpa.CreateConfig(ctx, configs, c1)).To(Succeed())
				c2 := newValidConfig(testAppID)
				c2.EnvName = "prod"
				Expect(gpa.CreateConfig(ctx, configs, c2)).To(Succeed())

				configs, err := configs.List(ctx, gpa.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(HaveLen(2))
			})
		})
	})
})

var _ = Describe("GPAConfig", func() {
	Describe("Validate", func() {
		It("should accept a config with two distinct metrics", func() {
			config := &gpa.GPAConfig{
				Name:        "c",
				AppID:       "app",
				EnvName:     "dev",
				MinReplicas: 1,
				MaxReplicas: 5,
				Metrics: []gpa.GPAMetric{
					{Resource: gpa.ResourceCPU, AverageUtilization: 60},
					{Resource: gpa.ResourceMemory, AverageUtilization: 70},
				},
			}
			Expect(config.Validate()).To(Succeed())
		})

		It("should reject an invalid resource name", func() {
			config := newValidConfig("app")
			config.Name = "c"
			config.Metrics = []gpa.GPAMetric{
				{Resource: gpa.ResourceName("disk"), AverageUtilization: 60},
			}
			Expect(config.Validate()).To(MatchError(ContainSubstring("metric resource must be cpu or memory")))
		})
	})
})
