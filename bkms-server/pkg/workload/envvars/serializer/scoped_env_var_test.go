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
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	previewpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/serializer"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Env var output serializers", func() {
	It("masks sensitive available env var values", func() {
		output := new(serializer.EnvVarOutputObj).FromModel(envvartypes.EnvVariableObj{
			Key:         "SECRET_KEY",
			Value:       "real-secret",
			Description: "secret env var",
			IsSensitive: true,
		})

		Expect(output.Value).To(Equal(envvartypes.SensitiveValueMask))
		Expect(output.IsSensitive).To(BeTrue())
	})

	It("omits empty conflict info from detailed app env var JSON", func() {
		createdAt := time.Date(2026, 7, 20, 10, 30, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 7, 20, 11, 30, 0, 0, time.UTC)
		output := new(serializer.AppEnvVarDetailedOutputObj).FromModel(appmodel.Variable{
			Key:         "NORMAL_KEY",
			Value:       "normal-value",
			Description: "normal app env var",
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}, envvartypes.EnvVarConflictedInfo{})

		payload, err := json.Marshal(output)
		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"appEnvVar": {
				"key": "NORMAL_KEY",
				"value": "normal-value",
				"description": "normal app env var",
				"isSensitive": false,
				"createdAt": "2026-07-20T10:30:00Z",
				"updatedAt": "2026-07-20T11:30:00Z"
			}
		}`))
	})

	It("serializes preview scope info as nested objects without defaulted summary", func() {
		output := new(serializer.EnvVarImportPreviewOutputObj).FromModel(&previewpkg.ImportPreview{
			Items: []previewpkg.ImportPreviewItem{{
				Key:           "SHARED_KEY",
				Value:         "override-dev-value",
				OriginalValue: "development-value",
				Description:   "demo",
				DeclaredScope: &previewpkg.ImportPreviewScope{
					Type:  "envType",
					Value: "development",
				},
				EffectiveScope: &previewpkg.ImportPreviewScope{
					Type:  "envType",
					Value: "development",
				},
				Action:      previewpkg.ImportActionOverwrite,
				EffectScope: previewpkg.ImportEffectScopeApplied,
			}},
			Summary: previewpkg.ImportPreviewSummary{
				Total:     1,
				Overwrite: 1,
			},
		})

		payload, err := json.Marshal(output)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(payload)).NotTo(ContainSubstring("defaulted"))
		Expect(payload).To(MatchJSON(`{
			"items": [{
				"key": "SHARED_KEY",
				"value": "override-dev-value",
				"originalValue": "development-value",
				"description": "demo",
				"declaredScope": {
					"type": "envType",
					"value": "development"
				},
				"effectiveScope": {
					"type": "envType",
					"value": "development"
				},
				"action": "overwrite",
				"effectScope": "applied"
			}],
			"summary": {
				"total": 1,
				"new": 0,
				"overwrite": 1
			}
		}`))
	})

	It("returns timestamps for app-defined env vars", func() {
		createdAt := time.Date(2026, 7, 20, 10, 30, 0, 0, time.UTC)
		updatedAt := time.Date(2026, 7, 20, 11, 30, 0, 0, time.UTC)
		output := new(serializer.AppDefinedEnvVarOutputObj).FromModel(appmodel.Variable{
			Key:       "NORMAL_KEY",
			Value:     "normal-value",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})

		payload, err := json.Marshal(output)
		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"key": "NORMAL_KEY",
			"value": "normal-value",
			"description": "",
			"isSensitive": false,
			"createdAt": "2026-07-20T10:30:00Z",
			"updatedAt": "2026-07-20T11:30:00Z"
		}`))
	})
})

