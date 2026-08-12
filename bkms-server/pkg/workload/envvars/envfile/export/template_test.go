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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TemplateRendering", func() {
	It("renders scoped import template", func() {
		content := RenderScopedImportTemplate()

		Expect(content).To(ContainSubstring("# - Each variable uses KEY=VALUE format."))
		Expect(content).To(ContainSubstring("# - Optional # desc: applies to the next variable."))
		Expect(content).To(ContainSubstring("# - Scoped import requires # scopeType: workspace or envType."))
		Expect(content).To(ContainSubstring("# - # scopeValue is required when scopeType=envType."))
		Expect(content).To(ContainSubstring(
			"# - scopeValue for envType can be: development, test, staging, production.",
		))
		Expect(content).To(ContainSubstring("# scopeType: workspace"))
		Expect(content).To(ContainSubstring("BKMS_SHARED_NAMESPACE=bk-shared"))
		Expect(content).To(ContainSubstring("# scopeType: envType"))
		Expect(content).To(ContainSubstring("# scopeValue: development"))
		Expect(content).To(ContainSubstring("FEATURE_FLAG=true"))
		Expect(content).To(ContainSubstring("# scopeValue: production"))
		Expect(content).To(ContainSubstring("JAVA_OPTS=\"-Xms1024m -Xmx2048m\""))
	})

	It("renders shared env and app import template", func() {
		content := RenderEnvAppImportTemplate()

		Expect(content).To(ContainSubstring("# - Each variable uses KEY=VALUE format."))
		Expect(content).To(ContainSubstring("# - Optional # desc: applies to the next variable."))
		Expect(content).To(ContainSubstring("# - Do not add # scopeType or # scopeValue in env/app import."))
		Expect(content).To(ContainSubstring("APP_NAME=demo-service"))
		Expect(content).To(ContainSubstring("FEATURE_FLAG=true"))
		Expect(content).To(ContainSubstring("APP_PORT=8080"))
		Expect(content).To(ContainSubstring("WELCOME_MESSAGE=\"hello bkms\""))
		Expect(content).To(ContainSubstring("EMPTY_VALUE=\"\""))
		Expect(content).NotTo(ContainSubstring("# scopeType:"))
		Expect(content).NotTo(ContainSubstring("# scopeValue:"))
		Expect(content).To(HaveSuffix("\n"))
	})
})
