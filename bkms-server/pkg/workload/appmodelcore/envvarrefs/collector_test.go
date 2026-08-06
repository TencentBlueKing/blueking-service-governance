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

package envvarrefs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

var _ = Describe("Collector", func() {
	It("aggregates, deduplicates, and sorts undefined env vars", func() {
		collector := envvarrefs.NewCollector(map[string]string{"EMPTY": ""})

		Expect(collector.Collect(
			`${{ env.ZED }} ${{ env.SHARED }} ${{ env.EMPTY }} ${{ bkms.IGNORED }}`,
			envvarrefs.Source{Type: envvarrefs.SourceAppConfigFile, Name: "production"},
		)).To(Succeed())
		Expect(collector.Collect(
			`${{ env.SHARED }} ${{ env.SHARED }}`,
			envvarrefs.Source{Type: envvarrefs.SourceComponent, Name: "z-component"},
		)).To(Succeed())
		Expect(collector.Collect(
			`${{ env.SHARED }}`,
			envvarrefs.Source{Type: envvarrefs.SourceComponent, Name: "a-component"},
		)).To(Succeed())

		Expect(collector.UndefinedEnvVars()).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "SHARED",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceAppConfigFile, Name: "production"},
					{Type: envvarrefs.SourceComponent, Name: "a-component"},
					{Type: envvarrefs.SourceComponent, Name: "z-component"},
				},
			},
			{
				Key: "ZED",
				Sources: []envvarrefs.Source{{
					Type: envvarrefs.SourceAppConfigFile,
					Name: "production",
				}},
			},
		}))
	})

	It("returns a non-nil empty result when all references are defined", func() {
		collector := envvarrefs.NewCollector(map[string]string{"DEFINED": "value"})

		Expect(collector.Collect(
			`${{ env.DEFINED }}`,
			envvarrefs.Source{Type: envvarrefs.SourcePolaris, Name: "main"},
		)).To(Succeed())
		Expect(collector.UndefinedEnvVars()).To(BeEmpty())
		Expect(collector.UndefinedEnvVars()).NotTo(BeNil())
	})

	It("returns extraction errors", func() {
		collector := envvarrefs.NewCollector(nil)

		err := collector.Collect(
			`${{ env.BROKEN }`,
			envvarrefs.Source{Type: envvarrefs.SourceComponent, Name: "broken"},
		)

		Expect(err).To(HaveOccurred())
	})
})
