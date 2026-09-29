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

// Package tenant 提供 MongoDB Collection 的租户隔离包装：查询/写入从 ctx 注入 tenant_id。
package tenant

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// Collection 包装 mongo.Collection，按租户上下文注入过滤与写入字段。
type Collection struct {
	inner      *mongo.Collection
	name       string
	isPlatform bool // 缓存：集合是否为平台级（不做租户隔离）
}

// Wrap 包装原始集合。租户 ID 一律从 ctx 读取，是否多租户由入口中间件写入 ctx。
func Wrap(coll *mongo.Collection) *Collection {
	name := coll.Name()
	return &Collection{
		inner:      coll,
		name:       name,
		isPlatform: isPlatform(name),
	}
}

// Find 按租户注入后查询。
func (c *Collection) Find(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.FindOptions],
) (*mongo.Cursor, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	return c.inner.Find(ctx, filter, opts...)
}

// FindOne 按租户注入后查询单条。
func (c *Collection) FindOne(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.FindOneOptions],
) *mongo.SingleResult {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return mongo.NewSingleResultFromDocument(nil, err, nil)
	}
	return c.inner.FindOne(ctx, filter, opts...)
}

// InsertOne 按租户填充后写入。
func (c *Collection) InsertOne(
	ctx context.Context,
	document any,
	opts ...options.Lister[options.InsertOneOptions],
) (*mongo.InsertOneResult, error) {
	document, err := c.applyDocumentWithTenant(ctx, document)
	if err != nil {
		return nil, err
	}
	return c.inner.InsertOne(ctx, document, opts...)
}

// UpdateOne 仅在过滤条件上注入租户，不改写更新内容。
func (c *Collection) UpdateOne(
	ctx context.Context,
	filter any,
	update any,
	opts ...options.Lister[options.UpdateOneOptions],
) (*mongo.UpdateResult, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	return c.inner.UpdateOne(ctx, filter, update, opts...)
}

// DeleteOne 按租户注入后删除。
func (c *Collection) DeleteOne(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.DeleteOneOptions],
) (*mongo.DeleteResult, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	return c.inner.DeleteOne(ctx, filter, opts...)
}

// CountDocuments 按租户注入后计数。
func (c *Collection) CountDocuments(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.CountOptions],
) (int64, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return 0, err
	}
	return c.inner.CountDocuments(ctx, filter, opts...)
}

// Aggregate 在管道前追加租户 $match。
func (c *Collection) Aggregate(
	ctx context.Context,
	pipeline any,
	opts ...options.Lister[options.AggregateOptions],
) (*mongo.Cursor, error) {
	pipeline, err := c.applyPipelineWithTenant(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	return c.inner.Aggregate(ctx, pipeline, opts...)
}

// InsertMany 按租户填充后批量写入。
func (c *Collection) InsertMany(
	ctx context.Context,
	documents any,
	opts ...options.Lister[options.InsertManyOptions],
) (*mongo.InsertManyResult, error) {
	documents, err := c.applyDocumentsWithTenant(ctx, documents)
	if err != nil {
		return nil, err
	}
	return c.inner.InsertMany(ctx, documents, opts...)
}

// UpdateMany 仅在过滤条件上注入租户，不改写更新内容。
func (c *Collection) UpdateMany(
	ctx context.Context,
	filter any,
	update any,
	opts ...options.Lister[options.UpdateManyOptions],
) (*mongo.UpdateResult, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	return c.inner.UpdateMany(ctx, filter, update, opts...)
}

// DeleteMany 按租户注入后批量删除。
func (c *Collection) DeleteMany(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.DeleteManyOptions],
) (*mongo.DeleteResult, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	return c.inner.DeleteMany(ctx, filter, opts...)
}

// ReplaceOne 过滤条件与替换文档都注入租户。
func (c *Collection) ReplaceOne(
	ctx context.Context,
	filter any,
	replacement any,
	opts ...options.Lister[options.ReplaceOptions],
) (*mongo.UpdateResult, error) {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return nil, err
	}
	replacement, err = c.applyDocumentWithTenant(ctx, replacement)
	if err != nil {
		return nil, err
	}
	return c.inner.ReplaceOne(ctx, filter, replacement, opts...)
}

// FindOneAndUpdate 仅在过滤条件上注入租户，不改写更新内容。
func (c *Collection) FindOneAndUpdate(
	ctx context.Context,
	filter any,
	update any,
	opts ...options.Lister[options.FindOneAndUpdateOptions],
) *mongo.SingleResult {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return mongo.NewSingleResultFromDocument(nil, err, nil)
	}
	return c.inner.FindOneAndUpdate(ctx, filter, update, opts...)
}

// FindOneAndDelete 按租户注入后查找并删除。
func (c *Collection) FindOneAndDelete(
	ctx context.Context,
	filter any,
	opts ...options.Lister[options.FindOneAndDeleteOptions],
) *mongo.SingleResult {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return mongo.NewSingleResultFromDocument(nil, err, nil)
	}
	return c.inner.FindOneAndDelete(ctx, filter, opts...)
}

