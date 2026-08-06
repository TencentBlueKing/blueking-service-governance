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

package serializer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/serializer"
)

var _ = Describe("Resources serializer", func() {
	It("should convert nil model to nil output", func() {
		Expect(new(serializer.AppSpecResourcesOutput).FromModel(nil)).To(BeNil())
	})

	It("should preserve pointer fields when converting model to output", func() {
		replicas := int32(2)
		cpuRequests := "1"
		cpuLimits := "2"
		spec := &appspec.ResourcesSpec{
			Replicas:    &replicas,
			CPURequests: &cpuRequests,
			CPULimits:   &cpuLimits,
		}

		output := new(serializer.AppSpecResourcesOutput).FromModel(spec)

		Expect(output.Replicas).NotTo(BeNil())
		Expect(*output.Replicas).To(Equal(int32(2)))
		Expect(output.CPURequests).NotTo(BeNil())
		Expect(*output.CPURequests).To(Equal("1"))
		Expect(output.CPULimits).NotTo(BeNil())
		Expect(*output.CPULimits).To(Equal("2"))
		Expect(output.MemoryRequests).To(BeNil())
		Expect(output.MemoryLimits).To(BeNil())
	})

	It("should convert default input to model with explicit fields", func() {
		input := &serializer.AppSpecResourcesInput{
			Replicas:       1,
			CPURequests:    "500m",
			CPULimits:      "1",
			MemoryRequests: "256Mi",
			MemoryLimits:   "512Mi",
		}

		spec := input.ToModel()

		Expect(spec.Replicas).NotTo(BeNil())
		Expect(*spec.Replicas).To(Equal(int32(1)))
		Expect(spec.CPURequests).NotTo(BeNil())
		Expect(*spec.CPURequests).To(Equal("500m"))
		Expect(spec.CPULimits).NotTo(BeNil())
		Expect(*spec.CPULimits).To(Equal("1"))
		Expect(spec.MemoryRequests).NotTo(BeNil())
		Expect(*spec.MemoryRequests).To(Equal("256Mi"))
		Expect(spec.MemoryLimits).NotTo(BeNil())
		Expect(*spec.MemoryLimits).To(Equal("512Mi"))
	})
})
