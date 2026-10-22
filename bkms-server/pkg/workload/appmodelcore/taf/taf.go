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

// Package taf converts TAF application inputs into application models and specifications.
package taf

import (
	"context"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// CreateParams 创建 TAF 应用的参数
type CreateParams struct {
	// Command 容器启动命令
	Command []string
	// Args 容器启动参数
	Args []string
	// EnvVars 容器环境变量
	EnvVars []appmodel.Variable
	// TafConfig TAF 特有配置
	TafConfig *TafConfigParams
}

// TafConfigParams TAF 配置参数
type TafConfigParams struct {
	// FileName 配置文件名，只写入 framework def，不写入 app model。
	// todo 兼容：Update 仍用它同步 def，可为空。更新改走 def 接口后，只有 Create 使用。
	FileName string
	// FilePath 配置文件路径，只写入 framework def，不写入 app model。
	// todo 兼容：Update 仍用它同步 def，可为空。更新改走 def 接口后，只有 Create 使用。
	FilePath string
	// FileContent 配置文件内容
	FileContent string
}

// UpdateParams 更新 TAF 应用的参数
type UpdateParams struct {
	// Command 容器启动命令
	Command []string
	// Args 容器启动参数
	Args []string
	// TafConfig TAF 特有配置
	TafConfig *TafConfigParams
}

// Service TAF 应用服务
type Service struct {
	db                   *mongo.Database
	appConfigFileService *appcfg.AppConfigFileService
}

// NewService 创建服务实例
func NewService(
	db *mongo.Database,
) *Service {
	return &Service{
		db: db,
		appConfigFileService: appcfg.NewAppConfigFileService(
			db,
		),
	}
}

// Create 创建 TAF 应用资源（AppModel + AppConfigFile + App）
func (s *Service) Create(ctx context.Context, app *bkmsapp.Application, params *CreateParams) error {
	// 在写入任何应用数据前读取初始化规则，避免读取失败时留下部分应用数据。
	resolved, err := appdefaults.Resolve(ctx, s.db, app.WorkspaceID, app.ID, app.Type)
	if err != nil {
		return errors.Wrap(err, "resolve application defaults")
	}

	// 设置配置文件内容
	var fileContent *string
	if params.TafConfig != nil && params.TafConfig.FileContent != "" {
		fileContent = &params.TafConfig.FileContent
	}

	// 创建时 def 还不存在，文件名和挂载目录由 spec 带入，只写入 framework def。
	cfgFileName := appcfg.DefaultAppConfigFileName
	cfgMountDir := ""
	if params.TafConfig != nil {
		if params.TafConfig.FileName != "" {
			cfgFileName = params.TafConfig.FileName
		}
		cfgMountDir = appcfg.NormalizeMountDir(params.TafConfig.FilePath)
	}

	if _, err = s.appConfigFileService.Create(
		ctx,
		appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              cfgFileName,
			MountDir:          cfgMountDir,
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatTAF,
			Content:           fileContent,
			Creator:           appcfg.CfgSystemUser,
			Description:       appcfg.CfgSystemVersionDescription,
			ConfigKind:        appcfg.ConfigKindFramework,
			AppType:           app.Type,
		},
	); err != nil {
		return errors.Wrap(err, "create default config file")
	}

	// 初始化 AppModel
	appModel := &appmodel.AppModel{
		AppID: app.ID,
		Workload: appmodel.Workload{
			Name:    app.Name,
			Type:    appmodel.WorkloadTypeTaf,
			Command: params.Command,
			Args:    params.Args,
			EnvVars: params.EnvVars,
		},
	}

	// 文件名和挂载目录写在 framework def 上，不再写入 app model。
	if params.TafConfig != nil {
		appModel.Workload.TafConfig = appmodel.TafConfig{}
	}

	return s.persistCreatedApp(ctx, app, appModel, resolved)
}

func (s *Service) persistCreatedApp(
	ctx context.Context,
	app *bkmsapp.Application,
	appModel *appmodel.AppModel,
	resolved *appdefaults.ResolvedAppSpec,
) error {
	appspec.ApplyToAppModel(&resolved.Default, appModel)
	if err := appmodel.CreateAppModel(ctx, s.db, appModel); err != nil {
		return errors.Wrapf(err, "create app(%s) model", app.Name)
	}
	specs := store.Of(s.db, appspec.AppSpecMeta)
	if _, err := specs.Upsert(
		ctx, &resolved.Default, appspec.ByAppEnv(resolved.Default.AppID, resolved.Default.EnvName),
	); err != nil {
		return errors.Wrap(err, "create default app spec")
	}
	for _, spec := range resolved.Environments {
		if _, err := specs.Upsert(ctx, spec, appspec.ByAppEnv(spec.AppID, spec.EnvName)); err != nil {
			return errors.Wrapf(err, "create app spec for environment %q", spec.EnvName)
		}
	}
	if err := bkmsapp.CreateApp(ctx, s.db, app); err != nil {
		return errors.Wrap(err, "create app")
	}
	return nil
}

// Update 更新 TAF 应用配置，返回 oldModel 和 newModel 供 handler 层进行审计
func (s *Service) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	params *UpdateParams,
) (*appmodel.AppModel, *appmodel.AppModel, error) {
	// 获取应用模型
	appModel, err := store.Of(s.db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get app(%s) model", app.Name)
	}

	// 保存旧模型副本用于审计
	var oldAppModel appmodel.AppModel
	if copyErr := copier.Copy(&oldAppModel, appModel); copyErr != nil {
		return nil, nil, errors.Wrap(copyErr, "copying app model")
	}

	// todo 兼容：fileName、filePath 仍从 spec 同步到 def，未传时保持 def 原值。见 SyncFrameworkFileLocation。
	if params.TafConfig != nil {
		if err = s.appConfigFileService.SyncFrameworkFileLocation(
			ctx, app.ID, app.Type, params.TafConfig.FileName, params.TafConfig.FilePath,
		); err != nil {
			return nil, nil, errors.Wrapf(err, "sync framework file location for app %s", app.ID)
		}
		appModel.Workload.TafConfig = appmodel.TafConfig{}
	}

	// 更新通用配置
	appModel.Workload.Command = params.Command
	appModel.Workload.Args = params.Args

	// 保存更新
	if err = appmodel.UpdateAppModel(ctx, s.db, appModel); err != nil {
		return nil, nil, errors.Wrapf(err, "update app(%s) model", app.Name)
	}

	return &oldAppModel, appModel, nil
}
