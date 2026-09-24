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

package tenant

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	reqtenant "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

func newTestCollection(name string) *Collection {
	return &Collection{name: name, isPlatform: isPlatform(name)}
}

var _ = Describe("TenantAwareCollection", func() {
	const tenantA = "tenant-a"

	Describe("applyFilterWithTenant", func() {
		It("injects tenant_id from context and overwrites caller-supplied tenant_id", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			filter := bson.M{"id": "ws-1", reqtenant.FieldTenantID: "forged"}

			got, err := c.applyFilterWithTenant(ctx, filter)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws-1", reqtenant.FieldTenantID: tenantA}))
			Expect(filter).To(Equal(bson.M{"id": "ws-1", reqtenant.FieldTenantID: "forged"}))
		})

		It("returns error when context has no tenant", func() {
			c := newTestCollection("workspaces")

			_, err := c.applyFilterWithTenant(context.Background(), bson.M{"id": "ws-1"})
			Expect(err).To(MatchError(reqtenant.ErrTenantIDRequired))
		})

		It("skips inject for platform collections", func() {
			c := newTestCollection("cluster_addon_defs")

			got, err := c.applyFilterWithTenant(context.Background(), bson.M{"key": "x"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"key": "x"}))
		})
	})

	Describe("applyDocumentWithTenant", func() {
		It("fills tenant_id from context and overwrites document tenant_id", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)

			got, err := c.applyDocumentWithTenant(ctx, bson.M{"id": "ws-1", reqtenant.FieldTenantID: "forged"})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.M{"id": "ws-1", reqtenant.FieldTenantID: tenantA}))
		})

		It("returns error when context has no tenant", func() {
			c := newTestCollection("workspaces")

			_, err := c.applyDocumentWithTenant(context.Background(), bson.M{"id": "ws-1"})
			Expect(err).To(MatchError(reqtenant.ErrTenantIDRequired))
		})

		It("converts struct to bson.D and overwrites tenant_id", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			doc := &struct {
				ID       string `bson:"id"`
				TenantID string `bson:"tenant_id,omitempty"`
			}{ID: "ws-1", TenantID: "forged"}

			got, err := c.applyDocumentWithTenant(ctx, doc)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.D{
				{Key: "id", Value: "ws-1"},
				{Key: reqtenant.FieldTenantID, Value: tenantA},
			}))
			Expect(doc.TenantID).To(Equal("forged"))
		})

		It("rejects unsupported document types", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)

			_, err := c.applyDocumentWithTenant(ctx, 1)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("applyPipelineWithTenant", func() {
		It("prepends tenant $match from context", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			pipeline := mongo.Pipeline{{{Key: "$match", Value: bson.M{"state": "Ready"}}}}

			got, err := c.applyPipelineWithTenant(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(mongo.Pipeline{
				{{Key: "$match", Value: bson.M{reqtenant.FieldTenantID: tenantA}}},
				{{Key: "$match", Value: bson.M{"state": "Ready"}}},
			}))
		})

		It("supports bson.A pipelines", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			pipeline := bson.A{
				bson.M{"$match": bson.M{"state": "Ready"}},
				bson.M{"$sort": bson.M{"createdAt": -1}},
			}

			got, err := c.applyPipelineWithTenant(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(bson.A{
				bson.M{"$match": bson.M{reqtenant.FieldTenantID: tenantA}},
				bson.M{"$match": bson.M{"state": "Ready"}},
				bson.M{"$sort": bson.M{"createdAt": -1}},
			}))
		})
	})

	Describe("applyDocumentsWithTenant", func() {
		It("fills tenant_id on each document from context", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			docs := []any{bson.M{"id": "ws-1"}, bson.M{"id": "ws-2", reqtenant.FieldTenantID: "forged"}}

			got, err := c.applyDocumentsWithTenant(ctx, docs)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal([]any{
				bson.M{"id": "ws-1", reqtenant.FieldTenantID: tenantA},
				bson.M{"id": "ws-2", reqtenant.FieldTenantID: tenantA},
			}))
		})
	})

	Describe("applyWriteModelsWithTenant", func() {
		It("injects tenant into insert document and update/delete/replace filters", func() {
			c := newTestCollection("workspaces")
			ctx := reqtenant.WithTenantID(context.Background(), tenantA)
			models := []mongo.WriteModel{
				mongo.NewInsertOneModel().SetDocument(bson.M{"id": "ws-1"}),
				mongo.NewUpdateOneModel().
					SetFilter(bson.M{"id": "ws-2"}).
					SetUpdate(bson.M{"$set": bson.M{"name": "n"}}),
				mongo.NewUpdateManyModel().
					SetFilter(bson.M{"state": "Ready"}).
					SetUpdate(bson.M{"$set": bson.M{"n": 1}}),
				mongo.NewDeleteOneModel().SetFilter(bson.M{"id": "ws-3"}),
				mongo.NewDeleteManyModel().SetFilter(bson.M{"state": "Disabled"}),
				mongo.NewReplaceOneModel().
					SetFilter(bson.M{"id": "ws-4"}).
					SetReplacement(bson.M{"id": "ws-4"}),
			}

			got, err := c.applyWriteModelsWithTenant(ctx, models)
			Expect(err).NotTo(HaveOccurred())
			Expect(got[0].(*mongo.InsertOneModel).Document).To(Equal(
				bson.M{"id": "ws-1", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[1].(*mongo.UpdateOneModel).Filter).To(Equal(
				bson.M{"id": "ws-2", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[1].(*mongo.UpdateOneModel).Update).To(Equal(bson.M{"$set": bson.M{"name": "n"}}))
			Expect(got[2].(*mongo.UpdateManyModel).Filter).To(Equal(
				bson.M{"state": "Ready", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[3].(*mongo.DeleteOneModel).Filter).To(Equal(
				bson.M{"id": "ws-3", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[4].(*mongo.DeleteManyModel).Filter).To(Equal(
				bson.M{"state": "Disabled", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[5].(*mongo.ReplaceOneModel).Filter).To(Equal(
				bson.M{"id": "ws-4", reqtenant.FieldTenantID: tenantA},
			))
			Expect(got[5].(*mongo.ReplaceOneModel).Replacement).To(Equal(
				bson.M{"id": "ws-4", reqtenant.FieldTenantID: tenantA},
			))
		})
	})
})
