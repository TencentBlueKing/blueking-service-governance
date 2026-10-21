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

// spec 一条规则的固定文案和按钮，命中后填进诊断分组
type spec struct {
	// id 规则标识，作标签 key，不直接展示
	id string
	// name 标签文案，展示为「规则名 · 实例数」
	name string
	// category 出错原因旁的分类，如「配置下发」
	category string
	// confidence 置信度，面板不展示
	confidence string
	// cause 出错原因主句，原样展示
	cause string
	// causeDetail 主句下的说明，原样展示
	causeDetail string
	// solutions 处理步骤，按数组顺序编号展示
	solutions []string
	// actions 解决方案按钮，id 决定跳转，label 是按钮文案
	actions []serializer.DiagnosisActionObj
}

func buildGroup(
	rule spec,
	snap *Snapshot,
	inst *Instance,
	evidence []string,
	configs []*serializer.RelatedConfigRowObj,
) *serializer.DiagnosisGroupObj {
	if evidence == nil {
		evidence = []string{}
	}
	if configs == nil {
		configs = []*serializer.RelatedConfigRowObj{}
	}
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
		Instances: []*serializer.DiagnosisInstanceObj{
			{
				ID:             inst.ID,
				Status:         inst.Status,
				RestartCount:   inst.RestartCount,
				Evidence:       evidence,
				RelatedConfigs: configs,
				LogSource:      inst.LogSource,
				LogHint:        inst.LogHint,
				LogLines:       append([]string{}, inst.LogLines...),
				RelatedEvents:  relatedEvents(snap.Events, inst.ID),
			},
		},
	}
}

func buildActions(rule spec) []*serializer.DiagnosisActionObj {
	// 重新部署始终下发；没权限点下去走现有部署弹窗报错，诊断接口不预裁
	actions := make([]*serializer.DiagnosisActionObj, 0, len(rule.actions))
	for i := range rule.actions {
		copied := rule.actions[i]
		actions = append(actions, &copied)
	}
	return actions
}

func relatedEvents(events []Event, instanceID string) []*serializer.DiagnosisEventObj {
	out := make([]*serializer.DiagnosisEventObj, 0)
	for _, ev := range events {
		if ev.InstanceID != "" && ev.InstanceID != instanceID {
			continue
		}
		out = append(out, &serializer.DiagnosisEventObj{
			CreatedAt: ev.CreatedAt,
			Level:     ev.Level,
			Content:   ev.Content,
		})
		if len(out) >= maxRelatedEvents {
			break
		}
	}
	return out
}
