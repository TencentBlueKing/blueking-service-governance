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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// setTenantField 把 tenant_id 写进 filter 或 document 的副本。
// 调用方传入的 map、bson.D、结构体都不会被改写。
func setTenantField(value any, tenantID string) (any, error) {
	switch v := value.(type) {
	case nil:
		return bson.M{tenant.FieldTenantID: tenantID}, nil
	case bson.M:
		copied := make(bson.M, len(v)+1)
		for key, item := range v {
			copied[key] = item
		}
		copied[tenant.FieldTenantID] = tenantID
		return copied, nil
	case map[string]any:
		copied := make(bson.M, len(v)+1)
		for key, item := range v {
			copied[key] = item
		}
		copied[tenant.FieldTenantID] = tenantID
		return copied, nil
	case bson.D:
		copied := make(bson.D, 0, len(v)+1)
		for _, elem := range v {
			if elem.Key == tenant.FieldTenantID {
				continue
			}
			copied = append(copied, elem)
		}
		copied = append(copied, bson.E{Key: tenant.FieldTenantID, Value: tenantID})
		return copied, nil
	default:
		raw, err := bson.Marshal(v)
		if err != nil {
			return nil, errors.Wrap(err, "marshal document")
		}
		var doc bson.D
		if err = bson.Unmarshal(raw, &doc); err != nil {
			return nil, errors.Wrap(err, "unmarshal document")
		}
		return setTenantField(doc, tenantID)
	}
}
