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

// Package postrenderer provides Helm PostRenderer implementations.
package postrenderer

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"helm.sh/helm/v3/pkg/postrender"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	helmcomp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

// Build 统一构建 PostRenderer（组件 PostRenderer → BSCP PostRenderer → 泳道 PostRenderer）
// 按顺序依次执行，如果全部为 nil，返回 nil
func Build(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
) (postrender.PostRenderer, error) {
	compRenderer, err := buildComponentPostRenderer(ctx, app, env)
	if err != nil {
		return nil, errors.Wrap(err, "build component post renderer")
	}

	bscpRenderer, err := buildBscpPostRenderer(ctx, app.ID, env.Name)
	if err != nil {
		return nil, errors.Wrap(err, "build bscp post renderer")
	}

	laneRenderer, err := buildLanePostRenderer(ctx, app.WorkspaceID, env.Name, trafficLaneName)
	if err != nil {
		return nil, errors.Wrap(err, "build lane post renderer")
	}

	chain := NewChainPostRenderer(compRenderer, bscpRenderer, laneRenderer)
	if chain == nil {
		return nil, nil
	}
	return chain, nil
}

// buildComponentPostRenderer 根据应用和环境查询组件配置，构建组件 PostRenderer
// 如果没有配置组件，返回 nil
func buildComponentPostRenderer(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) (*ComponentPostRenderer, error) {
	compStore, err := helmcomp.NewDbHelmAppComponentStore(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create helm app component store")
	}

	comps, err := compStore.ListByAppAndEnv(ctx, app.ID, env.Name)
	if err != nil {
		return nil, errors.Wrap(err, "list helm app components")
	}
	if len(comps) == 0 {
		return nil, nil
	}

	workspaceCompsStore, err := workspace.NewWorkspaceCompsStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create workspace component store")
	}
	workspaceComps, err := workspaceCompsStore.ListByWorkspace(ctx, app.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "list workspace components")
	}

	compApplier, err := component.CreateDefaultApplier()
	if err != nil {
		return nil, errors.Wrap(err, "create component applier")
	}

	items, err := buildComponentPatches(ctx, app, env, comps, workspaceComps, compApplier)
	if err != nil {
		return nil, err
	}
	return NewComponentPostRenderer(items), nil
}

type componentEvaluator interface {
	Evaluate(
		ctx context.Context,
		app *bkmsapp.Application,
		comp component.Component,
		envID bson.ObjectID,
		vars map[string]string,
		collector *envvarrefs.Collector,
	) (*component.EvaluatedComponent, error)
}

func buildComponentPatches(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	comps []*helmcomp.HelmAppComponent,
	workspaceComps []*workspace.Component,
	compApplier componentEvaluator,
) ([]ComponentPatch, error) {
	if len(comps) == 0 {
		return nil, nil
	}

	sort.SliceStable(comps, func(i, j int) bool {
		return comps[i].Priority < comps[j].Priority
	})

	workspaceCompMap := make(map[string]*workspace.Component, len(workspaceComps))
	for _, comp := range workspaceComps {
		workspaceCompMap[comp.Name] = comp
	}

	patches := make([]ComponentPatch, 0, len(comps))
	for _, comp := range comps {
		resolvedComp, resolveErr := resolveHelmComponent(comp, workspaceCompMap, env.Name)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if resolvedComp == nil {
			continue
		}
		patch, getErr := buildPatch(ctx, app, env.ID, comp.Name, comp.Target, *resolvedComp, compApplier)
		if getErr != nil {
			return nil, getErr
		}
		patches = append(patches, patch)
	}

	return patches, nil
}

func resolveHelmComponent(
	comp *helmcomp.HelmAppComponent,
	workspaceCompMap map[string]*workspace.Component,
	envName string,
) (*component.Component, error) {
	if comp.RefWorkspaceCompName == "" {
		return &comp.Component, nil
	}

	workspaceComp, ok := workspaceCompMap[comp.RefWorkspaceCompName]
	if !ok {
		return nil, errors.Errorf("referenced workspace component not found: %s", comp.RefWorkspaceCompName)
	}
	if !workspaceComp.IsAvailableInEnv(envName) {
		return nil, nil
	}

	return &component.Component{
		Name: workspaceComp.Name,
		ComponentInst: component.ComponentInst{
			Type:       workspaceComp.Type,
			Version:    workspaceComp.Version,
			Properties: workspaceComp.Properties,
		},
	}, nil
}

func buildPatch(
	ctx context.Context,
	app *bkmsapp.Application,
	envID bson.ObjectID,
	name string,
	target helmcomp.TargetResourceSelector,
	comp component.Component,
	compApplier componentEvaluator,
) (ComponentPatch, error) {
	// FIXME: Helm post-renderer 当前没有接入 env vars 渲染上下文，因此这里传 nil。
	// 这意味着组件属性中的 ${{KEY}} / {{.BKMS.ENV.xxx}} 不会按当前环境展开。
	// 这么做的原因是目前 Helm 应用不支持环境变量，采用的是 values + helm 独有变量的形式，
	// 如果后续 Helm 组件需要支持环境变量占位符，需要在构建 patch 前生成并传入 vars。
	// 当前也不收集未定义变量引用，因此 collector 同样传 nil。
	evaluated, err := compApplier.Evaluate(ctx, app, comp, envID, nil, nil)
	if err != nil {
		return ComponentPatch{}, errors.Wrapf(err, "evaluate component %q", name)
	}

	return ComponentPatch{
		Name:     name,
		Target:   target,
		Patchers: evaluated.Patchers,
		Specs:    evaluated.Specs,
	}, nil
}

// buildBscpPostRenderer 根据应用配置构建 BSCP PostRenderer
// 如果应用未配置 BSCP 配置管理，返回 nil
func buildBscpPostRenderer(
	ctx context.Context,
	appID, envName string,
) (*BscpPostRenderer, error) {
	store, err := bscpcfg.NewStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create bscp config store")
	}
	return NewBscpPostRendererFromStore(ctx, store, appID, envName)
}

// buildLanePostRenderer 根据泳道配置构建 PostRenderer
// 如果 trafficLaneName 为空，返回 nil
func buildLanePostRenderer(
	ctx context.Context,
	workspaceID, envName, trafficLaneName string,
) (*LanePostRenderer, error) {
	if trafficLaneName == "" {
		return nil, nil
	}

	lane, err := trafficmanager.New().GetTrafficLane(ctx, workspaceID, envName, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "get traffic lane %s", trafficLaneName)
	}

	labels := make(map[string]string)
	for k, v := range lane.LaneServiceVersionLabels {
		labels[k] = v
	}

	return NewLanePostRenderer(labels), nil
}
