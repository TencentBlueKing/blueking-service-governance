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

package taf

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
)

// Plugin provides workload extensions for TAF applications.
type Plugin struct {
	appConfigFileStore appcfg.AppConfigFileStore
}

// NewPlugin creates a new TAF plugin with the given dependencies.
func NewPlugin(appConfigFileStore appcfg.AppConfigFileStore) *Plugin {
	return &Plugin{appConfigFileStore: appConfigFileStore}
}

// Type returns the workload type handled by this plugin.
func (p *Plugin) Type() string {
	return appmodel.WorkloadTypeTaf
}

// Start initializes a plugin session for this build.
func (p *Plugin) Start(
	ctx context.Context,
	env *envmodel.Environment,
	app *bkmsapp.Application,
	appModel *appmodel.AppModel,
	renderCtx plugin.RenderContext,
) (plugin.WorkloadPluginSession, error) {
	// Compute and set the TAF config file content based on environment
	sourceName, content, err := p.computeTafConfig(ctx, app, env, *appModel)
	if err != nil {
		return nil, errors.Wrap(err, "computing TAF config")
	}

	// Collect undefined env var references before rendering.
	if err = renderCtx.Collector.Collect(content, envvarrefs.Source{
		Type: envvarrefs.SourceAppConfigFile,
		Name: sourceName,
	}); err != nil {
		return nil, errors.Wrap(err, "collecting env vars from TAF config")
	}

	// Render template variables in the TAF config content
	content, err = render.New(render.SetEnvContext(renderCtx.EnvVars)).Render(content)
	if err != nil {
		return nil, errors.Wrap(err, "rendering TAF config content")
	}
	appModel.Workload.TafConfig.FileContent = content

	if appModel.Workload.TafConfig.FileName == "" {
		return &runtimerender.Config{}, nil
	}

	return buildTafConfig(appModel), nil
}

// computeTafConfig computes the TAF configuration content based on environment.
// It queries AppConfigFile with the following priority:
// 1. Environment-specific config (envName = current environment name)
// 2. Application-level default config (envName = "")
// It also returns the selected logical file name for reference reporting.
func (p *Plugin) computeTafConfig(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	appModel appmodel.AppModel,
) (string, string, error) {
	if p.appConfigFileStore == nil {
		return appModel.Workload.TafConfig.FileName, appModel.Workload.TafConfig.FileContent, nil
	}
	acf, content, err := appcfg.GetEnvContent(ctx, p.appConfigFileStore, app.ID, env.Name)
	if err != nil {
		return "", "", err
	}
	return acf.Name, content, nil
}

// buildTafConfig builds the TAF spec configuration with init container support for runtime
// variable rendering.
//
// The build produces:
//   - A ConfigMap volume mounted at a temporary path (template source for init container)
//   - An emptyDir volume mounted at the final config path (rendered output)
//   - An init container that runs sed to replace __VAR_NAME__ placeholders with runtime values
//
// Runtime variables (BKMS_POD_IP, BKMS_POD_NAME, BKMS_NODE_IP) are rendered as special
// placeholders (e.g., __#VAR_PLACEHOLDER#__BKMS_POD_IP__) at compile time. The init container then replaces
// these placeholders with actual values from the Kubernetes Downward API at pod startup.
func buildTafConfig(appModel *appmodel.AppModel) *runtimerender.Config {
	config := appModel.Workload.TafConfig
	return runtimerender.BuildConfig(runtimerender.ConfigParams{
		WorkloadType:  appmodel.WorkloadTypeTaf,
		ConfigMapName: appModel.Workload.Name,
		FileName:      config.FileName,
		FilePath:      config.FilePath,
		FileContent:   config.FileContent,
	})
}
