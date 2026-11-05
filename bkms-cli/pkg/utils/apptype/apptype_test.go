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

package apptype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/apptype"
)

func TestAppType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "apptype Suite")
}

var _ = Describe("IsAppModel", func() {
	DescribeTable("reports whether the app is managed by AppModel",
		func(appType string, expected bool) {
			Expect(apptype.IsAppModel(appType)).To(Equal(expected))
		},
		Entry("trpc", "trpc", true),
		Entry("taf", "taf", true),
		Entry("bkmsApp", "bkmsApp", true),
		Entry("helm", "helm", false),
		Entry("agones", "agones", false),
		Entry("empty", "", false),
	)
})

var _ = Describe("IsBKMSApp", func() {
	It("only matches bkmsApp", func() {
		Expect(apptype.IsBKMSApp("bkmsApp")).To(BeTrue())
		Expect(apptype.IsBKMSApp("trpc")).To(BeFalse())
		Expect(apptype.IsBKMSApp("")).To(BeFalse())
	})
})

var _ = Describe("FrameworkOf", func() {
	DescribeTable("resolves the framework of an app",
		func(appType, framework, expected string) {
			Expect(apptype.FrameworkOf(appType, framework)).To(Equal(expected))
		},
		Entry("bkmsApp uses its own framework", "bkmsApp", "blank", "blank"),
		// 旧应用没有独立的 framework 字段时回落到 type
		Entry("legacy trpc without framework", "trpc", "", "trpc"),
		Entry("legacy taf without framework", "taf", "", "taf"),
		Entry("legacy trpc with backfilled framework", "trpc", "trpc", "trpc"),
		Entry("bkmsApp without framework", "bkmsApp", "", ""),
		Entry("helm has no framework", "helm", "", ""),
	)
})
