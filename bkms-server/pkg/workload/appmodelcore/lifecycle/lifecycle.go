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

// Package lifecycle 编排 bkmsApp 应用的创建与修改，只写新模型字段。
//
// 新 API 与兼容运行的旧 API（创建、trpc-spec / taf-spec）都使用本包。
package lifecycle

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/datatype"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// CreateParams 创建 bkmsApp 应用的参数。
type CreateParams struct {
	// Command 容器启动命令。
	Command []string
	// Args 容器启动参数。
	Args []string
	// ShCommand 容器启动脚本，与 Command、Args 互斥。
	ShCommand string
	// EnvVars 应用环境变量。
	EnvVars []appmodel.Variable
	// FrameworkConfig 框架扩展配置，为空时写入空对象。
	FrameworkConfig map[string]any
	// FrameworkFile 框架配置文件。框架支持框架配置文件时必填，否则必须为空。
	FrameworkFile *FrameworkFileParams
}

// FrameworkFileParams 框架配置文件的初始内容与挂载位置。
type FrameworkFileParams struct {
	// Name 文件名。
	Name string
	// MountDir 挂载目录。
	MountDir string
	// Content 文件内容，为空时只创建文件定义。
	Content string
}

// Service bkmsApp 应用的创建与修改。
type Service struct {
	db         *mongo.Database
	cfgService *appcfg.AppConfigFileService
}

// NewService 创建 Service。
func NewService(db *mongo.Database) *Service {
	return &Service{db: db, cfgService: appcfg.NewAppConfigFileService(db)}
}

// Create 创建 bkmsApp 应用：框架配置文件（框架支持时）、AppModel、AppSpec，最后写入 Application。
//
// 涉及集合：app_config_file_defs、app_config_files、app_config_file_versions、app_models、app_specs、applications。
// 构建配置由调用方在此之前写入。
func (s *Service) Create(ctx context.Context, app *bkmsapp.Application, params *CreateParams) error {
	if err := ValidateCreate(app, params); err != nil {
		return err
	}
	runtimeMods := mods.Of(app)
	// 在写入任何应用数据前读取初始化规则，避免读取失败时留下部分应用数据。
	resolved, err := appdefaults.Resolve(ctx, s.db, app.WorkspaceID, app.ID, runtimeMods.DevMode.Props)
	if err != nil {
		return errors.Wrapf(err, "resolve defaults for app %s", app.ID)
	}

	if runtimeMods.FrameworkFile.Supported {
		if err = s.createFrameworkFile(ctx, app, runtimeMods, params.FrameworkFile); err != nil {
			return err
		}
	}

	appModel := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Name:                   app.Name,
			Command:                params.Command,
			Args:                   params.Args,
			ShCommand:              params.ShCommand,
			EnvVars:                params.EnvVars,
			FrameworkConfig:        datatype.CloneMap(normalizeFrameworkConfig(params.FrameworkConfig)),
			FrameworkConfigVersion: appruntime.FrameworkConfigVersionV1,
		},
	}
	appspec.ApplyToAppModel(&resolved.Default, appModel)
	if err = appmodel.CreateAppModel(ctx, s.db, appModel); err != nil {
		return errors.Wrapf(err, "create model of app %s", app.ID)
	}

	specs := store.Of(s.db, appspec.AppSpecMeta)
	if _, err = specs.Upsert(
		ctx, &resolved.Default, appspec.ByAppEnv(resolved.Default.AppID, resolved.Default.EnvName),
	); err != nil {
		return errors.Wrapf(err, "create default spec of app %s", app.ID)
	}
	for _, spec := range resolved.Environments {
		if _, err = specs.Upsert(ctx, spec, appspec.ByAppEnv(spec.AppID, spec.EnvName)); err != nil {
			return errors.Wrapf(err, "create spec of app %s for environment %q", app.ID, spec.EnvName)
		}
	}

	// Application 最后写入，之前任一步失败都不会留下可见的应用。
	if err = bkmsapp.CreateApp(ctx, s.db, app); err != nil {
		return errors.Wrapf(err, "create app %s", app.ID)
	}
	return nil
}

// createFrameworkFile 按能力声明的格式创建框架配置文件。
func (s *Service) createFrameworkFile(
	ctx context.Context,
	app *bkmsapp.Application,
	runtimeMods mods.Set,
	file *FrameworkFileParams,
) error {
	var content *string
	if file.Content != "" {
		content = &file.Content
	}
	if _, err := s.cfgService.Create(ctx, appcfg.CreateCfgFileParams{
		AppID:             app.ID,
		EnvName:           appcfg.EnvNameDefault,
		Name:              file.Name,
		MountDir:          appcfg.NormalizeMountDir(file.MountDir),
		Type:              appcfg.AppConfigFileTypeNormal,
		ContentSourceType: appcfg.ContentSourceTypeLocal,
		Format:            appcfg.FileFormat(runtimeMods.FrameworkFile.Props.Format),
		Content:           content,
		Creator:           appcfg.CfgSystemUser,
		Description:       appcfg.CfgSystemVersionDescription,
		ConfigKind:        appcfg.ConfigKindFramework,
		AppType:           app.Type,
		Framework:         app.Framework,
	}); err != nil {
		return errors.Wrapf(err, "create framework config file of app %s", app.ID)
	}
	return nil
}
