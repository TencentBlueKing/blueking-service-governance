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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// DeleteDeployBatch 批量卸载指定环境，逐个执行并汇总失败。
func DeleteDeployBatch(
	ctx context.Context,
	cli client.Client,
	appType, appID string,
	envNames []string,
	deployID string,
) error {
	var errs []string
	for _, env := range envNames {
		if err := deleteDeploy(ctx, cli, appType, appID, env, deployID); err != nil {
			errs = append(errs, fmt.Sprintf("env %s: %v", env, err))
			console.Error("✗ Deployment delete failed for env: %s (%v)", env, err)
		}
	}
	if len(errs) > 0 {
		return clierr.Reportedf("delete deploy failed for some envs:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// deleteDeploy 根据应用类型路由到对应的删除接口，执行部署删除并打印成功消息
func deleteDeploy(ctx context.Context, cli client.Client, appType, appID, envName, deployID string) error {
	var err error
	switch appType {
	case constant.AppTypeHelm:
		if deployID == "" {
			return errors.New("--deploy-id is required for helm applications (use 'deploy list' to get deploy IDs)")
		}
		err = cli.DeleteHelmDeploy(ctx, appID, envName, deployID)
	case constant.AppTypeTrpc:
		err = cli.DeleteTrpcDeploy(ctx, appID, envName)
	case constant.AppTypeTaf:
		err = cli.DeleteTafDeploy(ctx, appID, envName)
	default:
		return errors.Errorf("unsupported app type '%s' for deploy delete", appType)
	}
	if err != nil {
		return errors.Wrap(err, "delete deploy")
	}

	console.Info("✓ Deployment deleted for app %s in env %s", appID, envName)
	return nil
}
