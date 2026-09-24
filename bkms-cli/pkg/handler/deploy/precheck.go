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

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// Precheck 根据应用类型路由到对应的部署前检查接口。
// helm 类型不支持预检，返回错误；trpc/taf 返回检查结果。
func Precheck(ctx context.Context, appID, envName string) (*client.DeployPrecheckResult, error) {
	cli := client.New()

	// 校验环境合法性并按可见环境名单拦截，同时复用返回的应用详情做类型路由
	app, err := validateDeployEnvs(ctx, cli, appID, []string{envName})
	if err != nil {
		return nil, err
	}

	switch app.Type {
	case constant.AppTypeTrpc:
		return cli.PreCheckTrpcDeploy(ctx, appID, envName)
	case constant.AppTypeTaf:
		return cli.PreCheckTafDeploy(ctx, appID, envName)
	default:
		return nil, errors.Errorf("precheck is not supported for app type '%s' (only trpc/taf)", app.Type)
	}
}