// BulkWrite 按模型类型注入 filter 或 document，不改写 update payload。
func (c *Collection) BulkWrite(
	ctx context.Context,
	models []mongo.WriteModel,
	opts ...options.Lister[options.BulkWriteOptions],
) (*mongo.BulkWriteResult, error) {
	models, err := c.applyWriteModelsWithTenant(ctx, models)
	if err != nil {
		return nil, err
	}
	return c.inner.BulkWrite(ctx, models, opts...)
}

// --- 注入辅助函数 ---
func (c *Collection) applyFilterWithTenant(ctx context.Context, filter any) (any, error) {
	return c.withTenant(ctx, filter)
}

func (c *Collection) applyDocumentWithTenant(ctx context.Context, document any) (any, error) {
	return c.withTenant(ctx, document)
}

// withTenant 是 applyFilterWithTenant / applyDocumentWithTenant 的公共实现。
func (c *Collection) withTenant(ctx context.Context, in any) (any, error) {
	if c.isPlatform {
		return in, nil
	}
	tenantID, err := c.requireTenant(ctx)
	if err != nil {
		return nil, c.wrapErr(err)
	}
	out, err := setTenantField(in, tenantID)
	if err != nil {
		return nil, c.wrapErr(err)
	}
	return out, nil
}

func (c *Collection) applyPipelineWithTenant(ctx context.Context, pipeline any) (any, error) {
	if c.isPlatform {
		return pipeline, nil
	}
	tenantID, err := c.requireTenant(ctx)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	matchDoc := bson.D{{Key: "$match", Value: bson.M{tenant.FieldTenantID: tenantID}}}
	matchMap := bson.M{"$match": bson.M{tenant.FieldTenantID: tenantID}}

	switch p := pipeline.(type) {
	case mongo.Pipeline:
		out := make(mongo.Pipeline, 0, len(p)+1)
		out = append(out, matchDoc)
		out = append(out, p...)
		return out, nil
	case []bson.D:
		out := make([]bson.D, 0, len(p)+1)
		out = append(out, matchDoc)
		out = append(out, p...)
		return out, nil
	case []bson.M:
		out := make([]bson.M, 0, len(p)+1)
		out = append(out, matchMap)
		out = append(out, p...)
		return out, nil
	case bson.A:
		out := make(bson.A, 0, len(p)+1)
		out = append(out, matchMap)
		out = append(out, p...)
		return out, nil
	default:
		// 未知 pipeline 类型直接报错，不再兜底包装为 $match。
		return nil, c.wrapErr(errors.Errorf("unsupported pipeline type %T", pipeline))
	}
}

func (c *Collection) applyDocumentsWithTenant(ctx context.Context, documents any) (any, error) {
	if c.isPlatform {
		return documents, nil
	}
	tenantID, err := c.requireTenant(ctx)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	// 常见切片类型不走反射
	switch docs := documents.(type) {
	case []any:
		return c.injectDocs(docs, tenantID)
	case []bson.M:
		out := make([]any, len(docs))
		for i, d := range docs {
			injected, err := setTenantField(d, tenantID)
			if err != nil {
				return nil, c.wrapErr(err)
			}
			out[i] = injected
		}
		return out, nil
	case []bson.D:
		out := make([]any, len(docs))
		for i, d := range docs {
			injected, err := setTenantField(d, tenantID)
			if err != nil {
				return nil, c.wrapErr(err)
			}
			out[i] = injected
		}
		return out, nil
	}

	// 反射兜底
	v := reflect.ValueOf(documents)
	if v.Kind() != reflect.Slice {
		return nil, c.wrapErr(errors.Errorf("InsertMany documents must be a slice, got %T", documents))
	}
	out := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		injected, err := setTenantField(v.Index(i).Interface(), tenantID)
		if err != nil {
			return nil, c.wrapErr(err)
		}
		out[i] = injected
	}
	return out, nil
}

func (c *Collection) injectDocs(docs []any, tenantID string) (any, error) {
	out := make([]any, len(docs))
	for i, d := range docs {
		injected, err := setTenantField(d, tenantID)
		if err != nil {
			return nil, c.wrapErr(err)
		}
		out[i] = injected
	}
	return out, nil
}

func (c *Collection) applyWriteModelsWithTenant(
	ctx context.Context,
	models []mongo.WriteModel,
) ([]mongo.WriteModel, error) {
	if c.isPlatform {
		return models, nil
	}
	tenantID, err := c.requireTenant(ctx)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	out := make([]mongo.WriteModel, len(models))
	for i, model := range models {
		injected, err := injectWriteModel(model, tenantID)
		if err != nil {
			return nil, c.wrapErr(err)
		}
		out[i] = injected
	}
	return out, nil
}

// wrapErr 给错误加上 collection 上下文，方便排查。
func (c *Collection) wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "collection %q", c.name)
}

func (c *Collection) requireTenant(ctx context.Context) (string, error) {
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", tenant.ErrTenantIDRequired
	}
	return tenantID, nil
}
