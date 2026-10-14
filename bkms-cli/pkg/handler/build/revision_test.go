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

package build_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/build"
)

var _ = Describe("ResolveCodeRevision", func() {
	DescribeTable(
		"accepts exactly one source",
		func(branch, tag, commit, wantType, wantValue string) {
			gotType, gotValue, err := build.ResolveCodeRevision(branch, tag, commit)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotType).To(Equal(wantType))
			Expect(gotValue).To(Equal(wantValue))
		},
		Entry("branch", "main", "", "", "branch", "main"),
		Entry("tag", "", "v1.0.0", "", "tag", "v1.0.0"),
		Entry("short commit", "", "", "a1b2c3d", "commit", "a1b2c3d"),
		Entry("upper-case commit", "", "", "A1B2C3D4E5F6", "commit", "A1B2C3D4E5F6"),
		Entry(
			"full sha",
			"",
			"",
			"0123456789abcdef0123456789abcdef01234567",
			"commit",
			"0123456789abcdef0123456789abcdef01234567",
		),
	)

	DescribeTable("rejects a missing, duplicated, or illegal source before any request",
		func(branch, tag, commit, message string) {
			_, _, err := build.ResolveCodeRevision(branch, tag, commit)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
		Entry("none", "", "", "", "exactly one of --branch, --tag, or --commit"),
		Entry("branch and tag", "main", "v1.0.0", "", "exactly one of --branch, --tag, or --commit"),
		Entry("too short", "", "", "a1b2c3", "7-40 hexadecimal"),
		Entry("too long", "", "", "0123456789abcdef0123456789abcdef012345678", "7-40 hexadecimal"),
		Entry("non-hex", "", "", "g1b2c3d", "7-40 hexadecimal"),
	)
})
