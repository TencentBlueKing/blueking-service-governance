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

package semver

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("CounterStoreMongo", func() {
	var (
		ctx   context.Context
		store CounterStore
		appID string
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = NewCounterStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		appID = "semver-test-" + stringx.Random(8)
	})

	expectNext := func(bump BumpType, expected string) {
		v, err := store.Next(ctx, appID, bump)
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal(expected))
	}

	Describe("BumpPatch", func() {
		It("should increment patch sequentially starting from 0.0.1", func() {
			expectNext(BumpPatch, "0.0.1")
			expectNext(BumpPatch, "0.0.2")
			expectNext(BumpPatch, "0.0.3")
		})
	})

	Describe("BumpMinor", func() {
		It("should reset patch when bumping minor", func() {
			expectNext(BumpPatch, "0.0.1")
			expectNext(BumpPatch, "0.0.2")
			expectNext(BumpPatch, "0.0.3")
			expectNext(BumpMinor, "0.1.0")
		})

		It("should increment minor sequentially", func() {
			expectNext(BumpMinor, "0.1.0")
			expectNext(BumpMinor, "0.2.0")
			expectNext(BumpMinor, "0.3.0")
		})
	})

	Describe("BumpMajor", func() {
		It("should reset minor and patch when bumping major", func() {
			expectNext(BumpMinor, "0.1.0")
			expectNext(BumpMinor, "0.2.0")
			expectNext(BumpPatch, "0.2.1")
			expectNext(BumpPatch, "0.2.2")
			expectNext(BumpMajor, "1.0.0")
		})

		It("should increment major sequentially", func() {
			expectNext(BumpMajor, "1.0.0")
			expectNext(BumpMajor, "2.0.0")
			expectNext(BumpMajor, "3.0.0")
		})
	})

	Describe("independent counters", func() {
		It("should maintain independent counters for different appIDs", func() {
			expectNext(BumpPatch, "0.0.1")
			expectNext(BumpPatch, "0.0.2")

			otherID := "semver-test-" + stringx.Random(8)
			v, err := store.Next(ctx, otherID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("0.0.1"))

			expectNext(BumpPatch, "0.0.3")
		})
	})

	Describe("unknown bump type", func() {
		It("should return error", func() {
			_, err := store.Next(ctx, appID, BumpType("unknown"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown bump type"))
		})
	})

	Describe("mixed bump sequence", func() {
		It("should handle mixed bump types correctly", func() {
			expectNext(BumpPatch, "0.0.1")
			expectNext(BumpPatch, "0.0.2")
			expectNext(BumpPatch, "0.0.3")
			expectNext(BumpMinor, "0.1.0")
			expectNext(BumpPatch, "0.1.1")
			expectNext(BumpPatch, "0.1.2")
			expectNext(BumpMajor, "1.0.0")
			expectNext(BumpPatch, "1.0.1")
			expectNext(BumpMinor, "1.1.0")
			expectNext(BumpMajor, "2.0.0")
		})
	})

	Describe("Get", func() {
		It("should return zero-value counter when no record exists", func() {
			c, err := store.Get(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Major).To(Equal(int64(0)))
			Expect(c.Minor).To(Equal(int64(0)))
			Expect(c.Patch).To(Equal(int64(0)))
			Expect(c.FormatSemver()).To(Equal("0.0.0"))
		})

		It("should return current counter value after Next calls", func() {
			_, err := store.Next(ctx, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Next(ctx, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Next(ctx, appID, BumpMinor)
			Expect(err).NotTo(HaveOccurred())

			c, err := store.Get(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Major).To(Equal(int64(0)))
			Expect(c.Minor).To(Equal(int64(1)))
			Expect(c.Patch).To(Equal(int64(0)))
			Expect(c.FormatSemver()).To(Equal("0.1.0"))
		})

		It("should not increment counter on Get", func() {
			_, err := store.Next(ctx, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())

			c1, err := store.Get(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			c2, err := store.Get(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c1.Patch).To(Equal(c2.Patch))
		})
	})
})
