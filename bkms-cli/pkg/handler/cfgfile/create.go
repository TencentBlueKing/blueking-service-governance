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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// CreateOptions 创建 plain 配置文件的参数。
type CreateOptions struct {
	// Name 文件名。
	Name string
	// MountDir 容器内挂载目录。
	MountDir string
	// Content 初始内容，nil 表示空文件。
	Content *string
	// MountEnvNames 挂载环境范围，nil 表示挂载到所有环境。
	MountEnvNames []string
	// EnableEnvVarRender 是否启用环境变量渲染，nil 表示使用后端默认值（关闭）。
	EnableEnvVarRender *bool
	// Description 版本描述。
	Description string
}

// Create 创建 plain 配置文件。
// framework 文件随应用创建，不通过 CLI 新建。
func Create(
	ctx context.Context,
	cli client.Client,
	appID string,
	opts CreateOptions,
) (*client.AppConfigFileDef, error) {
	body := client.CreateAppConfigFileDefBody{
		Name:       opts.Name,
		ConfigKind: client.ConfigKindPlain,
		MountDir:   opts.MountDir,
		// plain 文件不校验内容格式，后端仍要求 fileFormat 等字段，按固定值传入
		FileFormat:         "yaml",
		FileType:           "normal",
		ContentSourceType:  "local",
		Content:            opts.Content,
		EnableEnvVarRender: opts.EnableEnvVarRender,
		Description:        opts.Description,
	}
	if opts.MountEnvNames != nil {
		envNames, err := resolveMountEnvNames(ctx, cli, appID, opts.MountEnvNames)
		if err != nil {
			return nil, err
		}
		body.MountedEnvNames = &envNames
	}

	def, err := cli.CreateAppConfigFileDef(ctx, appID, body)
	if err != nil {
		return nil, errors.Wrap(err, "create app config file")
	}
	return def, nil
}
