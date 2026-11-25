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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/resname"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
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
	// UnsupportedComponents 共用命名空间的环境不支持使用组件，列出应用在该环境下生效的组件名
	UnsupportedComponents []string
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

// Check reports undefined env vars, missing required cluster addons and components
// unsupported in an environment sharing namespace.
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
	modelForBuild.Workload.Name = resname.ForEnv(env).Build(app.Name)
	modelForBuild.Workload.Image = deployPreCheckImage
	builder := workload.NewBuilder(c.builderService, app, &modelForBuild)

	result := &DeployPreCheckResult{}
	if result.UnsupportedComponents, err = listUnsupportedComponents(ctx, builder, env, &modelForBuild); err != nil {
		return nil, err
	}
	// 存在不支持的组件时构建必然失败，跳过构建，以阻断项而非错误的形式返回给前端
	if len(result.UnsupportedComponents) == 0 {
		buildResult, buildErr := builder.Build(ctx, env)
		if buildErr != nil {
			return nil, errors.Wrap(buildErr, "building workload for deployment pre-check")
		}
		result.UndefinedVars = buildResult.UndefinedEnvVars
	}

	defs, err := clusteraddon.ListDefs(ctx, c.clusterAddonDefs)
	if err != nil {
		return nil, err
	}
	result.MissingRequiredClusterAddons, err = clusteraddon.InspectRequiredAddons(
		ctx, defs, clusteraddon.AddonAppTypesOf(app), env, "",
	)
	if err != nil {
		return nil, errors.Wrapf(err, "pre-check required addons for app %s", app.ID)
	}
	return result, nil
}

// listUnsupportedComponents 返回共用命名空间的环境中应用生效的组件名（含已解析的空间组件引用）；
// 组件生成的资源名无法带 resPrefix 前缀，这类环境不支持使用组件。
func listUnsupportedComponents(
	ctx context.Context, builder *workload.Builder, env *envmodel.Environment, appModel *appmodel.AppModel,
) ([]string, error) {
	if !env.SharesNamespace() {
		return nil, nil
	}
	comps, err := builder.ListComponents(ctx, env, appModel)
	if err != nil {
		return nil, errors.Wrap(err, "list components for deployment pre-check")
	}
	return lo.Map(comps, func(comp *component.Component, _ int) string { return comp.Name }), nil
}
