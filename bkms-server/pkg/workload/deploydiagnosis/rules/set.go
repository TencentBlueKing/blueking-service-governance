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

// Hit 某条规则命中后交给自己的 Render 使用的证据
type Hit interface {
	evidenceLines() []string
}

// InstanceRule 针对单个异常实例。规则集按顺序调用 Match，命中后只调这一条的 Render
type InstanceRule interface {
	// ID 规则标识，分组输出时按规则集中的顺序排列
	ID() string
	Match(snap *Snapshot, inst *Instance) (Hit, bool)
	Render(snap *Snapshot, inst *Instance, hit Hit) *serializer.DiagnosisGroupObj
}

// StageRule 针对没有异常实例的快照，例如部署已失败但没有 Pod
type StageRule interface {
	Match(snap *Snapshot) bool
	Render(snap *Snapshot) *serializer.DiagnosisGroupObj
}

// InstanceRuleSet 按顺序匹配异常实例，第一条命中即停
type InstanceRuleSet []InstanceRule

// StageRuleSet 没有异常实例时按顺序匹配
type StageRuleSet []StageRule

var (
	// InstanceRules 异常实例规则。unknownRule 必须放最后
	InstanceRules = InstanceRuleSet{
		configFileRule{},
		unknownRule{},
	}
	// StageRules 没有异常实例时的规则。部署失败的兜底放在这里
	StageRules = StageRuleSet{
		deployFailedRule{},
	}
)

// Run 按顺序匹配，把第一条命中的规则渲染成分组
func (s InstanceRuleSet) Run(snap *Snapshot, inst *Instance) *serializer.DiagnosisGroupObj {
	for _, rule := range s {
		hit, ok := rule.Match(snap, inst)
		if !ok {
			continue
		}
		return rule.Render(snap, inst, hit)
	}
	return nil
}

// Run 按顺序匹配，把第一条命中的阶段规则渲染成分组
func (s StageRuleSet) Run(snap *Snapshot) (*serializer.DiagnosisGroupObj, bool) {
	for _, rule := range s {
		if !rule.Match(snap) {
			continue
		}
		return rule.Render(snap), true
	}
	return nil, false
}
