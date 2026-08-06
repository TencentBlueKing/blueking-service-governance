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

package repo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

var _ = Describe("Test Index.ListPaginatedChartEntries", func() {
	const targetChart = "sample-app"

	// newFixtureIndex 构造一个固定的 in-memory IndexFile，包含 5 个版本
	newFixtureIndex := func() *repo.IndexFile {
		index := repo.NewIndexFile()
		versions := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"}
		for _, v := range versions {
			index.Entries[targetChart] = append(index.Entries[targetChart], &repo.ChartVersion{
				Metadata: &chart.Metadata{Name: targetChart, Version: v},
				Digest:   "digest-" + v,
			})
		}
		return index
	}

	Context("keyword filter", func() {
		It("should match version substring case-insensitively", func() {
			repoIndex := NewIndex(newFixtureIndex())
			result := repoIndex.ListPaginatedChartEntries(targetChart, "2.", 1, 10)
			Expect(result.TotalCount).To(Equal(int64(3)))
			Expect(result.Entries).To(HaveLen(3))
		})

		It("should return all entries when keyword is empty", func() {
			repoIndex := NewIndex(newFixtureIndex())
			result := repoIndex.ListPaginatedChartEntries(targetChart, "", 1, 10)
			Expect(result.TotalCount).To(Equal(int64(5)))
			Expect(result.Entries).To(HaveLen(5))
		})
	})

	Context("page out of range", func() {
		It("should return empty entries but correct total count when start exceeds size", func() {
			repoIndex := NewIndex(newFixtureIndex())
			result := repoIndex.ListPaginatedChartEntries(targetChart, "", 100, 10)
			Expect(result.TotalCount).To(Equal(int64(5)))
			Expect(result.Entries).NotTo(BeNil())
			Expect(result.Entries).To(BeEmpty())
		})

		It("should return empty entries when page or pageSize is non-positive", func() {
			repoIndex := NewIndex(newFixtureIndex())
			result := repoIndex.ListPaginatedChartEntries(targetChart, "", 0, 10)
			Expect(result.TotalCount).To(Equal(int64(5)))
			Expect(result.Entries).NotTo(BeNil())
			Expect(result.Entries).To(BeEmpty())
		})
	})

	Context("chart not found", func() {
		It("should return an empty (non-nil) slice when chart name is missing from index", func() {
			repoIndex := NewIndex(newFixtureIndex())
			entries := repoIndex.ListChartEntries("non-existent-chart")
			Expect(entries).NotTo(BeNil())
			Expect(entries).To(BeEmpty())

			result := repoIndex.ListPaginatedChartEntries("non-existent-chart", "", 1, 10)
			Expect(result.TotalCount).To(Equal(int64(0)))
			Expect(result.Entries).NotTo(BeNil())
			Expect(result.Entries).To(BeEmpty())
		})
	})
})
