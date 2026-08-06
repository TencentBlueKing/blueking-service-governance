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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// ListDeploy 查询部署记录，支持多环境（逗号分隔）。
// 多环境时将各环境的记录合并为一个扁平数组返回，保持与单环境一致的表格输出格式。
func ListDeploy(ctx context.Context, workspaceID, appID, envName, trafficLane, keyword string) (interface{}, error) {
	// 解析多环境名称
	envNames := parseEnvNames(envName)
	if len(envNames) == 0 {
		return nil, errors.New("env name is required")
	}

	cli := client.New()

	// 校验所有环境名称合法性
	if err := validateEnvNames(ctx, cli, workspaceID, envNames); err != nil {
		return nil, err
	}

	// 获取应用信息（只需一次）
	app, err := cli.GetAppMinimal(ctx, workspaceID, appID)
	if err != nil {
		return nil, err
	}

	// 单环境时直接返回
	if len(envNames) == 1 {
		return listDeployForEnv(ctx, cli, app, envNames[0], trafficLane, keyword)
	}

	// 多环境时将各环境的记录合并为一个扁平数组
	var merged []any
	var errs []string
	for _, env := range envNames {
		records, listErr := listDeployForEnv(ctx, cli, app, env, trafficLane, keyword)
		if listErr != nil {
			errs = append(errs, fmt.Sprintf("env %s: %v", env, listErr))
			continue
		}
		merged = append(merged, records...)
	}

	if len(errs) > 0 {
		if len(merged) > 0 {
			return merged, errors.Errorf("list deploy failed for some envs:\n  %s", strings.Join(errs, "\n  "))
		}
		return nil, errors.Errorf("list deploy failed for some envs:\n  %s", strings.Join(errs, "\n  "))
	}

	return merged, nil
}

// listDeployForEnv 查询单个环境的部署记录，返回 []any 以便多环境时直接合并
func listDeployForEnv(
	ctx context.Context,
	cli client.Client,
	app *client.AppMinimal,
	envName, trafficLane, keyword string,
) ([]any, error) {
	switch app.Type {
	case constant.AppTypeHelm:
		records, err := cli.ListHelmDeployRecords(ctx, app.ID, envName, trafficLane, keyword)
		if err != nil {
			return nil, err
		}
		return toAnySlice(records), nil

	case constant.AppTypeTrpc:
		records, err := cli.ListTrpcDeployRecords(ctx, app.ID, envName, keyword, trafficLane)
		if err != nil {
			return nil, err
		}
		return toAnySlice(records), nil

	case constant.AppTypeTaf:
		records, err := cli.ListTafDeployRecords(ctx, app.ID, envName, keyword, trafficLane)
		if err != nil {
			return nil, err
		}
		return toAnySlice(records), nil

	default:
		return nil, errors.Errorf("unsupported app type %s", app.Type)
	}
}

// toAnySlice 将具体类型的切片转换为 []any
func toAnySlice[T any](items []T) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}
