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

// EditOptions 修改配置内容的参数。
type EditOptions struct {
	// Content 写入可编辑字段的内容。
	Content string
	// Description 版本描述。
	Description string
}

// EditResult 修改配置内容的结果，写入内容失败时仍会返回，以便调用方得知模式已切换。
type EditResult struct {
	// Def 被修改的配置文件定义。
	Def *client.AppConfigFileDef
	// EnvName 目标环境，空字符串表示默认文件。
	EnvName string
	// SwitchedToPerEnv 为写入指定环境，文件从统一配置切换为按环境独立配置。
	SwitchedToPerEnv bool
	// CreatedEnvInstance 本次写入为该环境新建了独立实例。
	CreatedEnvInstance bool
	// UpdateResult 后端返回的写入结果。
	UpdateResult *client.AppConfigFileDefContentUpdateResult
}

// Edit 修改配置内容：envName 为空时写默认文件，否则只写该环境。
//
// 统一配置下后端会把环境内容写进默认文件（影响所有环境），因此指定环境时先把文件切换为按环境独立配置。
// 切换本身不删除数据，未单独配置的环境仍使用默认内容。
func Edit(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	envName string,
	opts EditOptions,
) (*EditResult, error) {
	target, err := loadTarget(ctx, cli, appID, ref, envName)
	if err != nil {
		return nil, err
	}
	result := &EditResult{Def: target.Def, EnvName: envName}

	if envName != "" && target.Def.IsUnifiedConfig {
		if _, err = cli.UpdateAppConfigFileDef(ctx, appID, target.Def.ID, client.UpdateAppConfigFileDefBody{
			IsUnifiedConfig: lo.ToPtr(false),
		}); err != nil {
			return nil, errors.Wrap(err, "switch app config file to per-env config")
		}
		result.SwitchedToPerEnv = true
	}

	body := client.UpdateAppConfigFileDefContentBody{Content: opts.Content, Description: opts.Description}
	// 环境尚无独立实例时，后端会新建实例，此时没有可比较的版本号
	result.CreatedEnvInstance = result.SwitchedToPerEnv || target.SharesDefault()
	if !result.CreatedEnvInstance {
		body.CurrentVersion = lo.ToPtr(target.Detail.CurrentVersion)
	}

	result.UpdateResult, err = cli.UpdateAppConfigFileDefContent(ctx, appID, target.Def.ID, envName, body)
	if err != nil {
		// 已切换的环境配置模式不回滚（切换本身不改变生效内容），返回 result 供调用方提示
		return result, errors.Wrap(err, "update app config file content")
	}
	return result, nil
}
