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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
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
		ctx = testutil.WithTenantOrDefault(context.Background())

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

	Describe("tenancy", func() {
		var ctxOther context.Context
		var builtin *component.ComponentDef

		BeforeEach(func() {
			ctxOther = tenant.WithTenantID(context.Background(), "tenant-"+stringx.Random(6))

			platform, err := tenant.PlatformTenantID()
			Expect(err).NotTo(HaveOccurred())
			builtin = &component.ComponentDef{
				Name:      "Builtin-" + stringx.Random(6),
				Version:   component.DefaultComponentDefVersion,
				Patchers:  []string{"foo: bar\n"},
				ScopeType: component.ScopeTypeGlobal,
				IsBuiltin: true,
			}
			Expect(component.UpsertDef(ctx, compDefStore, builtin, store.TenantOf(platform))).To(Succeed())
			DeferCleanup(func() {
				Expect(compDefStore.DeleteIfExists(ctx, component.ByNameVersion(builtin.Name, builtin.Version),
					store.TenantAll())).To(Succeed())
			})
		})

		It("should share builtin definitions of the platform tenant with other tenants", func() {
			got, err := component.GetDef(ctxOther, compDefStore, builtin.Name, builtin.Version)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.IsBuiltin).To(BeTrue())
			Expect(component.BuiltinDefExists(ctxOther, compDefStore, builtin.Name, builtin.Version)).To(BeTrue())
		})

		It("should not share custom definitions of the platform tenant", func() {
			Expect(compDefStore.Create(ctx, compDef)).To(Succeed())
			_, err := component.GetDef(ctxOther, compDefStore, compDef.Name, compDef.Version)
			Expect(err).To(MatchError(component.ErrComponentDefNotFound))
		})

		It("should allow the same name in different tenants", func() {
			Expect(compDefStore.Create(ctx, compDef)).To(Succeed())
			other := *compDef
			Expect(compDefStore.Create(ctxOther, &other)).To(Succeed())
			DeferCleanup(func() {
				Expect(compDefStore.DeleteIfExists(ctxOther,
					component.ByNameVersion(other.Name, other.Version))).To(Succeed())
			})
		})

		Context("a tenant definition shadows a builtin one of the same name", func() {
			BeforeEach(func() {
				owned := &component.ComponentDef{
					Name:        builtin.Name,
					Version:     builtin.Version,
					Description: "owned",
					Patchers:    []string{"foo: owned\n"},
					ScopeType:   component.ScopeTypeGlobal,
				}
				Expect(compDefStore.Create(ctxOther, owned)).To(Succeed())
				DeferCleanup(func() {
					Expect(compDefStore.DeleteIfExists(ctxOther,
						component.ByNameVersion(owned.Name, owned.Version))).To(Succeed())
				})
			})

			It("should resolve to the tenant definition", func() {
				got, err := component.GetDef(ctxOther, compDefStore, builtin.Name, builtin.Version)
				Expect(err).NotTo(HaveOccurred())
				Expect(got.IsBuiltin).To(BeFalse())
				Expect(got.Description).To(Equal("owned"))
			})

			It("should hide the shadowed builtin definition from the list", func() {
				exclude, err := component.ExcludeShadowedBuiltins(ctxOther, compDefStore)
				Expect(err).NotTo(HaveOccurred())
				defs, err := compDefStore.List(ctxOther, store.Where("name", builtin.Name), exclude)
				Expect(err).NotTo(HaveOccurred())
				Expect(defs).To(HaveLen(1))
				Expect(defs[0].IsBuiltin).To(BeFalse())
			})

			It("should keep the builtin definition visible to other tenants", func() {
				exclude, err := component.ExcludeShadowedBuiltins(ctx, compDefStore)
				Expect(err).NotTo(HaveOccurred())
				defs, err := compDefStore.List(ctx, store.Where("name", builtin.Name), exclude)
				Expect(err).NotTo(HaveOccurred())
				Expect(defs).To(HaveLen(1))
				Expect(defs[0].IsBuiltin).To(BeTrue())
			})
		})
	})
})

var _ = Describe("InstanceCount", func() {
	var counts *component.InstanceCountStore
	var ctx, ctxOther context.Context
	var name string

	countOf := func(ctx context.Context) component.InstanceCount {
		got, err := component.ListInstanceCounts(ctx, counts, []component.ComponentDef{
			{Name: name, Version: component.DefaultComponentDefVersion},
		})
		Expect(err).NotTo(HaveOccurred())
		return got[name+":"+component.DefaultComponentDefVersion]
	}
	update := func(ctx context.Context, instanceType component.InstanceType, delta int) {
		Expect(component.UpdateInstanceCount(ctx, counts, name, instanceType, delta)).To(Succeed())
	}

	BeforeEach(func() {
		counts = testutil.NewStore(GinkgoT(), component.InstanceCountMeta)
		ctx = testutil.WithTenantOrDefault(context.Background())
		ctxOther = tenant.WithTenantID(context.Background(), "tenant-"+stringx.Random(6))
		name = "Comp-" + stringx.Random(6)
		DeferCleanup(func() {
			Expect(counts.DeleteMany(ctx, component.ByDef(name, component.DefaultComponentDefVersion),
				store.TenantAll())).Error().NotTo(HaveOccurred())
		})
	})

	It("should create the record on the first increment and accumulate", func() {
		for range 3 {
			update(ctx, component.FieldAppCompInstance, 1)
		}
		update(ctx, component.FieldWorkspaceCompInstance, 2)
		update(ctx, component.FieldAppCompInstance, -1)

		got := countOf(ctx)
		Expect(got.AppCompInstanceCount).To(Equal(int32(2)))
		Expect(got.WorkspaceCompInstanceCount).To(Equal(int32(2)))
	})

	It("should ignore a decrement without a record", func() {
		update(ctx, component.FieldAppCompInstance, -1)
		Expect(counts.Exists(ctx, component.ByDef(name, component.DefaultComponentDefVersion))).To(BeFalse())
	})

	It("should count each tenant separately", func() {
		update(ctx, component.FieldAppCompInstance, 1)
		update(ctxOther, component.FieldAppCompInstance, 5)

		Expect(countOf(ctx).AppCompInstanceCount).To(Equal(int32(1)))
		Expect(countOf(ctxOther).AppCompInstanceCount).To(Equal(int32(5)))
	})

	It("should reject an unknown instance type", func() {
		Expect(component.UpdateInstanceCount(ctx, counts, name, "unknown", 1)).
			To(MatchError(ContainSubstring("invalid instance type")))
	})
})
