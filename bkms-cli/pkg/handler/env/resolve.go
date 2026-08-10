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

// Package env 提供环境相关的公共处理能力。
package env

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ResolveEnvIDByName 通过环境名称解析出环境 ID。
// 该函数通过 ListEnvs 获取工作空间下的环境列表，按 name 匹配找到目标环境并返回其 ID。
func ResolveEnvIDByName(ctx context.Context, cli client.Client, workspaceID, envName string) (string, error) {
	envs, err := cli.ListEnvs(ctx, workspaceID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to list envs for workspace %s", workspaceID)
	}

	for i := range envs {
		if envs[i].Name == envName {
			return envs[i].ID, nil
		}
	}

	return "", errors.Errorf("environment '%s' not found in workspace", envName)
}

// ResolveScopedEnvVarID 通过 key + scopeType + scopeValue 从公共环境变量列表中查找变量 ID。
// 用于 public 级别的 update/delete 命令，通过人类可读的标识定位变量。
func ResolveScopedEnvVarID(
	ctx context.Context, cli client.Client, workspaceID, key, scopeType, scopeValue string,
) (string, error) {
	envVars, err := cli.ListPublicEnvVars(ctx, workspaceID)
	if err != nil {
		return "", errors.Wrap(err, "list public env vars")
	}

	var matched []client.ScopedEnvVar
	for i := range envVars {
		if envVars[i].Key == key && envVars[i].ScopeType == scopeType && envVars[i].ScopeValue == scopeValue {
			matched = append(matched, envVars[i])
		}
	}

	switch len(matched) {
	case 0:
		if scopeValue != "" {
			return "", errors.Errorf(
				"environment variable '%s' not found (scopeType=%s, scopeValue=%s)", key, scopeType, scopeValue)
		}
		return "", errors.Errorf("environment variable '%s' not found (scopeType=%s)", key, scopeType)
	case 1:
		return matched[0].ID, nil
	default:
		return "", errors.Errorf(
			"ambiguous match: found %d variables with key '%s' (scopeType=%s, scopeValue=%s)",
			len(matched), key, scopeType, scopeValue)
	}
}

// ResolveEnvScopedEnvVarID 通过 envID + key 从环境级别环境变量列表中查找变量 ID。
// 用于 env 级别的 update/delete 命令，通过人类可读的标识定位变量。
func ResolveEnvScopedEnvVarID(ctx context.Context, cli client.Client, envID, key string) (string, error) {
	envVars, err := cli.ListEnvScopedEnvVars(ctx, envID)
	if err != nil {
		return "", errors.Wrap(err, "list env scoped env vars")
	}

	var matched []client.ScopedEnvVarDetailed
	for i := range envVars {
		if envVars[i].ScopedEnvVar != nil && envVars[i].ScopedEnvVar.Key == key {
			matched = append(matched, envVars[i])
		}
	}

	switch len(matched) {
	case 0:
		return "", errors.Errorf("environment variable '%s' not found in env", key)
	case 1:
		return matched[0].ScopedEnvVar.ID, nil
	default:
		return "", errors.Errorf("ambiguous match: found %d variables with key '%s' in env", len(matched), key)
	}
}

// ParseScope 解析 --scope 参数值，格式为 "workspace" 或 "envType:<value>"。
// 未指定时默认为 workspace 级别。
// 返回 scopeType 和 scopeValue。
func ParseScope(scope string) (scopeType, scopeValue string, err error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "workspace", "", nil
	}

	// 不含冒号：仅支持 "workspace"
	if !strings.Contains(scope, ":") {
		if scope != "workspace" {
			return "", "", errors.Errorf(
				"invalid --scope %q: expected 'workspace' or 'envType:<value>'", scope)
		}
		return "workspace", "", nil
	}

	// 含冒号：按第一个冒号分割
	parts := strings.SplitN(scope, ":", 2)
	scopeType = strings.TrimSpace(parts[0])
	scopeValue = strings.TrimSpace(parts[1])

	if scopeType != "envType" {
		return "", "", errors.Errorf(
			"invalid --scope %q: only 'workspace' and 'envType:<value>' are supported", scope)
	}
	if scopeValue == "" {
		return "", "", errors.New("--scope 'envType:' requires a non-empty value, e.g. 'envType:development'")
	}

	return scopeType, scopeValue, nil
}
