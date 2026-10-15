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
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// logicalOps 子条件同样以字段名为 key 的逻辑操作符
var logicalOps = []string{"$and", "$or", "$nor"}

// isOperatorKey 判断 key 是否是 $in、$and 这样的操作符
func isOperatorKey(key string) bool {
	return strings.HasPrefix(key, "$")
}

// toDoc 将 bson.D / bson.M / map[string]any 形式的文档统一为 bson.D，map 按 key 排序；其它类型返回 false
func toDoc(v any) (bson.D, bool) {
	switch d := v.(type) {
	case bson.D:
		return d, true
	case bson.M:
		return sortedDoc(d), true
	case map[string]any:
		return sortedDoc(d), true
	}
	return nil, false
}

// sortedDoc 将 map 转换为按 key 排序的 bson.D，保证生成的条件与错误信息稳定
func sortedDoc(m map[string]any) bson.D {
	keys := lo.Keys(m)
	slices.Sort(keys)
	return lo.Map(keys, func(k string, _ int) bson.E { return bson.E{Key: k, Value: m[k]} })
}

// toArray 将 bson.A、[]bson.D、[]bson.M 等数组统一为 []any；bson.D 是文档而不是数组，返回 false
func toArray(v any) ([]any, bool) {
	switch a := v.(type) {
	case bson.A:
		return a, true
	case bson.D, []byte:
		return nil, false
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

// documentHasKey 只沿查询操作符递归查找字段，普通字段值中的同名 key 属于嵌套文档，不是顶层条件
func documentHasKey(doc bson.D, key string) bool {
	for _, e := range doc {
		if e.Key == key {
			return true
		}
		if isOperatorKey(e.Key) && valueHasKey(e.Value, key) {
			return true
		}
	}
	return false
}

func valueHasKey(value any, key string) bool {
	if d, ok := toDoc(value); ok {
		return documentHasKey(d, key)
	}
	items, _ := toArray(value)
	return lo.ContainsBy(items, func(item any) bool { return valueHasKey(item, key) })
}

// isOperatorDoc 判断条件的值是否是 {"$in": ...} 这样的操作符文档
func isOperatorDoc(v any) bool {
	d, ok := toDoc(v)
	return ok && lo.ContainsBy(d, func(e bson.E) bool { return isOperatorKey(e.Key) })
}

// equalityKeys 返回过滤条件中以等值方式出现的字段（含 $and 中的），upsert 插入文档时 MongoDB 会写入这些字段。
// 只识别直接写值的条件，{"$eq": v} 等操作符形式有意不识别，见 Upsert 的说明
func equalityKeys(conds bson.D) map[string]bool {
	keys := map[string]bool{}
	for _, e := range conds {
		if e.Key == "$and" {
			items, _ := toArray(e.Value)
			for _, item := range items {
				if d, ok := toDoc(item); ok {
					maps.Copy(keys, equalityKeys(d))
				}
			}
			continue
		}
		if !isOperatorKey(e.Key) && !isOperatorDoc(e.Value) {
			keys[e.Key] = true
		}
	}
	return keys
}

// mapFilterKeys 用 fn 转换过滤条件中的字段名。除 $and / $or / $nor 中的子条件外，其余操作符
// 及字段的值（包括 $elemMatch 等操作符的参数）都原样保留
func mapFilterKeys(conds bson.D, fn func(string) (string, error)) (bson.D, error) {
	if len(conds) == 0 {
		return conds, nil
	}
	out := make(bson.D, 0, len(conds))
	for _, e := range conds {
		var err error
		switch {
		case slices.Contains(logicalOps, e.Key):
			e.Value, err = mapSubFilterKeys(e.Value, fn)
		case !isOperatorKey(e.Key):
			e.Key, err = fn(e.Key)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// mapSubFilterKeys 转换逻辑操作符的子条件数组，无法识别的形态原样返回
func mapSubFilterKeys(value any, fn func(string) (string, error)) (any, error) {
	items, ok := toArray(value)
	if !ok {
		return value, nil
	}
	out := make(bson.A, 0, len(items))
	for _, item := range items {
		sub, ok := toDoc(item)
		if !ok {
			return value, nil
		}
		mapped, err := mapFilterKeys(sub, fn)
		if err != nil {
			return nil, err
		}
		out = append(out, mapped)
	}
	return out, nil
}

// updateDoc 组装更新文档，跳过空的操作符
func updateDoc(set, unset, onInsert bson.M) bson.M {
	update := bson.M{}
	for op, fields := range map[string]bson.M{"$set": set, "$unset": unset, "$setOnInsert": onInsert} {
		if len(fields) > 0 {
			update[op] = fields
		}
	}
	return update
}
