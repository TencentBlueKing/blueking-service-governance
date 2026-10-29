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
	"fmt"
	"sync"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Counter", func() {
	var (
		ctx   context.Context
		db    *mongo.Database
		appID string
	)

	BeforeEach(func() {
		ctx = testutil.WithTenantOrDefault(context.Background())
		db = testutil.NewDB(GinkgoT())
		appID = "semver-test-" + stringx.Random(8)
	})

	expectNext := func(bump BumpType, expected string) {
		v, err := Next(ctx, db, appID, bump)
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
			v, err := Next(ctx, db, otherID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("0.0.1"))

			expectNext(BumpPatch, "0.0.3")
		})
	})

	Describe("concurrency", func() {
		It("should hand out unique versions when the counter is created concurrently", func() {
			const n = 10
			var (
				wg  sync.WaitGroup
				mu  sync.Mutex
				got []string
			)
			for range n {
				wg.Go(func() {
					defer GinkgoRecover()
					v, err := Next(ctx, db, appID, BumpPatch)
					Expect(err).NotTo(HaveOccurred())
					mu.Lock()
					got = append(got, v)
					mu.Unlock()
				})
			}
			wg.Wait()
			Expect(lo.Uniq(got)).To(HaveLen(n))
			expectNext(BumpPatch, fmt.Sprintf("0.0.%d", n+1))
		})
	})

	Describe("unknown bump type", func() {
		It("should return error", func() {
			_, err := Next(ctx, db, appID, BumpType("unknown"))
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
			c, err := Get(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Major).To(Equal(int64(0)))
			Expect(c.Minor).To(Equal(int64(0)))
			Expect(c.Patch).To(Equal(int64(0)))
			Expect(c.FormatSemver()).To(Equal("0.0.0"))
		})

		It("should return current counter value after Next calls", func() {
			_, err := Next(ctx, db, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			_, err = Next(ctx, db, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())
			_, err = Next(ctx, db, appID, BumpMinor)
			Expect(err).NotTo(HaveOccurred())

			c, err := Get(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Major).To(Equal(int64(0)))
			Expect(c.Minor).To(Equal(int64(1)))
			Expect(c.Patch).To(Equal(int64(0)))
			Expect(c.FormatSemver()).To(Equal("0.1.0"))
		})

		It("should not increment counter on Get", func() {
			_, err := Next(ctx, db, appID, BumpPatch)
			Expect(err).NotTo(HaveOccurred())

			c1, err := Get(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			c2, err := Get(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c1.Patch).To(Equal(c2.Patch))
		})
	})

	Describe("tenant", func() {
		It("should require a tenant in ctx", func() {
			_, err := Next(context.Background(), db, appID, BumpPatch)
			Expect(errors.Is(err, store.ErrTenantIDRequired)).To(BeTrue(), "got %v", err)
		})

		It("should reject a ctx that requires an explicit tenant scope", func() {
			_, err := Next(store.RequireExplicitTenantScope(ctx), db, appID, BumpPatch)
			Expect(errors.Is(err, store.ErrTenantScopeRequired)).To(BeTrue(), "got %v", err)
		})

		It("should write the counter under the tenant in ctx", func() {
			expectNext(BumpPatch, "0.0.1")

			otherCtx := tenant.WithTenantID(context.Background(), "tenant-other")
			c, err := Get(otherCtx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.FormatSemver()).To(Equal("0.0.0"))

			c, err = Get(ctx, db, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.FormatSemver()).To(Equal("0.0.1"))
		})
	})
})
