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

// Package database 数据库相关封装，目前基于 mongodb 数据库
package database

import (
	"context"
	"net"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

var (
	// The global mongo client instance, the client use a connection pool under the hood
	// so it is safe to use a single instance across the application
	// See: https://www.mongodb.com/docs/manual/administration/connection-pool-overview/
	mongoClient *mongo.Client
	dbName      string

	initOnce sync.Once
)

// Client 获取 mongodb 客户端
func Client() *mongo.Client {
	if mongoClient == nil {
		log.Fatal("mongodb client not initialized")
	}
	return mongoClient
}

// Name 获取 mongodb 数据库名
func Name() string {
	if dbName == "" {
		log.Fatal("mongodb database not initialized")
	}
	return dbName
}

// InitClient 初始化
func InitClient(ctx context.Context, cfg config.MongoConfig) {
	if mongoClient != nil {
		return
	}
	initOnce.Do(func() {
		var err error
		// 自动为所有 MongoDB 操作生成 OTel 子 Span
		clientOptions := options.Client().ApplyURI(cfg.GetURI()).SetMonitor(otelmongo.NewMonitor())
		// Set the global database client object
		mongoClient, err = mongo.Connect(clientOptions)
		if err != nil {
			log.Fatalf("failed to create mongo client: %v", err)
		}
		if err = mongoClient.Ping(ctx, nil); err != nil {
			log.Fatalf("failed to ping mongo client: %v", err)
		} else {
			log.Infof(ctx, "connected to mongo at %s", net.JoinHostPort(cfg.Host, cfg.Port))
		}
		// Set the global database name
		dbName = cfg.Database
	})
}
