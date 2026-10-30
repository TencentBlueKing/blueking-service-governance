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
)

// Precheck 执行部署前检查。
// helm 类型不支持预检，返回错误；trpc、taf、bkmsApp 统一走 bkmsapp-deploys/precheck。
func Precheck(ctx context.Context, appID, envName string) (*client.DeployPrecheckResult, error) {
	cli := client.New()

	app, err := cli.GetApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "get app")
	}

	if !client.IsAppModelType(app.Type) {
		return nil, errors.Errorf("precheck is not supported for app type '%s' (only appmodel apps)", app.Type)
	}
	return cli.PreCheckAppModelDeploy(ctx, appID, envName)
}
