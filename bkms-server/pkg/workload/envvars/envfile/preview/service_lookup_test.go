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

package preview

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Import preview existing lookup", func() {
	It("treats same key in different public scope as new", func() {
		preview, err := buildImportPreview(
			[]parserpkg.ParsedEnvVarRecord{{
				Key:                "SHARED_KEY",
				Value:              "override-dev-value",
				DeclaredScopeType:  lookupPtr(string(envvartypes.ScopeTypeEnvType)),
				DeclaredScopeValue: lookupPtr("development"),
			}},
			buildPublicExistingVarLookup([]envvars.ScopedEnvVar{
				{
					ScopeType:  envvartypes.ScopeTypeWorkspace,
					ScopeValue: "",
					Key:        "SHARED_KEY",
					Value:      "workspace-value",
				},
				{
					ScopeType:  envvartypes.ScopeTypeEnvType,
					ScopeValue: "production",
					Key:        "SHARED_KEY",
					Value:      "production-value",
				},
			}),
			ResolvePublicRecord,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))
		Expect(preview.Items[0].Action).To(Equal(ImportActionNew))
		Expect(preview.Items[0].OriginalValue).To(BeEmpty())
		Expect(preview.Summary.New).To(Equal(1))
		Expect(preview.Summary.Overwrite).To(Equal(0))
	})

	It("masks original value for sensitive scoped env var overwrite", func() {
		preview, err := buildImportPreview(
			[]parserpkg.ParsedEnvVarRecord{{
				Key:               "SECRET_KEY",
				Value:             "new-secret",
				DeclaredScopeType: lookupPtr(string(envvartypes.ScopeTypeWorkspace)),
			}},
			buildPublicExistingVarLookup([]envvars.ScopedEnvVar{{
				ScopeType:   envvartypes.ScopeTypeWorkspace,
				ScopeValue:  "",
				Key:         "SECRET_KEY",
				Value:       "old-secret",
				IsSensitive: true,
			}}),
			ResolvePublicRecord,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))
		Expect(preview.Items[0].Action).To(Equal(ImportActionOverwrite))
		Expect(preview.Items[0].OriginalValue).To(Equal(envvartypes.SensitiveValueMask))
	})

	It("masks original value for sensitive app env var overwrite", func() {
		preview, err := buildImportPreview(
			[]parserpkg.ParsedEnvVarRecord{{
				Key:   "APP_SECRET_KEY",
				Value: "new-secret",
			}},
			buildExistingVarLookupFromAppVars([]appmodel.Variable{{
				Key:         "APP_SECRET_KEY",
				Value:       "old-secret",
				IsSensitive: true,
			}}),
			ResolveAppRecord,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))
		Expect(preview.Items[0].Action).To(Equal(ImportActionOverwrite))
		Expect(preview.Items[0].OriginalValue).To(Equal(envvartypes.SensitiveValueMask))
	})
})

func lookupPtr(value string) *string {
	return &value
}
