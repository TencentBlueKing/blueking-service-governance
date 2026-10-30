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
	// 开关型 section 只有一个 enabled 布尔值，不接受 YAML 文件输入
	if section == client.AppSpecSectionTkeRouteEni || section == client.AppSpecSectionDevMode {
		return errors.Errorf("section %s does not support edit, use enable/disable instead", section)
	}

	cli := client.New()
	switch section {
	case client.AppSpecSectionResources:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*ResourcesInput, error) { return ParseResourcesFile(specFile) },
			func(v *ResourcesInput) any {
				return &SetDefaultResourcesRequest{AppSpecResources: v}
			},
		)
	case client.AppSpecSectionUpdateStrategy:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*UpdateStrategyInput, error) { return ParseUpdateStrategyFile(specFile) },
			func(v *UpdateStrategyInput) any {
				return &SetDefaultUpdateStrategyRequest{AppSpecUpdateStrategy: v}
			},
		)
	case client.AppSpecSectionLifecycle:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*LifecycleInput, error) { return ParseLifecycleFile(specFile) },
			func(v *LifecycleInput) any {
				return &SetDefaultLifecycleRequest{AppSpecLifecycle: v}
			},
		)
	case client.AppSpecSectionProbe:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*ProbeInput, error) { return ParseProbeFile(specFile) },
			func(v *ProbeInput) any {
				return &SetDefaultProbeRequest{AppSpecProbe: v}
			},
		)
	case client.AppSpecSectionLabels:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*LabelsInput, error) { return ParseLabelsFile(specFile) },
			func(v *LabelsInput) any { return v },
		)
	case client.AppSpecSectionAnnotations:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			func() (*AnnotationsInput, error) { return ParseAnnotationsFile(specFile) },
			func(v *AnnotationsInput) any { return v },
		)
	default:
		return errors.Errorf("unsupported section: %s", section)
	}
}

// SetEnabledHandler 设置开关型 section（underlay-ip / dev-mode）的启用状态。
func SetEnabledHandler(
	ctx context.Context,
	appID, envName string,
	section client.AppSpecSectionName,
	enabled bool,
) error {
	return setEnabledHandler(ctx, client.New(), appID, envName, section, enabled)
}

func setEnabledHandler(
	ctx context.Context,
	cli client.Client,
	appID, envName string,
	section client.AppSpecSectionName,
	enabled bool,
) error {
	load := func() (*EnabledInput, error) { return &EnabledInput{Enabled: enabled}, nil }

	switch section {
	case client.AppSpecSectionTkeRouteEni:
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			load,
			// 非标准行为：tkeRouteEni 的默认级与环境级请求体都是扁平结构，不带 appSpecXxx 包装
			func(v *EnabledInput) any { return v },
		)
	case client.AppSpecSectionDevMode:
		if envName == "" {
			return errors.Errorf("%s only supports env-level configuration, --env is required", section)
		}
		return setSectionHandler(
			ctx,
			cli,
			appID,
			envName,
			section,
			load,
			func(v *EnabledInput) any {
				return &SetEnvDevModeRequest{AppSpecDevMode: v}
			},
		)
	default:
		return errors.Errorf("section %s does not support enable/disable", section)
	}
}

// EditStartCommandHandler 从 YAML 文件更新应用启动命令。
func EditStartCommandHandler(ctx context.Context, appID, specFile string) error {
	cli := client.New()
	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return err
	}

	if !client.IsAppModelType(app.Type) {
		return errors.Errorf("app type %q does not support start command (only appmodel apps are supported)", app.Type)
	}

	input, err := ParseStartCommandFile(specFile)
	if err != nil {
		return err
	}

	// bkmsApp 应用使用 startup-config，请求体平铺，且没有框架 spec 需要回填
	if client.IsBKMSApp(app.Type) {
		body := &StartCommandInput{
			Command:   input.Command,
			Args:      input.Args,
			ShCommand: input.ShCommand,
		}
		return cli.UpdateAppStartCommand(ctx, appID, app.Type, body)
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

// setSectionHandler 获取 section 输入，包装为请求体后写入默认级或环境级配置。
// load 负责产出输入（从 YAML 文件解析，或直接由命令行参数构造），
// buildReq 负责包装为服务端要求的请求体结构。
func setSectionHandler[T any](
	ctx context.Context,
	cli client.Client,
	appID, envName string,
	section client.AppSpecSectionName,
	load func() (*T, error),
	buildReq func(*T) any,
) error {
	input, err := load()
	if err != nil {
		return err
	}
	body := buildReq(input)
	if envName == "" {
		return cli.SetAppSpecDefaultSection(ctx, appID, section, body)
	}
	return cli.SetAppSpecEnvSection(ctx, appID, envName, section, body)
}
