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

package export

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvarsstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ExportUnitHelpers", func() {
	It("filters out sensitive scoped vars", func() {
		vars := []envvarsstore.ScopedEnvVar{
			{Key: "PUBLIC", Value: "visible", IsSensitive: false},
			{Key: "SECRET", Value: "hidden", IsSensitive: true},
		}

		filtered := filterOutSensitiveScopedVars(vars)
		Expect(filtered).To(HaveLen(1))
		Expect(filtered[0].Key).To(Equal("PUBLIC"))
	})

	It("filters out sensitive app vars", func() {
		vars := []appmodel.Variable{
			{Key: "APP_MODE", Value: "prod", IsSensitive: false},
			{Key: "APP_SECRET", Value: "hidden", IsSensitive: true},
		}

		filtered := filterOutSensitiveAppVars(vars)
		Expect(filtered).To(HaveLen(1))
		Expect(filtered[0].Key).To(Equal("APP_MODE"))
	})

	It("filters out sensitive effective vars", func() {
		vars := envvartypes.EnvVariableList{
			{Key: "PUBLIC", Value: "visible", IsSensitive: false},
			{Key: "SECRET", Value: "hidden", IsSensitive: true},
		}

		filtered := filterOutSensitiveEffectiveVars(vars)
		Expect(filtered).To(HaveLen(1))
		Expect(filtered[0].Key).To(Equal("PUBLIC"))
	})

	It("escapes multiline descriptions in rendered records", func() {
		content := renderRecords([]renderRecord{
			{
				Key:         "KEY",
				Value:       "value",
				Description: "line1\nline2",
			},
		})
		Expect(content).To(Equal("# desc: \"line1\\nline2\"\nKEY=value\n"))
	})

	It("does not insert blank lines between rendered records", func() {
		content := renderRecords([]renderRecord{
			{
				Key:         "FIRST_KEY",
				Value:       "first",
				Description: "first desc",
			},
			{
				Key:         "SECOND_KEY",
				Value:       "second",
				Description: "second desc",
			},
		})
		Expect(content).To(Equal("# desc: first desc\nFIRST_KEY=first\n# desc: second desc\nSECOND_KEY=second\n"))
	})

	It("renders scoped import template with multiple examples", func() {
		content := RenderScopedImportTemplate()
		expectedLines := []string{
			"# scopeType: workspace",
			"BKMS_SHARED_NAMESPACE=bk-shared",
			"# scopeValue: development",
			"FEATURE_FLAG=true",
			"# scopeValue: production",
			"JAVA_OPTS=\"-Xms1024m -Xmx2048m\"",
		}
		for _, line := range expectedLines {
			Expect(strings.Contains(content, line)).To(BeTrue(), "expected scoped template to contain %q", line)
		}
	})

	It("renders env and app import template with multiple examples", func() {
		content := RenderEnvAppImportTemplate()
		expectedLines := []string{
			"APP_NAME=demo-service",
			"FEATURE_FLAG=true",
			"APP_PORT=8080",
			"WELCOME_MESSAGE=\"hello bkms\"",
			"EMPTY_VALUE=\"\"",
		}
		for _, line := range expectedLines {
			Expect(strings.Contains(content, line)).To(BeTrue(), "expected env/app template to contain %q", line)
		}
	})

	It("keeps the last occurrence when env variable keys repeat", func() {
		vars := envvartypes.EnvVariableList{
			{Key: "SHARED", Value: "workspace"},
			{Key: "OTHER", Value: "other"},
			{Key: "SHARED", Value: "app"},
		}

		deduped := vars.ToDeduplicatedList()
		Expect(deduped).To(HaveLen(2))
		Expect(deduped[0]).To(Equal(envvartypes.EnvVariableObj{Key: "OTHER", Value: "other"}))
		Expect(deduped[1]).To(Equal(envvartypes.EnvVariableObj{Key: "SHARED", Value: "app"}))
	})
})
