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

// Package migrate 承载 bkiam 相关数据存储的版本化迁移逻辑，基于 golang-migrate 实现
package migrate

import (
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mongodb"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/migrate/migrations"
)

// MongoConfig describes the MongoDB connection used to persist the
// golang-migrate version-history collection.
//
// Note: this is intentionally decoupled from bkms-server's global mongo
// config, so the migrate library remains self-contained and can be reused
// by command handlers without forcing the caller to share bkms-server's
// global state.
type MongoConfig struct {
	User       string
	Password   string
	Host       string
	Port       string
	Database   string
	Collection string
}

// dsn builds a mongodb URI used by the golang-migrate mongodb backend.
func (c MongoConfig) dsn() string {
	userPassword := url.UserPassword(c.User, c.Password)
	return fmt.Sprintf(
		"mongodb://%s@%s:%s/%s?connect=direct&authSource=admin&x-migrations-collection=%s",
		userPassword.String(), c.Host, c.Port, c.Database, c.Collection,
	)
}

// Migrate runs all pending IAM-model migrations against BlueKing IAM, using
// the given mongo connection for migration-history bookkeeping and the
// given iam Config for talking to the IAM backend.
//
// Returns nil when there is nothing to apply (migrate.ErrNoChange).
func Migrate(mongoCfg MongoConfig, iamCfg Config) error {
	sourceInstance, err := iofs.New(migrations.MigrationFS, ".")
	if err != nil {
		return errors.Wrap(err, "init iam migrations source")
	}

	mongoBackend := &mongodb.Mongo{}
	d, err := mongoBackend.Open(mongoCfg.dsn())
	if err != nil {
		return errors.Wrap(err, "open mongodb migration backend")
	}
	defer func() { _ = d.Close() }()

	m, err := migrate.NewWithInstance("migrations_source", sourceInstance, "", NewIAMDriver(d, iamCfg))
	if err != nil {
		return errors.Wrap(err, "new iam migrate instance")
	}

	if err = m.Up(); err != nil {
		if err.Error() == migrate.ErrNoChange.Error() {
			return nil
		}
		return errors.Wrap(err, "run iam migrate up")
	}
	return nil
}
