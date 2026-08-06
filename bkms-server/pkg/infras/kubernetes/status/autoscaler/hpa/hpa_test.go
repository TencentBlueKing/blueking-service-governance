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

package hpa

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// buildManifest 构造一个含有指定 conditions 列表的 HPA manifest
func buildManifest(conditions []any) map[string]any {
	return map[string]any{
		"status": map[string]any{
			"conditions": conditions,
		},
	}
}

var _ = Describe("HPA Parse", func() {
	Context("when AbleToScale and ScalingActive are both True", func() {
		It("returns Healthy", func() {
			manifest := buildManifest([]any{
				map[string]any{"type": "AbleToScale", "status": "True"},
				map[string]any{"type": "ScalingActive", "status": "True"},
				map[string]any{"type": "ScalingLimited", "status": "True"},
			})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Healthy))
		})
	})

	Context("when AbleToScale is False", func() {
		It("returns Degraded", func() {
			manifest := buildManifest([]any{
				map[string]any{"type": "AbleToScale", "status": "False"},
				map[string]any{"type": "ScalingActive", "status": "True"},
			})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when ScalingActive is False", func() {
		It("returns Degraded", func() {
			manifest := buildManifest([]any{
				map[string]any{"type": "AbleToScale", "status": "True"},
				map[string]any{"type": "ScalingActive", "status": "False"},
			})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when only ScalingLimited condition exists", func() {
		It("returns Unknown because key conditions are missing", func() {
			manifest := buildManifest([]any{
				map[string]any{"type": "ScalingLimited", "status": "True"},
			})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when conditions slice is empty", func() {
		It("returns Unknown", func() {
			manifest := buildManifest([]any{})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when status.conditions field is missing", func() {
		It("returns Unknown", func() {
			Expect(Parse(map[string]any{}).Code).To(Equal(k8sstatus.Unknown))
		})
	})

	Context("when a key condition has status Unknown", func() {
		It("treats it as non-True and returns Degraded", func() {
			manifest := buildManifest([]any{
				map[string]any{"type": "AbleToScale", "status": "Unknown"},
				map[string]any{"type": "ScalingActive", "status": "True"},
			})
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Degraded))
		})
	})

	Context("when manifest is nil", func() {
		It("does not panic and returns Unknown", func() {
			Expect(Parse(nil).Code).To(Equal(k8sstatus.Unknown))
		})
	})
})
