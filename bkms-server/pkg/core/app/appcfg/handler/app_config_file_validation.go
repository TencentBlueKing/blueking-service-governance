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

package handler

import (
	"context"
	"errors"

	pkgerrors "github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

func (h *Handler) validateAndGetAppConfigFile(
	ctx context.Context,
	appID string,
	id string,
	permType perm.Type,
) (*bkmsapp.Application, *appcfg.AppConfigFile, error) {
	app, err := perm.ValidateAppByID(ctx, h.registry, appID, permType)
	if err != nil {
		return nil, nil, err
	}

	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "parsing ID")
	}
	acf, err := h.configFiles().GetByID(ctx, oid)
	if err != nil {
		return nil, nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "getting values file")
	}
	return app, acf, nil
}

func (h *Handler) validateBaseAppConfigFileID(
	ctx context.Context,
	appID string,
	fileType string,
	baseID string,
) (*bson.ObjectID, error) {
	if appcfg.AppConfigFileType(fileType) != appcfg.AppConfigFileTypeOverlay {
		return nil, nil
	}
	if baseID == "" {
		return nil, errors.New("baseAppConfigFileID is required for overlay type")
	}

	baseObjID, err := bson.ObjectIDFromHex(baseID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "invalid baseAppConfigFileID")
	}
	obj, err := h.configFiles().GetByID(ctx, baseObjID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "getting the base app config file")
	}
	if obj.Type != appcfg.AppConfigFileTypeNormal {
		return nil, errors.New("base app config file must be of type 'normal'")
	}
	if obj.AppID != appID {
		return nil, errors.New("base app config file does not belong to the app")
	}
	return &obj.ID, nil
}

func (h *Handler) validateBSCPConfig(
	ctx context.Context,
	sourceType appcfg.ContentSourceType,
	cfg *slz.BSCPAppConfigFileConfig,
	fileFormat appcfg.FileFormat,
) (*appcfg.BSCPConfig, error) {
	var ref *appcfg.BSCPConfig
	if cfg != nil {
		ref = &appcfg.BSCPConfig{BizID: cfg.BizID, ServiceID: cfg.ServiceID, ConfigID: cfg.ID}
	}
	return appcfg.ResolveBSCPConfig(ctx, sourceType, ref, fileFormat)
}

func validateFileContent(content string, format appcfg.FileFormat) error {
	if content == "" {
		return nil
	}

	var output any
	if err := yaml.Unmarshal([]byte(content), &output); err != nil {
		return pkgerrors.Wrap(err, "values file content is not valid yaml")
	}
	return nil
}
