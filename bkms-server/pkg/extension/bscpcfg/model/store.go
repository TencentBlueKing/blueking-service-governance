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

// Package model 定义了应用配置管理相关的方法。
package model

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// GetSnapshot 获取指定 app+env 的聚合快照（Metadata + EnvBinding）。
// 如果 Metadata 不存在或 EnvBinding 不存在，返回 nil, nil。
func GetSnapshot(ctx context.Context, db *mongo.Database, appID, envName string) (*Snapshot, error) {
	meta, err := store.Of(db, MetadataMeta).Get(ctx, ByApp(appID))
	if err != nil {
		if errors.Is(err, ErrMetadataNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get metadata of app %s", appID)
	}

	binding, err := store.Of(db, EnvBindingMeta).Get(ctx, ByAppAndEnv(appID, envName))
	if err != nil {
		if errors.Is(err, ErrEnvBindingNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get env binding of app %s env %s", appID, envName)
	}

	return &Snapshot{Metadata: meta, EnvBinding: binding}, nil
}
