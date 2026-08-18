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

package serializer_test

import (
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/instancestats"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/serializer"
)

var _ = Describe("PolarisConfigOutputObj", func() {
	Describe("FromModel", func() {
		It("should map core config fields to the output object", func() {
			now := time.Now()
			config := polaris.PolarisConfig{
				Name:  "cfg-1",
				AppID: "app-1",
				Properties: polaris.Properties{
					InstanceKey:       "k1",
					PolarisName:       "polaris-1",
					PolarisNamespace:  "ns-1",
					PolarisToken:      "token-1",
					ServicePort:       8080,
					Direct:            true,
					KeepNotReadyPod:   false,
					EnableHealthCheck: true,
					ServiceLabels:     map[string]string{"env": "test"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			}

			out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
			Expect(out.Name).To(Equal("cfg-1"))
			Expect(out.AppID).To(Equal("app-1"))
			Expect(out.InstanceKey).To(Equal("k1"))
			Expect(out.ServicePort).To(Equal(int32(8080)))
			Expect(out.PolarisToken).To(Equal("token-1"))
			Expect(out.ServiceLabels).To(Equal(map[string]string{"env": "test"}))
		})

		It("should report the immediate register mode", func() {
			out := new(serializer.PolarisConfigOutputObj).FromModel(polaris.PolarisConfig{
				Name:       "cfg-immediate",
				Properties: polaris.Properties{RegisterMode: polaris.RegisterModeImmediate},
			}, nil)
			Expect(out.RegisterMode).To(Equal(polaris.RegisterModeImmediate))
		})

		It("should fall back to on_deploy for configs stored before the mode was introduced", func() {
			out := new(serializer.PolarisConfigOutputObj).FromModel(polaris.PolarisConfig{Name: "cfg-legacy"}, nil)
			Expect(out.RegisterMode).To(Equal(polaris.RegisterModeOnDeploy))
		})

		It("should render an empty envStates object when no environments are relevant", func() {
			out := new(serializer.PolarisConfigOutputObj).FromModel(polaris.PolarisConfig{Name: "cfg-1"}, nil)
			Expect(out.EnvStates).ToNot(BeNil())
			Expect(out.EnvStates).To(BeEmpty())
			Expect(out.EnvWeights).ToNot(BeNil())
			Expect(out.EnvWeights).To(BeEmpty())
		})

		It("should synthesize pending-create states for scoped environments without snapshots", func() {
			config := polaris.PolarisConfig{
				Name: "cfg-1",
				Properties: polaris.Properties{
					InstanceKey: "k1", PolarisToken: "token", ServicePort: 8080,
				},
				ScopeEnvNames: []string{"dev"},
				EnvWeights:    map[string]int32{"dev": 100},
			}

			out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
			Expect(out.EnvStates).To(HaveLen(1))
			Expect(out.EnvStates["dev"].AppliedFields).To(BeNil())
			Expect(out.EnvStates["dev"].PolarisTokenChanged).To(BeFalse())
			Expect(out.EnvStates["dev"].LastError).To(BeEmpty())
			Expect(out.EnvStates["dev"].UpdatedAt).To(BeEmpty())
			Expect(out.EnvStates["dev"].Status).To(Equal(polaris.PolarisEnvStatusPendingCreate))
			Expect(out.EnvWeights["dev"]).To(Equal(int32(100)))
		})

		It("should expose per-environment weights as a top-level field", func() {
			config := polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev", "staging"},
				EnvWeights:    map[string]int32{"dev": 35, "staging": 100},
			}

			out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
			Expect(out.EnvStates["dev"].Status).To(Equal(polaris.PolarisEnvStatusPendingCreate))
			Expect(out.EnvWeights).To(Equal(map[string]int32{"dev": 35, "staging": 100}))
		})

		It("should preserve matching deployment snapshots and mark them as deployed", func() {
			updatedAt := time.Date(2026, time.July, 27, 8, 30, 0, 0, time.UTC)
			config := polaris.PolarisConfig{
				Properties: polaris.Properties{
					InstanceKey: "k1", PolarisToken: "token", ServicePort: 8080,
				},
				ScopeEnvNames: []string{"dev"},
				EnvWeights:    map[string]int32{"dev": 35},
				EnvStates: map[string]polaris.PolarisEnvState{
					"dev": {
						AppliedFields: &polaris.RedeployRequiredFields{
							InstanceKey: "k1", PolarisToken: "token", ServicePort: 8080,
						},
						LastError: "previous apply failed",
						UpdatedAt: updatedAt,
					},
				},
			}

			out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
			state := out.EnvStates["dev"]
			Expect(state.AppliedFields).To(Equal(&serializer.RedeployRequiredFieldsOutput{
				InstanceKey:  "k1",
				PolarisToken: "******",
				ServicePort:  8080,
			}))
			Expect(state.PolarisTokenChanged).To(BeFalse())
			Expect(state.LastError).To(Equal("previous apply failed"))
			Expect(state.UpdatedAt).To(Equal(updatedAt.Format(time.RFC3339)))
			Expect(state.Status).To(Equal(polaris.PolarisEnvStatusDeployed))
			Expect(out.EnvWeights["dev"]).To(Equal(int32(35)))
		})

		DescribeTable(
			"should mark changed deployment fields as pending modify",
			func(applied *polaris.RedeployRequiredFields, tokenChanged bool) {
				config := polaris.PolarisConfig{
					Properties: polaris.Properties{
						InstanceKey: "current-key", PolarisToken: "current-token", ServicePort: 9090,
					},
					ScopeEnvNames: []string{"dev"},
					EnvStates: map[string]polaris.PolarisEnvState{
						"dev": {AppliedFields: applied, LastError: "previous apply failed"},
					},
				}

				out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
				Expect(out.EnvStates["dev"].Status).To(Equal(polaris.PolarisEnvStatusPendingModify))
				Expect(out.EnvStates["dev"].PolarisTokenChanged).To(Equal(tokenChanged))
			},
			Entry(
				"when instanceKey changed",
				&polaris.RedeployRequiredFields{
					InstanceKey: "old-key", PolarisToken: "current-token", ServicePort: 9090,
				},
				false,
			),
			Entry(
				"when polarisToken changed",
				&polaris.RedeployRequiredFields{
					InstanceKey: "current-key", PolarisToken: "old-token", ServicePort: 9090,
				},
				true,
			),
			Entry(
				"when servicePort changed",
				&polaris.RedeployRequiredFields{
					InstanceKey: "current-key", PolarisToken: "current-token", ServicePort: 8080,
				},
				false,
			),
		)

		It("should mark deployed environments outside scope as pending delete", func() {
			applied := &polaris.RedeployRequiredFields{
				InstanceKey: "k1", PolarisToken: "token", ServicePort: 8080,
			}
			config := polaris.PolarisConfig{
				ScopeEnvNames: []string{"dev"},
				EnvStates: map[string]polaris.PolarisEnvState{
					"prod": {AppliedFields: applied},
				},
			}

			out := new(serializer.PolarisConfigOutputObj).FromModel(config, nil)
			Expect(out.EnvStates).To(HaveLen(2))
			Expect(out.EnvStates["dev"].Status).To(Equal(polaris.PolarisEnvStatusPendingCreate))
			Expect(out.EnvStates["prod"].Status).To(Equal(polaris.PolarisEnvStatusPendingDelete))
		})

		It("should convert a zero DepSvcInstID to an empty string", func() {
			out := new(serializer.PolarisConfigOutputObj).FromModel(polaris.PolarisConfig{Name: "cfg-1"}, nil)
			Expect(out.DepSvcInstID).To(BeEmpty())
		})

		It("should convert a non-zero DepSvcInstID to its hex string", func() {
			id := bson.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
			out := new(serializer.PolarisConfigOutputObj).FromModel(
				polaris.PolarisConfig{Name: "cfg-1", DepSvcInstID: id}, nil,
			)
			Expect(out.DepSvcInstID).To(Equal("0102030405060708090a0b0c"))
		})

		It("should carry warnings through to the output", func() {
			out := new(serializer.PolarisConfigOutputObj).FromModel(
				polaris.PolarisConfig{Name: "cfg-1"}, []string{"name mismatch"},
			)
			Expect(out.Warnings).To(Equal([]string{"name mismatch"}))
		})
	})
})

var _ = Describe("PatchAppPolarisConfigOutput", func() {
	Describe("FromModel", func() {
		It("should map the updated config and derive all relevant environment states", func() {
			applied := &polaris.RedeployRequiredFields{InstanceKey: "k1", PolarisToken: "old", ServicePort: 8080}
			config := &polaris.PolarisConfig{
				Name: "cfg-1",
				Properties: polaris.Properties{
					InstanceKey: "k2", PolarisToken: "new", ServicePort: 9090,
				},
				ScopeEnvNames: []string{"dev", "staging"},
				EnvStates: map[string]polaris.PolarisEnvState{
					"dev":     {UpdatedAt: time.Now()},
					"staging": {AppliedFields: applied, UpdatedAt: time.Now()},
					"prod":    {AppliedFields: applied, UpdatedAt: time.Now()},
				},
			}

			out := new(serializer.PatchAppPolarisConfigOutput).FromModel(config)
			Expect(out.Data.Name).To(Equal("cfg-1"))
			Expect(out.Data.InstanceKey).To(Equal("k2"))
			Expect(out.Data.EnvStates).To(HaveLen(3))
			Expect(out.Data.EnvStates["dev"].Status).To(Equal(polaris.PolarisEnvStatusPendingCreate))
			Expect(out.Data.EnvStates["staging"].Status).To(Equal(polaris.PolarisEnvStatusPendingModify))
			Expect(out.Data.EnvStates["prod"].Status).To(Equal(polaris.PolarisEnvStatusPendingDelete))
		})
	})
})

var _ = Describe("PutEnvWeightInput", func() {
	var validate *validator.Validate

	BeforeEach(func() {
		validate = validator.New()
		validate.SetTagName("binding")
	})

	It("should accept an explicit zero weight", func() {
		weight := int32(0)
		input := serializer.PutEnvWeightInput{Weight: &weight}

		Expect(validate.Struct(input)).To(Succeed())
	})

	It("should reject an omitted weight", func() {
		Expect(validate.Struct(serializer.PutEnvWeightInput{})).NotTo(Succeed())
	})

	It("should reject a weight above the upper bound", func() {
		weight := int32(10001)
		input := serializer.PutEnvWeightInput{Weight: &weight}

		Expect(validate.Struct(input)).NotTo(Succeed())
	})
})

var _ = Describe("CreateAppPolarisConfigInput", func() {
	inputWithMode := func(mode *string) serializer.CreateAppPolarisConfigInput {
		return serializer.CreateAppPolarisConfigInput{
			InstanceKey:      "k1",
			PolarisName:      "polaris-1",
			PolarisNamespace: "Test",
			ServicePort:      8080,
			RegisterMode:     mode,
		}
	}

	DescribeTable(
		"registerMode validation",
		func(mode *string, wantErr bool) {
			err := binding.Validator.ValidateStruct(inputWithMode(mode))
			if wantErr {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("accepts an omitted mode", nil, false),
		Entry("accepts immediate", lo.ToPtr(polaris.RegisterModeImmediate), false),
		Entry("accepts on_deploy", lo.ToPtr(polaris.RegisterModeOnDeploy), false),
		Entry("rejects an unknown mode", lo.ToPtr("whenever"), true),
		Entry("rejects an explicit empty mode", lo.ToPtr(""), true),
	)
})

var _ = Describe("AppConfigEnvNameURIInput", func() {
	DescribeTable(
		"URI slug validation",
		func(envName string, wantErr bool) {
			input := serializer.AppConfigEnvNameURIInput{
				AppID:      "demo-app",
				ConfigName: "demo-config",
				EnvName:    envName,
			}

			err := binding.Validator.ValidateStruct(input)
			if wantErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed on the 'uri_slug' tag"))
				return
			}
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("accepts a URI slug environment name", "env_123-Test", false),
		Entry("rejects an environment name containing a dot", "env.test", true),
		Entry("rejects an environment name containing a dollar sign", "env$test", true),
	)
})

var _ = Describe("GetEnvInstanceStatsOutput", func() {
	It("maps per-environment instance counts and healthy totals", func() {
		output := new(serializer.GetEnvInstanceStatsOutput).FromModel(&instancestats.Result{
			EnvStats: map[string]instancestats.Stats{
				"stable": {
					HealthyInstanceCount:          3,
					HealthyInstanceWeight:         300,
					IsolatedInstanceCount:         1,
					TotalInstanceCount:            4,
					WeightOverriddenInstanceCount: 2,
				},
			},
			TotalHealthyInstanceCount:  7,
			TotalHealthyInstanceWeight: 520,
		})

		Expect(output.Data.EnvStats["stable"]).To(Equal(serializer.EnvInstanceStatsOutput{
			HealthyInstanceCount:          3,
			HealthyInstanceWeight:         300,
			IsolatedInstanceCount:         1,
			TotalInstanceCount:            4,
			WeightOverriddenInstanceCount: 2,
		}))
		Expect(output.Data.TotalHealthyInstanceCount).To(Equal(int32(7)))
		Expect(output.Data.TotalHealthyInstanceWeight).To(Equal(int32(520)))
	})
})
