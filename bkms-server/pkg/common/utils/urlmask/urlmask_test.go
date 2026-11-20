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

package urlmask_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/urlmask"
)

var _ = Describe("Query", func() {
	DescribeTable("masks sensitive values",
		func(raw, expected string) {
			Expect(urlmask.Query(raw)).To(Equal(expected))
		},
		Entry("empty", "", ""),
		Entry("no sensitive keys", "keyword=foo&page=1", "keyword=foo&page=1"),
		Entry("token and ticket", "access_token=a&bk_ticket=b", "access_token=***&bk_ticket=***"),
		Entry("secret, password and credential", "appSecret=a&Password=b&credential=c",
			"appSecret=***&Password=***&credential=***"),
		Entry("key suffix", "api_key=a&apiKey=b&keyword=c", "api_key=***&apiKey=***&keyword=c"),
		Entry("repeated values", "token=a&token=b", "token=***&token=***"),
		Entry("keeps order and encoding", "z=1&q=a+b%20c&token=x&a=%E4%B8%AD", "z=1&q=a+b%20c&token=***&a=%E4%B8%AD"),
		Entry("escaped sensitive key", "access%5Ftoken=x", "access%5Ftoken=***"),
		Entry("empty values", "keyword=&token=", "keyword=&token=***"),
		Entry("key without value", "token&page=1", "token&page=1"),
		Entry("malformed values", "a=%zz&token=%zz", "a=%zz&token=***"),
	)

	It("truncates long query", func() {
		Expect(urlmask.Query("q=" + strings.Repeat("a", 2000))).To(HaveLen(1024))
	})
})

var _ = Describe("URL", func() {
	DescribeTable("masks query of url",
		func(raw, expected string) {
			Expect(urlmask.URL(raw)).To(Equal(expected))
		},
		Entry("empty", "", ""),
		Entry("without query", "https://bkms.example.com/apps", "https://bkms.example.com/apps"),
		Entry("with sensitive query", "https://bkms.example.com/apps?bk_token=x&tab=env",
			"https://bkms.example.com/apps?bk_token=***&tab=env"),
		Entry("malformed", "http://[::1", urlmask.Masked),
	)
})
