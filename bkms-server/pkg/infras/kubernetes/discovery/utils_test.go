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

package discovery

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// filterResByKind 的纯单元测试，不依赖真实集群，重点验证 ErrKindNotFound sentinel 归一。
var _ = Describe("filterResByKind", func() {
	Context("when the kind exists in the resource list", func() {
		It("should resolve the group/version/resource", func() {
			all := []*metav1.APIResourceList{
				{
					GroupVersion: "autoscaling.tkex.tencent.com/v1alpha1",
					APIResources: []metav1.APIResource{
						{Kind: "GeneralPodAutoscaler", Name: "generalpodautoscalers"},
					},
				},
			}
			gvr, err := filterResByKind("GeneralPodAutoscaler", all)
			Expect(err).NotTo(HaveOccurred())
			Expect(gvr.Group).To(Equal("autoscaling.tkex.tencent.com"))
			Expect(gvr.Version).To(Equal("v1alpha1"))
			Expect(gvr.Resource).To(Equal("generalpodautoscalers"))
		})
	})

	Context("when the kind is missing", func() {
		It("should return an error that is ErrKindNotFound", func() {
			all := []*metav1.APIResourceList{
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{Kind: "Deployment", Name: "deployments"},
					},
				},
			}
			_, err := filterResByKind("GeneralPodAutoscaler", all)
			Expect(err).To(HaveOccurred())
			// 调用方据此 sentinel 精确判定资源类型不存在，而非依赖脆弱的文本匹配
			Expect(errors.Is(err, ErrKindNotFound)).To(BeTrue())
			// 仍保留 kind 信息便于排障
			Expect(err.Error()).To(ContainSubstring("GeneralPodAutoscaler"))
		})
	})
})
