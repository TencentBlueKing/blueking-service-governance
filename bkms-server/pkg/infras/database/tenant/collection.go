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

// Package tenant 提供 MongoDB Collection 的租户隔离包装。
// 默认从 ctx 注入 tenant_id；WithTenant 派生出绑定指定租户的视图，AllTenants 跳过注入。
package tenant

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// Scope 调整 Collection 的租户访问范围，返回新实例，不修改入参。
type Scope func(*Collection) *Collection

// WithTenant 返回只访问指定租户的 Scope，不再从 ctx 读取租户。
func WithTenant(id string) Scope {
	return func(c *Collection) *Collection {
		return c.withSpecifiedTenant(id)
	}
}

// AllTenants 返回跨租户视图，读写都不注入 tenant_id。
func AllTenants() Scope {
	return func(c *Collection) *Collection {
		next := c.clone()
		if next == nil {
			return nil
		}
		next.scope = scopeCrossTenant
		next.specifiedTenantID = nil
		return next
	}
}

// Collection 包装 mongo.Collection，按租户上下文注入过滤与写入字段。
type Collection struct {
	inner *mongo.Collection
	name  string
	// 访问视角：tenant 表示默认租户隔离，global / cross_tenant 表示跳过注入
	scope scopeMode
	// specifiedTenantID 非 nil 时使用该值，不再从 ctx 读取。nil 表示未指定。
	specifiedTenantID *string
}

// scopeMode 定义 Collection 的租户访问视角。
type scopeMode string

const (
	// scopeTenant 表示该集合按租户上下文隔离访问。
	scopeTenant scopeMode = "tenant"
	// scopeGlobal 表示该集合是全局表，没有 tenant_id，不注入。
	scopeGlobal scopeMode = "global"
	// scopeCrossTenant 表示该集合允许调用方以跨租户视角访问。
	scopeCrossTenant scopeMode = "cross_tenant"
)

// WrapTenant 包装原始集合，默认做租户隔离；若命中全局表白名单，则自动跳过注入。所有表均以此使用，不再使用 mongo.Collection
func WrapTenant(coll *mongo.Collection) *Collection {
	name := coll.Name()
	return &Collection{
		inner: coll,
		name:  name,
		scope: tenantScopeForCollection(name),
	}
}

// withSpecifiedTenant 返回指定租户的视图。全局表没有 tenant_id，保持原范围。
func (c *Collection) withSpecifiedTenant(id string) *Collection {
	next := c.clone()
	if next == nil || isGlobal(next.name) {
		return next
	}
	next.scope = scopeTenant
	next.specifiedTenantID = &id
	return next
}

