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

package helm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

var _ = Describe("Status", func() {
	Describe("IsStable", func() {
		Context("with stable native statuses", func() {
			DescribeTable("should return true",
				func(status helmrelease.Status) {
					Expect(IsStable(status)).To(BeTrue())
				},
				Entry("deployed", StatusDeployed),
				Entry("uninstalled", StatusUninstalled),
				Entry("superseded", StatusSuperseded),
				Entry("failed", StatusFailed),
			)
		})

		Context("with stable custom statuses", func() {
			DescribeTable("should return true",
				func(status helmrelease.Status) {
					Expect(IsStable(status)).To(BeTrue())
				},
				Entry("polling-timeout", StatusPollingTimeout),
				Entry("polling-broken", StatusPollingBroken),
			)
		})

		Context("with non-stable statuses", func() {
			DescribeTable("should return false",
				func(status helmrelease.Status) {
					Expect(IsStable(status)).To(BeFalse())
				},
				Entry("uninstalling", StatusUninstalling),
				Entry("pending-install", StatusPendingInstall),
				Entry("pending-upgrade", StatusPendingUpgrade),
				Entry("pending-rollback", StatusPendingRollback),
				Entry("unknown", StatusUnknown),
			)
		})
	})
})
