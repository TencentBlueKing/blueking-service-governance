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

package helmcomponent

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	appcomponent "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

func TestHelmComponent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helm Component Suite")
}

var _ = BeforeSuite(func() {
	if err := testutil.SetUpGlobalDatabase(); err != nil {
		panic("failed to set up global database: " + err.Error())
	}
})

var _ = AfterSuite(func() {
	if err := testutil.TeardownGlobalDatabase(); err != nil {
		panic("failed to teardown global database: " + err.Error())
	}
})

var _ = Describe("DbHelmAppComponentStore", func() {
	var (
		store *DbHelmAppComponentStore
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = NewDbHelmAppComponentStore(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		// 清理测试数据
		err = testutil.CleanupCollection(helmAppComponentCollectionName)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Add", func() {
		It("should insert a component and auto-generate name", func() {
			comp := &HelmAppComponent{
				AppID:   "app-001",
				EnvName: "prod",
				Component: appcomponent.Component{
					ComponentInst: appcomponent.ComponentInst{
						Type:       "sidecar-injector",
						Version:    "v1.0.0",
						Properties: map[string]any{"image": "envoy:latest"},
					},
				},
				Target: TargetResourceSelector{
					Kind: "Deployment",
					Name: "my-nginx",
				},
				Priority: 0,
			}

			err := store.Add(ctx, comp)
			Expect(err).NotTo(HaveOccurred())
			Expect(comp.ID).NotTo(BeZero())
			Expect(comp.Name).NotTo(BeEmpty())
			Expect(comp.CreatedAt).NotTo(BeZero())
			Expect(comp.UpdatedAt).NotTo(BeZero())
		})

		It("should reject duplicate name in same app+env", func() {
			comp := &HelmAppComponent{
				AppID:   "app-001",
				EnvName: "prod",
				Component: appcomponent.Component{
					Name: "fixed-name",
					ComponentInst: appcomponent.ComponentInst{
						Type:    "sidecar-injector",
						Version: "v1.0.0",
					},
				},
				Target: TargetResourceSelector{Kind: "Deployment", Name: "nginx"},
			}
			err := store.Add(ctx, comp)
			Expect(err).NotTo(HaveOccurred())

			// 同名组件应该失败
			comp2 := &HelmAppComponent{
				AppID:   "app-001",
				EnvName: "prod",
				Component: appcomponent.Component{
					Name: "fixed-name",
					ComponentInst: appcomponent.ComponentInst{
						Type:    "another-comp",
						Version: "v1.0.0",
					},
				},
				Target: TargetResourceSelector{Kind: "Deployment", Name: "other"},
			}
			err = store.Add(ctx, comp2)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Get", func() {
		It("should return the component by ID", func() {
			comp := &HelmAppComponent{
				AppID:   "app-002",
				EnvName: "staging",
				Component: appcomponent.Component{
					Name: "my-comp",
					ComponentInst: appcomponent.ComponentInst{
						Type:    "config-injector",
						Version: "v1.0.0",
					},
				},
				Target:   TargetResourceSelector{Kind: "Deployment", Name: "web"},
				Priority: 1,
			}
			err := store.Add(ctx, comp)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, comp.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AppID).To(Equal("app-002"))
			Expect(got.Name).To(Equal("my-comp"))
			Expect(got.Target.Kind).To(Equal("Deployment"))
		})

		It("should return error when not found", func() {
			_, err := store.Get(ctx, bson.NewObjectID())
			Expect(err).To(MatchError(ErrHelmAppComponentNotFound))
		})
	})

	Describe("ListByAppAndEnv", func() {
		It("should return components sorted by priority then createdAt", func() {
			comps := []*HelmAppComponent{
				{
					AppID:   "app-003",
					EnvName: "prod",
					Component: appcomponent.Component{
						Name:          "comp-b",
						ComponentInst: appcomponent.ComponentInst{Type: "b", Version: "v1.0.0"},
					},
					Target:   TargetResourceSelector{Kind: "Deployment", Name: "b"},
					Priority: 10,
				},
				{
					AppID:   "app-003",
					EnvName: "prod",
					Component: appcomponent.Component{
						Name:          "comp-a",
						ComponentInst: appcomponent.ComponentInst{Type: "a", Version: "v1.0.0"},
					},
					Target:   TargetResourceSelector{Kind: "Deployment", Name: "a"},
					Priority: 1,
				},
			}
			for _, c := range comps {
				Expect(store.Add(ctx, c)).To(Succeed())
			}

			list, err := store.ListByAppAndEnv(ctx, "app-003", "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(2))
			Expect(list[0].Name).To(Equal("comp-a"))
			Expect(list[1].Name).To(Equal("comp-b"))
		})

		It("should return empty list when no components exist", func() {
			list, err := store.ListByAppAndEnv(ctx, "non-existent", "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})
	})

	Describe("Update", func() {
		It("should update properties and target", func() {
			comp := &HelmAppComponent{
				AppID:   "app-004",
				EnvName: "prod",
				Component: appcomponent.Component{
					Name:          "updatable",
					ComponentInst: appcomponent.ComponentInst{Type: "x", Version: "v1.0.0"},
				},
				Target: TargetResourceSelector{Kind: "Deployment", Name: "old"},
			}
			Expect(store.Add(ctx, comp)).To(Succeed())

			newPriority := 5
			err := store.Update(ctx, comp.ID, &UpdateData{
				Properties: map[string]any{"key": "value"},
				Target:     &TargetResourceSelector{Kind: "StatefulSet", Name: "new"},
				Priority:   &newPriority,
			})
			Expect(err).NotTo(HaveOccurred())

			got, err := store.Get(ctx, comp.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Target.Kind).To(Equal("StatefulSet"))
			Expect(got.Target.Name).To(Equal("new"))
			Expect(got.Priority).To(Equal(5))
		})

		It("should return error when component not found", func() {
			err := store.Update(ctx, bson.NewObjectID(), &UpdateData{})
			Expect(err).To(MatchError(ErrHelmAppComponentNotFound))
		})
	})

	Describe("Delete", func() {
		It("should delete the component", func() {
			comp := &HelmAppComponent{
				AppID:   "app-005",
				EnvName: "prod",
				Component: appcomponent.Component{
					Name:          "deletable",
					ComponentInst: appcomponent.ComponentInst{Type: "y", Version: "v1.0.0"},
				},
				Target: TargetResourceSelector{Kind: "Deployment", Name: "del"},
			}
			Expect(store.Add(ctx, comp)).To(Succeed())

			err := store.Delete(ctx, comp.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = store.Get(ctx, comp.ID)
			Expect(err).To(MatchError(ErrHelmAppComponentNotFound))
		})

		It("should be idempotent (no error when not found)", func() {
			err := store.Delete(ctx, bson.NewObjectID())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DeleteByApp", func() {
		It("should delete all components for the app", func() {
			for i := range 3 {
				comp := &HelmAppComponent{
					AppID:   "app-006",
					EnvName: "prod",
					Component: appcomponent.Component{
						ComponentInst: appcomponent.ComponentInst{
							Type:    "comp-" + string(rune('a'+i)),
							Version: "v1.0.0",
						},
					},
					Target: TargetResourceSelector{Kind: "Deployment", Name: "d"},
				}
				Expect(store.Add(ctx, comp)).To(Succeed())
			}

			err := store.DeleteByApp(ctx, "app-006")
			Expect(err).NotTo(HaveOccurred())

			list, err := store.ListByAppAndEnv(ctx, "app-006", "prod")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})
	})
})