func (c *Collection) clone() *Collection {
	if c == nil {
		return nil
	}
	next := *c
	return &next
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
		return singleResultErr(err)
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

// UpdateOne 在过滤条件上注入租户，并拒绝改写 tenant_id。
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
	if err = c.rejectTenantUpdate(update); err != nil {
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
// 当前实现只覆盖单集合的安全聚合；换集合的 stage，以及必须位于管道首位的
// stage，先 fail closed 直接拒绝，避免在尚未完成子管道改写前引入跨租户读风险。
//
// TODO: 后续如果 tenant collection 需要支持更复杂的聚合，按以下方向演进：
//  1. 先将传入 pipeline 标准化为统一结构，避免后续同时兼容多种表示方式。
//  2. 将 tenant 注入从“根 pipeline prepend $match”升级为“递归改写 pipeline”。
//  3. 第一批优先支持只读的跨集合 stage，例如 $lookup.pipeline / $unionWith.pipeline；
//     遇到 tenant 表时在子管道中补租户条件，全局表则保持原样。
//  4. $out / $merge 等写出型 stage 继续显式拒绝，直到有单独的语义与约束设计。
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

// UpdateMany 在过滤条件上注入租户，并拒绝改写 tenant_id。
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
	if err = c.rejectTenantUpdate(update); err != nil {
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

// FindOneAndUpdate 在过滤条件上注入租户，并拒绝改写 tenant_id。
func (c *Collection) FindOneAndUpdate(
	ctx context.Context,
	filter any,
	update any,
	opts ...options.Lister[options.FindOneAndUpdateOptions],
) *mongo.SingleResult {
	filter, err := c.applyFilterWithTenant(ctx, filter)
	if err != nil {
		return singleResultErr(err)
	}
	if err = c.rejectTenantUpdate(update); err != nil {
		return singleResultErr(err)
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
		return singleResultErr(err)
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
	if !c.shouldInjectTenant() {
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
	if !c.shouldInjectTenant() {
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
		if err = rejectUnsupportedPipelineStagesFromDocs(p); err != nil {
			return nil, c.wrapErr(err)
		}
		out := make(mongo.Pipeline, 0, len(p)+1)
		out = append(out, matchDoc)
		out = append(out, p...)
		return out, nil
	case []bson.D:
		if err = rejectUnsupportedPipelineStagesFromDocs(p); err != nil {
			return nil, c.wrapErr(err)
		}
		out := make([]bson.D, 0, len(p)+1)
		out = append(out, matchDoc)
		out = append(out, p...)
		return out, nil
	case []bson.M:
		if err = rejectUnsupportedPipelineStagesFromMaps(p); err != nil {
			return nil, c.wrapErr(err)
		}
		out := make([]bson.M, 0, len(p)+1)
		out = append(out, matchMap)
		out = append(out, p...)
		return out, nil
	case bson.A:
		if err = rejectUnsupportedPipelineStagesFromArray(p); err != nil {
			return nil, c.wrapErr(err)
		}
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
	if !c.shouldInjectTenant() {
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
	if !c.shouldInjectTenant() {
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
			return nil, c.wrapErr(errors.Wrapf(err, "inject tenant into bulk model #%d (%T)", i, model))
		}
		out[i] = injected
	}
	return out, nil
}

// singleResultErr 把错误放进 SingleResult。document 必须非 nil，
// 否则驱动丢弃 err，改返回 ErrNilDocument。
func singleResultErr(err error) *mongo.SingleResult {
	return mongo.NewSingleResultFromDocument(bson.D{}, err, nil)
}

// wrapErr 给错误加上 collection 上下文，方便排查。
func (c *Collection) wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "collection %q", c.name)
}

func (c *Collection) requireTenant(ctx context.Context) (string, error) {
	if c.specifiedTenantID != nil {
		if *c.specifiedTenantID == "" {
			return "", tenant.ErrTenantIDRequired
		}
		return *c.specifiedTenantID, nil
	}
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", tenant.ErrTenantIDRequired
	}
	return tenantID, nil
}

func (c *Collection) shouldInjectTenant() bool {
	return c.scope == scopeTenant
}

// TenantID 返回本次写入会注入的租户。全局表和跨租户视图不注入。
// 已通过 WithTenant 绑定的视图返回绑定值，不再读取 ctx。
func (c *Collection) TenantID(ctx context.Context) (string, bool) {
	if !c.shouldInjectTenant() {
		return "", false
	}
	if c.specifiedTenantID != nil {
		if *c.specifiedTenantID == "" {
			return "", false
		}
		return *c.specifiedTenantID, true
	}
	return tenant.GetTenantID(ctx)
}

func tenantScopeForCollection(name string) scopeMode {
	if isGlobal(name) {
		return scopeGlobal
	}
	return scopeTenant
}

func (c *Collection) rejectTenantUpdate(update any) error {
	if !c.shouldInjectTenant() {
		return nil
	}
	if err := ensureTenantImmutable(update); err != nil {
		return c.wrapErr(err)
	}
	return nil
}

// ensureTenantImmutable 拒绝会改写 tenant_id 的 update。
// pipeline 形式可以用 $set / $unset / $replaceWith 改写整份文档，这里整体拒绝。
func ensureTenantImmutable(update any) error {
	// pipeline update 能 $replaceWith / $set 整份文档，整段拒绝，目前项目中无此类使用方法
	switch update.(type) {
	case mongo.Pipeline, []bson.D, []bson.M, bson.A:
		return errors.New("pipeline-style update is not allowed on tenant collection")
	}

	doc, err := asBSONDocument(update)
	if err != nil {
		return errors.Wrap(err, "decode update document")
	}
	for _, op := range doc {
		// replacement 风格：顶层直接写字段，不能出现 tenant_id。
		if isTenantKey(op.Key) {
			return errors.Errorf("update must not modify %s", tenant.FieldTenantID)
		}
		if !strings.HasPrefix(op.Key, "$") {
			continue
		}
		// 操作符风格：$set / $unset / $inc 等的字段名里不能带 tenant_id。
		fields, err := asBSONDocument(op.Value)
		if err != nil {
			return errors.Wrapf(err, "decode update operator %s", op.Key)
		}
		for _, field := range fields {
			if isTenantKey(field.Key) {
				return errors.Errorf("update must not modify %s via %s", tenant.FieldTenantID, op.Key)
			}
			// $rename 的 value 是目标字段名，改成 tenant_id 同样拒绝。
			target, ok := field.Value.(string)
			if ok && op.Key == "$rename" && isTenantKey(target) {
				return errors.Errorf("update must not rename field into %s", tenant.FieldTenantID)
			}
		}
	}
	return nil
}

func isTenantKey(key string) bool {
	return key == tenant.FieldTenantID || strings.HasPrefix(key, tenant.FieldTenantID+".")
}

// allowedPipelineStages 是当前 tenant collection 明确支持的安全聚合 stage。
// 不在白名单内的 stage 可能会换集合、写到别处，或要求位于管道首位，统一拒绝。
var allowedPipelineStages = map[string]struct{}{
	"$match":   {},
	"$group":   {},
	"$sort":    {},
	"$project": {},
	"$skip":    {},
	"$limit":   {},
	"$count":   {},
	"$unwind":  {},
}

func rejectUnsupportedPipelineStagesFromDocs(stages []bson.D) error {
	for _, stage := range stages {
		for _, elem := range stage {
			if strings.HasPrefix(elem.Key, "$") {
				if err := rejectUnsupportedPipelineStage(elem.Key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func rejectUnsupportedPipelineStagesFromMaps(stages []bson.M) error {
	for _, stage := range stages {
		for key := range stage {
			if strings.HasPrefix(key, "$") {
				if err := rejectUnsupportedPipelineStage(key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func rejectUnsupportedPipelineStagesFromArray(stages bson.A) error {
	for _, stage := range stages {
		switch s := stage.(type) {
		case bson.D:
			if err := rejectUnsupportedPipelineStagesFromDocs([]bson.D{s}); err != nil {
				return err
			}
		case bson.M:
			if err := rejectUnsupportedPipelineStagesFromMaps([]bson.M{s}); err != nil {
				return err
			}
		default:
			doc, err := asBSONDocument(stage)
			if err != nil {
				return err
			}
			if err := rejectUnsupportedPipelineStagesFromDocs([]bson.D{doc}); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectUnsupportedPipelineStage(name string) error {
	if _, ok := allowedPipelineStages[name]; !ok {
		return errors.Errorf("pipeline stage %s is not allowed on tenant collection", name)
	}
	return nil
}
