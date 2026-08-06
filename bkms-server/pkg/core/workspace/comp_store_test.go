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
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test WorkspaceCompsStoreMongo", func() {
	var ctx context.Context

	var store WorkspaceCompsStore

	BeforeEach(func() {
		var err error

		ctx = context.Background()

		store, err = NewWorkspaceCompsStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Test WorkspaceCompsStoreMongo methods", func() {
		var workspaceID1 string
		var workspaceID2 string

		BeforeEach(func() {
			workspaceID1 = stringx.Random(10)
			workspaceID2 = stringx.Random(10)
		})

		AfterEach(func() {
			// 清理测试数据
			_ = store.DeleteByWorkspace(ctx, workspaceID1)
			_ = store.DeleteByWorkspace(ctx, workspaceID2)
		})

		Context("test add component", func() {
			It("test add component successfully", func() {
				comp1 := &Component{
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
					WorkspaceID: workspaceID1,
				}
				err := store.Add(ctx, comp1)
				Expect(err).NotTo(HaveOccurred())

				// 不同类型的组件
				comp2 := &Component{
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
					WorkspaceID: workspaceID1,
					Name:        stringx.Random(10),
				}
				err = store.Add(ctx, comp2)
				Expect(err).NotTo(HaveOccurred())

				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(2))

				comp, err := store.GetByName(ctx, workspaceID1, comp2.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(comp).NotTo(BeNil())
				Expect(comp.Name).To(Equal(comp2.Name))
				Expect(comp.Type).To(Equal(comp2.Type))
				Expect(comp.Version).To(Equal(comp2.Version))

				_, err = store.GetByName(ctx, workspaceID1, "not-exist")
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrComponentNotFound)).To(BeTrue())
			})

			It("test add component with properties", func() {
				properties := map[string]any{
					"key1": "value1",
					"key2": 123,
					"key3": true,
					"key4": map[string]string{"key5": "value5"},
					"key6": float64(123.456),
				}

				comp := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:       stringx.Random(10),
						Version:    stringx.Random(10),
						Properties: properties,
					},
				}
				err := store.Add(ctx, comp)
				Expect(err).NotTo(HaveOccurred())

				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(1))
				Expect(comps[0].Properties["key1"]).To(Equal(properties["key1"]))
				Expect(comps[0].Properties["key3"]).To(BeTrue())
				// TODO 确认是否有更好方法，目前转换会导致复杂类型丢失
				Expect(comps[0].Properties["key4"]).To(Equal(map[string]any{"key5": "value5"}))
				Expect(comps[0].Properties["key6"]).To(Equal(float64(123.456)))
			})

			It("test add component with scope settings", func() {
				comp := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{"dev", "test", "prod"},
				}
				err := store.Add(ctx, comp)
				Expect(err).NotTo(HaveOccurred())

				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(1))
				Expect(comps[0].ScopeType).To(Equal(component.ScopeTypeEnvironment))
				Expect(comps[0].ScopeEnvNames).To(Equal([]string{"dev", "test", "prod"}))
			})
		})

		Context("test add components batch", func() {
			It("test add components batch successfully", func() {
				comps := []*Component{
					{
						WorkspaceID: workspaceID1,
						ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
					{
						WorkspaceID: workspaceID1,
						ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
					{
						WorkspaceID: workspaceID1,
						ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
				}
				err := store.Add(ctx, comps...)
				Expect(err).NotTo(HaveOccurred())

				result, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(3))
			})
		})

		Context("test update component", func() {
			var compID bson.ObjectID

			BeforeEach(func() {
				comp1 := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
					ScopeType:     component.ScopeTypeEnvironment,
					ScopeEnvNames: []string{"dev", "staging"},
				}
				err := store.Add(ctx, comp1)
				Expect(err).NotTo(HaveOccurred())

				comps, _ := store.ListByWorkspace(ctx, workspaceID1)
				compID = comps[0].ID
			})

			It("test update component successfully", func() {
				newVersion := stringx.Random(10)
				newName := stringx.Random(10)
				updateData := &ComponentUpdateData{
					Name:       lo.ToPtr(newName),
					Version:    &newVersion,
					Properties: map[string]any{"foo": "bar"},
				}
				err := store.Update(ctx, compID, updateData)
				Expect(err).NotTo(HaveOccurred())

				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())

				// 找到更新后的组件
				var updatedComp *Component
				for i := range comps {
					if comps[i].ID == compID {
						updatedComp = comps[i]
						break
					}
				}
				Expect(updatedComp).NotTo(BeNil())
				Expect(updatedComp.Version).To(Equal(newVersion))
				Expect(updatedComp.Properties["foo"]).To(Equal("bar"))
				Expect(updatedComp.Name).To(Equal(newName))
			})

			It("test update component failed when component not found", func() {
				updateData := &ComponentUpdateData{
					Version: lo.ToPtr(stringx.Random(10)),
				}
				err := store.Update(ctx, bson.NewObjectID(), updateData)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("component not found"))
			})

			It("test update component scope settings", func() {
				updateData := &ComponentUpdateData{
					ScopeType: lo.ToPtr(component.ScopeTypeGlobal),
				}
				err := store.Update(ctx, compID, updateData)
				Expect(err).NotTo(HaveOccurred())

				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())

				// 找到更新后的组件
				var updatedComp *Component
				for i := range comps {
					if comps[i].ID == compID {
						updatedComp = comps[i]
						break
					}
				}
				Expect(updatedComp).NotTo(BeNil())
				Expect(updatedComp.ScopeType).To(Equal(component.ScopeTypeGlobal))
				Expect(updatedComp.ScopeEnvNames).To(BeEmpty())

				updateData = &ComponentUpdateData{
					ScopeType:     lo.ToPtr(component.ScopeTypeEnvironment),
					ScopeEnvNames: []string{"dev", "staging"},
				}

				err = store.Update(ctx, compID, updateData)
				Expect(err).NotTo(HaveOccurred())

				comps, err = store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				for i := range comps {
					if comps[i].ID == compID {
						updatedComp = comps[i]
						break
					}
				}
				Expect(updatedComp.ScopeType).To(Equal(component.ScopeTypeEnvironment))
				Expect(updatedComp.ScopeEnvNames).To(Equal([]string{"dev", "staging"}))
			})
		})

		Context("test remove component", func() {
			var compID1, compID2 bson.ObjectID

			BeforeEach(func() {
				comp1 := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
				}
				err := store.Add(ctx, comp1)
				Expect(err).NotTo(HaveOccurred())

				comp2 := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
				}
				err = store.Add(ctx, comp2)
				Expect(err).NotTo(HaveOccurred())

				comps, _ := store.ListByWorkspace(ctx, workspaceID1)
				compID1 = comps[0].ID
				compID2 = comps[1].ID
			})

			It("test remove component successfully", func() {
				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(2))

				err = store.Remove(ctx, compID1)
				Expect(err).NotTo(HaveOccurred())

				comps, err = store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(1))
				Expect(comps[0].ID).To(Equal(compID2))
			})

			It("test remove component failed when component not found", func() {
				err := store.Remove(ctx, bson.NewObjectID())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("component not found"))
			})
		})

		Context("test remove components batch", func() {
			var compID1, compID2, compID3 bson.ObjectID

			BeforeEach(func() {
				comps := []*Component{
					{
						WorkspaceID: workspaceID1,
						ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
					{
						WorkspaceID: workspaceID1,
						ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
					{
						WorkspaceID: workspaceID1, ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
				}
				err := store.Add(ctx, comps...)
				Expect(err).NotTo(HaveOccurred())

				result, _ := store.ListByWorkspace(ctx, workspaceID1)
				compID1 = result[0].ID
				compID2 = result[1].ID
				compID3 = result[2].ID
			})

			It("test remove components batch successfully", func() {
				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(3))

				err = store.Remove(ctx, compID1, compID2)
				Expect(err).NotTo(HaveOccurred())

				comps, err = store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(1))
				Expect(comps[0].ID).To(Equal(compID3))
			})

			It("test remove components batch failed when all not found", func() {
				err := store.Remove(ctx, bson.NewObjectID(), bson.NewObjectID())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("component not found"))
			})
		})

		Context("test get by workspace", func() {
			It("test get by workspace returns empty when no components", func() {
				comps, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps).To(HaveLen(0))
			})

			It("test get by workspace returns components for specific workspace only", func() {
				comp1 := &Component{
					WorkspaceID: workspaceID1,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
				}
				err := store.Add(ctx, comp1)
				Expect(err).NotTo(HaveOccurred())

				comp2 := &Component{
					WorkspaceID: workspaceID2,
					ComponentInst: component.ComponentInst{
						Type:    stringx.Random(10),
						Version: stringx.Random(10),
					},
				}
				err = store.Add(ctx, comp2)
				Expect(err).NotTo(HaveOccurred())

				comps1, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps1).To(HaveLen(1))
				Expect(comps1[0].Type).To(Equal(comp1.Type))

				comps2, err := store.ListByWorkspace(ctx, workspaceID2)
				Expect(err).NotTo(HaveOccurred())
				Expect(comps2).To(HaveLen(1))
				Expect(comps2[0].Type).To(Equal(comp2.Type))
			})
		})

		Context("test delete by workspace", func() {
			It("test delete by workspace removes all components", func() {
				comps := []*Component{
					{
						WorkspaceID: workspaceID1, ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
					{
						WorkspaceID: workspaceID1, ComponentInst: component.ComponentInst{
							Type:    stringx.Random(10),
							Version: stringx.Random(10),
						},
					},
				}
				err := store.Add(ctx, comps...)
				Expect(err).NotTo(HaveOccurred())

				result, err := store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(2))

				err = store.DeleteByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())

				result, err = store.ListByWorkspace(ctx, workspaceID1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(0))
			})
		})
	})
})
