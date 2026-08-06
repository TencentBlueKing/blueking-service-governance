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

// Package migrate provides a library to register / update bkms IAM models
// (resource types, instance selections, actions) into BlueKing IAM.
//
// This package is provided as a library only. Command registration is owned by
// bkms-server's migration command layer.
//
// The package depends on github.com/golang-migrate/migrate/v4 and
// github.com/TencentBlueKing/iam-go-sdk's IAMBackendClient. Note that the
// IAM backend used here is the IAM model-management endpoint, which is a
// different surface from the IAM gateway client provided by
// pkg/infras/cloudapi/iam (which targets the IsAllowed / GradeManager APIs).
package migrate

import (
	"context"
	"io"
	"time"

	"github.com/TencentBlueKing/iam-go-sdk/client"
	"github.com/TencentBlueKing/iam-go-sdk/iammigrate"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/pkg/errors"
)

// defaultMigrateTimeout is the default per-migration step timeout.
const defaultMigrateTimeout = 5 * time.Minute

// Config holds the inputs the IAM model migration needs at runtime.
//
// All fields are required. Command handlers typically populate them from
// bkms-server's global config, e.g.:
//   - BkApiGatewayURL <- iam.BuildIAMGatewayURL(
//     config.G.BkPlatUrls.BkApiUrlTmpl, config.G.BkApiStages.BkIAM,
//     ) // NOT the raw BkApiUrlTmpl template
//   - BkmsSystemID    <- config.G.BkIAMSystemIDs.Bkms
//   - AppCode         <- config.G.BkApp.Code
//   - AppSecret       <- config.G.BkApp.Secret
//   - BkmsHost        <- the externally reachable bkms-server URL that IAM
//     will use as the resource-callback base
type Config struct {
	// BkApiGatewayURL is the concrete BlueKing IAM gateway URL used by the
	// IAM backend client to reach the IAM model-management endpoint.
	//
	// NOTE: this MUST be a fully rendered URL, i.e. the `{api_name}`
	// placeholder has already been replaced with `bk-iam` and the stage
	// path (e.g. `prod`) has already been joined. It is NOT the raw
	// `BkApiUrlTmpl` template. The caller is expected to render it via
	// pkg/infras/cloudapi/iam.BuildIAMGatewayURL (kept in the iam gateway
	// client package to avoid a reverse import from this package).
	BkApiGatewayURL string
	// BkmsSystemID is the bkms system id registered in BlueKing IAM.
	BkmsSystemID string
	// AppCode is the bkms app code used to authenticate to IAM.
	AppCode string
	// AppSecret is the bkms app secret used to authenticate to IAM.
	AppSecret string
	// BkmsHost is the externally reachable bkms-server URL used by IAM as
	// the resource provider callback base, i.e. the value rendered into
	// `provider_config.host` of the system model.
	BkmsHost string
}

// IAMDriver migrates bkms IAM models into BlueKing IAM while persisting the
// migration history into the embedded migrate database driver.
type IAMDriver struct {
	database.Driver
	iamClient      client.IAMBackendClient
	cfg            Config
	migrateTimeout time.Duration
}

// NewIAMDriver creates a new IAMDriver wrapping the given migrate database
// driver, using the supplied Config to talk to the IAM backend.
func NewIAMDriver(d database.Driver, cfg Config) *IAMDriver {
	return &IAMDriver{
		Driver: d,
		iamClient: client.NewIAMBackendClient(
			cfg.BkApiGatewayURL,
			cfg.BkmsSystemID,
			cfg.AppCode,
			cfg.AppSecret,
		),
		cfg:            cfg,
		migrateTimeout: defaultMigrateTimeout,
	}
}

// Run executes a single migration step, registering the model fragment in
// IAM via the IAM backend client.
func (d *IAMDriver) Run(migration io.Reader) error {
	migr, err := io.ReadAll(migration)
	if err != nil {
		return errors.Wrap(err, "read migration content")
	}

	ctx := context.Background()
	if d.migrateTimeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.migrateTimeout)
		defer cancel()
	}

	version, _, err := d.Version()
	if err != nil {
		return errors.Wrap(err, "read current migration version")
	}

	// Render the embedded JSON template with the bkms-specific placeholders
	// and call iam-go-sdk to register the model data into IAM.
	if err = iammigrate.DoMigate(ctx, d.iamClient, migr, map[string]string{
		"BKMSSystemID": d.cfg.BkmsSystemID,
		"APP_CODE":     d.cfg.AppCode,
		"BKMS_HOST":    d.cfg.BkmsHost,
	}, version); err != nil {
		return database.Error{OrigErr: err, Err: "migration failed", Query: migr}
	}

	return nil
}
