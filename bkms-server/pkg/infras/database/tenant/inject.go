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
	"reflect"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// setTenantField 返回一份新的值，附带/覆盖 tenant_id 字段；不会修改入参。
//
// 支持的入参类型：
//   - nil：返回 bson.M{tenant_id: tenantID}
//   - bson.M：返回新的 bson.M
//   - map[string]any：返回新的 map[string]any
//   - bson.D：返回新的 bson.D（去重后追加）
//   - 其他：通过 bson.Marshal/Unmarshal 转成 bson.D 处理
func setTenantField(value any, tenantID string) (any, error) {
	field := tenant.FieldTenantID

	switch v := value.(type) {
	case nil:
		// nil filter 是合法用法（等价于匹配全部），构造只含 tenant_id 的过滤条件。
		return bson.M{field: tenantID}, nil

	case bson.M:
		out := make(bson.M, len(v)+1)
		for k, item := range v {
			if k == field { // 与 bson.D 分支保持一致：显式去重
				continue
			}
			out[k] = item
		}
		out[field] = tenantID
		return out, nil

	case map[string]any:
		out := make(map[string]any, len(v)+1)
		for k, item := range v {
			if k == field {
				continue
			}
			out[k] = item
		}
		out[field] = tenantID
		return out, nil

	case bson.D:
		return replaceDocumentField(v, field, tenantID), nil

	default:
		doc, err := asBSONDocument(value)
		if err != nil {
			return nil, err
		}
		return replaceDocumentField(doc, field, tenantID), nil
	}
}

// replaceDocumentField 在 bson.D 中删除同名字段后追加新值。
func replaceDocumentField(document bson.D, field string, value any) bson.D {
	out := make(bson.D, 0, len(document)+1)
	for _, elem := range document {
		if elem.Key == field {
			continue
		}
		out = append(out, elem)
	}
	return append(out, bson.E{Key: field, Value: value})
}

// asBSONDocument 通过 bson.Marshal/Unmarshal 将任意 struct/指针转成 bson.D。
// bson.M / bson.D 由上层直接处理，这里不再重复分支。
func asBSONDocument(value any) (bson.D, error) {
	if isNilValue(value) {
		return nil, errors.Errorf("cannot convert nil %T to BSON document", value)
	}
	raw, err := bson.Marshal(value)
	if err != nil {
		return nil, errors.Wrapf(err, "marshal %T as BSON document", value)
	}
	var document bson.D
	if err := bson.Unmarshal(raw, &document); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %T as BSON document", value)
	}
	return document, nil
}

// isNilValue 同时识别 untyped nil 和 typed nil。
func isNilValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	}
	return false
}
