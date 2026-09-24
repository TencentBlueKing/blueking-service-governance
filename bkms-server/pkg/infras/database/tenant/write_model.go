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

// injectWriteModel 根据模型类型注入 filter 或 document，并返回一份拷贝。
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
		return copyUpdateOne(m, tenantID)
	case *mongo.UpdateManyModel:
		return copyUpdateMany(m, tenantID)
	case *mongo.DeleteOneModel:
		return copyDeleteOne(m, tenantID)
	case *mongo.DeleteManyModel:
		return copyDeleteMany(m, tenantID)

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

// —— 下面几个是纯 boilerplate 的收敛，只做 Filter 注入 ——

func copyUpdateOne(m *mongo.UpdateOneModel, tenantID string) (*mongo.UpdateOneModel, error) {
	f, err := setTenantField(m.Filter, tenantID)
	if err != nil {
		return nil, err
	}
	cp := *m
	cp.Filter = f
	return &cp, nil
}

func copyUpdateMany(m *mongo.UpdateManyModel, tenantID string) (*mongo.UpdateManyModel, error) {
	f, err := setTenantField(m.Filter, tenantID)
	if err != nil {
		return nil, err
	}
	cp := *m
	cp.Filter = f
	return &cp, nil
}

func copyDeleteOne(m *mongo.DeleteOneModel, tenantID string) (*mongo.DeleteOneModel, error) {
	f, err := setTenantField(m.Filter, tenantID)
	if err != nil {
		return nil, err
	}
	cp := *m
	cp.Filter = f
	return &cp, nil
}

func copyDeleteMany(m *mongo.DeleteManyModel, tenantID string) (*mongo.DeleteManyModel, error) {
	f, err := setTenantField(m.Filter, tenantID)
	if err != nil {
		return nil, err
	}
	cp := *m
	cp.Filter = f
	return &cp, nil
}
