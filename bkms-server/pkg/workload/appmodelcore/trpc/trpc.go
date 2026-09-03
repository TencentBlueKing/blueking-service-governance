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

// Package trpc converts tRPC application inputs into application models and specifications.
package trpc

import (
	"context"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// CreateParams 创建 tRPC 应用的参数（不依赖 API 层）
type CreateParams struct {
	// Command 容器启动命令
	Command []string
	// Args 容器启动参数
	Args []string
	// EnvVars 容器环境变量
	EnvVars []appmodel.Variable
	// TrpcConfig tRPC 特有配置
	TrpcConfig *TrpcConfigParams
}

// TrpcConfigParams tRPC 配置参数
type TrpcConfigParams struct {
	// Language tRPC 框架语言
	Language string
	// FileName 配置文件名
	FileName string
	// FilePath 配置文件路径
	FilePath string
	// FileContent 配置文件内容
	FileContent string
}

// UpdateParams 更新 tRPC 应用的参数
type UpdateParams struct {
	// Command 容器启动命令
	Command []string
	// Args 容器启动参数
	Args []string
	// TrpcConfig tRPC 特有配置
	TrpcConfig *TrpcConfigParams
}

// Service tRPC 应用服务
type Service struct {
	appModelStore        appmodel.AppModelStore
	appSpecStore         appspec.AppSpecStore
	appDefaultRuleStore  appdefaults.RuleStore
	envStore             envmodel.EnvironmentStore
	appStore             bkmsapp.ApplicationStore
	appConfigFileService *appcfg.AppConfigFileService
}

// NewService 创建服务实例
func NewService(
	appModelStore appmodel.AppModelStore,
	appSpecStore appspec.AppSpecStore,
	appDefaultRuleStore appdefaults.RuleStore,
	envStore envmodel.EnvironmentStore,
	appConfigFileStore appcfg.AppConfigFileStore,
	appConfigFileDefStore appcfg.AppConfigFileDefStore,
	appConfigFileVersionStore appcfg.AppConfigFileVersionStore,
	appStore bkmsapp.ApplicationStore,
) *Service {
	return &Service{
		appModelStore:       appModelStore,
		appSpecStore:        appSpecStore,
		appDefaultRuleStore: appDefaultRuleStore,
		envStore:            envStore,
		appStore:            appStore,
		appConfigFileService: appcfg.NewAppConfigFileService(
			appConfigFileStore,
			appConfigFileDefStore,
			appConfigFileVersionStore,
		),
	}
}

// Create 创建 tRPC 应用资源（AppModel + AppConfigFile + App）
func (s *Service) Create(ctx context.Context, app *bkmsapp.Application, params *CreateParams) error {
	// 在写入任何应用数据前读取初始化规则，避免读取失败时留下部分应用数据。
	resolved, err := appdefaults.Resolve(
		ctx,
		s.appDefaultRuleStore,
		s.envStore,
		app.WorkspaceID,
		app.ID,
	)
	if err != nil {
		return errors.Wrap(err, "resolve application defaults")
	}

	// 设置配置文件内容
	var fileContent *string
	if params.TrpcConfig != nil && params.TrpcConfig.FileContent != "" {
		fileContent = &params.TrpcConfig.FileContent
	}

	if _, err = s.appConfigFileService.Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              appcfg.DefaultAppConfigFileName,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           fileContent,
			Creator:           appcfg.CfgSystemUser,
			Description:       appcfg.CfgSystemVersionDescription,
		},
	); err != nil {
		return errors.Wrap(err, "create default config file")
	}

	// 初始化 AppModel
	appModel := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Name:    app.Name,
			Type:    appmodel.WorkloadTypeTrpc,
			Command: params.Command,
			Args:    params.Args,
			EnvVars: params.EnvVars,
		},
	}

	// 设置 tRPC 配置
	if params.TrpcConfig != nil {
		appModel.Workload.TrpcConfig = appmodel.TrpcConfig{
			FileName: params.TrpcConfig.FileName,
			FilePath: params.TrpcConfig.FilePath,
			Language: params.TrpcConfig.Language,
		}
	}

	// 将平台默认 AppSpec 应用到 AppModel。
	appspec.ApplyToAppModel(&resolved.Default, appModel)

	// 创建 AppModel
	if err = s.appModelStore.CreateAppModel(ctx, appModel); err != nil {
		return errors.Wrapf(err, "create app(%s) model", app.Name)
	}

	// 插入 appspec 初始配置
	if err = s.appSpecStore.Upsert(ctx, &resolved.Default); err != nil {
		return errors.Wrap(err, "create default app spec")
	}
	for _, spec := range resolved.Environments {
		if err = s.appSpecStore.Upsert(ctx, spec); err != nil {
			return errors.Wrapf(err, "create app spec for environment %q", spec.EnvName)
		}
	}

	// 创建应用基础数据
	if err := s.appStore.CreateApp(ctx, app); err != nil {
		return errors.Wrap(err, "create app")
	}

	return nil
}

// Update 更新 tRPC 应用配置，返回 oldModel 和 newModel 供 handler 层进行审计
func (s *Service) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	params *UpdateParams,
) (*appmodel.AppModel, *appmodel.AppModel, error) {
	// 获取应用模型
	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get app(%s) model", app.Name)
	}

	// 保存旧模型副本用于审计
	var oldAppModel appmodel.AppModel
	if copyErr := copier.Copy(&oldAppModel, appModel); copyErr != nil {
		return nil, nil, errors.Wrap(copyErr, "copying app model")
	}

	// 更新 tRPC 配置
	if params.TrpcConfig != nil {
		appModel.Workload.TrpcConfig = appmodel.TrpcConfig{
			FileName: params.TrpcConfig.FileName,
			FilePath: params.TrpcConfig.FilePath,
			Language: params.TrpcConfig.Language,
		}
	}

	// 更新通用配置
	appModel.Workload.Command = params.Command
	appModel.Workload.Args = params.Args

	// 保存更新
	if err = s.appModelStore.UpdateAppModel(ctx, appModel); err != nil {
		return nil, nil, errors.Wrapf(err, "update app(%s) model", app.Name)
	}

	return &oldAppModel, appModel, nil
}
