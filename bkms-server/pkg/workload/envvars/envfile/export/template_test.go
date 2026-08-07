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

package export_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	exporter "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/export"
)

var _ = Describe("TemplateRendering", func() {
	It("renders scoped import template", func() {
		content := exporter.RenderScopedImportTemplate()

		Expect(content).To(ContainSubstring("# - Each variable uses KEY=VALUE format."))
		Expect(content).To(ContainSubstring("# - Optional # desc: applies to the next variable."))
		Expect(content).To(ContainSubstring("# - Scoped import requires # scopeType: workspace or envType."))
		Expect(content).To(ContainSubstring("# - # scopeValue is required when scopeType=envType."))
		Expect(content).To(ContainSubstring("# scopeType: workspace"))
		Expect(content).To(ContainSubstring("# scopeType: envType"))
		Expect(content).To(ContainSubstring("# scopeValue: production"))
		Expect(content).To(ContainSubstring("BKMS_NAMESPACE=bk-prod"))
		Expect(content).To(ContainSubstring("BKMS_ENV_TYPE=production"))
	})

	It("renders shared env and app import template", func() {
		content := exporter.RenderEnvAppImportTemplate()

		Expect(content).To(ContainSubstring("# - Each variable uses KEY=VALUE format."))
		Expect(content).To(ContainSubstring("# - Optional # desc: applies to the next variable."))
		Expect(content).To(ContainSubstring("# - Do not add # scopeType or # scopeValue in env/app import."))
		Expect(content).To(ContainSubstring("# desc: example description"))
		Expect(content).To(ContainSubstring("EXAMPLE_KEY=example-value"))
		Expect(content).NotTo(ContainSubstring("# scopeType:"))
		Expect(content).NotTo(ContainSubstring("# scopeValue:"))
		Expect(content).To(HaveSuffix("\n"))
	})
})
