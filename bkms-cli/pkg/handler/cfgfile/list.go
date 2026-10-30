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

package cfgfile

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// DefOutput 是配置文件列表中的单行输出。
type DefOutput struct {
	Name               string `json:"name" yaml:"name"`
	ConfigKind         string `json:"configKind" yaml:"configKind"`
	MountDir           string `json:"mountDir" yaml:"mountDir"`
	MountEnvs          string `json:"mountEnvs" yaml:"mountEnvs"`
	EnvMode            string `json:"envMode" yaml:"envMode"`
	EnvContentStrategy string `json:"envContentStrategy,omitempty" yaml:"envContentStrategy,omitempty" table:"-"`
	EnableEnvVarRender bool   `json:"enableEnvVarRender" yaml:"enableEnvVarRender"`
	FileFormat         string `json:"fileFormat" yaml:"fileFormat" table:"-"`
	CurrentVersion     int64  `json:"currentVersion" yaml:"currentVersion"`
	Updater            string `json:"updater" yaml:"updater" table:"-"`
	UpdatedAt          string `json:"updatedAt" yaml:"updatedAt"`
	ID                 string `json:"id" yaml:"id"`
	FileID             string `json:"fileId" yaml:"fileId" table:"-"`
}

// List 返回应用下所有配置文件定义。
func List(ctx context.Context, cli client.Client, appID string) ([]DefOutput, error) {
	defs, err := cli.ListAppConfigFileDefs(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list app config files")
	}
	return lo.Map(defs, func(def client.AppConfigFileDef, _ int) DefOutput { return toDefOutput(&def) }), nil
}

func toDefOutput(def *client.AppConfigFileDef) DefOutput {
	return DefOutput{
		Name:               def.Name,
		ConfigKind:         def.ConfigKind,
		MountDir:           def.MountDir,
		MountEnvs:          formatMountEnvs(def),
		EnvMode:            formatEnvMode(def),
		EnvContentStrategy: def.EnvContentStrategy,
		EnableEnvVarRender: def.EnableEnvVarRender,
		FileFormat:         def.FileFormat,
		CurrentVersion:     def.CurrentVersion,
		Updater:            def.Updater,
		UpdatedAt:          def.UpdatedAt,
		ID:                 def.ID,
		FileID:             def.FileID,
	}
}
