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

package dal

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var _ = Describe("Table tenant scope", func() {
	It("keeps a global table from injecting even after ForTenant", func() {
		table := newTable("cluster_addon_defs", nil).ForTenant("tenant-a")
		id, ok := table.TenantID(tenant.WithTenantID(context.Background(), "tenant-b"))
		Expect(ok).To(BeFalse())
		Expect(id).To(BeEmpty())
	})

	It("uses ForTenant instead of the tenant in context", func() {
		table := newTable("workspaces", nil).ForTenant("tenant-a")
		id, ok := table.TenantID(tenant.WithTenantID(context.Background(), "tenant-b"))
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-a"))
	})

	It("reads the tenant from context when no scope is set", func() {
		table := newTable("workspaces", nil)
		id, ok := table.TenantID(tenant.WithTenantID(context.Background(), "tenant-b"))
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-b"))
	})

	It("returns false when context has no tenant", func() {
		table := newTable("workspaces", nil)
		_, ok := table.TenantID(context.Background())
		Expect(ok).To(BeFalse())
	})

	It("applies AllTenants through Scopes without changing the original store", func() {
		store := newScopedStore(newTable("workspaces", nil))
		ctx := tenant.WithTenantID(context.Background(), "tenant-a")

		_, ok := store.Scopes(AllTenants()).table().TenantID(ctx)
		Expect(ok).To(BeFalse())

		id, ok := store.table().TenantID(ctx)
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-a"))
	})

	It("applies ForTenant through Scope without changing the original store", func() {
		store := newScopedStore(newTable("workspaces", nil))
		ctx := tenant.WithTenantID(context.Background(), "tenant-b")

		id, ok := store.Scopes(ForTenant("tenant-a")).table().TenantID(ctx)
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-a"))

		id, ok = store.table().TenantID(ctx)
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-b"))
	})

	It("uses ForTenant through Scope when context has no tenant", func() {
		store := newScopedStore(newTable("workspaces", nil))
		id, ok := store.Scopes(ForTenant("tenant-a")).table().TenantID(context.Background())
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-a"))
	})

	It("keeps a global table from injecting after Scope ForTenant", func() {
		store := newScopedStore(newTable("cluster_addon_defs", nil))
		_, ok := store.Scopes(ForTenant("tenant-a")).table().TenantID(context.Background())
		Expect(ok).To(BeFalse())
	})

	It("returns ErrTenantIDRequired when Scope ForTenant is empty", func() {
		store := newScopedStore(newTable("workspaces", nil))
		ctx := tenant.WithTenantID(context.Background(), "tenant-b")
		_, err := store.Scopes(ForTenant("")).table().Find(nil).Count(ctx)
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})

	It("returns the scope error on the next operation when Scope is called twice", func() {
		store := newScopedStore(newTable("workspaces", nil))
		scoped := store.Scopes(ForTenant("tenant-a"))

		_, err := scoped.Scopes(ForTenant("tenant-c")).table().Find(nil).Count(context.Background())
		Expect(err).To(MatchError(ErrScopeAlreadySet))

		id, ok := store.Scopes(AllTenants()).table().TenantID(context.Background())
		Expect(ok).To(BeFalse())
		Expect(id).To(BeEmpty())
	})

	It("returns the scope error on the next operation when the scope is nil", func() {
		store := newScopedStore(newTable("workspaces", nil))
		_, err := store.Scopes(nil).table().Find(nil).Count(context.Background())
		Expect(err).To(MatchError(ErrScopeRequired))
	})

	It("returns the scope error on the next operation when the table scope is set twice", func() {
		table := newTable("workspaces", nil)
		scoped := table.ForTenant("tenant-a")

		_, err := scoped.AllTenants().Find(nil).Count(context.Background())
		Expect(err).To(MatchError(ErrScopeAlreadySet))
		_, err = scoped.FromContext().Find(nil).Count(context.Background())
		Expect(err).To(MatchError(ErrScopeAlreadySet))

		_, ok := table.AllTenants().TenantID(context.Background())
		Expect(ok).To(BeFalse())
	})

	It("reads the tenant from context through Scope FromContext", func() {
		store := newScopedStore(newTable("workspaces", nil))
		scoped := store.Scopes(FromContext())

		id, ok := scoped.table().TenantID(tenant.WithTenantID(context.Background(), "tenant-b"))
		Expect(ok).To(BeTrue())
		Expect(id).To(Equal("tenant-b"))

		_, err := scoped.table().Find(nil).Count(context.Background())
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})

	It("keeps a global table from injecting after Scope FromContext", func() {
		store := newScopedStore(newTable("cluster_addon_defs", nil))
		_, ok := store.Scopes(FromContext()).table().TenantID(tenant.WithTenantID(context.Background(), "tenant-b"))
		Expect(ok).To(BeFalse())
	})

	It("returns ErrTenantIDRequired when Find runs without a tenant", func() {
		table := newTable("workspaces", nil)
		err := table.Find(bson.M{"id": "ws"}).One(context.Background(), &bson.M{})
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})

	It("returns ErrTenantIDRequired when ForTenant is empty", func() {
		table := newTable("workspaces", nil).ForTenant("")
		_, err := table.Find(nil).Count(context.Background())
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})
})

