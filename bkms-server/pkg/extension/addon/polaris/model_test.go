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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
)

var _ = Describe("PolarisConfig", func() {
	Describe("IsAvailableInEnv", func() {
		It("should return true only for environments in ScopeEnvNames", func() {
			config := &polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev", "staging"},
			}
			Expect(config.IsAvailableInEnv("dev")).To(BeTrue())
			Expect(config.IsAvailableInEnv("staging")).To(BeTrue())
			Expect(config.IsAvailableInEnv("production")).To(BeFalse())
		})

		It("should return false when ScopeEnvNames is empty", func() {
			config := &polaris.PolarisConfig{}
			Expect(config.IsAvailableInEnv("any-env")).To(BeFalse())
		})
	})

	Describe("EnvNamesOutsideScope", func() {
		It("should return recorded environments that are no longer in scope", func() {
			config := &polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev"},
				EnvStates: map[string]polaris.PolarisEnvState{
					"dev":  {},
					"prod": {},
					"stag": {},
				},
			}
			Expect(config.EnvNamesOutsideScope()).To(Equal([]string{"prod", "stag"}))
		})

		It("should return empty when every recorded environment is still in scope", func() {
			config := &polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev"},
				EnvStates:     map[string]polaris.PolarisEnvState{"dev": {}},
			}
			Expect(config.EnvNamesOutsideScope()).To(BeEmpty())
		})
	})

	Describe("TrackedEnvNames", func() {
		It("should union scoped environments with recorded environments", func() {
			config := &polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev", "stag"},
				EnvStates: map[string]polaris.PolarisEnvState{
					"dev":  {},
					"prod": {},
				},
			}
			Expect(config.TrackedEnvNames()).To(Equal([]string{"dev", "prod", "stag"}))
		})
	})

	Describe("GetVars", func() {
		It("should return token and service port for on_deploy configs", func() {
			config := &polaris.PolarisConfig{
				Properties: polaris.Properties{
					InstanceKey:  "svc",
					PolarisToken: "token",
					ServicePort:  8080,
				},
			}
			Expect(config.GetVars()).To(Equal([]polaris.ConfigVar{
				{Key: "svc_polarisToken", Value: "token"},
				{Key: "svc_serviceport", Value: "8080"},
			}))
		})

		It("should return no vars for immediate-register configs", func() {
			config := &polaris.PolarisConfig{
				Properties: polaris.Properties{
					InstanceKey:  "svc",
					PolarisToken: "token",
					ServicePort:  8080,
					RegisterMode: polaris.RegisterModeImmediate,
				},
			}
			Expect(config.GetVars()).To(BeEmpty())
		})
	})

	Describe("GetEnvWeight", func() {
		It("should use the fixed default when the environment has no explicit value", func() {
			config := &polaris.PolarisConfig{}

			Expect(config.GetEnvWeight("dev")).To(Equal(polaris.DefaultEnvWeight))
		})

		It("should prefer an explicit environment weight including zero", func() {
			config := &polaris.PolarisConfig{
				EnvWeights: map[string]int32{"dev": 0},
			}

			Expect(config.GetEnvWeight("dev")).To(BeZero())
		})
	})
})
