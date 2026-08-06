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

package migrate

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const componentDefsCollection = "component_defs"

type currentComponentDef struct {
	Name     string   `bson:"name"`
	Version  string   `bson:"version"`
	Output   *string  `bson:"output"`
	Patchers []string `bson:"patchers"`
	Specs    []string `bson:"specs"`
}

func (m *Migrator) listCurrentComponentDefs(ctx context.Context) ([]currentComponentDef, error) {
	cursor, err := m.db.Collection(componentDefsCollection).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{
		{Key: "name", Value: 1}, {Key: "version", Value: 1},
	}))
	if err != nil {
		return nil, fmt.Errorf("list component definitions: %w", err)
	}
	defer cursor.Close(ctx)

	items := make([]currentComponentDef, 0)
	for cursor.Next(ctx) {
		var item currentComponentDef
		if err = cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode component definition: %w", err)
		}
		if item.Name == "" || item.Version == "" {
			return nil, fmt.Errorf("component definition has empty name or version")
		}
		items = append(items, item)
	}
	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate component definitions: %w", err)
	}
	return items, nil
}
