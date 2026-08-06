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

package postrenderer

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	helmcomp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	wlbscpcfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

type fakeComponentEvaluator struct {
	envIDs []bson.ObjectID
	comps  []component.Component
}

func (g *fakeComponentEvaluator) Evaluate(
	_ context.Context,
	_ *bkmsapp.Application,
	comp component.Component,
	envID bson.ObjectID,
	_ map[string]string,
	_ *envvarrefs.Collector,
) (*component.EvaluatedComponent, error) {
	g.envIDs = append(g.envIDs, envID)
	g.comps = append(g.comps, comp)
	return &component.EvaluatedComponent{Patchers: []map[string]any{{
		"metadata": map[string]any{
			"annotations": map[string]any{"component-type": comp.Type},
		},
	}}}, nil
}

var _ = Describe("buildComponentItems", func() {
	It("should render helm components with the current environment id", func() {
		envID := bson.NewObjectID()
		getter := &fakeComponentEvaluator{}
		comps := []*helmcomp.HelmAppComponent{
			{
				Component: component.Component{
					Name: "direct",
					ComponentInst: component.ComponentInst{
						Type:    "DirectComp",
						Version: "v1.0.0",
					},
				},
				Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
			},
		}

		items, err := buildComponentPatches(
			context.Background(),
			&bkmsapp.Application{ID: "app-1", WorkspaceID: "ws-1"},
			&envmodel.Environment{ID: envID, Name: "prod"},
			comps,
			nil,
			getter,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(items).To(HaveLen(1))
		Expect(getter.envIDs).To(Equal([]bson.ObjectID{envID}))
		Expect(items[0].Target).To(Equal(helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"}))
	})

	It("should resolve workspace component references before rendering", func() {
		envID := bson.NewObjectID()
		getter := &fakeComponentEvaluator{}
		comps := []*helmcomp.HelmAppComponent{
			{
				Component: component.Component{
					Name: "ref",
					ComponentRef: component.ComponentRef{
						RefWorkspaceCompName: "shared",
					},
				},
				Target: helmcomp.TargetResourceSelector{Kind: "Deployment", Name: "web"},
			},
		}
		workspaceComps := []*workspace.Component{
			{
				Name:      "shared",
				ScopeType: component.ScopeTypeGlobal,
				ComponentInst: component.ComponentInst{
					Type:       "SharedComp",
					Version:    "v1.0.0",
					Properties: map[string]any{"k": "v"},
				},
			},
		}

		items, err := buildComponentPatches(
			context.Background(),
			&bkmsapp.Application{ID: "app-1", WorkspaceID: "ws-1"},
			&envmodel.Environment{ID: envID, Name: "prod"},
			comps,
			workspaceComps,
			getter,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(items).To(HaveLen(1))
		Expect(getter.comps).To(HaveLen(1))
		Expect(getter.comps[0].Name).To(Equal("shared"))
		Expect(getter.comps[0].Type).To(Equal("SharedComp"))
		Expect(getter.comps[0].Properties).To(Equal(map[string]any{"k": "v"}))
	})

	It("should return nil when helm app components are empty", func() {
		envID := bson.NewObjectID()
		getter := &fakeComponentEvaluator{}

		items, err := buildComponentPatches(
			context.Background(),
			&bkmsapp.Application{ID: "app-1", WorkspaceID: "ws-1"},
			&envmodel.Environment{ID: envID, Name: "prod"},
			nil,
			nil,
			getter,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(items).To(BeNil())
		Expect(getter.comps).To(BeEmpty())
	})
})

var _ = Describe("Build chain ordering with BSCP", func() {
	It("should place BSCP renderer between component and lane renderers in the chain", func() {
		// 构造各个 PostRenderer
		compRenderer := NewComponentPostRenderer([]ComponentPatch{
			{Name: "test-comp", Patchers: []map[string]any{{"metadata": map[string]any{}}}},
		})
		bscpFragment := wlbscpcfg.Build(wlbscpcfg.Params{
			BscpBizID: "100",
			AppNames:  "svc-a",
			MountPath: "/cfg",
			FeedAddr:  "feed:9510",
			Token:     "tok",
		})
		bscpFragment.WorkloadName = "my-workload"
		bscpRenderer := NewBscpPostRenderer(bscpFragment)
		laneRenderer := NewLanePostRenderer(map[string]string{"lane": "v1"})

		// 构建链
		chain := NewChainPostRenderer(compRenderer, bscpRenderer, laneRenderer)

		Expect(chain).NotTo(BeNil())
		Expect(chain.renderers).To(HaveLen(3))
	})

	It("should include typed nil BSCP renderer in chain but Run handles it gracefully", func() {
		bscpRenderer := NewBscpPostRenderer(nil)
		laneRenderer := NewLanePostRenderer(map[string]string{"lane": "v1"})
		Expect(bscpRenderer).To(BeNil())

		chain := NewChainPostRenderer(bscpRenderer, laneRenderer)
		Expect(chain).NotTo(BeNil())
	})

	It("should return nil chain when no renderers are configured", func() {
		// 所有 renderer 都是 untyped nil
		chain := NewChainPostRenderer(nil, nil, nil)

		Expect(chain).To(BeNil())
	})
})
