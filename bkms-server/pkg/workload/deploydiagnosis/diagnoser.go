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

package deploydiagnosis

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

// Diagnoser 聚合已有数据并跑规则，输出一次诊断结论
type Diagnoser struct {
	registry *storereg.Registry
}

// NewDiagnoser 创建诊断器
func NewDiagnoser(registry *storereg.Registry) *Diagnoser {
	return &Diagnoser{registry: registry}
}

// Diagnose 聚合实例 / 日志 / 事件 / 部署记录，按规则输出可渲染结论
func (d *Diagnoser) Diagnose(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
) (*serializer.DeployDiagnosisOutputObj, error) {
	snap, err := d.collectSnapshot(ctx, app, env, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "collect diagnosis snapshot for app %s env %s", app.ID, env.Name)
	}
	return d.diagnose(snap), nil
}
