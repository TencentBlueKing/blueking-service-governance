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

// EnvInstanceOutput 是配置文件下单个文件实例的输出。
type EnvInstanceOutput struct {
	EnvName        string  `json:"envName" yaml:"envName"`
	Type           string  `json:"type" yaml:"type"`
	CurrentVersion int64   `json:"currentVersion" yaml:"currentVersion"`
	Updater        string  `json:"updater" yaml:"updater"`
	UpdatedAt      string  `json:"updatedAt" yaml:"updatedAt"`
	FileID         string  `json:"fileId" yaml:"fileId"`
	Content        *string `json:"content,omitempty" yaml:"content,omitempty" table:"-"`
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty" table:"-"`
}

// ListEnvInstances 列出配置文件的默认文件及所有环境独立实例。
func ListEnvInstances(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
) ([]EnvInstanceOutput, error) {
	def, err := ResolveDef(ctx, cli, appID, ref)
	if err != nil {
		return nil, err
	}
	instances, err := cli.ListAppConfigFileEnvInstances(ctx, appID, def.ID)
	if err != nil {
		return nil, errors.Wrap(err, "list app config file env instances")
	}
	return lo.Map(instances, func(item client.AppConfigFileEnvInstance, _ int) EnvInstanceOutput {
		return EnvInstanceOutput{
			EnvName:        formatEnvName(item.EnvName),
			Type:           item.Type,
			CurrentVersion: item.CurrentVersion,
			Updater:        item.Updater,
			UpdatedAt:      item.UpdatedAt,
			FileID:         item.FileID,
			Content:        item.Content,
			OverlayContent: item.OverlayContent,
		}
	}), nil
}

// listEnvInstanceNames 返回配置文件下所有环境独立实例的环境名（不含默认文件）。
func listEnvInstanceNames(ctx context.Context, cli client.Client, appID, defID string) ([]string, error) {
	instances, err := cli.ListAppConfigFileEnvInstances(ctx, appID, defID)
	if err != nil {
		return nil, errors.Wrap(err, "list app config file env instances")
	}
	return lo.FilterMap(instances, func(item client.AppConfigFileEnvInstance, _ int) (string, bool) {
		return item.EnvName, item.EnvName != ""
	}), nil
}
