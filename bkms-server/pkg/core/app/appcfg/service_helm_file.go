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

package appcfg

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateHelmFile 保存 Helm 编辑表单。名称有变化时只更新 def，不追加文件版本。
// 基础文件或 BSCP 有变化时才追加一条文件版本。
func (s *AppCfgFileDefService) UpdateHelmFile(
	ctx context.Context,
	def *AppConfigFileDef,
	file *AppConfigFile,
	update HelmFileUpdate,
) (*HelmFileUpdateResult, error) {
	if def == nil || file == nil {
		return nil, errors.New("def and file are required")
	}
	if file.DefID != def.ID {
		return nil, errors.Errorf("file %s does not belong to def %s", file.ID.Hex(), def.ID.Hex())
	}

	result := &HelmFileUpdateResult{Def: def, File: file}
	previousName := def.Name
	if update.Name != previousName {
		if err := s.UpdateAppCfgFileDef(ctx, def, FileDefUpdate{
			Name:     &update.Name,
			AppType:  update.AppType,
			Operator: update.Operator,
		}); err != nil {
			return nil, errors.Wrapf(err, "update helm file name for def %s", def.ID.Hex())
		}
		result.NameChanged = true
	}

	if !helmFileFieldsChanged(file, update) {
		return result, nil
	}

	if update.BaseAppConfigFileID != nil {
		file.BaseAppConfigFileID = update.BaseAppConfigFileID
	}
	if update.BSCPConfig != nil {
		file.BSCPConfig = update.BSCPConfig
	}
	if err := s.UpdateFile(ctx, file, def.Name, update.Operator, UpdateCfgFileOptions{
		OperationType:          AppConfigFileVersionOperationTypeUpdate,
		Description:            update.Description,
		ExpectedCurrentVersion: update.ExpectedCurrentVersion,
	}); err != nil {
		if result.NameChanged {
			if restoreErr := s.UpdateAppCfgFileDef(ctx, def, FileDefUpdate{
				Name:     &previousName,
				AppType:  update.AppType,
				Operator: update.Operator,
			}); restoreErr != nil {
				return nil, errors.Wrapf(
					restoreErr,
					"restore helm file name %q for def %s after file update failed: %v",
					previousName, def.ID.Hex(), err,
				)
			}
		}
		return nil, errors.Wrapf(err, "update helm file %s", file.ID.Hex())
	}
	result.FileChanged = true
	return result, nil
}

func helmFileFieldsChanged(file *AppConfigFile, update HelmFileUpdate) bool {
	if update.BaseAppConfigFileID != nil && !sameObjectID(file.BaseAppConfigFileID, update.BaseAppConfigFileID) {
		return true
	}
	if update.BSCPConfig != nil && !sameBSCPConfig(file.BSCPConfig, update.BSCPConfig) {
		return true
	}
	return false
}

func sameObjectID(left, right *bson.ObjectID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameBSCPConfig(left, right *BSCPConfig) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.BizID == right.BizID &&
		left.ServiceID == right.ServiceID &&
		left.VersionID == right.VersionID &&
		left.ConfigID == right.ConfigID
}
