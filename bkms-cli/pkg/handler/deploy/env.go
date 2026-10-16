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

package deploy

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// parseEnvNames 解析逗号分隔的环境名称，去空格、去空、去重。
func parseEnvNames(envName string) []string {
	names := lo.Map(strings.Split(envName, ","), func(part string, _ int) string {
		return strings.TrimSpace(part)
	})
	return lo.Uniq(lo.Filter(names, func(name string, _ int) bool {
		return name != ""
	}))
}

// validateEnvNames 校验环境名称是否存在于应用可用环境。
func validateEnvNames(ctx context.Context, cli client.Client, appID string, envNames []string) error {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "failed to list envs for app %s", appID)
	}
	// 构建已存在的环境名称集合
	envSet := lo.SliceToMap(envs, func(env client.Env) (string, bool) {
		return env.Name, true
	})

	// 校验所有输入的环境名称
	notFound := lo.Filter(envNames, func(name string, _ int) bool {
		return !envSet[name]
	})

	if len(notFound) > 0 {
		return errors.Errorf("env(s) not found: %v", notFound)
	}

	return nil
}

// resolveEnvNames 解析目标环境名称，envName 按名称、envType 按类型，二者互斥。
func resolveEnvNames(
	ctx context.Context,
	cli client.Client,
	appID, envName, envType string,
) ([]string, error) {
	if envName != "" && envType != "" {
		return nil, errors.New("--env and --env-type are mutually exclusive")
	}

	if envName != "" {
		names := parseEnvNames(envName)
		if len(names) == 0 {
			return nil, errors.New("env name is required")
		}
		return names, nil
	}

	if envType == "" {
		return nil, errors.New("env name or env type is required")
	}

	types := parseEnvNames(envType)
	if len(types) == 0 {
		return nil, errors.New("env type is required")
	}

	return resolveEnvNamesByTypes(ctx, cli, appID, types)
}

// ResolveAndValidateEnvNames 解析目标环境名称，并按名称指定时校验其合法性。
func ResolveAndValidateEnvNames(
	ctx context.Context,
	cli client.Client,
	appID, envName, envType string,
) ([]string, error) {
	envNames, err := resolveEnvNames(ctx, cli, appID, envName, envType)
	if err != nil {
		return nil, err
	}

	if envType == "" {
		if err := validateEnvNames(ctx, cli, appID, envNames); err != nil {
			return nil, err
		}
	}

	return envNames, nil
}

// resolveEnvNamesByTypes 按环境类型展开为应用可用环境名称列表。
func resolveEnvNamesByTypes(
	ctx context.Context,
	cli client.Client,
	appID string,
	envTypes []string,
) ([]string, error) {
	for _, t := range envTypes {
		if !isValidEnvType(t) {
			return nil, errors.Errorf(
				"invalid env type %q: valid types are development, test, staging, production", t)
		}
	}

	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list envs for app %s", appID)
	}

	typeSet := lo.SliceToMap(envTypes, func(t string) (string, bool) {
		return t, true
	})

	names := lo.FilterMap(envs, func(env client.Env, _ int) (string, bool) {
		return env.Name, typeSet[env.Type]
	})

	if len(names) == 0 {
		return nil, errors.Errorf("no envs found for type(s): %s", strings.Join(envTypes, ","))
	}

	return names, nil
}

// isValidEnvType 判断环境类型是否合法。
func isValidEnvType(envType string) bool {
	switch envType {
	case constant.EnvTypeDevelopment, constant.EnvTypeTest, constant.EnvTypeStaging, constant.EnvTypeProduction:
		return true
	default:
		return false
	}
}
