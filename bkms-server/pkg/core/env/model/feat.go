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

package model

import (
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// FeatureEnvCounterCollection 特性环境序号计数器的集合名，文档的 _id 为应用 ID
const FeatureEnvCounterCollection = "feature_environment_counters"

// NewFeatureEnvCounter 返回为特性环境按应用分配递增序号的计数器，Next 的 key 为应用 ID
func NewFeatureEnvCounter(db *mongo.Database) *store.Counter {
	return store.NewCounter(db, FeatureEnvCounterCollection)
}
