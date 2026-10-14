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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("ComponentDefStore", func() {
	var compDefStore *component.ComponentDefStore
	var ctx context.Context
	var compDef *component.ComponentDef

	get := func(name, version string) (*component.ComponentDef, error) {
		return compDefStore.Get(ctx, component.ByNameVersion(name, version))
	}
	deleteDef := func(d *component.ComponentDef) {
		Expect(compDefStore.DeleteIfExists(ctx, component.ByNameVersion(d.Name, d.Version))).To(Succeed())
	}
	listNames := func(opts ...store.Opt) []string {
		results, err := compDefStore.List(ctx, opts...)
		Expect(err).NotTo(HaveOccurred())
		names := make([]string, 0, len(results))
		for _, r := range results {
			names = append(names, r.Name)
		}
		return names
	}

	BeforeEach(func() {
		compDefStore = testutil.NewStore(GinkgoT(), component.ComponentDefMeta)
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
		deleteDef(compDef)
	})

	Describe("Create", func() {
		Context("creating a valid component-def", func() {
			DescribeTable("should persist fragment fields as arrays",
				func(patchers, specs []string) {
					compDef.Patchers = patchers
					compDef.Specs = specs
					Expect(compDefStore.Create(ctx, compDef)).To(Succeed())

					retrieved, err := get(compDef.Name, compDef.Version)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Patchers).NotTo(BeNil())
					Expect(retrieved.Specs).NotTo(BeNil())
				},
				Entry("patchers only", []string{"spec:\n  replicas: 1\n"}, nil),
				Entry("specs only", nil, []string{"apiVersion: v1\nkind: ConfigMap\n"}),
			)

			It("should return ErrComponentDefAlreadyExists when name+version already exists", func() {
				Expect(compDefStore.Create(ctx, compDef)).To(Succeed())

				err := compDefStore.Create(ctx, compDef)
				Expect(err).To(MatchError(component.ErrComponentDefAlreadyExists))
			})
		})

		It("should reject an invalid component-def", func() {
			compDef.Patchers = []string{"not: [valid"}
			Expect(compDefStore.Create(ctx, compDef)).To(MatchError(ContainSubstring("component validation failed")))
			_, err := get(compDef.Name, compDef.Version)
			Expect(err).To(MatchError(component.ErrComponentDefNotFound))
		})
	})

	Describe("UpsertDef", func() {
		It("should create the component-def when it does not exist", func() {
			Expect(component.UpsertDef(ctx, compDefStore, compDef)).To(Succeed())

			retrieved, err := get(compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.Description).To(Equal(compDef.Description))
			Expect(retrieved.Specs).To(BeEmpty())
			Expect(retrieved.Specs).NotTo(BeNil())
		})

		It("should keep instance counts maintained by UpdateInstanceCount", func() {
			Expect(compDefStore.Create(ctx, compDef)).To(Succeed())
			Expect(component.UpdateInstanceCount(
				ctx, compDefStore, compDef.Name, component.FieldWorkspaceCompInstance, 2,
			)).To(Succeed())

			// 调用方对象中的计数是旧值（0），覆盖时不应写回
			compDef.DisplayName = "Updated Name"
			Expect(component.UpsertDef(ctx, compDefStore, compDef)).To(Succeed())

			retrieved, err := get(compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.DisplayName).To(Equal("Updated Name"))
			Expect(retrieved.WorkspaceCompInstanceCount).To(Equal(int32(2)))
		})

		It("should update an existing component-def", func() {
			err := compDefStore.Create(ctx, compDef)
			Expect(err).NotTo(HaveOccurred())

			compDef.DisplayName = "Updated Name"
			err = component.UpsertDef(ctx, compDefStore, compDef)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := get(compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved.DisplayName).To(Equal("Updated Name"))
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
			deleteDef(compDef2)
			deleteDef(compDef3)
		})

		Context("filtering by workspace ID", func() {
			It("should return global + workspace-specific component-defs", func() {
				err := compDefStore.Create(ctx, compDef2) // global
				Expect(err).NotTo(HaveOccurred())
				err = compDefStore.Create(ctx, compDef3) // workspace-specific
				Expect(err).NotTo(HaveOccurred())

				names := listNames(component.UsableInWorkspace("ws-test-001"))
				// 应该包含 global 和 workspace-specific 的组件
				Expect(names).To(ContainElement(compDef2.Name))
				Expect(names).To(ContainElement(compDef3.Name))
				// 对于没配置可见性/不满足可见性的组件，不应该返回
				Expect(names).To(Not(ContainElement(compDef.Name)))
			})

			It("should not return workspace-specific component-defs for other workspaces", func() {
				err := compDefStore.Create(ctx, compDef3) // workspace-specific for ws-test-001, ws-test-002
				Expect(err).NotTo(HaveOccurred())

				// 不应该包含 compDef3
				Expect(listNames(component.UsableInWorkspace("ws-other"))).NotTo(ContainElement(compDef3.Name))
			})

			It("should return only component-defs that can be managed by the workspace", func() {
				err := compDefStore.Create(ctx, compDef3) // workspace-specific for ws-test-001, ws-test-002
				Expect(err).NotTo(HaveOccurred())

				Expect(listNames(component.ManagedByWorkspace("ws-test-001"))).To(Equal([]string{compDef3.Name}))
			})
		})

		Context("filtering by keyword", func() {
			It("should search in name", func() {
				err := compDefStore.Create(ctx, compDef2)
				Expect(err).NotTo(HaveOccurred())

				Expect(listNames(component.KeywordMatches("list-test"))).To(ContainElement(compDef2.Name))
			})

			It("should search in displayName (case insensitive)", func() {
				err := compDefStore.Create(ctx, compDef3)
				Expect(err).NotTo(HaveOccurred())

				Expect(listNames(component.KeywordMatches("workspace specific"))).To(ContainElement(compDef3.Name))
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
				names := listNames(component.KeywordMatches("Workspace"))
				Expect(names).To(ContainElement(compDef3.Name))
				Expect(names).NotTo(ContainElement(compDef2.Name))

				// 工作空间范围与关键字各自是一组 $or，组合后两组条件都要满足
				names = listNames(component.UsableInWorkspace("ws-test-001"), component.KeywordMatches("list test"))
				Expect(names).To(ContainElement(compDef2.Name))
				Expect(names).NotTo(ContainElement(compDef3.Name))
				Expect(names).NotTo(ContainElement(compDef.Name))
			})
		})
	})

	Describe("UpdateInstanceCount", func() {
		Context("incrementing appCompInstanceCount", func() {
			It("should accumulate multiple increments", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				for i := 0; i < 5; i++ {
					err = component.UpdateInstanceCount(
						ctx,
						compDefStore,
						compDef.Name,
						component.FieldAppCompInstance,
						1,
					)
					Expect(err).NotTo(HaveOccurred())
				}

				retrieved, err := get(compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.AppCompInstanceCount).To(Equal(int32(5)))
			})

			It("should decrement appCompInstanceCount with negative delta", func() {
				err := compDefStore.Create(ctx, compDef)
				Expect(err).NotTo(HaveOccurred())

				// 先增加 3
				err = component.UpdateInstanceCount(ctx, compDefStore, compDef.Name, component.FieldAppCompInstance, 3)
				Expect(err).NotTo(HaveOccurred())

				// 再减少 2
				err = component.UpdateInstanceCount(
					ctx,
					compDefStore,
					compDef.Name,
					component.FieldAppCompInstance,
					-2,
				)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := get(compDef.Name, compDef.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.AppCompInstanceCount).To(Equal(int32(1)))
			})
		})

		It("should not refresh updatedAt of the definition", func() {
			Expect(compDefStore.Create(ctx, compDef)).To(Succeed())
			before, err := get(compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())

			err = component.UpdateInstanceCount(
				ctx,
				compDefStore,
				compDef.Name,
				component.FieldWorkspaceCompInstance,
				1,
			)
			Expect(err).NotTo(HaveOccurred())

			after, err := get(compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())
			Expect(after.WorkspaceCompInstanceCount).To(Equal(before.WorkspaceCompInstanceCount + 1))
			Expect(after.UpdatedAt).To(Equal(before.UpdatedAt))
		})

		It("should ignore a missing definition", func() {
			err := component.UpdateInstanceCount(ctx, compDefStore, "not-exists", component.FieldAppCompInstance, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
