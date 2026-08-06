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

package component_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ComponentDefStoreMongo", func() {
	var compDefStore component.ComponentDefStore
	var ctx context.Context
	var compDef *component.ComponentDef

	BeforeEach(func() {
		var err error

		compDefStore, err = component.NewComponentDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()

		// The default component for testing
		compDef = &component.ComponentDef{
			Name:        "SetReplicas-" + stringx.Random(6),
			Version:     "v1.0.0",
			Description: "Set the number of replicas for containers",
			Properties:  []component.Property{{Name: "replicas", Type: "INT", Description: "Number of replicas"}},
			Patchers:    []string{"spec:\n  replicas: {{ .replicas }}"},
			Creator:     "admin",
			Updater:     "admin",
		}
	})

	AfterEach(func() {
		_, _ = compDefStore.Delete(ctx, compDef.Name, compDef.Version)
	})

	Describe("Create", func() {
		Context("creating a valid component-def", func() {
			It("should create and get successfully", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal(compDef.Name))
				Expect(retrieved.Version).To(Equal(compDef.Version))
				Expect(retrieved.DisplayName).To(Equal(compDef.DisplayName))
			})

			DescribeTable("should persist fragment fields as arrays",
				func(patchers, specs []string) {
					compDef.Patchers = patchers
					compDef.Specs = specs
					Expect(compDefStore.Create(ctx, compDef)).To(Succeed())

					retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Patchers).NotTo(BeNil())
					Expect(retrieved.Specs).NotTo(BeNil())
				},
				Entry("patchers only", []string{"spec:\n  replicas: 1\n"}, nil),
				Entry("specs only", nil, []string{"apiVersion: v1\nkind: ConfigMap\n"}),
			)

			It("createdAt and updatedAt fields should behaviour normally", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.CreatedAt.IsZero()).To(BeFalse())
				Expect(retrieved.UpdatedAt.IsZero()).To(BeFalse())

				oldCreatedAt := retrieved.CreatedAt
				oldUpdatedAt := retrieved.UpdatedAt
				// Update the component and check the time related fields
				compDef.DisplayName = "Updated Name"
				err = compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err = compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				// CreatedAt should remain unchanged, UpdatedAt should be later than before
				Expect(retrieved.CreatedAt).To(Equal(oldCreatedAt))
				Expect(retrieved.UpdatedAt.Compare(oldUpdatedAt)).To(BeElementOf(0, 1))
			})
		})

		Context("Update existing component-def", func() {
			It("should update successfully", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				// Update the component
				compDef.DisplayName = "Updated Name"
				err = compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.DisplayName).To(Equal("Updated Name"))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing component-def", func() {
			It("should delete successfully", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				cnt, err := compDefStore.Delete(ctx, compDef.Name, compDef.Version)
				Expect(cnt).To(Equal(int64(1)))
				Expect(err).NotTo(HaveOccurred())

				_, err = compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).To(Equal(component.ErrComponentDefNotFound))
			})
		})
	})

	Describe("List", func() {
		var compDef2, compDef3 *component.ComponentDef

		BeforeEach(func() {
			// 创建多个测试组件
			compDef2 = &component.ComponentDef{
				Name:        "list-test-" + stringx.Random(6),
				Version:     "v1.0.0",
				DisplayName: "List Test Component",
				Description: "Test component for list",
				Properties:  []component.Property{{Name: "key", Type: "STRING", Description: "A key"}},
				Patchers:    []string{"key: {{ .key }}"},
				ScopeType:   component.ScopeTypeGlobal,
				Creator:     "admin",
				Updater:     "admin",
			}
			compDef3 = &component.ComponentDef{
				Name:                  "workspace-scope-comp-" + stringx.Random(6),
				Version:               "v1.0.0",
				DisplayName:           "Workspace Specific",
				Description:           "Test component for workspace scope",
				Properties:            []component.Property{{Name: "value", Type: "INT", Description: "A value"}},
				Patchers:              []string{"value: {{ .value }}"},
				ScopeType:             component.ScopeTypeWorkspace,
				ScopeWorkspaceIDs:     []string{"ws-test-001", "ws-test-002"},
				Creator:               "admin",
				Updater:               "admin",
				ManagedByWorkspaceIDs: []string{"ws-test-001"},
			}
		})

		AfterEach(func() {
			_, _ = compDefStore.Delete(ctx, compDef2.Name, compDef2.Version)
			_, _ = compDefStore.Delete(ctx, compDef3.Name, compDef3.Version)
		})

		Context("listing all component-defs without filters", func() {
			It("should return all component-defs", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())
				err = compDefStore.Create(ctx, compDef2)
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, nil)
				Expect(err).NotTo(HaveOccurred())
				// builtin 会引入默认组件，因此不能直接通过 list 返回的总量判断
				count := 0
				for _, r := range results {
					if r.Name == compDef.Name && r.Version == compDef.Version {
						count++
					}
					if r.Name == compDef2.Name && r.Version == compDef2.Version {
						count++
					}
				}
				Expect(count).To(Equal(2))
			})
		})

		Context("filtering by workspace ID", func() {
			It("should return global + workspace-specific component-defs", func() {
				err := compDefStore.Create(ctx, compDef2) // global
				Expect(err).NotTo(HaveOccurred())
				err = compDefStore.Create(ctx, compDef3) // workspace-specific
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, &component.ListOptions{
					ScopeWorkspaceID: "ws-test-001",
				})
				Expect(err).NotTo(HaveOccurred())

				// 应该包含 global 和 workspace-specific 的组件
				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(compDef2.Name))
				Expect(names).To(ContainElement(compDef3.Name))
				// 对于没配置可见性/不满足可见性的组件，不应该返回
				Expect(names).To(Not(ContainElement(compDef.Name)))
			})

			It("should not return workspace-specific component-defs for other workspaces", func() {
				err := compDefStore.Create(ctx, compDef3) // workspace-specific for ws-test-001, ws-test-002
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, &component.ListOptions{
					ScopeWorkspaceID: "ws-other",
				})
				Expect(err).NotTo(HaveOccurred())

				// 不应该包含 compDef3
				for _, r := range results {
					Expect(r.Name).NotTo(Equal(compDef3.Name))
				}
			})

			It("should return only component-defs that can be managed by the workspace", func() {
				err := compDefStore.Create(ctx, compDef3) // workspace-specific for ws-test-001, ws-test-002
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, &component.ListOptions{
					ManagedByWorkspaceID: "ws-test-001",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results)).To(Equal(1))
				Expect(results[0].Name).To(Equal(compDef3.Name))
			})
		})

		Context("filtering by keyword", func() {
			It("should search in name", func() {
				err := compDefStore.Create(ctx, compDef2)
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, &component.ListOptions{
					Keyword: "list-test",
				})
				Expect(err).NotTo(HaveOccurred())

				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(compDef2.Name))
			})

			It("should search in displayName (case insensitive)", func() {
				err := compDefStore.Create(ctx, compDef3)
				Expect(err).NotTo(HaveOccurred())

				results, err := compDefStore.List(ctx, &component.ListOptions{
					Keyword: "workspace specific",
				})
				Expect(err).NotTo(HaveOccurred())

				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(compDef3.Name))
			})
		})

		Context("combining multiple filters", func() {
			It("should apply all filters correctly", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())
				err = compDefStore.Create(ctx, compDef2)
				Expect(err).NotTo(HaveOccurred())
				err = compDefStore.Create(ctx, compDef3)
				Expect(err).NotTo(HaveOccurred())

				// 按 keyword 过滤
				results, err := compDefStore.List(ctx, &component.ListOptions{
					Keyword: "Workspace",
				})
				Expect(err).NotTo(HaveOccurred())

				// 应该只返回 compDef3
				names := make([]string, 0, len(results))
				for _, r := range results {
					names = append(names, r.Name)
				}
				Expect(names).To(ContainElement(compDef3.Name))
				Expect(names).NotTo(ContainElement(compDef2.Name))
			})
		})
	})

	Describe("UpdateInstanceCount", func() {
		Context("incrementing appCompInstanceCount", func() {
			It("should accumulate multiple increments", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < 5; i++ {
					err = compDefStore.UpdateInstanceCount(ctx, compDef.Name, component.FieldAppCompInstance, 1)
					Expect(err).NotTo(HaveOccurred())
				}

				retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.AppCompInstanceCount).To(Equal(int32(5)))
			})

			It("should decrement appCompInstanceCount with negative delta", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				// 先增加 3
				err = compDefStore.UpdateInstanceCount(ctx, compDef.Name, component.FieldAppCompInstance, 3)
				Expect(err).NotTo(HaveOccurred())

				// 再减少 2
				err = compDefStore.UpdateInstanceCount(ctx, compDef.Name, component.FieldAppCompInstance, -2)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.AppCompInstanceCount).To(Equal(int32(1)))
			})
		})
	})
})
