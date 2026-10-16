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
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// FxModule 向整个 fx 应用公开提供基于全局数据库连接的 *mongo.Database，一个 fx 应用中只应引入一次，
// 主要供测试通过 fx.Populate 获取 db 或 Store。
// 功能模块无需依赖它：database.PrivateFxModule 已在模块内部私有地提供 *mongo.Database
var FxModule = fx.Module("store",
	fx.Provide(database.DB),
)

// Provide 以 *GenericStore[T] 的形式向 fx 提供模型 T 的 Store。
// Store 在应用启动时立即构造，使 Meta 配置错误在启动阶段暴露
func Provide[T any](meta Meta[T]) fx.Option {
	return fx.Options(
		fx.Provide(func(db *mongo.Database) (*GenericStore[T], error) { return New(db, meta) }),
		fx.Invoke(func(*GenericStore[T]) {}),
	)
}
