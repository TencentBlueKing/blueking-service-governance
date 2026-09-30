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

package releasestrategy_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

var _ = Describe("ParseBatchTarget", func() {
	DescribeTable("accepts positive integers and percentages",
		func(target string, expected intstr.IntOrString) {
			parsed, err := releasestrategy.ParseBatchTarget(target)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed).To(Equal(expected))
		},
		Entry("single instance", "1", intstr.FromInt32(1)),
		Entry("multiple instances", "120", intstr.FromInt32(120)),
		Entry("lower percentage", "1%", intstr.FromString("1%")),
		Entry("two digit percentage", "25%", intstr.FromString("25%")),
		Entry("full percentage", "100%", intstr.FromString("100%")),
	)

	DescribeTable("rejects invalid targets",
		func(target string) {
			_, err := releasestrategy.ParseBatchTarget(target)
			Expect(errors.Is(err, releasestrategy.ErrInvalidBatchTarget)).To(BeTrue())
		},
		Entry("empty", ""),
		Entry("zero", "0"),
		Entry("negative", "-1"),
		Entry("leading zero", "01"),
		Entry("explicit plus sign", "+1"),
		Entry("decimal", "1.5"),
		Entry("zero percentage", "0%"),
		Entry("percentage over 100", "101%"),
		Entry("decimal percentage", "12.5%"),
		Entry("overflow", "99999999999"),
		Entry("not a number", "abc"),
	)
})
