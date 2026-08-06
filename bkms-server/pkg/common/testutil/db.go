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
	"context"
	"os"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	dbmigration "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database/migration"
)

const (
	// EnvServerConfigPath is the environment variable key for the server configuration file path,
	// It's used for setting up the test environment.
	//
	// Q: Why not use the flag "--svrCfg" like other commands did?
	// A: It's would be too complicated for ginkgo command to support the flag.
	EnvServerConfigPath = "BKMS_SERVER_CONFIG_PATH"
)

// SetUpGlobalDatabase sets up the global database client for the project, it tries to connect
// to the database specified by server configuration file.
//
// IMPORTANT: It didn't connect to the database in the config file, but to a new test database
// that is created for testing purpose. For example, the database name in config is "bkms", then
// the test database name would be "bkms_for_test".
func SetUpGlobalDatabase() error {
	svrCfgPath := os.Getenv(EnvServerConfigPath)
	if svrCfgPath == "" {
		return errors.New("server config path missing")
	}
	ctx := context.Background()
	cfg, err := config.Load(ctx, svrCfgPath)
	if err != nil {
		return errors.Wrap(err, "loading config")
	}
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return errors.Wrap(err, "init logger")
	}

	// Force using kubeconfig for cluster access in tests
	// TODO: Update tests to not depend on this setting if possible
	cfg.Development.UseKubeConfigCluster = true

	// Change the database name and initialize the global database client
	mongoCfg := cfg.Mongo
	mongoCfg.Database = mongoCfg.Database + "_for_test"
	log.Infof(ctx, "Initializing the database, the DATABASE NAME is %s", mongoCfg.Database)
	log.Infof(ctx, "Applying all pending database migrations to %s", mongoCfg.Database)
	if err = dbmigration.UpAll(ctx, mongoCfg, cfg.Development.AllowSkipNewerDBMigration); err != nil {
		return errors.Wrap(err, "apply test database migrations")
	}
	database.InitClient(ctx, mongoCfg)
	return nil
}

// CleanupCollection removes all documents from the specified collection.
// This is useful for cleaning up test data to avoid issues like decryption errors
// when data was encrypted with different keys in previous test runs.
func CleanupCollection(collectionName string) error {
	if database.Client() == nil {
		return errors.New("database client not initialized")
	}

	ctx := context.Background()
	collection := database.Client().Database(database.Name()).Collection(collectionName)
	_, err := collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return errors.Wrapf(err, "deleting documents from collection %s", collectionName)
	}

	log.Infof(ctx, "Cleaned up collection: %s", collectionName)
	return nil
}

// TeardownGlobalDatabase is supposed to tear down the global database; it did nothing in current implementation.
// TODO: drop the test database created in SetUpGlobalDatabase if configured to.
func TeardownGlobalDatabase() error {
	// Disconnect the global database client
	if database.Client() != nil {
		_ = database.Client().Disconnect(context.Background())
	}
	return nil
}
