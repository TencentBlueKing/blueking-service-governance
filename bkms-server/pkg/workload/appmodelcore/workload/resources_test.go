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

package workload

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("buildResourceRequirements", func() {
	It("should return nil when resources is empty", func() {
		reqs, err := buildResourceRequirements(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(reqs).To(BeNil())
	})

	It("should parse requests and limits", func() {
		reqs, err := buildResourceRequirements(map[string]string{
			"cpu":    "100m-200m",
			"memory": "256Mi-512Mi",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reqs).NotTo(BeNil())
		Expect(lo.ToPtr(reqs.Requests[corev1.ResourceCPU]).Cmp(resource.MustParse("100m"))).To(BeZero())
		Expect(lo.ToPtr(reqs.Limits[corev1.ResourceCPU]).Cmp(resource.MustParse("200m"))).To(BeZero())
		Expect(lo.ToPtr(reqs.Requests[corev1.ResourceMemory]).Cmp(resource.MustParse("256Mi"))).To(BeZero())
		Expect(lo.ToPtr(reqs.Limits[corev1.ResourceMemory]).Cmp(resource.MustParse("512Mi"))).To(BeZero())
	})

	It("should use the same value when no separator is used", func() {
		reqs, err := buildResourceRequirements(map[string]string{
			"cpu": "250m",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(reqs).NotTo(BeNil())
		Expect(lo.ToPtr(reqs.Requests[corev1.ResourceCPU]).Cmp(resource.MustParse("250m"))).To(BeZero())
		Expect(lo.ToPtr(reqs.Limits[corev1.ResourceCPU]).Cmp(resource.MustParse("250m"))).To(BeZero())
	})

	It("should reject invalid values", func() {
		_, err := buildResourceRequirements(map[string]string{
			"cpu": "200m-100m",
		})
		Expect(err).To(HaveOccurred())
	})
})
