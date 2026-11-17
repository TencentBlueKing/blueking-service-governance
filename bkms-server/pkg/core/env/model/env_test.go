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

package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment resource naming", func() {
	DescribeTable("PrefixName and TrimPrefix",
		func(resourceKey, base, want string) {
			env := Environment{ResourceKey: resourceKey}
			Expect(env.SharesNamespace()).To(Equal(resourceKey != ""))
			Expect(env.PrefixName(base)).To(Equal(want))
			Expect(env.TrimPrefix(want)).To(Equal(base))
		},
		Entry("keeps the base name without resource key", "", "web", "web"),
		Entry("prefixes the resource key", "fab12", "web", "fab12-web"),
	)

	It("leaves names without the prefix unchanged when trimming", func() {
		env := Environment{ResourceKey: "fab12"}
		Expect(env.TrimPrefix("web")).To(Equal("web"))
		Expect(env.TrimPrefix("fab12web")).To(Equal("fab12web"))
	})
})
