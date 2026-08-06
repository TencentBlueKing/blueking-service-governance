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

	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ClusterAddonDefStoreMongo", func() {
	var (
		ctx      context.Context
		store    clusteraddon.ClusterAddonDefStore
		addonDef *clusteraddon.ClusterAddonDef
	)

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		store, err = clusteraddon.NewClusterAddonDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		addonDef = &clusteraddon.ClusterAddonDef{
			Name:        "test-addon-" + stringx.Random(6),
			DisplayName: "Test Addon",
			Description: "A test addon for unit testing",
			ChartInfo: clusteraddon.HelmChartInfo{
				ChartName:           "test-chart-" + stringx.Random(6),
				DefaultChartVersion: "1.0.0",
				DefaultNamespace:    "test-ns",
			},
			RequiredForAppTypes: []string{"webserver"},
			OptionalForAppTypes: []string{"worker"},
			Creator:             "admin",
		}
	})

	AfterEach(func() {
		_, _ = store.Delete(ctx, addonDef.Name)
	})

	Describe("Create", func() {
		Context("creating a valid addon def", func() {
			It("should create and get successfully", func() {
				err := store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				got, err := store.Get(ctx, addonDef.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.Name).To(Equal(addonDef.Name))
				Expect(got.DisplayName).To(Equal(addonDef.DisplayName))
				Expect(got.ChartInfo.ChartName).To(Equal(addonDef.ChartInfo.ChartName))
				Expect(got.ChartInfo.DefaultChartVersion).To(Equal("1.0.0"))
				Expect(got.RequiredForAppTypes).To(Equal([]string{"webserver"}))
				Expect(got.OptionalForAppTypes).To(Equal([]string{"worker"}))
			})

			It("createdAt and updatedAt fields should behave normally", func() {
				err := store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := store.Get(ctx, addonDef.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.CreatedAt.IsZero()).To(BeFalse())
				Expect(retrieved.UpdatedAt.IsZero()).To(BeFalse())

				oldCreatedAt := retrieved.CreatedAt
				oldUpdatedAt := retrieved.UpdatedAt

				addonDef.DisplayName = "Updated Name"
				err = store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err = store.Get(ctx, addonDef.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.CreatedAt).To(Equal(oldCreatedAt))
				Expect(retrieved.UpdatedAt.Compare(oldUpdatedAt)).To(BeElementOf(0, 1))
			})
		})

		Context("updating an existing addon def", func() {
			It("should update successfully via upsert", func() {
				err := store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				addonDef.DisplayName = "Updated Name"
				addonDef.ChartInfo.DefaultChartVersion = "2.0.0"
				addonDef.RequiredForAppTypes = []string{"webserver", "worker"}
				err = store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				got, err := store.Get(ctx, addonDef.Name)
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
				_, err := store.Get(ctx, "non-existent-"+stringx.Random(6))
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
			_, _ = store.Delete(ctx, addonDef2.Name)
			_, _ = store.Delete(ctx, addonDef3.Name)
		})

		Context("listing all addon defs", func() {
			It("should return all created addon defs", func() {
				err := store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())
				err = store.Create(ctx, addonDef2)
				Expect(err).NotTo(HaveOccurred())

				results, err := store.List(ctx)
				Expect(err).NotTo(HaveOccurred())

				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(addonDef.Name))
				Expect(names).To(ContainElement(addonDef2.Name))
			})

			It("should return results sorted by name", func() {
				err := store.Create(ctx, addonDef2)
				Expect(err).NotTo(HaveOccurred())
				err = store.Create(ctx, addonDef3)
				Expect(err).NotTo(HaveOccurred())

				results, err := store.List(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(BeNumerically(">=", 2))

				for i := 1; i < len(results); i++ {
					Expect(results[i].Name >= results[i-1].Name).To(BeTrue(),
						"expected %s >= %s", results[i].Name, results[i-1].Name)
				}
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing addon def", func() {
			It("should delete successfully", func() {
				err := store.Create(ctx, addonDef)
				Expect(err).NotTo(HaveOccurred())

				count, err := store.Delete(ctx, addonDef.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(1)))

				_, err = store.Get(ctx, addonDef.Name)
				Expect(err).To(MatchError(clusteraddon.ErrClusterAddonDefNotFound))
			})
		})

		Context("when deleting a non-existent addon def", func() {
			It("should return 0 deleted count", func() {
				count, err := store.Delete(ctx, "non-existent-"+stringx.Random(6))
				Expect(err).NotTo(HaveOccurred())
				Expect(count).To(Equal(int64(0)))
			})
		})
	})
})
