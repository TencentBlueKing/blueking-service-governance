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

// MountPreviewOutput 是挂载预览中的单个文件输出。
type MountPreviewOutput struct {
	Name          string `json:"name" yaml:"name"`
	ConfigKind    string `json:"configKind" yaml:"configKind"`
	MountDir      string `json:"mountDir" yaml:"mountDir"`
	ContentSource string `json:"contentSource" yaml:"contentSource"`
	HasEnvFile    bool   `json:"hasEnvFile" yaml:"hasEnvFile"`
	ID            string `json:"id" yaml:"id"`
}

// MountPreview 返回指定环境下最终挂载到容器的配置文件。
func MountPreview(ctx context.Context, cli client.Client, appID, envName string) ([]MountPreviewOutput, error) {
	if err := ensureAppEnv(ctx, cli, appID, envName); err != nil {
		return nil, err
	}
	items, err := cli.GetAppConfigFileMountPreview(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "get app config file mount preview")
	}
	return lo.Map(items, func(item client.AppConfigFileMountPreviewItem, _ int) MountPreviewOutput {
		return MountPreviewOutput{
			Name:          item.Name,
			ConfigKind:    item.ConfigKind,
			MountDir:      item.MountDir,
			ContentSource: item.ContentSource,
			HasEnvFile:    item.HasEnvFile,
			ID:            item.DefID,
		}
	}), nil
}
