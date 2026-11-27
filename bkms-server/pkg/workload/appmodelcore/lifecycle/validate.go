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
	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/framework/blank"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/framework/taf"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/framework/trpc"
)

// ErrInvalidParams 创建或修改参数不合法，调用方据此返回参数错误。
var ErrInvalidParams = errors.New("invalid bkmsApp params")

// frameworkConfigDecoders 各框架 frameworkConfig 的严格解码。
var frameworkConfigDecoders = map[string]func(cfg map[string]any, version int) error{
	appruntime.FrameworkTRPC: func(cfg map[string]any, version int) error {
		_, err := trpc.DecodeConfig(cfg, version)
		return err
	},
	appruntime.FrameworkTAF: func(cfg map[string]any, version int) error {
		_, err := taf.DecodeConfig(cfg, version)
		return err
	},
	appruntime.FrameworkBlank: func(cfg map[string]any, version int) error {
		_, err := blank.DecodeConfig(cfg, version)
		return err
	},
}

// ValidateCreate 校验创建参数，不访问数据库。调用方可在写入其他数据（如构建配置）前先行调用。
func ValidateCreate(app *bkmsapp.Application, params *CreateParams) error {
	if err := ValidateStack(app); err != nil {
		return err
	}
	if err := validateFrameworkConfig(app.Framework, params.FrameworkConfig); err != nil {
		return err
	}
	supported := mods.Of(app).FrameworkFile.Supported
	switch {
	case supported && params.FrameworkFile == nil:
		return errors.Wrapf(ErrInvalidParams, "framework %q requires a framework config file", app.Framework)
	case !supported && params.FrameworkFile != nil:
		return errors.Wrapf(ErrInvalidParams, "framework %q does not support framework config file", app.Framework)
	case supported && params.FrameworkFile.Name == "":
		return errors.Wrap(ErrInvalidParams, "framework config file name is required")
	}
	return nil
}

// ValidateStack 校验应用为 bkmsApp，且框架与语言是框架目录中的合法组合。
func ValidateStack(app *bkmsapp.Application) error {
	if app.Type != appruntime.AppTypeBKMSApp {
		return errors.Wrapf(ErrInvalidParams, "app %s: type %q is not bkmsApp", app.ID, app.Type)
	}
	if err := appruntime.ValidateStack(appruntime.Stack{
		Type:      app.Type,
		Framework: app.Framework,
		Language:  app.Language,
	}); err != nil {
		return errors.Wrapf(ErrInvalidParams, "app %s: %v", app.ID, err)
	}
	return nil
}

// validateFrameworkConfig 按框架严格解码 frameworkConfig。
func validateFrameworkConfig(framework string, cfg map[string]any) error {
	decode, ok := frameworkConfigDecoders[framework]
	if !ok {
		return errors.Wrapf(ErrInvalidParams, "no frameworkConfig decoder for framework %q", framework)
	}
	if err := decode(normalizeFrameworkConfig(cfg), appruntime.FrameworkConfigVersionV1); err != nil {
		return errors.Wrapf(ErrInvalidParams, "frameworkConfig: %v", err)
	}
	return nil
}

// normalizeFrameworkConfig 未提供时规范化为空对象。
func normalizeFrameworkConfig(cfg map[string]any) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	return cfg
}
