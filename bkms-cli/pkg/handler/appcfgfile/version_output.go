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

package appcfgfile

import (
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// VersionOutput is the structured output object for app config file version data.
type VersionOutput struct {
	// 基础字段
	Name            string `json:"name" yaml:"name"`
	ID              string `json:"id" yaml:"id"`
	AppConfigFileID string `json:"appConfigFileID" yaml:"appConfigFileID"`
	EnvName         string `json:"envName" yaml:"envName"`
	Type            string `json:"type" yaml:"type"`
	Description     string `json:"description" yaml:"description"`
	// 版本相关字段
	Version             int64  `json:"version" yaml:"version"`
	BaseVersion         *int64 `json:"baseVersion,omitempty" yaml:"baseVersion,omitempty" table:"-"`
	RollbackFromVersion *int64 `json:"rollbackFromVersion,omitempty" yaml:"rollbackFromVersion,omitempty" table:"-"`
	OperationType       string `json:"operationType" yaml:"operationType"`
	// 内容相关字段
	Content        *string `json:"content,omitempty" yaml:"content,omitempty" table:"-"`
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty" table:"-"`
}

func toVersionOutput(version client.AppConfigFileVersion) (VersionOutput, error) {
	output := VersionOutput{}
	if err := copier.Copy(&output, &version); err != nil {
		return VersionOutput{}, errors.Wrap(err, "copy app config file version output")
	}

	output.EnvName = formatEnvName(version.EnvName)
	return output, nil
}
