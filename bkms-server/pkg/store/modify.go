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

package store

import (
	"context"
	"slices"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// modifyOps Modify 接受的更新操作符，按字母序排列。其中 $set / $unset 只能用于数组位置路径，
// 普通字段须走 Update，从而经过 Encode
var modifyOps = []string{
	"$addToSet", "$inc", "$max", "$min", "$mul", "$pop", "$pull", "$pullAll", "$push", "$set", "$unset",
}

// ModifyResult Modify / ModifyMany 的结果
type ModifyResult struct {
	// Matched 匹配的文档数量
	Matched int64
	// Modified 实际发生变化的文档数量，如 $addToSet 的元素已存在、$pull 没有可删除的元素时不计入。
	// 刷新 updatedAt 也算变化，因此只有传入 KeepUpdatedAt 时它才能反映操作符本身是否生效
	Modified int64
}

// Modify 用 update 中的操作符修改匹配的一个文档，用于 Update 无法表达的数组与计数类修改；不存在时返回 NotFound 错误。
//
// update 的 key 只能是以下操作符：$push、$addToSet、$pull、$pullAll、$pop、$inc、$min、$max、$mul，
// 以及作用于数组位置路径（含 $、$[]、$[id] 段）的 $set / $unset；普通字段的 $set / $unset 请使用 Update，
// 以免绕过 Encode。操作符中的字段名可以写 bson 路径或 Go 字段名，必须存在于模型中，_id 与 createdAt 不能修改。
// 字段的值原样传给 MongoDB：不经过 Encode，其中的字段名也不做转换，需要加密的值须由调用方处理。
// updatedAt 默认刷新为当前时间，传入 KeepUpdatedAt 时不刷新；唯一键冲突时返回 Duplicate 错误。例如：
//
//	// 将环境加入应用的可见环境，已存在时不重复添加
//	_, err := apps.Modify(ctx, bson.M{"$addToSet": bson.M{"visibleEnvNames": env}}, ByID(appID))
//	// 删除指定名称的组件，res.Modified 为 0 表示组件不存在
//	res, err := models.Modify(ctx, bson.M{"$pull": bson.M{"components": bson.M{"name": name}}},
//		ByApp(appID), store.KeepUpdatedAt())
//
// 支持的选项：CondOpt、TimestampOpt、ArrayFilterOpt
func (s *GenericStore[T]) Modify(ctx context.Context, update bson.M, opts ...Opt) (ModifyResult, error) {
	q, doc, err := s.prepareModify("modify", update, opts)
	if err != nil {
		return ModifyResult{}, err
	}
	updateOpts := options.UpdateOne()
	if len(q.arrayFilters) > 0 {
		updateOpts.SetArrayFilters(q.arrayFilters)
	}
	res, err := s.coll.UpdateOne(ctx, q.filter(), doc, updateOpts)
	if err != nil {
		return ModifyResult{}, s.wrapWriteErr(err, "modify")
	}
	if res.MatchedCount == 0 {
		return ModifyResult{}, s.notFound(q)
	}
	return ModifyResult{Matched: res.MatchedCount, Modified: res.ModifiedCount}, nil
}

// ModifyMany 用 update 中的操作符修改全部匹配的文档，没有匹配时不返回错误。
// update 的规则同 Modify。支持的选项：CondOpt、TimestampOpt、ArrayFilterOpt
func (s *GenericStore[T]) ModifyMany(ctx context.Context, update bson.M, opts ...Opt) (ModifyResult, error) {
	q, doc, err := s.prepareModify("modify many in", update, opts)
	if err != nil {
		return ModifyResult{}, err
	}
	updateOpts := options.UpdateMany()
	if len(q.arrayFilters) > 0 {
		updateOpts.SetArrayFilters(q.arrayFilters)
	}
	res, err := s.coll.UpdateMany(ctx, q.filter(), doc, updateOpts)
	if err != nil {
		return ModifyResult{}, s.wrapWriteErr(err, "modify many in")
	}
	return ModifyResult{Matched: res.MatchedCount, Modified: res.ModifiedCount}, nil
}

func (s *GenericStore[T]) prepareModify(op string, update bson.M, opts []Opt) (*query, bson.D, error) {
	q, err := s.writeQuery(op, opts, modifyOpts)
	if err != nil {
		return nil, nil, err
	}
	doc, err := s.modifyDoc(update, q.keepUpdatedAt)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "%s %s", op, s.coll.Name())
	}
	return q, doc, nil
}

// modifyDoc 校验 update 中的操作符与字段，将字段名转换为 bson 路径，并按需追加刷新 updatedAt 的 $set
func (s *GenericStore[T]) modifyDoc(update bson.M, keepUpdatedAt bool) (bson.D, error) {
	if len(update) == 0 {
		return nil, ErrNoFieldsToUpdate
	}
	doc := make(bson.D, 0, len(update)+1)
	for _, e := range sortedDoc(update) {
		if !isOperatorKey(e.Key) {
			return nil, errors.Errorf("Modify only accepts update operators, got field %q, use Update instead", e.Key)
		}
		if !slices.Contains(modifyOps, e.Key) {
			return nil, errors.Errorf(
				"update operator %q is not supported by Modify, expected one of %v",
				e.Key,
				modifyOps,
			)
		}
		fields, ok := toDoc(e.Value)
		if !ok || len(fields) == 0 {
			return nil, errors.Errorf("value of %s must be a non-empty document of fields", e.Key)
		}
		mapped := make(bson.D, 0, len(fields))
		for _, f := range fields {
			p, err := s.schema.resolveUpdatePath(f.Key)
			if err != nil {
				return nil, err
			}
			if (e.Key == "$set" || e.Key == "$unset") && !p.Positional() {
				return nil, errors.Errorf(
					"%s of %q is only supported for positional array paths in Modify, use Update so that Encode is applied",
					e.Key,
					f.Key,
				)
			}
			mapped = append(mapped, bson.E{Key: p.String(), Value: f.Value})
		}
		doc = append(doc, bson.E{Key: e.Key, Value: mapped})
	}

	if s.schema.updatedAt == nil || keepUpdatedAt {
		return doc, nil
	}
	touch := bson.E{Key: keyUpdatedAt, Value: time.Now()}
	if i := slices.IndexFunc(doc, func(e bson.E) bool { return e.Key == "$set" }); i >= 0 {
		doc[i].Value = append(doc[i].Value.(bson.D), touch)
		return doc, nil
	}
	return append(doc, bson.E{Key: "$set", Value: bson.D{touch}}), nil
}
