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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Builtin ClusterAddonDefs Tests", func() {
	const testAddonsPath = "./assets/testaddons"
	var store clusteraddon.ClusterAddonDefStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = clusteraddon.NewClusterAddonDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("LoadBuiltinFromFolder valid input", func() {
		AfterEach(func() {
			_, _ = store.Delete(ctx, "bkms-test-chart")
			_, _ = store.Delete(ctx, "no-namespace-addon")
		})

		It("should load valid addon defs from directory successfully", func() {
			err := clusteraddon.LoadBuiltinFromFolder(ctx, store, filepath.Join(testAddonsPath, "valid"))
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, "bkms-test-chart")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("bkms-test-chart"))
			Expect(got.DisplayName).To(Equal("测试集群插件"))
			Expect(got.ChartInfo.ChartName).To(Equal("bkms-test-chart"))
			Expect(got.ChartInfo.DefaultNamespace).To(Equal("bcs-system"))
			Expect(got.ChartInfo.ExampleValues).To(ContainSubstring("test"))
			Expect(got.OptionalForAppTypes).To(Equal([]string{"trpc", "taf"}))
			Expect(got.Creator).To(Equal("admin"))
		})
	})

	Context("LoadBuiltinFromFolder invalid input", func() {
		It("should fail with invalid YAML", func() {
			err := clusteraddon.LoadBuiltinFromFolder(
				ctx, store, filepath.Join(testAddonsPath, "broken/bad-yaml.yaml"),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unmarshal yaml"))
		})

		It("should fail with non-existent path", func() {
			err := clusteraddon.LoadBuiltinFromFolder(ctx, store, "/non/existent/path")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("stating path"))
		})
	})
})
