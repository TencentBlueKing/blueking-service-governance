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

package appcfgfiledef

// 本文件的 validateBaseAppConfigFileID 与旧 handler（handler/app_config_file_validation.go）中的逻辑一致。
// 为避免对旧接口做结构性改动，本次单独实现；待旧接口废弃后统一收敛到公共层。

import (
	"context"
	stderrors "errors"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
)

// validateConfigKindForApp 校验应用能否新建该种类的配置文件：
//   - plain 只在 AppModel 渲染链路中挂载，Helm / Agones 创建后不会生效；
//   - AppModel 应用的 framework 文件由框架渲染插件挂载，不支持框架配置文件的技术栈（如 blank）创建后不会生效。
func validateConfigKindForApp(app *bkmsapp.Application, kind appcfg.ConfigKind) error {
	isAppModel := appruntime.IsAppModelType(app.Type)
	switch {
	case kind == appcfg.ConfigKindPlain && !isAppModel:
		return bkerrs.New(
			bkerrs.ErrCodeInvalidArgument, i18n.Msg("plain config files are only supported for appmodel applications"),
		)
	case kind == appcfg.ConfigKindFramework && isAppModel && !mods.Of(app).FrameworkFile.Supported:
		return bkerrs.Errorf(
			bkerrs.ErrCodeInvalidArgument,
			i18n.Msg("framework %q does not support framework config files"),
			mods.Of(app).Stack.Framework,
		)
	}
	return nil
}

// validateBaseAppConfigFileID 校验 overlay 文件的 base 引用。
// 非 overlay 类型直接返回 nil；overlay 必须指定合法的同应用 normal 文件。
func (h *Handler) validateBaseAppConfigFileID(
	ctx context.Context,
	appID string,
	fileType appcfg.AppConfigFileType,
	baseIDHex string,
) (*bson.ObjectID, error) {
	if fileType != appcfg.AppConfigFileTypeOverlay {
		return nil, nil
	}
	if baseIDHex == "" {
		return nil, stderrors.New("baseAppConfigFileId is required for overlay type")
	}
	baseObjID, err := bson.ObjectIDFromHex(baseIDHex)
	if err != nil {
		return nil, errors.Wrap(err, "invalid baseAppConfigFileId")
	}
	obj, err := h.configFiles().GetByID(ctx, baseObjID)
	if err != nil {
		return nil, errors.Wrap(err, "getting the base app config file")
	}
	if obj.Type != appcfg.AppConfigFileTypeNormal {
		return nil, stderrors.New("base app config file must be of type 'normal'")
	}
	if obj.AppID != appID {
		return nil, stderrors.New("base app config file does not belong to the app")
	}
	return &obj.ID, nil
}

// validateBSCPConfig 校验 BSCP 来源配置并返回可持久化的 BSCPConfig。
// 非 BSCP 来源直接返回 nil。
func (h *Handler) validateBSCPConfig(
	ctx context.Context,
	sourceType appcfg.ContentSourceType,
	cfg *BSCPConfigInput,
	fileFormat appcfg.FileFormat,
) (*appcfg.BSCPConfig, error) {
	var ref *appcfg.BSCPConfig
	if cfg != nil {
		ref = &appcfg.BSCPConfig{BizID: cfg.BizID, ServiceID: cfg.ServiceID, ConfigID: cfg.ID}
	}
	return appcfg.ResolveBSCPConfig(ctx, sourceType, ref, fileFormat)
}
