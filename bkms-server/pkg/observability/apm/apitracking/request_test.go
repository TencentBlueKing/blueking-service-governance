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

package apitracking

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("clientType", func() {
	DescribeTable("classifies by credential",
		func(cred auth.UserCredential, expected string) {
			Expect(clientType(cred)).To(Equal(expected))
		},
		Entry("bk_token", auth.UserCredential{BkToken: "x"}, ClientWeb),
		Entry("bk_ticket", auth.UserCredential{BkTicket: "x"}, ClientWeb),
		Entry("access token", auth.UserCredential{AccessToken: "x"}, ClientToken),
		Entry("empty", auth.UserCredential{}, ClientUnknown),
	)
})
