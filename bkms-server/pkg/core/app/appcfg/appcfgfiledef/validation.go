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

// 本文件的校验函数与旧 handler（handler/app_config_file_validation.go）中的
// validateBaseAppConfigFileID、validateBSCPConfig 逻辑一致。
// 为避免对旧接口做结构性改动，本次单独实现；待旧接口废弃后统一收敛到公共层。

import (
	"context"
	stderrors "errors"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

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
	obj, err := h.registry.AppConfigFileStore.GetByID(ctx, baseObjID)
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
	if sourceType != appcfg.ContentSourceTypeBSCP {
		return nil, nil
	}
	if cfg == nil {
		return nil, stderrors.New("bscpConfig is required when contentSourceType is bscp")
	}

	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "initializing bscp client")
	}
	versions, err := client.ListServiceVersions(ctx, cfg.BizID, cfg.ServiceID)
	if err != nil {
		return nil, errors.Wrap(err, "listing bscp service versions")
	}
	ver := versions.LatestFullyReleased()
	if ver == nil {
		return nil, stderrors.New("no fully released bscp version")
	}
	svcCfg, err := client.GetServiceConfig(ctx, cfg.BizID, cfg.ServiceID, ver.ID, cfg.ID)
	if err != nil {
		return nil, errors.Wrap(err, "getting bscp service config")
	}
	content, err := svcCfg.Content(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetching bscp config content")
	}
	// BSCP 内容也需要做格式校验
	if content != "" && fileFormat == appcfg.FileFormatYAML {
		var out any
		if err := yaml.Unmarshal([]byte(content), &out); err != nil {
			return nil, errors.Wrap(err, "bscp config content is not valid YAML")
		}
	}

	return &appcfg.BSCPConfig{
		BizID:     cfg.BizID,
		ServiceID: cfg.ServiceID,
		VersionID: ver.ID,
		ConfigID:  cfg.ID,
	}, nil
}
