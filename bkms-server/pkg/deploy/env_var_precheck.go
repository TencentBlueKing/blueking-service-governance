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

package deploy

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

const envVarPreCheckImage = "precheck.invalid/bkms:latest"

// EnvVarPreCheckResult contains all referenced but undefined env vars.
type EnvVarPreCheckResult struct {
	UndefinedVars []envvarrefs.UndefinedEnvVar
}

// EnvVarPreChecker checks env var references against variables available to an app deployment.
type EnvVarPreChecker struct {
	appModelStore  appmodel.AppModelStore
	builderService *workload.BuilderService
}

// NewEnvVarPreChecker creates an EnvVarPreChecker.
func NewEnvVarPreChecker(
	appModelStore appmodel.AppModelStore,
	builderService *workload.BuilderService,
) *EnvVarPreChecker {
	return &EnvVarPreChecker{
		appModelStore:  appModelStore,
		builderService: builderService,
	}
}

// Check checks the effective application configuration for undefined env var references.
// It performs a complete in-memory workload build and returns its undefined-variable report.
func (c *EnvVarPreChecker) Check(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) (*EnvVarPreCheckResult, error) {
	if app == nil || env == nil {
		return nil, errors.New("app and environment are required")
	}
	appModel, err := c.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s model", app.ID)
	}
	// Persisted models do not contain the deployment image. Use a copy with a placeholder
	// so the full build can run without persisting it.
	modelForBuild := *appModel
	modelForBuild.Workload.Image = envVarPreCheckImage
	buildResult, err := workload.NewBuilder(c.builderService, app, &modelForBuild).Build(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "building workload for deployment env var pre-check")
	}
	return &EnvVarPreCheckResult{UndefinedVars: buildResult.UndefinedEnvVars}, nil
}
