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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// injectWriteModel 按模型类型把一条 BulkWrite 模型改写成带租户约束的拷贝。
//
// 规则与单条写接口保持一致：
//   - InsertOne：给 Document 注入 tenant_id
//   - Update/Delete：只给 Filter 注入 tenant_id
//   - ReplaceOne：同时给 Filter 和 Replacement 注入 tenant_id
//
// 调用方传入的原始 model 不会被修改，可安全复用原始切片。
func injectWriteModel(model mongo.WriteModel, tenantID string) (mongo.WriteModel, error) {
	if model == nil {
		return nil, errors.New("bulk write model cannot be nil")
	}

	switch m := model.(type) {
	case *mongo.InsertOneModel:
		doc, err := setTenantField(m.Document, tenantID)
		if err != nil {
			return nil, err
		}
		cp := *m
		cp.Document = doc
		return &cp, nil

	case *mongo.UpdateOneModel:
		return copyWriteModelWithInjectedFilter(m, m.Filter, tenantID, func(cp *mongo.UpdateOneModel, filter any) {
			cp.Filter = filter
		})
	case *mongo.UpdateManyModel:
		return copyWriteModelWithInjectedFilter(m, m.Filter, tenantID, func(cp *mongo.UpdateManyModel, filter any) {
			cp.Filter = filter
		})
	case *mongo.DeleteOneModel:
		return copyWriteModelWithInjectedFilter(m, m.Filter, tenantID, func(cp *mongo.DeleteOneModel, filter any) {
			cp.Filter = filter
		})
	case *mongo.DeleteManyModel:
		return copyWriteModelWithInjectedFilter(m, m.Filter, tenantID, func(cp *mongo.DeleteManyModel, filter any) {
			cp.Filter = filter
		})

	case *mongo.ReplaceOneModel:
		filter, err := setTenantField(m.Filter, tenantID)
		if err != nil {
			return nil, err
		}
		replacement, err := setTenantField(m.Replacement, tenantID)
		if err != nil {
			return nil, err
		}
		cp := *m
		cp.Filter = filter
		cp.Replacement = replacement
		return &cp, nil

	default:
		return nil, errors.Errorf("unsupported bulk write model %T", model)
	}
}

// copyWriteModelWithInjectedFilter 复制一条写模型，并仅改写其中的 Filter。
//
// Bulk 的 update/delete 模型共用同一条租户规则：只收窄命中文档范围，不改写
// update payload 本身，避免把 tenant 字段错误写进 $set / $inc 等更新语义中。
func copyWriteModelWithInjectedFilter[T any](
	model *T,
	filter any,
	tenantID string,
	setFilter func(*T, any),
) (*T, error) {
	f, err := setTenantField(filter, tenantID)
	if err != nil {
		return nil, err
	}
	cp := *model
	setFilter(&cp, f)
	return &cp, nil
}
