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

package revision_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/revision"
)

const fullSHA = "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678"

var _ = Describe("CodeRevision", func() {
	Context("Resolve", func() {
		It("should default to branch and fall back to legacy branch", func() {
			rev, err := revision.Resolve("", "", "master")
			Expect(err).NotTo(HaveOccurred())
			Expect(rev).To(Equal(revision.CodeRevision{Type: revision.TypeBranch, Value: "master"}))
		})

		It("should prefer revision over legacy branch", func() {
			rev, err := revision.Resolve("tag", "v1.0.0", "master")
			Expect(err).NotTo(HaveOccurred())
			Expect(rev).To(Equal(revision.CodeRevision{Type: revision.TypeTag, Value: "v1.0.0"}))
		})

		// 兼容字段只代表分支，不能补给显式选择的 Tag / CommitID；空值留给调用方按构建模式判断
		DescribeTable("should not fill legacy branch into tag or commit",
			func(typ, legacyBranch string, wantType revision.Type) {
				rev, err := revision.Resolve(typ, "", legacyBranch)
				Expect(err).NotTo(HaveOccurred())
				Expect(rev).To(Equal(revision.CodeRevision{Type: wantType}))
			},
			Entry("tag ignores legacy branch", "tag", "main", revision.TypeTag),
			Entry("commit ignores hex-like legacy branch", "commit", "a1b2c3d", revision.TypeCommit),
		)

		DescribeTable("should validate commit id format",
			func(value string, valid bool) {
				rev, err := revision.Resolve("commit", value, "")
				if valid {
					Expect(err).NotTo(HaveOccurred())
					Expect(rev).To(Equal(revision.CodeRevision{Type: revision.TypeCommit, Value: value}))
					return
				}
				Expect(errors.Is(err, revision.ErrInvalidCommitID)).To(BeTrue())
			},
			Entry("7 chars short sha", "a1b2c3d", true),
			Entry("upper case", "A1B2C3D4E5F6", true),
			Entry("40 chars full sha", fullSHA, true),
			Entry("6 chars", "a1b2c3", false),
			Entry("41 chars", fullSHA+"a", false),
			Entry("non hex char", "g1b2c3d", false),
		)
	})

	Context("CheckoutBy and TypeFromCheckoutBy", func() {
		DescribeTable("should map both ways",
			func(typ revision.Type, checkoutBy string) {
				Expect(revision.CodeRevision{Type: typ}.CheckoutBy()).To(Equal(checkoutBy))
				Expect(revision.TypeFromCheckoutBy(checkoutBy)).To(Equal(typ))
			},
			Entry("branch", revision.TypeBranch, "BRANCH"),
			Entry("tag", revision.TypeTag, "TAG"),
			Entry("commit", revision.TypeCommit, "COMMIT_ID"),
		)

		// 本功能上线前的构建记录没有检出方式参数，必须按分支处理才能免迁移
		It("should treat missing checkout as branch", func() {
			Expect(revision.TypeFromCheckoutBy("")).To(Equal(revision.TypeBranch))
		})
	})

	Context("ImageTagSegment", func() {
		It("should keep first 7 chars of commit id", func() {
			rev := revision.CodeRevision{Type: revision.TypeCommit, Value: fullSHA}
			Expect(rev.ImageTagSegment()).To(Equal(fullSHA[:7]))
		})

		It("should not truncate branch", func() {
			longBranch := "feature/" + strings.Repeat("x", 20)
			Expect(revision.CodeRevision{Type: revision.TypeBranch, Value: longBranch}.ImageTagSegment()).
				To(Equal(longBranch))
		})
	})
})
