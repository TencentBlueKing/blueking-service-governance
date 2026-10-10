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

package resourcevalue_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/resourcevalue"
)

func TestResourceValue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Value Suite")
}

var _ = Describe("ParseResourceBounds", func() {
	It("splits requests and limits and keeps the original text", func() {
		bounds, err := resourcevalue.ParseResourceBounds("100m-200m")
		Expect(err).NotTo(HaveOccurred())
		Expect(bounds.Request).To(Equal("100m"))
		Expect(bounds.Limit).To(Equal("200m"))
		expectSameQuantity(bounds.RequestQuantity, "100m")
		expectSameQuantity(bounds.LimitQuantity, "200m")

		bounds, err = resourcevalue.ParseResourceBounds("512Mi-2048Mi")
		Expect(err).NotTo(HaveOccurred())
		Expect(bounds.Request).To(Equal("512Mi"))
		Expect(bounds.Limit).To(Equal("2048Mi"))
		expectSameQuantity(bounds.RequestQuantity, "512Mi")
		expectSameQuantity(bounds.LimitQuantity, "2048Mi")
	})

	It("uses one value as both request and limit", func() {
		bounds, err := resourcevalue.ParseResourceBounds("0.2")
		Expect(err).NotTo(HaveOccurred())
		Expect(bounds.Request).To(Equal("0.2"))
		Expect(bounds.Limit).To(Equal("0.2"))
		expectSameQuantity(bounds.RequestQuantity, "0.2")
		expectSameQuantity(bounds.LimitQuantity, "0.2")
	})

	DescribeTable("rejects values that are not a single request-limit pair",
		func(raw string) {
			_, err := resourcevalue.ParseResourceBounds(raw)
			Expect(err).To(HaveOccurred())
		},
		Entry("request greater than limit", "200m-100m"),
		Entry("more than one separator", "100m-200m-300m"),
		Entry("invalid request", "bad-200m"),
		Entry("invalid limit", "100m-bad"),
		Entry("empty limit", "100m-"),
		Entry("empty value", ""),
	)
})

func expectSameQuantity(got resource.Quantity, raw string) {
	want := resource.MustParse(raw)
	Expect(got.Cmp(want)).To(Equal(0))
}
