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

package annotations

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("Annotations AppModel sync", func() {
	Describe("FromAppModel", func() {
		It("returns nil for a nil app model", func() {
			Expect(FromAppModel(nil)).To(BeNil())
		})

		It("returns nil when app model has no annotations", func() {
			Expect(FromAppModel(&appmodel.AppModel{})).To(BeNil())
		})

		It("builds a spec from app model annotations", func() {
			am := &appmodel.AppModel{Annotations: map[string]string{"desc": "my-app"}}
			Expect(FromAppModel(am)).To(Equal(&Spec{Annotations: map[string]string{"desc": "my-app"}}))
		})
	})

	Describe("ApplyToAppModel", func() {
		It("clears annotations for a nil spec", func() {
			am := &appmodel.AppModel{Annotations: map[string]string{"desc": "my-app"}}
			ApplyToAppModel(nil, am)
			Expect(am.Annotations).To(BeNil())
		})

		It("fully replaces annotations", func() {
			am := &appmodel.AppModel{Annotations: map[string]string{"old": "v"}}
			ApplyToAppModel(&Spec{Annotations: map[string]string{"desc": "my-app"}}, am)
			Expect(am.Annotations).To(Equal(map[string]string{"desc": "my-app"}))
		})

		It("does not share the backing map with the spec", func() {
			spec := &Spec{Annotations: map[string]string{"desc": "my-app"}}
			am := &appmodel.AppModel{}
			ApplyToAppModel(spec, am)
			spec.Annotations["desc"] = "changed"
			Expect(am.Annotations["desc"]).To(Equal("my-app"))
		})
	})

	Describe("round-trip", func() {
		It("preserves annotations through FromAppModel and ApplyToAppModel", func() {
			am := &appmodel.AppModel{Annotations: map[string]string{"a": "1", "b": "2"}}
			spec := FromAppModel(am)

			dst := &appmodel.AppModel{}
			ApplyToAppModel(spec, dst)
			Expect(dst.Annotations).To(Equal(am.Annotations))
		})
	})
})
