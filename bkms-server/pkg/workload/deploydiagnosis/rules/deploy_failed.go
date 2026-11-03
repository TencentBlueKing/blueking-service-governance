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

package rules

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
)

// BuildFailedRule 一键构建部署在构建阶段失败：没有产出新版本，环境里跑的仍是旧版本或者什么都没有
type BuildFailedRule struct{}

// ID 返回 IDBuildFailed
func (BuildFailedRule) ID() string { return IDBuildFailed }

// Diagnose 只认构建阶段的失败；进入部署阶段后的失败交给 DeployFailedRule
func (BuildFailedRule) Diagnose(snap *Snapshot) *Group {
	if !isFailedAt(snap.Deploy, autodeploy.StageBuild) {
		return nil
	}

	// DeployID 为空说明构建前从未有过部署记录
	detail := buildFailedDetailKeepPrevious
	if snap.Deploy.DeployID == "" {
		detail = buildFailedDetailNeverDeploy
	}
	return newGroup(buildFailedSpec.withDetail(detail), snap.Events)
}

// DeployFailedRule 部署已失败、但实例都已就绪或没有实例时的兜底结论
type DeployFailedRule struct{}

// ID 返回 IDDeployFailed
func (DeployFailedRule) ID() string { return IDDeployFailed }

// Diagnose 认直接部署、Helm 部署以及一键构建部署阶段的失败，工作负载事件挂到分组上
func (DeployFailedRule) Diagnose(snap *Snapshot) *Group {
	// 构建阶段的失败由排在前面的 BuildFailedRule 处理，这里再挡一次，避免规则集顺序被调整后重复认领
	if snap.Deploy == nil || snap.Deploy.Phase != deploystatus.DeployPhaseFailed ||
		snap.Deploy.Stage == autodeploy.StageBuild {
		return nil
	}

	detail := deployFailedDetailWithInstances
	if len(snap.Instances) == 0 {
		detail = deployFailedDetailNoInstance
	}
	return newGroup(deployFailedSpec.withDetail(detail), snap.Events)
}

// isFailedAt 判断最新部署是否在指定的一键构建部署阶段失败
func isFailedAt(deploy *Deploy, stage autodeploy.Stage) bool {
	return deploy != nil && deploy.Phase == deploystatus.DeployPhaseFailed && deploy.Stage == stage
}
