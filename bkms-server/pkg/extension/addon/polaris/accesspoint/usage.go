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

package accesspoint

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ConfigUsage 一份固定使用某个接入点的北极星配置
type ConfigUsage struct {
	// AppID 配置所属应用 ID
	AppID string
	// ConfigName 配置名称
	ConfigName string
}

// ConfigUsageLister 返回使用指定 accessPointID 的全部北极星配置，没有时返回空切片。
//
// 北极星配置的 accessPointID 创建后不变，即使生效环境已清空、集群已解绑，服务仍在该接入点上，
// 因此删除接入点时除了绑定集群，还必须检查这里，否则配置会因接入点解析失败而无法再删除服务。
//
// 实现是 polaris.ListConfigUsagesByAccessPoint，由 handler 传给删除逻辑：
// addon/polaris 依赖本包，本包不能直接查询北极星配置。
type ConfigUsageLister func(ctx context.Context, db *mongo.Database, accessPointID string) ([]ConfigUsage, error)
