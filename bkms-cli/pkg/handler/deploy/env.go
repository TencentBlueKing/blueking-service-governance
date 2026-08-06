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

// validateEnvNames 通过 ListEnvs API 校验所有环境名称是否真实存在。
// 一次性拉取环境列表并校验，不存在的环境名称汇总后返回错误。
func validateEnvNames(ctx context.Context, cli client.Client, workspaceID string, envNames []string) error {
	envs, err := cli.ListEnvs(ctx, workspaceID)
	if err != nil {
		return errors.Wrapf(err, "failed to list envs for workspace %s", workspaceID)
	}
	// 构建已存在的环境名称集合
	envSet := make(map[string]bool, len(envs))

	for _, env := range envs {
		envSet[env.Name] = true
	}

	// 校验所有输入的环境名称
	var notFound []string
	for _, name := range envNames {
		if _, ok := envSet[name]; !ok {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		return errors.Errorf("env(s) not found: %v", notFound)
	}

	return nil
}
