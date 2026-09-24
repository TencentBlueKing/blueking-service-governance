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
)

// parseEnvNames 解析逗号分隔的环境名称字符串，返回去重后的环境名称切片。
// 自动去除每个名称前后的空格，忽略空字符串。
func parseEnvNames(envName string) []string {
	parts := strings.Split(envName, ",")
	seen := make(map[string]bool)
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}

	return result
}

// validateEnvNames 通过应用可用环境列表校验标准环境及应用专属特性环境。
// 一次性拉取环境列表并校验，不存在的环境名称汇总后返回错误。
func validateEnvNames(ctx context.Context, cli client.Client, appID string, envNames []string) error {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "failed to list envs for app %s", appID)
	}
	return checkEnvsExist(envs, envNames)
}

// validateDeployEnvs 校验部署目标环境：先确认环境存在，再按应用可见环境名单拦截。
// 两项校验共用同一份环境列表；返回拉取到的应用详情，precheck 用它做类型路由。
// 环境不存在时直接返回，不再退化成可见环境错误；拉取应用详情失败时不得跳过可见环境校验。
func validateDeployEnvs(
	ctx context.Context,
	cli client.Client,
	appID string,
	envNames []string,
) (*client.AppFull, error) {
	envs, err := cli.ListAppEnvs(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list envs for app %s", appID)
	}
	if err = checkEnvsExist(envs, envNames); err != nil {
		return nil, err
	}

	app, err := cli.GetApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get app %s", appID)
	}
	if err = checkVisibleEnvs(app, envs, envNames); err != nil {
		return nil, err
	}
	return app, nil
}

// checkEnvsExist 汇总不在应用可用环境列表中的名称。
func checkEnvsExist(envs []client.Env, envNames []string) error {
	envSet := lo.SliceToMap(envs, func(env client.Env) (string, struct{}) {
		return env.Name, struct{}{}
	})
	notFound := lo.Filter(envNames, func(name string, _ int) bool {
		_, ok := envSet[name]
		return !ok
	})
	if len(notFound) > 0 {
		return errors.Errorf("env(s) not found: %v", notFound)
	}
	return nil
}

// checkVisibleEnvs 按应用可见标准环境名单拦截目标环境。
// 名单为空表示未配置，不做限制；应用自己的特性环境无需写入名单，始终允许。
func checkVisibleEnvs(app *client.AppFull, envs []client.Env, envNames []string) error {
	if len(app.VisibleEnvNames) == 0 {
		return nil
	}

	envByName := lo.SliceToMap(envs, func(env client.Env) (string, client.Env) {
		return env.Name, env
	})
	denied := lo.Filter(envNames, func(name string, _ int) bool {
		env := envByName[name]
		isOwnFeatureEnv := env.Kind == client.EnvKindFeature && env.OwnerAppID == app.ID
		return !isOwnFeatureEnv && !lo.Contains(app.VisibleEnvNames, name)
	})
	if len(denied) == 0 {
		return nil
	}
	return errors.Errorf("env(s) not in the application's visible environments: %v", denied)
}
