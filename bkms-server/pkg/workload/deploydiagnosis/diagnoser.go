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

// Package deploydiagnosis 部署诊断：采集环境最新部署的实例、日志、事件和配置，按规则给出出错原因与处理建议
package deploydiagnosis

import (
	"context"
	"time"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
)

// 没有分组时的空态，有结论时 EmptyReason 为空字符串
const (
	// EmptyReasonNoDeploy 该环境从未部署过
	EmptyReasonNoDeploy = "no_deploy"
	// EmptyReasonDeploying 构建或部署仍在进行，滚动中的实例不代表最终状态，不做诊断
	EmptyReasonDeploying = "deploying"
	// EmptyReasonAllHealthy 实例均已启动并通过健康检查
	EmptyReasonAllHealthy = "all_healthy"
	// EmptyReasonNoInstance 已经列过实例，当前环境没有实例
	EmptyReasonNoInstance = "no_instance"
	// EmptyReasonInstancesUnavailable 有部署记录但无法列出实例，例如 Helm 部署记录没有标签选择器
	EmptyReasonInstancesUnavailable = "instances_unavailable"
)

// 采集未取全的数据源
const (
	// PartialReasonLogs 日志未取全：接口失败，或异常实例超过采集上限
	PartialReasonLogs = "logs"
	// PartialReasonEvents 事件未取全：接口失败，或条数超过一页
	PartialReasonEvents = "events"
)

// Diagnosis 一次环境诊断的结论
type Diagnosis struct {
	// GeneratedAt 诊断完成时间
	GeneratedAt time.Time
	// PartialReasons 未取全的数据源，取值见 PartialReason* 常量；为空表示证据完整
	PartialReasons []string
	// EmptyReason Groups 为空时的空态，取值见 EmptyReason* 常量
	EmptyReason string
	// Deploy 最新一次部署；从未部署过时为 nil
	Deploy *rules.Deploy
	// TotalCount 当前环境实例总数
	TotalCount int
	// AbnormalCount 异常实例数，与部署入口的异常数口径一致
	AbnormalCount int
	// Groups 按规则分组的结论，顺序与规则集一致
	Groups []*rules.Group
}

// Diagnoser 聚合已有数据并跑规则，输出一次诊断结论
type Diagnoser struct {
	registry  *storereg.Registry
	statusSvc *deploystatus.DeployStatusService
}

// NewDiagnoser 创建诊断器
func NewDiagnoser(registry *storereg.Registry) *Diagnoser {
	return &Diagnoser{
		registry:  registry,
		statusSvc: deploystatus.NewDeployStatusService(registry.DB),
	}
}

// Diagnose 诊断应用在指定环境、泳道下最新一次部署；只有列实例失败才返回错误，日志和事件失败记入 PartialReasons
func (d *Diagnoser) Diagnose(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
) (*Diagnosis, error) {
	snap, err := d.collectSnapshot(ctx, app, env, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "collect diagnosis snapshot for app %s env %s", app.ID, env.Name)
	}
	return d.diagnose(snap), nil
}
