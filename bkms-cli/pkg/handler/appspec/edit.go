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

// Package appspec provides AppSpec CLI handler logic.
package appspec

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// EditHandler 从 YAML 文件更新指定 section 配置。
func EditHandler(ctx context.Context, appID, envName, specFile string, section client.AppSpecSectionName) error {
	cli := client.New()
	switch section {
	case client.AppSpecSectionResources:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseResourcesFile,
			func(v *ResourcesInput) any {
				return &SetDefaultResourcesRequest{AppSpecResources: v}
			},
		)
	case client.AppSpecSectionUpdateStrategy:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseUpdateStrategyFile,
			func(v *UpdateStrategyInput) any {
				return &SetDefaultUpdateStrategyRequest{AppSpecUpdateStrategy: v}
			},
		)
	case client.AppSpecSectionLifecycle:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseLifecycleFile,
			func(v *LifecycleInput) any {
				return &SetDefaultLifecycleRequest{AppSpecLifecycle: v}
			},
		)
	case client.AppSpecSectionProbe:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseProbeFile,
			func(v *ProbeInput) any {
				return &SetDefaultProbeRequest{AppSpecProbe: v}
			},
		)
	case client.AppSpecSectionLabels:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseLabelsFile,
			func(v *LabelsInput) any {
				return &SetDefaultLabelsRequest{AppSpecLabels: v}
			},
		)
	case client.AppSpecSectionAnnotations:
		return editSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			specFile,
			section,
			ParseAnnotationsFile,
			func(v *AnnotationsInput) any {
				return &SetDefaultAnnotationsRequest{AppSpecAnnotations: v}
			},
		)
	default:
		return errors.Errorf("unsupported section: %s", section)
	}
}

// EditStartCommandHandler 从 YAML 文件更新应用启动命令。
func EditStartCommandHandler(ctx context.Context, appID, specFile string) error {
	cli := client.New()
	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return err
	}

	if !isAppModelType(app.Type) {
		return errors.Errorf("app type %q does not support start command (only trpc/taf apps are supported)", app.Type)
	}

	input, err := ParseStartCommandFile(specFile)
	if err != nil {
		return err
	}

	// 服务端要求 trpcSpec/tafSpec 必填，从当前配置中获取并回填
	if app.AppModelSpec != nil {
		if input.TrpcSpec == nil && app.AppModelSpec.TrpcSpec != nil {
			input.TrpcSpec = app.AppModelSpec.TrpcSpec
		}
		if input.TafSpec == nil && app.AppModelSpec.TafSpec != nil {
			input.TafSpec = app.AppModelSpec.TafSpec
		}
	}

	body := &UpdateStartCommandRequest{AppModelSpec: input}
	return cli.UpdateAppStartCommand(ctx, appID, app.Type, body)
}

// ResetHandler 删除环境级 section 覆盖配置，恢复为默认值。
func ResetHandler(ctx context.Context, appID, envName string, section client.AppSpecSectionName) error {
	cli := client.New()
	return cli.DeleteAppSpecEnvSection(ctx, appID, envName, section)
}

// --- internal helpers ---

func isAppModelType(appType string) bool {
	return appType == "trpc" || appType == "taf"
}

func editSectionHandler[T any](
	ctx context.Context,
	cli client.Client,
	appID, envName, specFile string,
	section client.AppSpecSectionName,
	parse func(string) (*T, error),
	buildReq func(*T) any,
) error {
	input, err := parse(specFile)
	if err != nil {
		return err
	}
	body := buildReq(input)
	if envName == "" {
		return cli.SetAppSpecDefaultSection(ctx, appID, section, body)
	}
	return cli.SetAppSpecEnvSection(ctx, appID, envName, section, body)
}
