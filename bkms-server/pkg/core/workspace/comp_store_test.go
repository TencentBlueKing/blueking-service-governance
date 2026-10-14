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

package workspace

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Workspace component store", func() {
	var (
		ctx          context.Context
		db           *mongo.Database
		comps        *WorkspaceCompsStore
		defs         *component.ComponentDefStore
		workspaceID1 string
		workspaceID2 string
		// compType 对应一个已注册的组件定义，用于观察引用计数
		compType string
	)

	newComp := func(workspaceID, typ string) *Component {
		return &Component{
			WorkspaceID: workspaceID,
			ComponentInst: component.ComponentInst{
				Type:    typ,
				Version: stringx.Random(10),
			},
		}
	}

	instanceCount := func() int32 {
		def, err := defs.Get(ctx, store.Where("name", compType),
			store.Where("version", component.DefaultComponentDefVersion))
		Expect(err).NotTo(HaveOccurred())
		return def.WorkspaceCompInstanceCount
	}

	listOf := func(workspaceID string) []Component {
		result, err := comps.List(ctx, ComponentsIn(workspaceID))
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		comps = store.Of(db, ComponentMeta)
		defs = testutil.NewStore(GinkgoT(), component.ComponentDefMeta)
		workspaceID1 = stringx.Random(10)
		workspaceID2 = stringx.Random(10)

		compType = "ws-comp-" + stringx.Random(8)
		Expect(defs.Create(ctx, &component.ComponentDef{
			Name:     compType,
			Version:  component.DefaultComponentDefVersion,
			Patchers: []string{"metadata:\n  labels:\n    foo: bar\n"},
			Creator:  "admin",
			Updater:  "admin",
		})).To(Succeed())
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(ComponentMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(component.ComponentDefMeta)).To(Succeed())
	})

	Context("AddComponents", func() {
		It("adds components, fills defaults and increments the def ref count", func() {
			comp1 := newComp(workspaceID1, compType)
			comp2 := newComp(workspaceID1, compType)
			comp2.Name = stringx.Random(10)
			Expect(AddComponents(ctx, db, comp1, comp2)).To(Succeed())

			// 未指定名称时自动生成，Properties 初始化为空 map
			Expect(comp1.Name).NotTo(BeEmpty())
			Expect(comp1.Properties).NotTo(BeNil())
			Expect(listOf(workspaceID1)).To(HaveLen(2))
			Expect(instanceCount()).To(Equal(int32(2)))

			got, err := comps.Get(ctx, ComponentByName(workspaceID1, comp2.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Type).To(Equal(comp2.Type))
			Expect(got.Version).To(Equal(comp2.Version))

			_, err = comps.Get(ctx, ComponentByName(workspaceID1, "not-exist"))
			Expect(err).To(MatchError(ErrComponentNotFound))
		})

		It("round-trips properties with normalized BSON types", func() {
			properties := map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
				"key4": map[string]string{"key5": "value5"},
				"key6": float64(123.456),
			}
			comp := newComp(workspaceID1, compType)
			comp.Properties = properties
			Expect(AddComponents(ctx, db, comp)).To(Succeed())

			result := listOf(workspaceID1)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Properties["key1"]).To(Equal(properties["key1"]))
			Expect(result[0].Properties["key3"]).To(BeTrue())
			// TODO 确认是否有更好方法，目前转换会导致复杂类型丢失
			Expect(result[0].Properties["key4"]).To(Equal(map[string]any{"key5": "value5"}))
			Expect(result[0].Properties["key6"]).To(Equal(float64(123.456)))
		})

		It("persists scope settings", func() {
			comp := newComp(workspaceID1, compType)
			comp.ScopeType = component.ScopeTypeEnvironment
			comp.ScopeEnvNames = []string{"dev", "test", "prod"}
			Expect(AddComponents(ctx, db, comp)).To(Succeed())

			result := listOf(workspaceID1)
			Expect(result).To(HaveLen(1))
			Expect(result[0].ScopeType).To(Equal(component.ScopeTypeEnvironment))
			Expect(result[0].ScopeEnvNames).To(Equal([]string{"dev", "test", "prod"}))
		})

		It("rejects a duplicate name in the same workspace", func() {
			comp := newComp(workspaceID1, compType)
			comp.Name = "dup"
			Expect(AddComponents(ctx, db, comp)).To(Succeed())

			dup := newComp(workspaceID1, compType)
			dup.Name = "dup"
			Expect(AddComponents(ctx, db, dup)).To(MatchError(ErrComponentAlreadyExists))
			Expect(instanceCount()).To(Equal(int32(1)))

			// 不同工作空间允许同名
			other := newComp(workspaceID2, compType)
			other.Name = "dup"
			Expect(AddComponents(ctx, db, other)).To(Succeed())
		})

		It("ignores components whose def does not exist", func() {
			Expect(AddComponents(ctx, db, newComp(workspaceID1, "unknown-"+stringx.Random(6)))).To(Succeed())
			Expect(listOf(workspaceID1)).To(HaveLen(1))
			Expect(instanceCount()).To(BeZero())
		})

		It("is a no-op without components", func() {
			Expect(AddComponents(ctx, db)).To(Succeed())
		})
	})

	Context("UpdateComponent", func() {
		var compID bson.ObjectID

		BeforeEach(func() {
			comp := newComp(workspaceID1, compType)
			comp.ScopeType = component.ScopeTypeEnvironment
			comp.ScopeEnvNames = []string{"dev", "staging"}
			Expect(AddComponents(ctx, db, comp)).To(Succeed())
			compID = comp.ID
		})

		It("updates version, properties and name", func() {
			newVersion := stringx.Random(10)
			newName := stringx.Random(10)
			Expect(UpdateComponent(ctx, db, compID, &ComponentUpdateData{
				Name:       lo.ToPtr(newName),
				Version:    &newVersion,
				Properties: map[string]any{"foo": "bar"},
			})).To(Succeed())

			updated, err := comps.Get(ctx, store.ByID(compID))
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Version).To(Equal(newVersion))
			Expect(updated.Properties["foo"]).To(Equal("bar"))
			Expect(updated.Name).To(Equal(newName))
		})

		It("clears scope env names when switching to global scope", func() {
			Expect(UpdateComponent(ctx, db, compID, &ComponentUpdateData{
				ScopeType: lo.ToPtr(component.ScopeTypeGlobal),
			})).To(Succeed())

			updated, err := comps.Get(ctx, store.ByID(compID))
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.ScopeType).To(Equal(component.ScopeTypeGlobal))
			Expect(updated.ScopeEnvNames).To(BeEmpty())

			Expect(UpdateComponent(ctx, db, compID, &ComponentUpdateData{
				ScopeType:     lo.ToPtr(component.ScopeTypeEnvironment),
				ScopeEnvNames: []string{"dev", "staging"},
			})).To(Succeed())

			updated, err = comps.Get(ctx, store.ByID(compID))
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.ScopeType).To(Equal(component.ScopeTypeEnvironment))
			Expect(updated.ScopeEnvNames).To(Equal([]string{"dev", "staging"}))
		})

		It("rejects an invalid scope type", func() {
			err := UpdateComponent(ctx, db, compID, &ComponentUpdateData{
				ScopeType: lo.ToPtr(component.ScopeType("invalid")),
			})
			Expect(err).To(MatchError(ContainSubstring("invalid scope type")))
		})
	})

	Context("RemoveComponents", func() {
		var compID1, compID2, compID3 bson.ObjectID

		BeforeEach(func() {
			added := []*Component{
				newComp(workspaceID1, compType),
				newComp(workspaceID1, compType),
				newComp(workspaceID1, compType),
			}
			Expect(AddComponents(ctx, db, added...)).To(Succeed())
			compID1, compID2, compID3 = added[0].ID, added[1].ID, added[2].ID
			Expect(instanceCount()).To(Equal(int32(3)))
		})

		It("removes a single component and decrements the def ref count", func() {
			Expect(RemoveComponents(ctx, db, compID1)).To(Succeed())

			result := listOf(workspaceID1)
			Expect(result).To(HaveLen(2))
			Expect(lo.Map(result, func(c Component, _ int) bson.ObjectID { return c.ID })).
				To(ConsistOf(compID2, compID3))
			Expect(instanceCount()).To(Equal(int32(2)))
		})

		It("removes components in batch", func() {
			Expect(RemoveComponents(ctx, db, compID1, compID2)).To(Succeed())

			result := listOf(workspaceID1)
			Expect(result).To(HaveLen(1))
			Expect(result[0].ID).To(Equal(compID3))
			Expect(instanceCount()).To(Equal(int32(1)))
		})

		It("returns ErrComponentNotFound and keeps the ref count when nothing matches", func() {
			err := RemoveComponents(ctx, db, bson.NewObjectID(), bson.NewObjectID())
			Expect(err).To(MatchError(ErrComponentNotFound))
			Expect(instanceCount()).To(Equal(int32(3)))
		})
	})

	Context("DeleteComponentsOf", func() {
		It("removes all components of the workspace and decrements the def ref count", func() {
			Expect(AddComponents(ctx, db,
				newComp(workspaceID1, compType),
				newComp(workspaceID1, compType),
				newComp(workspaceID2, compType),
			)).To(Succeed())
			Expect(instanceCount()).To(Equal(int32(3)))

			Expect(DeleteComponentsOf(ctx, db, workspaceID1)).To(Succeed())

			Expect(listOf(workspaceID1)).To(BeEmpty())
			Expect(listOf(workspaceID2)).To(HaveLen(1))
			Expect(instanceCount()).To(Equal(int32(1)))
		})

		It("succeeds when the workspace has no components", func() {
			Expect(DeleteComponentsOf(ctx, db, workspaceID1)).To(Succeed())
		})
	})
})
