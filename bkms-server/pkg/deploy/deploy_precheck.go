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
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

const deployPreCheckImage = "precheck.invalid/bkms:latest"

// DeployPreCheckResult contains deployment pre-check findings.
type DeployPreCheckResult struct {
	UndefinedVars                []envvarrefs.UndefinedEnvVar
	MissingRequiredClusterAddons []clusteraddon.AddonReference
}

// DeployPreChecker collects deployment pre-check findings for an app and environment.
type DeployPreChecker struct {
	appModels        *appmodel.AppModelStore
	builderService   *workload.BuilderService
	clusterAddonDefs *clusteraddon.ClusterAddonDefStore
}

// NewDeployPreChecker creates a DeployPreChecker.
func NewDeployPreChecker(
	builderService *workload.BuilderService,
	db *mongo.Database,
) *DeployPreChecker {
	return &DeployPreChecker{
		appModels:        store.Of(db, appmodel.AppModelMeta),
		builderService:   builderService,
		clusterAddonDefs: store.Of(db, clusteraddon.ClusterAddonDefMeta),
	}
}

// Check reports undefined env vars and missing required cluster addons.
func (c *DeployPreChecker) Check(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) (*DeployPreCheckResult, error) {
	appModel, err := c.appModels.Get(ctx, appmodel.ByApp(app.ID))
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s model", app.ID)
	}
	// Persisted models do not contain the deployment image. Use a copy with a placeholder
	// so the full build can run without persisting it.
	modelForBuild := *appModel
	modelForBuild.Workload.Image = deployPreCheckImage
	buildResult, err := workload.NewBuilder(c.builderService, app, &modelForBuild).Build(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "building workload for deployment pre-check")
	}
	defs, err := clusteraddon.ListDefs(ctx, c.clusterAddonDefs)
	if err != nil {
		return nil, err
	}
	missing, err := clusteraddon.InspectRequiredAddons(
		ctx, defs, app.Type, env, "",
	)
	if err != nil {
		return nil, errors.Wrapf(err, "pre-check required addons for app %s", app.ID)
	}
	return &DeployPreCheckResult{
		UndefinedVars:                buildResult.UndefinedEnvVars,
		MissingRequiredClusterAddons: missing,
	}, nil
}
