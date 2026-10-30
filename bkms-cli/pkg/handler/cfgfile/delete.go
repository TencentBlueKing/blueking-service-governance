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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// DeletePlan 是删除配置文件前收集的信息，供用户确认。
type DeletePlan struct {
	// Def 待删除的配置文件定义。
	Def *client.AppConfigFileDef
	// EnvInstances 将被一并删除的环境独立实例。
	EnvInstances []string
}

// PlanDelete 定位待删除的配置文件，并收集将被级联删除的环境实例。
func PlanDelete(ctx context.Context, cli client.Client, appID string, ref FileRef) (*DeletePlan, error) {
	def, err := ResolveDef(ctx, cli, appID, ref)
	if err != nil {
		return nil, err
	}
	instances, err := listEnvInstanceNames(ctx, cli, appID, def.ID)
	if err != nil {
		return nil, err
	}
	return &DeletePlan{Def: def, EnvInstances: instances}, nil
}

// ApplyDelete 删除配置文件定义及其所有文件实例和版本记录。
func ApplyDelete(ctx context.Context, cli client.Client, appID string, plan *DeletePlan) error {
	if err := cli.DeleteAppConfigFileDef(ctx, appID, plan.Def.ID); err != nil {
		return errors.Wrap(err, "delete app config file")
	}
	return nil
}
