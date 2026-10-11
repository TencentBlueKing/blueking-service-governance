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

package datascope_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database/datascope"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

type sampleStore struct {
	scope datascope.Scope
}

func (s *sampleStore) Scopes(opt datascope.Option) *sampleStore {
	next := *s
	next.scope = datascope.ApplyScope(s.scope, opt)
	return &next
}

func (s *sampleStore) filter(ctx context.Context, filter bson.M) (bson.M, error) {
	return s.scope.TenantFilter(ctx, filter)
}

func (s *sampleStore) writeTenant(ctx context.Context) (string, error) {
	return s.scope.TenantForWrite(ctx)
}

var _ = Describe("Datascope", func() {
	var (
		store     *sampleStore
		tenantCtx context.Context
	)

	BeforeEach(func() {
		store = &sampleStore{}
		tenantCtx = tenant.WithTenantID(context.Background(), "tenant-a")
	})

	Context("default scope", func() {
		It("adds the tenant from context to a copy of the filter", func() {
			input := bson.M{"id": "ws"}
			got, err := store.filter(tenantCtx, input)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws", tenant.FieldTenantID: "tenant-a"}))
			Expect(input).To(Equal(bson.M{"id": "ws"}))
		})

		It("overwrites the tenant_id set by the caller", func() {
			got, err := store.filter(tenantCtx, bson.M{tenant.FieldTenantID: "forged"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got[tenant.FieldTenantID]).To(Equal("tenant-a"))
		})

		It("accepts a nil filter", func() {
			got, err := store.filter(tenantCtx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{tenant.FieldTenantID: "tenant-a"}))
		})

		It("returns ErrTenantIDRequired when context has no tenant", func() {
			_, err := store.filter(context.Background(), bson.M{})
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
			_, err = store.writeTenant(context.Background())
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
		})

		It("writes with the tenant from context", func() {
			id, err := store.writeTenant(tenantCtx)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("tenant-a"))
		})
	})

	Context("ForTenant", func() {
		It("uses the given tenant instead of context", func() {
			scoped := store.Scopes(datascope.ForTenant("tenant-b"))

			got, err := scoped.filter(tenantCtx, bson.M{})
			Expect(err).NotTo(HaveOccurred())
			Expect(got[tenant.FieldTenantID]).To(Equal("tenant-b"))

			id, err := scoped.writeTenant(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("tenant-b"))
		})

		It("returns ErrTenantIDRequired for an empty tenant", func() {
			scoped := store.Scopes(datascope.ForTenant(""))
			_, err := scoped.filter(tenantCtx, bson.M{})
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
		})
	})

	Context("FromContext", func() {
		It("reads the tenant from context", func() {
			scoped := store.Scopes(datascope.FromContext())

			got, err := scoped.filter(tenantCtx, bson.M{"id": "ws"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws", tenant.FieldTenantID: "tenant-a"}))

			_, err = scoped.writeTenant(context.Background())
			Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
		})
	})

	Context("AllTenants", func() {
		It("leaves the filter without a tenant condition", func() {
			scoped := store.Scopes(datascope.AllTenants())
			got, err := scoped.filter(tenantCtx, bson.M{"id": "ws"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws"}))
		})

		It("refuses to resolve a tenant for writes", func() {
			scoped := store.Scopes(datascope.AllTenants())
			_, err := scoped.writeTenant(tenantCtx)
			Expect(err).To(MatchError(datascope.ErrWriteAllTenants))
		})
	})

	Context("Scopes", func() {
		It("leaves the original store unchanged", func() {
			_ = store.Scopes(datascope.AllTenants())
			got, err := store.filter(tenantCtx, bson.M{})
			Expect(err).NotTo(HaveOccurred())
			Expect(got[tenant.FieldTenantID]).To(Equal("tenant-a"))
		})

		It("returns the scope error on the next operation when Scopes is called twice", func() {
			scoped := store.Scopes(datascope.ForTenant("tenant-b"))

			_, err := scoped.Scopes(datascope.ForTenant("tenant-c")).filter(tenantCtx, bson.M{})
			Expect(err).To(MatchError(datascope.ErrScopeAlreadySet))

			got, err := store.Scopes(datascope.AllTenants()).filter(tenantCtx, bson.M{"id": "ws"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws"}))
		})

		It("returns the scope error on the next operation when the option is nil", func() {
			_, err := store.Scopes(nil).filter(tenantCtx, bson.M{})
			Expect(err).To(MatchError(datascope.ErrScopeRequired))
		})

		It("returns the scope error when ApplyScope gets a scoped value", func() {
			scoped := datascope.ApplyScope(datascope.Scope{}, datascope.FromContext())

			_, err := datascope.ApplyScope(scoped, datascope.AllTenants()).TenantFilter(tenantCtx, bson.M{})
			Expect(err).To(MatchError(datascope.ErrScopeAlreadySet))

			got, err := datascope.ApplyScope(datascope.Scope{}, datascope.AllTenants()).
				TenantFilter(tenantCtx, bson.M{"id": "ws"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws"}))
		})
	})
})
