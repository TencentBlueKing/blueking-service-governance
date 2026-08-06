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

package clusteraddon_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"

	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
)

var _ = Describe("RepoIndex", func() {
	var repoIndex *clusteraddon.RepoIndex

	BeforeEach(func() {
		indexFile := &repo.IndexFile{
			Entries: map[string]repo.ChartVersions{
				"bk-log-collector": {
					{Metadata: &chart.Metadata{Version: "1.2.0"}},
					{Metadata: &chart.Metadata{Version: "1.1.0"}},
					{Metadata: &chart.Metadata{Version: "1.0.0"}},
				},
				"bk-monitor-agent": {
					{Metadata: &chart.Metadata{Version: "2.0.0"}},
				},
			},
		}
		repoIndex = clusteraddon.NewRepoIndex(indexFile)
	})

	Describe("GetChartVersions", func() {
		It("should return all versions for an existing chart", func() {
			versions := repoIndex.ListChartVersions("bk-log-collector")
			Expect(versions).To(Equal([]string{"1.2.0", "1.1.0", "1.0.0"}))
		})

		It("should return single version when chart has one entry", func() {
			versions := repoIndex.ListChartVersions("bk-monitor-agent")
			Expect(versions).To(Equal([]string{"2.0.0"}))
		})

		It("should return nil for a non-existent chart", func() {
			versions := repoIndex.ListChartVersions("non-existent-chart")
			Expect(versions).To(BeNil())
		})
	})

	Describe("FetchRepoIndex with real helm registry (utdeps)", func() {
		It("should fetch and parse real helm repo index", func() {
			// 使用 utdeps 中的 ChartMuseum 服务测试真实的 Helm registry 交互
			realIndex, err := clusteraddon.FetchRepoIndex()
			if err != nil {
				Skip("Helm registry not available: " + err.Error())
			}

			Expect(realIndex).NotTo(BeNil())
			// sample-app 是 utdeps 中内置的示例 chart
			versions := realIndex.ListChartVersions("sample-app")
			Expect(versions).NotTo(BeEmpty())
			Expect(versions[0]).To(Equal("0.1.0"))
		})

		It("should list all chart names from real registry", func() {
			realIndex, err := clusteraddon.FetchRepoIndex()
			if err != nil {
				Skip("Helm registry not available: " + err.Error())
			}

			names := realIndex.ListChartNames()
			Expect(names).NotTo(BeEmpty())
			Expect(names).To(ContainElement("sample-app"))
		})
	})
})