var _ = Describe("tenant field injection", func() {
	It("overwrites tenant_id on a bson.M copy and leaves the caller unchanged", func() {
		filter := bson.M{"id": "ws", tenant.FieldTenantID: "forged"}
		got, err := setTenantField(filter, "tenant-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(bson.M{"id": "ws", tenant.FieldTenantID: "tenant-a"}))
		Expect(filter[tenant.FieldTenantID]).To(Equal("forged"))
	})

	It("replaces tenant_id on bson.D", func() {
		filter := bson.D{{Key: "id", Value: "ws"}, {Key: tenant.FieldTenantID, Value: "forged"}}
		got, err := setTenantField(filter, "tenant-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(bson.D{
			{Key: "id", Value: "ws"},
			{Key: tenant.FieldTenantID, Value: "tenant-a"},
		}))
	})

	It("writes tenant_id onto a struct copy", func() {
		type doc struct {
			ID       string `bson:"id"`
			TenantID string `bson:"tenant_id"`
		}
		got, err := setTenantField(doc{ID: "ws", TenantID: "forged"}, "tenant-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(bson.D{
			{Key: "id", Value: "ws"},
			{Key: tenant.FieldTenantID, Value: "tenant-a"},
		}))
	})
})

var _ = Describe("pipeline and update guards", func() {
	It("prepends a tenant match and rejects cross-collection stages", func() {
		got, err := applyPipeline([]bson.D{
			{{Key: "$group", Value: bson.M{"_id": "$state"}}},
		}, "tenant-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(got[0]).To(Equal(bson.D{{Key: "$match", Value: bson.M{tenant.FieldTenantID: "tenant-a"}}}))

		_, err = applyPipeline([]bson.D{{{Key: "$lookup", Value: bson.M{"from": "apps"}}}}, "tenant-a")
		Expect(err).To(MatchError(ContainSubstring("$lookup")))
	})

	It("rejects updates that rewrite tenant_id", func() {
		err := ensureTenantImmutable(bson.M{"$set": bson.M{tenant.FieldTenantID: "other"}})
		Expect(err).To(MatchError(ContainSubstring("tenant_id")))

		err = ensureTenantImmutable(bson.A{bson.D{{Key: "$set", Value: bson.M{"state": "Ready"}}}})
		Expect(err).To(MatchError(ContainSubstring("pipeline-style update")))
	})
})

type scopedStore struct {
	tbl Table
}

func newScopedStore(table Table) *scopedStore {
	return &scopedStore{tbl: table}
}

func (s *scopedStore) Scopes(scope Scope) *scopedStore {
	next := *s
	next.tbl = ApplyScope(s.tbl, scope)
	return &next
}

func (s *scopedStore) table() Table {
	return s.tbl
}
