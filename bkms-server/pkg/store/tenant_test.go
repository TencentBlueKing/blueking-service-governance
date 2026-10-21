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

package store_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Test tenant isolation", func() {
	var f *storeFixture

	BeforeEach(func() {
		f = setUpStores()
	})

	AfterEach(func() {
		f.tearDown()
	})

	Context("writes", func() {
		It("stores the tenant resolved from context", func() {
			f.mustCreateTicket(f.ctxA, "t1")
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
		})

		It("fills the tenant back into the object", func() {
			t := f.mustCreateTicket(f.ctxA, "t1")
			Expect(t.TenantID).To(Equal(tenantA))
		})

		It("overrides a tenant forged by the caller", func() {
			t := &Ticket{OwnerID: f.owner, Name: "t1", TenantID: tenantB}
			Expect(f.tickets.Create(f.ctxA, t)).To(Succeed())
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
			Expect(t.TenantID).To(Equal(tenantA))
		})

		It("stores the tenant for every object of CreateMany", func() {
			objs := []*Ticket{{OwnerID: f.owner, Name: "t1"}, {OwnerID: f.owner, Name: "t2"}}
			Expect(f.tickets.CreateMany(f.ctxA, objs)).To(Succeed())
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
			Expect(f.rawTicketTenant("t2")).To(Equal(tenantA))
		})

		It("keeps the tenant when a document is replaced", func() {
			t := f.mustCreateTicket(f.ctxA, "t1")
			t.Priority = 9
			t.TenantID = tenantB
			Expect(f.tickets.Replace(f.ctxA, t, f.byName("t1"))).To(Succeed())
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
		})

		It("stores the tenant on an inserting Upsert", func() {
			_, err := f.tickets.Upsert(f.ctxA, &Ticket{OwnerID: f.owner, Name: "t1"}, f.byName("t1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
		})

		It("does not let an Upsert of another tenant touch an existing document", func() {
			f.mustCreateTicket(f.ctxA, "t1")

			// B 租户 upsert 同名工单：匹配不到 A 的文档，插入一条属于 B 的新文档
			inserted, err := f.tickets.Upsert(f.ctxB, &Ticket{OwnerID: f.owner, Name: "t1"}, f.byName("t1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeTrue())

			n, err := f.tickets.Count(f.ctx, f.byName("t1"), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
		})
	})

	Context("reads and writes are scoped to the tenant in context", func() {
		BeforeEach(func() {
			f.mustCreateTicket(f.ctxA, "a1")
			f.mustCreateTicket(f.ctxB, "b1")
		})

		It("Get only sees its own tenant", func() {
			_, err := f.tickets.Get(f.ctxB, f.byName("a1"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
			Expect(errors.Is(err, ErrTicketNotFound)).To(BeTrue())
		})

		It("List only sees its own tenant", func() {
			list, err := f.tickets.List(f.ctxA, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"a1"}))
		})

		It("ListPage only sees its own tenant", func() {
			page, err := f.tickets.ListPage(f.ctxB, 1, 10, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(1)))
			Expect(ticketNames(page.Items)).To(Equal([]string{"b1"}))
		})

		It("Count only counts its own tenant", func() {
			n, err := f.tickets.Count(f.ctxA, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))
		})

		It("Exists does not leak across tenants", func() {
			ok, err := f.tickets.Exists(f.ctxB, f.byName("a1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("ListLatestBy only sees its own tenant", func() {
			list, err := f.tickets.ListLatestBy(f.ctxA, []string{"ownerID"}, f.byOwner(), store.Desc("createdAt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"a1"}))
		})

		It("Aggregate only sees its own tenant", func() {
			type row struct {
				Owner string `bson:"_id"`
				Count int64  `bson:"count"`
			}
			rows, err := store.Aggregate[row](f.ctxA, f.tickets, mongo.Pipeline{
				{{Key: "$group", Value: bson.M{"_id": "$ownerID", "count": bson.M{"$sum": 1}}}},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(HaveLen(1))
			Expect(rows[0].Count).To(Equal(int64(1)))
		})

		It("Update cannot reach another tenant", func() {
			err := f.tickets.Update(f.ctxB, bson.M{"priority": 9}, f.byName("a1"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
		})

		It("UpdateMany does not touch another tenant", func() {
			n, err := f.tickets.UpdateMany(f.ctxB, bson.M{"priority": 9}, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			a, err := f.tickets.Get(f.ctxA, f.byName("a1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(a.Priority).To(Equal(1))
		})

		It("Modify cannot reach another tenant", func() {
			_, err := f.tickets.Modify(f.ctxB, bson.M{"$inc": bson.M{"hits": 1}}, f.byName("a1"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
		})

		It("Delete cannot reach another tenant", func() {
			err := f.tickets.Delete(f.ctxB, f.byName("a1"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
		})

		It("DeleteMany does not remove another tenant's documents", func() {
			n, err := f.tickets.DeleteMany(f.ctxB, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			_, err = f.tickets.Get(f.ctxA, f.byName("a1"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("GetAndDelete cannot reach another tenant", func() {
			_, err := f.tickets.GetAndDelete(f.ctxB, f.byName("a1"))
			Expect(errors.Is(err, store.ErrNotFound)).To(BeTrue())
		})
	})

	Context("missing tenant", func() {
		// 漏挂中间件时每个操作都要立刻失败，而不是回落到默认租户悄悄写出错数据
		It("fails every operation when the context has no tenant", func() {
			expectTenantRequired := func(err error) {
				GinkgoHelper()
				Expect(errors.Is(err, store.ErrTenantIDRequired)).To(BeTrue(), "got %v", err)
			}

			_, err := f.tickets.Get(f.ctx, f.byName("t1"))
			expectTenantRequired(err)
			_, err = f.tickets.List(f.ctx, f.byOwner())
			expectTenantRequired(err)
			_, err = f.tickets.ListPage(f.ctx, 1, 10, f.byOwner())
			expectTenantRequired(err)
			_, err = f.tickets.Count(f.ctx, f.byOwner())
			expectTenantRequired(err)
			_, err = f.tickets.Exists(f.ctx, f.byOwner())
			expectTenantRequired(err)
			expectTenantRequired(f.tickets.Create(f.ctx, &Ticket{OwnerID: f.owner, Name: "t1"}))
			expectTenantRequired(f.tickets.CreateMany(f.ctx, []*Ticket{{OwnerID: f.owner, Name: "t1"}}))
			expectTenantRequired(f.tickets.Replace(f.ctx, &Ticket{OwnerID: f.owner, Name: "t1"}, f.byName("t1")))
			_, err = f.tickets.Upsert(f.ctx, &Ticket{OwnerID: f.owner, Name: "t1"}, f.byName("t1"))
			expectTenantRequired(err)
			expectTenantRequired(f.tickets.Update(f.ctx, bson.M{"priority": 2}, f.byName("t1")))
			_, err = f.tickets.Modify(f.ctx, bson.M{"$inc": bson.M{"hits": 1}}, f.byName("t1"))
			expectTenantRequired(err)
			expectTenantRequired(f.tickets.Delete(f.ctx, f.byName("t1")))
			_, err = f.tickets.DeleteMany(f.ctx, f.byOwner())
			expectTenantRequired(err)
		})
	})

	Context("scopes", func() {
		BeforeEach(func() {
			f.mustCreateTicket(f.ctxA, "a1")
			f.mustCreateTicket(f.ctxB, "b1")
		})

		It("AllTenants reads across tenants", func() {
			list, err := f.tickets.List(f.ctx, f.byOwner(), store.Asc("name"), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"a1", "b1"}))
		})

		It("AllTenants allows filtering by tenant explicitly", func() {
			list, err := f.tickets.List(f.ctx, f.byOwner(), store.Where("tenantID", tenantB), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"b1"}))
		})

		It("AllTenants is not accepted by operations writing a whole document", func() {
			err := f.tickets.Create(f.ctx, &Ticket{OwnerID: f.owner, Name: "t1"}, store.AllTenants())
			Expect(err).To(MatchError(ContainSubstring("all tenants")))

			err = f.tickets.Replace(f.ctx, &Ticket{OwnerID: f.owner, Name: "a1"}, f.byName("a1"), store.AllTenants())
			Expect(err).To(MatchError(ContainSubstring("all tenants")))

			_, err = f.tickets.Upsert(f.ctx, &Ticket{OwnerID: f.owner, Name: "a1"}, f.byName("a1"), store.AllTenants())
			Expect(err).To(MatchError(ContainSubstring("all tenants")))
		})

		It("AllTenants can still update and delete across tenants", func() {
			n, err := f.tickets.UpdateMany(f.ctx, bson.M{"priority": 7}, f.byOwner(), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))

			n, err = f.tickets.DeleteMany(f.ctx, f.byOwner(), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(2)))
		})

		It("ForTenant takes precedence over the context", func() {
			list, err := f.tickets.List(f.ctxA, f.byOwner(), store.ForTenant(tenantB))
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"b1"}))
		})

		It("ForTenant works without a tenant in the context", func() {
			list, err := f.tickets.List(f.ctx, f.byOwner(), store.ForTenant(tenantA))
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"a1"}))
		})

		It("ForTenant writes to the given tenant", func() {
			err := f.tickets.Create(f.ctx, &Ticket{OwnerID: f.owner, Name: "c1"}, store.ForTenant(tenantB))
			Expect(err).NotTo(HaveOccurred())

			list, err := f.tickets.List(f.ctxB, f.byOwner(), store.Asc("name"))
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"b1", "c1"}))
		})

		It("ForTenant with an empty id fails instead of matching everything", func() {
			_, err := f.tickets.List(f.ctx, f.byOwner(), store.ForTenant(""))
			Expect(errors.Is(err, store.ErrTenantIDRequired)).To(BeTrue())
		})

		It("rejects specifying the scope twice", func() {
			_, err := f.tickets.List(f.ctx, f.byOwner(), store.AllTenants(), store.ForTenant(tenantA))
			Expect(errors.Is(err, store.ErrTenantScopeDuplicated)).To(BeTrue())
		})

		It("rejects a scope on a collection that is not tenant isolated", func() {
			_, err := f.widgets.List(f.ctx, f.byOwner(), store.AllTenants())
			Expect(errors.Is(err, store.ErrTenantScopeNotSupported)).To(BeTrue())
		})

		It("rejects a scope hidden inside Combine", func() {
			hidden := store.Combine(f.byOwner(), store.AllTenants())
			_, err := f.tickets.List(f.ctx, hidden)
			Expect(errors.Is(err, store.ErrTenantScopeCombined)).To(BeTrue())
		})
	})

	Context("explicit tenant scope required", func() {
		var platCtx context.Context

		BeforeEach(func() {
			f.mustCreateTicket(f.ctxA, "a1")
			f.mustCreateTicket(f.ctxB, "b1")
			// 模拟平台管理请求：ctx 里带着操作者的租户，同时被要求显式选择范围
			platCtx = store.RequireExplicitTenantScope(f.ctxA)
		})

		It("rejects an operation without a scope instead of using the tenant in context", func() {
			_, err := f.tickets.List(platCtx, f.byOwner())
			Expect(errors.Is(err, store.ErrTenantScopeRequired)).To(BeTrue(), "got %v", err)

			err = f.tickets.Create(platCtx, &Ticket{OwnerID: f.owner, Name: "c1"})
			Expect(errors.Is(err, store.ErrTenantScopeRequired)).To(BeTrue(), "got %v", err)

			err = f.tickets.Update(platCtx, bson.M{"priority": 9}, f.byName("a1"))
			Expect(errors.Is(err, store.ErrTenantScopeRequired)).To(BeTrue(), "got %v", err)
		})

		It("accepts AllTenants", func() {
			list, err := f.tickets.List(platCtx, f.byOwner(), store.Asc("name"), store.AllTenants())
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"a1", "b1"}))
		})

		It("accepts ForTenant", func() {
			list, err := f.tickets.List(platCtx, f.byOwner(), store.ForTenant(tenantB))
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"b1"}))
		})

		It("does not affect collections that are not tenant isolated", func() {
			f.mustCreateWidget("w1")
			_, err := f.widgets.Get(platCtx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("tenantID is maintained by the store", func() {
		It("rejects tenantID as a condition under a tenant scope", func() {
			_, err := f.tickets.List(f.ctxA, store.Where("tenantID", tenantB))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be used as a condition"))
		})

		It("rejects updating tenantID", func() {
			f.mustCreateTicket(f.ctxA, "t1")

			err := f.tickets.Update(f.ctxA, bson.M{"tenantID": tenantB}, f.byName("t1"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be updated"))

			_, err = f.tickets.Modify(f.ctxA, bson.M{"$set": bson.M{"tenantID": tenantB}}, f.byName("t1"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be updated"))
		})

		It("does not rewrite tenantID when updating all fields", func() {
			t := f.mustCreateTicket(f.ctxA, "t1")
			t.Priority = 5
			t.TenantID = tenantB
			Expect(f.tickets.Update(f.ctxA, t, store.UpdateSelected("*"), f.byName("t1"))).To(Succeed())
			Expect(f.rawTicketTenant("t1")).To(Equal(tenantA))
		})
	})

	Context("safety nets kept intact", func() {
		It("still requires a filter for writes even though a tenant condition is injected", func() {
			_, err := f.tickets.DeleteMany(f.ctxA)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("filter is required"))
		})

		It("accepts All() as an explicit whole-tenant filter", func() {
			f.mustCreateTicket(f.ctxA, "a1")
			f.mustCreateTicket(f.ctxB, "b1")

			n, err := f.tickets.DeleteMany(f.ctxA, store.All())
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			list, err := f.tickets.List(f.ctxB, f.byOwner())
			Expect(err).NotTo(HaveOccurred())
			Expect(ticketNames(list)).To(Equal([]string{"b1"}))
		})

		It("rejects aggregation stages that read other collections", func() {
			_, err := store.Aggregate[bson.M](f.ctxA, f.tickets, mongo.Pipeline{
				{{Key: "$lookup", Value: bson.M{"from": widgetCollection, "as": "w"}}},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bypasses tenant isolation"))
		})

		It("keeps Upsert matching by the object _id", func() {
			t := f.mustCreateTicket(f.ctxA, "t1")
			t.Priority = 3
			inserted, err := f.tickets.Upsert(f.ctxA, t)
			Expect(err).NotTo(HaveOccurred())
			Expect(inserted).To(BeFalse())

			got, err := f.tickets.Get(f.ctxA, f.byName("t1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Priority).To(Equal(3))
			Expect(got.TenantID).To(Equal(tenantA))
		})
	})

	Context("global collections are unaffected", func() {
		It("reads and writes without a tenant in the context", func() {
			w := f.mustCreateWidget("w1")
			Expect(w.CreatedAt).NotTo(Equal(time.Time{}))

			got, err := f.widgets.Get(f.ctx, f.byName("w1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("w1"))
		})
	})
})

var _ = Describe("Test tenancy declaration", func() {
	var db *mongo.Database

	BeforeEach(func() {
		f := setUpStores()
		db = f.db
		DeferCleanup(f.tearDown)
	})

	It("rejects a tenant isolated meta whose model has no tenantID field", func() {
		_, err := store.New(db, store.Meta[Widget]{Collection: widgetCollection, TenantIsolated: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`requires a "tenantID" field`))
	})

	It("rejects a meta that has a tenantID field but is not declared isolated", func() {
		_, err := store.New(db, store.Meta[Ticket]{Collection: ticketCollection})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("set TenantIsolated to true"))
	})

	It("rejects a tenantID field that is not a string", func() {
		type badTenant struct {
			TenantID int `bson:"tenantID"`
		}
		_, err := store.New(db, store.Meta[badTenant]{Collection: "store_test_bad", TenantIsolated: true})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must be a string"))
	})
})
