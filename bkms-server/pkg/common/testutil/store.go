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

package testutil

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// StoreTB NewStore / NewDB 所需的测试上下文，GinkgoT() 与 *testing.T 均满足
type StoreTB interface {
	fxtest.TB
	Cleanup(func())
}

// NewStore 通过 fx 构造模型 T 的 GenericStore，使用 SetUpGlobalDatabase 初始化的测试数据库，
// fx 应用在测试结束时自动停止。用法：
//
//	projects := testutil.NewStore(GinkgoT(), bkci.ProjectMeta)
func NewStore[T any](t StoreTB, meta store.Meta[T]) *store.GenericStore[T] {
	var s *store.GenericStore[T]
	app := fxtest.New(t, store.FxModule, store.Provide(meta), fx.Populate(&s))
	app.RequireStart()
	t.Cleanup(app.RequireStop)
	return s
}

// CleanupMetaColl 清空 meta 描述的集合中的全部文档。清理有 Meta 的集合时优先使用它，
func CleanupMetaColl[T any](meta store.Meta[T]) error {
	return CleanupCollection(meta.Collection)
}

// NewDB 通过 fx 获取 SetUpGlobalDatabase 初始化的测试数据库，供接收 *mongo.Database 的函数使用，
// fx 应用在测试结束时自动停止。用法：
//
//	db := testutil.NewDB(GinkgoT())
func NewDB(t StoreTB) *mongo.Database {
	var db *mongo.Database
	app := fxtest.New(t, store.FxModule, fx.Populate(&db))
	app.RequireStart()
	t.Cleanup(app.RequireStop)
	return db
}
