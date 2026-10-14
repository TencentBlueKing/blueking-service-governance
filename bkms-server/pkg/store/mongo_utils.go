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
	"strings"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// documentHasKey 只沿查询操作符递归查找字段，普通字段值中的同名 key 属于嵌套文档，不是顶层条件
func documentHasKey(doc bson.D, key string) bool {
	for _, e := range doc {
		if e.Key == key {
			return true
		}
		if strings.HasPrefix(e.Key, "$") && valueHasKey(e.Value, key) {
			return true
		}
	}
	return false
}

func valueHasKey(value any, key string) bool {
	switch v := value.(type) {
	case bson.D:
		return documentHasKey(v, key)
	case bson.M:
		for field, nested := range v {
			if field == key || (strings.HasPrefix(field, "$") && valueHasKey(nested, key)) {
				return true
			}
		}
	case bson.A:
		return lo.ContainsBy(v, func(item any) bool { return valueHasKey(item, key) })
	}
	return false
}
