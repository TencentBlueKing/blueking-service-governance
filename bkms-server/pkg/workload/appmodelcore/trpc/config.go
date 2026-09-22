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

package trpc

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var configValidate = validator.New(validator.WithRequiredStructEnabled())

// Config is the typed tRPC frameworkConfig.
type Config struct {
	FileName string `json:"fileName" validate:"required"`
	FilePath string `json:"filePath" validate:"required"`
}

// DecodeConfig strictly decodes a frameworkConfig map into a tRPC config.
func DecodeConfig(cfg map[string]any, version int) (Config, error) {
	dest, err := appruntime.DecodeStrict[Config](cfg, version, appruntime.FrameworkConfigVersionV1)
	if err != nil {
		return Config{}, errors.Wrap(err, "tRPC")
	}
	if err = configValidate.Struct(dest); err != nil {
		return Config{}, errors.Wrap(err, "validate tRPC frameworkConfig")
	}
	return dest, nil
}

// ConfigMapFromLegacy copies file metadata from the compatibility DTO.
// Language and fileContent stay outside frameworkConfig.
func ConfigMapFromLegacy(cfg appmodel.TrpcConfig) map[string]any {
	return appruntime.FileMetaToMap(cfg.FileName, cfg.FilePath)
}
