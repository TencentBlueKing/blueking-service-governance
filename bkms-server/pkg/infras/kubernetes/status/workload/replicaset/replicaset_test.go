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

package replicaset

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

var _ = Describe("ReplicaSet Parse", func() {
	Context("when spec.replicas is zero and readyReplicas is zero", func() {
		It("returns Available as the desired state is reached", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"replicas": int64(0),
				},
				"status": map[string]any{
					"readyReplicas": int64(0),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Available))
		})
	})

	Context("when readyReplicas equals replicas and greater than zero", func() {
		It("returns Available", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"replicas": int64(3),
				},
				"status": map[string]any{
					"readyReplicas": int64(3),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Available))
		})
	})

	Context("when readyReplicas is less than replicas", func() {
		It("returns Progressing", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"replicas": int64(3),
				},
				"status": map[string]any{
					"readyReplicas": int64(1),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Progressing))
		})
	})

	Context("when spec.replicas field is missing", func() {
		It("treats it as zero and returns Available", func() {
			manifest := map[string]any{
				"status": map[string]any{
					"readyReplicas": int64(0),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Available))
		})
	})

	Context("when status.readyReplicas field is missing", func() {
		It("treats it as zero and returns Progressing when replicas greater than zero", func() {
			manifest := map[string]any{
				"spec": map[string]any{
					"replicas": int64(2),
				},
			}
			Expect(Parse(manifest).Code).To(Equal(k8sstatus.Progressing))
		})
	})

	Context("when manifest is nil", func() {
		It("does not panic and returns Unknown", func() {
			Expect(Parse(nil).Code).To(Equal(k8sstatus.Unknown))
		})
	})
})
