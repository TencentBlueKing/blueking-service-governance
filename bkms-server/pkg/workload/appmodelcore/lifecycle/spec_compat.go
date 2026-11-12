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

package lifecycle

import (
	"context"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// SpecCompatParams 旧 trpc-spec / taf-spec 接口的修改参数。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧 trpc-spec / taf-spec 接口使用。
type SpecCompatParams struct {
	// Framework 接口要求的框架，与应用框架不一致时拒绝。
	Framework string
	// Command 容器启动命令，整体替换。
	Command []string
	// Args 容器启动参数，整体替换。
	Args []string
	// ShCommand 容器启动脚本，整体替换。
	ShCommand string
	// Language 旧接口携带的语言，非空时必须与应用当前语言一致；应用没有记录语言时不校验，也不写入。
	Language string
	// FileName 框架配置文件名，非空时同步到框架 def。
	FileName string
	// FilePath 框架配置文件挂载目录，非空时同步到框架 def。
	FilePath string
}

// UpdateSpecCompat 处理旧 trpc-spec / taf-spec 接口的修改：修改启动命令、参数、脚本，同步框架配置文件位置。
// 旧类型应用与 bkmsApp 应用处理方式相同，只读写新模型字段；不允许切换语言。
// 返回修改前后的 AppModel，供调用方审计。
//
// 涉及集合：app_config_file_defs、app_models。
//
// Deprecated: bkmsApp 迁移兼容，仅供旧 trpc-spec / taf-spec 接口使用。
func (s *Service) UpdateSpecCompat(
	ctx context.Context,
	app *bkmsapp.Application,
	params *SpecCompatParams,
) (*appmodel.AppModel, *appmodel.AppModel, error) {
	stack := mods.Of(app).Stack
	if stack.Framework != params.Framework {
		return nil, nil, errors.Wrapf(ErrInvalidParams,
			"app %s: framework %q does not match %q", app.ID, stack.Framework, params.Framework)
	}
	if params.Language != "" && stack.Language != "" && params.Language != stack.Language {
		return nil, nil, errors.Wrapf(ErrInvalidParams,
			"app %s: changing language from %q to %q is not supported", app.ID, stack.Language, params.Language)
	}

	if params.FileName != "" || params.FilePath != "" {
		if err := s.cfgService.SyncFrameworkFileLocation(
			ctx, app.ID, stack.Framework, params.FileName, params.FilePath,
		); err != nil {
			return nil, nil, errors.Wrapf(err, "sync framework file location of app %s", app.ID)
		}
	}

	appModel, err := store.Of(s.db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "get model of app %s", app.ID)
	}
	var oldAppModel appmodel.AppModel
	if err = copier.CopyWithOption(&oldAppModel, appModel, copier.Option{DeepCopy: true}); err != nil {
		return nil, nil, errors.Wrapf(err, "copy model of app %s", app.ID)
	}
	appModel.Workload.Command = params.Command
	appModel.Workload.Args = params.Args
	appModel.Workload.ShCommand = params.ShCommand
	if err = appmodel.UpdateAppModel(ctx, s.db, appModel); err != nil {
		return nil, nil, errors.Wrapf(err, "update model of app %s", app.ID)
	}
	return &oldAppModel, appModel, nil
}
