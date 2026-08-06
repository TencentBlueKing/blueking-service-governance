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

package bkci

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildLog", func() {
	Describe("MaxLineNo", func() {
		It("returns the last line number because BKCI logs are ordered", func() {
			buildLog := &BuildLog{
				Logs: []LogLine{
					{LineNo: 10},
					{LineNo: 11},
					{LineNo: 12},
				},
			}

			Expect(buildLog.MaxLineNo()).To(Equal(int64(12)))
		})
	})

	Describe("IsComplete", func() {
		It("returns true only when the build is finished and there are no more logs", func() {
			Expect((&BuildLog{Finished: true, HasMore: false}).IsComplete()).To(BeTrue())
			Expect((&BuildLog{Finished: false, HasMore: false}).IsComplete()).To(BeFalse())
			Expect((&BuildLog{Finished: true, HasMore: true}).IsComplete()).To(BeFalse())
		})
	})

	Describe("ReachedCurrentTail", func() {
		It("returns true when the current batch has no more immediately available logs", func() {
			Expect((&BuildLog{HasMore: false}).ReachedCurrentTail()).To(BeTrue())
			Expect((&BuildLog{HasMore: true}).ReachedCurrentTail()).To(BeFalse())
		})
	})
})
