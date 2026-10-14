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

package database

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
)

// PrivateFxModule exposes the shared database client only inside the enclosing fx.Module.
//
// *mongo.Database is provided for GenericStore based models (store.Of(db, Meta)). Because all
// providers are private, every fx.Module may include PrivateFxModule independently, and it does
// not conflict with a public *mongo.Database provided elsewhere in the app (e.g. store.FxModule).
var PrivateFxModule = fx.Options(
	fx.Provide(
		func() *mongo.Client { return Client() },
		func() string { return Name() },
		func() *mongo.Database { return DB() },
		fx.Private,
	),
)
