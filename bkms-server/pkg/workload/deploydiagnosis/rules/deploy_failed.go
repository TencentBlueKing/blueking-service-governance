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

type deployFailedRule struct{}

func (deployFailedRule) Match(snap *Snapshot) bool {
	return IsDeployFailed(snap.Deploy.Status)
}

func (r deployFailedRule) Render(snap *Snapshot) *serializer.DiagnosisGroupObj {
	rule := r.spec(len(snap.Instances) == 0)
	return &serializer.DiagnosisGroupObj{
		RuleID:         rule.id,
		RuleName:       rule.name,
		Category:       rule.category,
		Confidence:     rule.confidence,
		Cause:          rule.cause,
		CauseDetail:    rule.causeDetail,
		Solutions:      append([]string{}, rule.solutions...),
		Actions:        buildActions(rule),
		RelatedConfigs: []*serializer.RelatedConfigRowObj{},
		Instances:      []*serializer.DiagnosisInstanceObj{},
	}
}

func (deployFailedRule) spec(noInstance bool) spec {
	causeDetail := "实例目前都已就绪，但最近一次部署没有成功。请对照部署记录说明，确认失败发生在哪一步。"
	if noInstance {
		causeDetail = "当前环境没有实例，最近一次部署也没有成功。请对照部署记录说明，确认失败发生在哪一步。"
	}
	return spec{
		id:          IDUnknown,
		name:        "未识别的部署失败",
		category:    "部署失败",
		confidence:  "low",
		cause:       "最近一次部署没有成功，尚未匹配到更具体的原因",
		causeDetail: causeDetail,
		solutions: []string{
			"查看该环境最新部署记录中的失败说明",
			"确认配置后重新部署",
		},
		actions: []serializer.DiagnosisActionObj{
			{ID: serializer.ActionDeployConfig, Label: "前往部署配置"},
			{ID: serializer.ActionRedeploy, Label: "重新部署"},
		},
	}
}
