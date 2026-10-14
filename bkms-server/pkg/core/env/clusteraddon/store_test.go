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
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
)

var _ = Describe("ClusterAddonDef store", func() {
	var (
		ctx      context.Context
		store    *clusteraddon.ClusterAddonDefStore
		addonDef *clusteraddon.ClusterAddonDef
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = testutil.NewStore(GinkgoT(), clusteraddon.ClusterAddonDefMeta)

		addonDef = &clusteraddon.ClusterAddonDef{
			Name:        "test-addon-" + stringx.Random(6),
			DisplayName: "Test Addon",
			Description: "A test addon for unit testing",
			ChartInfo: clusteraddon.HelmChartInfo{
				ChartName:           "test-chart-" + stringx.Random(6),
				DefaultChartVersion: "1.0.0",
				DefaultNamespace:    "test-ns",
			},
			RequiredForAppTypes:     []string{"webserver"},
			OptionalForAppTypes:     []string{"worker"},
			UnsupportedOnFederation: true,
			Creator:                 "admin",
		}
	})

	AfterEach(func() {
		_ = store.DeleteIfExists(ctx, clusteraddon.ByName(addonDef.Name))
	})

	Describe("Create", func() {
		Context("creating a valid addon def", func() {
			It("should create and get successfully", func() {
				err := clusteraddon.SaveDef(ctx, store, addonDef)
				Expect(err).NotTo(HaveOccurred())

				got, err := store.Get(ctx, clusteraddon.ByName(addonDef.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Name).To(Equal(addonDef.Name))
				Expect(got.DisplayName).To(Equal(addonDef.DisplayName))
				Expect(got.ChartInfo.ChartName).To(Equal(addonDef.ChartInfo.ChartName))
				Expect(got.ChartInfo.DefaultChartVersion).To(Equal("1.0.0"))
				Expect(got.RequiredForAppTypes).To(Equal([]string{"webserver"}))
				Expect(got.OptionalForAppTypes).To(Equal([]string{"worker"}))
				Expect(got.UnsupportedOnFederation).To(BeTrue())
			})
		})

		Context("updating an existing addon def", func() {
			It("should update successfully via upsert", func() {
				err := clusteraddon.SaveDef(ctx, store, addonDef)
				Expect(err).NotTo(HaveOccurred())

				addonDef.DisplayName = "Updated Name"
				addonDef.ChartInfo.DefaultChartVersion = "2.0.0"
				addonDef.RequiredForAppTypes = []string{"webserver", "worker"}
				err = clusteraddon.SaveDef(ctx, store, addonDef)
				Expect(err).NotTo(HaveOccurred())

				got, err := store.Get(ctx, clusteraddon.ByName(addonDef.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(got.DisplayName).To(Equal("Updated Name"))
				Expect(got.ChartInfo.DefaultChartVersion).To(Equal("2.0.0"))
				Expect(got.RequiredForAppTypes).To(Equal([]string{"webserver", "worker"}))
			})
		})
	})

	Describe("Get", func() {
		Context("when addon def does not exist", func() {
			It("should return ErrClusterAddonDefNotFound", func() {
				_, err := store.Get(ctx, clusteraddon.ByName("non-existent-"+stringx.Random(6)))
				Expect(err).To(MatchError(clusteraddon.ErrClusterAddonDefNotFound))
			})
		})
	})

	Describe("List", func() {
		var addonDef2, addonDef3 *clusteraddon.ClusterAddonDef

		BeforeEach(func() {
			addonDef2 = &clusteraddon.ClusterAddonDef{
				Name:      "list-b-" + stringx.Random(6),
				ChartInfo: clusteraddon.HelmChartInfo{ChartName: "chart-b"},
			}
			addonDef3 = &clusteraddon.ClusterAddonDef{
				Name:      "list-a-" + stringx.Random(6),
				ChartInfo: clusteraddon.HelmChartInfo{ChartName: "chart-a"},
			}
		})

		AfterEach(func() {
			_ = store.DeleteIfExists(ctx, clusteraddon.ByName(addonDef2.Name))
			_ = store.DeleteIfExists(ctx, clusteraddon.ByName(addonDef3.Name))
		})

		Context("listing all addon defs", func() {
			It("should return all created addon defs", func() {
				err := clusteraddon.SaveDef(ctx, store, addonDef)
				Expect(err).NotTo(HaveOccurred())
				err = clusteraddon.SaveDef(ctx, store, addonDef2)
				Expect(err).NotTo(HaveOccurred())

				results, err := clusteraddon.ListDefs(ctx, store)
				Expect(err).NotTo(HaveOccurred())

				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(addonDef.Name))
				Expect(names).To(ContainElement(addonDef2.Name))
			})

			It("should return results sorted by name", func() {
				err := clusteraddon.SaveDef(ctx, store, addonDef2)
				Expect(err).NotTo(HaveOccurred())
				err = clusteraddon.SaveDef(ctx, store, addonDef3)
				Expect(err).NotTo(HaveOccurred())

				results, err := clusteraddon.ListDefs(ctx, store)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(BeNumerically(">=", 2))

				for i := 1; i < len(results); i++ {
					Expect(results[i].Name >= results[i-1].Name).To(BeTrue(),
						"expected %s >= %s", results[i].Name, results[i-1].Name)
				}
			})
		})
	})
})