var _ = Describe("Scoped env var serializers", func() {
	DescribeTable(
		"CreateScopedEnvVarInput validation",
		func(input serializer.CreateScopedEnvVarInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid workspace scoped input", serializer.CreateScopedEnvVarInput{
			ScopeType: "workspace",
			Key:       "VALID_KEY",
		}, nil),
		Entry("workspace scope rejects explicit scopeValue", serializer.CreateScopedEnvVarInput{
			ScopeType:  "workspace",
			ScopeValue: "all",
			Key:        "VALID_KEY",
		}, []string{
			"CreateScopedEnvVarInput.ScopeValue",
			"failed on the 'scope_value_forbidden' tag",
		}),
		Entry("envType scope requires scopeValue", serializer.CreateScopedEnvVarInput{
			ScopeType: "envType",
			Key:       "VALID_KEY",
		}, []string{
			"CreateScopedEnvVarInput.ScopeValue",
			"failed on the 'scope_value_required' tag",
		}),
		Entry("missing scope type", serializer.CreateScopedEnvVarInput{
			Key: "VALID_KEY",
		}, []string{
			"CreateScopedEnvVarInput.ScopeType",
			"failed on the 'required' tag",
		}),
		Entry("invalid scope type", serializer.CreateScopedEnvVarInput{
			ScopeType: "invalid",
			Key:       "VALID_KEY",
		}, []string{
			"CreateScopedEnvVarInput.ScopeType",
			"failed on the 'oneof' tag",
		}),
		Entry("missing key", serializer.CreateScopedEnvVarInput{
			ScopeType: "workspace",
		}, []string{
			"CreateScopedEnvVarInput.Key",
			"failed on the 'required' tag",
		}),
		Entry("invalid env var key", serializer.CreateScopedEnvVarInput{
			ScopeType: "workspace",
			Key:       "1_INVALID",
		}, []string{
			"CreateScopedEnvVarInput.Key",
			"failed on the 'envvar_key' tag",
		}),
		Entry("invalid env var value length", serializer.CreateScopedEnvVarInput{
			ScopeType: "workspace",
			Key:       "VALID_KEY",
			Value:     strings.Repeat("v", 8193),
		}, []string{
			"CreateScopedEnvVarInput.Value",
			"failed on the 'envvar_value' tag",
		}),
	)

	It("distinguishes an omitted update value from an explicit empty value", func() {
		var omittedValue serializer.UpdateScopedEnvVarInput
		Expect(json.Unmarshal([]byte(`{"key":"VALID_KEY"}`), &omittedValue)).To(Succeed())
		Expect(omittedValue.Value).To(BeNil())

		var emptyValue serializer.UpdateScopedEnvVarInput
		Expect(json.Unmarshal([]byte(`{"key":"VALID_KEY","value":""}`), &emptyValue)).To(Succeed())
		Expect(emptyValue.Value).NotTo(BeNil())
		Expect(*emptyValue.Value).To(BeEmpty())
	})
})

var _ = Describe("App-defined env var serializers", func() {
	DescribeTable(
		"CreateAppDefinedEnvVarInput validation",
		func(input serializer.CreateAppDefinedEnvVarInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid create input", serializer.CreateAppDefinedEnvVarInput{
			Key:   "VALID_KEY",
			Value: "value",
		}, nil),
		Entry("missing key", serializer.CreateAppDefinedEnvVarInput{
			Value: "value",
		}, []string{
			"CreateAppDefinedEnvVarInput.Key",
			"failed on the 'required' tag",
		}),
		Entry("invalid key", serializer.CreateAppDefinedEnvVarInput{
			Key: "1INVALID",
		}, []string{
			"CreateAppDefinedEnvVarInput.Key",
			"failed on the 'envvar_key' tag",
		}),
		Entry("invalid value length", serializer.CreateAppDefinedEnvVarInput{
			Key:   "VALID_KEY",
			Value: strings.Repeat("v", 8193),
		}, []string{
			"CreateAppDefinedEnvVarInput.Value",
			"failed on the 'envvar_value' tag",
		}),
	)

	DescribeTable(
		"UpdateAppDefinedEnvVarInput validation",
		func(input serializer.UpdateAppDefinedEnvVarInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid update input", serializer.UpdateAppDefinedEnvVarInput{
			UpdatedKey: "VALID_KEY",
			Value:      lo.ToPtr("value"),
		}, nil),
		Entry("missing updated key", serializer.UpdateAppDefinedEnvVarInput{
			Value: lo.ToPtr("value"),
		}, []string{
			"UpdateAppDefinedEnvVarInput.UpdatedKey",
			"failed on the 'required' tag",
		}),
		Entry("invalid updated key", serializer.UpdateAppDefinedEnvVarInput{
			UpdatedKey: "1INVALID",
		}, []string{
			"UpdateAppDefinedEnvVarInput.UpdatedKey",
			"failed on the 'envvar_key' tag",
		}),
	)

	It("distinguishes an omitted app update value from an explicit empty value", func() {
		var omittedValue serializer.UpdateAppDefinedEnvVarInput
		Expect(json.Unmarshal([]byte(`{"updatedKey":"VALID_KEY"}`), &omittedValue)).To(Succeed())
		Expect(omittedValue.Value).To(BeNil())

		var emptyValue serializer.UpdateAppDefinedEnvVarInput
		Expect(json.Unmarshal([]byte(`{"updatedKey":"VALID_KEY","value":""}`), &emptyValue)).To(Succeed())
		Expect(emptyValue.Value).NotTo(BeNil())
		Expect(*emptyValue.Value).To(BeEmpty())
	})

	It("distinguishes an omitted app sensitivity from explicit false", func() {
		var omittedValue serializer.UpdateAppDefinedEnvVarInput
		Expect(json.Unmarshal([]byte(`{"updatedKey":"VALID_KEY"}`), &omittedValue)).To(Succeed())
		Expect(omittedValue.IsSensitive).To(BeNil())

		var falseValue serializer.UpdateAppDefinedEnvVarInput
		Expect(json.Unmarshal(
			[]byte(`{"updatedKey":"VALID_KEY","isSensitive":false}`),
			&falseValue,
		)).To(Succeed())
		Expect(falseValue.IsSensitive).NotTo(BeNil())
		Expect(*falseValue.IsSensitive).To(BeFalse())
	})
})
