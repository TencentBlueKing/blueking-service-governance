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

package timex_test

import (
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/timex"
)

var _ = Describe("Timex", func() {
	Describe("CalcDuration", func() {
		Context("when using standard datetime format", func() {
			It("should calculate duration for 1 second difference", func() {
				Expect(timex.CalcDuration("2022-01-01 12:00:00", "2022-01-01 12:00:01")).To(Equal("1s"))
			})

			It("should calculate duration for minutes and seconds", func() {
				Expect(timex.CalcDuration("2022-01-01 12:35:30", "2022-01-01 12:59:59")).To(Equal("24m29s"))
			})

			It("should calculate duration for hours and minutes", func() {
				Expect(timex.CalcDuration("2022-01-01 12:35:30", "2022-01-01 14:00:00")).To(Equal("1h24m"))
			})

			It("should calculate duration for days and hours", func() {
				Expect(timex.CalcDuration("2022-01-01 12:35:30", "2022-01-03 14:00:00")).To(Equal("2d1h"))
			})

			It("should calculate duration for 153 days", func() {
				Expect(timex.CalcDuration("2021-08-01 11:00:00", "2022-01-01 14:00:00")).To(Equal("153d3h"))
			})

			It("should calculate duration for 275 days", func() {
				Expect(timex.CalcDuration("2021-04-01 11:00:00", "2022-01-01 14:00:00")).To(Equal("275d3h"))
			})

			It("should calculate duration for 640 days", func() {
				Expect(timex.CalcDuration("2020-04-01 11:00:00", "2022-01-01 14:00:00")).To(Equal("640d3h"))
			})
		})

		Context("when using k8s manifest datetime format", func() {
			It("should calculate duration for 1 second difference", func() {
				Expect(timex.CalcDuration("2022-01-01T14:00:00Z", "2022-01-01T14:00:01Z")).To(Equal("1s"))
			})

			It("should calculate duration for minutes and seconds", func() {
				Expect(timex.CalcDuration("2022-01-01T14:45:30Z", "2022-01-01T14:59:59Z")).To(Equal("14m29s"))
			})

			It("should calculate duration for 275 days", func() {
				Expect(timex.CalcDuration("2021-04-01T11:00:00Z", "2022-01-01T14:00:00Z")).To(Equal("275d3h"))
			})
		})
	})

	Describe("CalcAge", func() {
		Context("when calculating age from a past date", func() {
			It("should return age greater than 1000 days for date from 2019", func() {
				age := timex.CalcAge("2019-01-01 11:00:00")
				dayCnt, _ := strconv.Atoi(strings.Split(age, "d")[0])
				Expect(dayCnt).To(BeNumerically(">", 1000))
			})
		})
	})

	Describe("NormalizeDatetime", func() {
		Context("when normalizing valid datetime formats", func() {
			It("should normalize k8s manifest format to standard format", func() {
				ret, err := timex.NormalizeDatetime("2022-01-01T12:00:00Z")
				Expect(err).ToNot(HaveOccurred())
				Expect(ret).To(Equal("2022-01-01 12:00:00"))
			})

			It("should keep standard format unchanged", func() {
				ret, err := timex.NormalizeDatetime("2022-01-02 14:00:00")
				Expect(err).ToNot(HaveOccurred())
				Expect(ret).To(Equal("2022-01-02 14:00:00"))
			})
		})

		Context("when normalizing invalid datetime format", func() {
			It("should return error for unsupported format", func() {
				_, err := timex.NormalizeDatetime("3/1/2021 12:00:00")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Current", func() {
		Context("when getting current time", func() {
			It("should return non-empty string", func() {
				ret := timex.Current()
				Expect(ret).ToNot(BeEmpty())
			})
		})
	})
})
