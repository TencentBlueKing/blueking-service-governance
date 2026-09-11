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

package env

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ResolveFeatureEnv 在应用可用环境中按名称或 ID 定位目标，并校验特性环境归属。
// 删除操作必须拒绝标准环境以及不属于当前应用的特性环境。
// nameOrID 的规范化和必填校验由调用方完成。
func ResolveFeatureEnv(ctx context.Context, cli client.Client, appID, nameOrID string) (*client.Env, error) {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list app envs")
	}
	env, found := lo.Find(envs, func(item client.Env) bool {
		return item.Name == nameOrID || item.ID == nameOrID
	})
	if !found {
		return nil, errors.Errorf("feature environment %q not found for app %q", nameOrID, appID)
	}
	if env.Kind != "feature" || env.OwnerAppID != appID {
		return nil, errors.New("only feature environments owned by this app can be deleted")
	}
	return &env, nil
}

// CreateFeatureEnv 在应用可用环境中按名称或 ID 解析源标准环境，然后创建特性环境。
// 调用方负责参数规范化和必填校验，此处校验来源类型，完整业务规则由服务端校验。
func CreateFeatureEnv(ctx context.Context, cli client.Client, appID, source, displayName string) (*client.Env, error) {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "list app envs")
	}
	sourceEnv, found := lo.Find(envs, func(item client.Env) bool {
		return item.Name == source || item.ID == source
	})
	if !found {
		return nil, errors.Errorf("source environment %q not found for app %q", source, appID)
	}
	if sourceEnv.Kind == "feature" || sourceEnv.OwnerAppID != "" {
		return nil, errors.New("source environment must be a standard environment")
	}

	// 创建接口使用源环境 ID，后续部署命令使用返回的环境 Name。
	created, err := cli.CreateFeatureEnv(ctx, appID, client.CreateFeatureEnvBody{
		SourceEnvID: sourceEnv.ID, DisplayName: displayName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create feature env")
	}
	return created, nil
}
