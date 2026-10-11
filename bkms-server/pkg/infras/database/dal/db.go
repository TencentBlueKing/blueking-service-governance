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

// Package dal 提供带租户注入的 Mongo 查询层。
// 连接由调用方传入，本包不读取 database 全局 client。
package dal

import "go.mongodb.org/mongo-driver/v2/mongo"

// DB 是一个数据库连接。租户范围在 Table 和 Store 的 Scope 上指定。
type DB interface {
	Table(name string) Table
}

type mongoDB struct {
	client *mongo.Client
	name   string
}

// New 用已有 client 和库名构造 DB。不会新建网络连接。
func New(client *mongo.Client, dbName string) DB {
	return &mongoDB{client: client, name: dbName}
}

func (d *mongoDB) Table(name string) Table {
	return newTable(name, d.client.Database(d.name).Collection(name))
}
