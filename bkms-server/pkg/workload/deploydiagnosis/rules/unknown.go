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

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"

type unknownHit struct{}

func (unknownHit) evidenceLines() []string { return nil }

type unknownRule struct{}

func (unknownRule) ID() string { return IDUnknown }

func (unknownRule) Match(*Snapshot, *Instance) (Hit, bool) {
	return unknownHit{}, true
}

func (r unknownRule) Render(snap *Snapshot, inst *Instance, _ Hit) *serializer.DiagnosisGroupObj {
	return buildGroup(r.spec(), snap, inst, nil, nil)
}

func (unknownRule) spec() spec {
	return spec{
		id:          IDUnknown,
		name:        "未识别的异常状态",
		category:    "启动失败",
		confidence:  "low",
		cause:       "实例处于异常状态，但尚未匹配到更具体的规则",
		causeDetail: "尚未匹配到更具体的规则。请结合下方日志、事件和部署配置继续排查。",
		solutions: []string{
			"查看下方日志与事件，确认容器退出原因",
			"检查部署配置后重新部署",
		},
		actions: []serializer.DiagnosisActionObj{
			{ID: serializer.ActionDeployConfig, Label: "前往部署配置"},
			{ID: serializer.ActionRedeploy, Label: "重新部署"},
		},
	}
}
