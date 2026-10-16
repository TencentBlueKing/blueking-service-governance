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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
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

var _ = Describe("HelmAppComponent store", func() {
	var (
		comps *Store
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		comps = testutil.NewStore(GinkgoT(), Meta)

		// 清理测试数据
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
	})

	Describe("Create", func() {
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

			Expect(comps.Create(ctx, comp)).To(Succeed())
			Expect(comp.Name).NotTo(BeEmpty())

			got, err := comps.GetByID(ctx, comp.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal(comp.Name))
			Expect(got.Properties).To(HaveKeyWithValue("image", "envoy:latest"))
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
			Expect(comps.Create(ctx, comp)).To(Succeed())

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
			Expect(comps.Create(ctx, comp2)).To(MatchError(store.ErrDuplicate))
		})
	})

	Describe("Get", func() {
		It("should return error when not found", func() {
			_, err := comps.GetByID(ctx, bson.NewObjectID())
			Expect(err).To(MatchError(ErrHelmAppComponentNotFound))
		})
	})

	Describe("ByAppAndEnv and InExecutionOrder", func() {
		It("should return components sorted by priority then createdAt", func() {
			newComp := func(name string, priority int) *HelmAppComponent {
				return &HelmAppComponent{
					AppID:   "app-003",
					EnvName: "prod",
					Component: appcomponent.Component{
						Name:          name,
						ComponentInst: appcomponent.ComponentInst{Type: name, Version: "v1.0.0"},
					},
					Target:   TargetResourceSelector{Kind: "Deployment", Name: name},
					Priority: priority,
				}
			}
			for _, c := range []*HelmAppComponent{newComp("comp-b", 10), newComp("comp-a", 1), newComp("comp-c", 10)} {
				Expect(comps.Create(ctx, c)).To(Succeed())
			}
			other := newComp("comp-x", 0)
			other.EnvName = "staging"
			Expect(comps.Create(ctx, other)).To(Succeed())

			list, err := comps.List(ctx, ByAppAndEnv("app-003", "prod"), InExecutionOrder())
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(3))
			Expect(list[0].Name).To(Equal("comp-a"))
			Expect(list[1].Name).To(Equal("comp-b"))
			Expect(list[2].Name).To(Equal("comp-c"))
		})
	})
})
