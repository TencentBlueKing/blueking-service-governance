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

// InstanceRule 针对单个异常实例的诊断规则
type InstanceRule interface {
	// ID 规则标识；同一规则命中的实例会合并成一组，组的输出顺序与规则集中的顺序一致
	ID() string
	// Diagnose 未命中返回 nil；命中时返回只含该实例的分组，文案和按钮可按命中的分支变化
	Diagnose(snap *Snapshot, inst *Instance) *Group
}

// StageRule 只在没有异常实例时运行的规则，处理部署 / 构建已失败、但实例看不出问题的情况
type StageRule interface {
	// ID 规则标识
	ID() string
	// Diagnose 未命中返回 nil；命中时返回不含实例的分组
	Diagnose(snap *Snapshot) *Group
}

// InstanceRuleSet 按顺序匹配异常实例，第一条命中即停
type InstanceRuleSet []InstanceRule

// StageRuleSet 没有异常实例时按顺序匹配，第一条命中即停
type StageRuleSet []StageRule

var (
	// InstanceRules 异常实例规则。UnknownRule 恒命中，必须放最后
	InstanceRules = InstanceRuleSet{
		FrameworkConfigRule{},
		UnknownRule{},
	}
	// StageRules 没有异常实例时的规则，构建失败比部署失败更具体，排在前面
	StageRules = StageRuleSet{
		BuildFailedRule{},
		DeployFailedRule{},
	}
)

// Diagnose 返回第一条命中规则的分组；规则集以 UnknownRule 结尾时不会返回 nil
func (s InstanceRuleSet) Diagnose(snap *Snapshot, inst *Instance) *Group {
	for _, rule := range s {
		if group := rule.Diagnose(snap, inst); group != nil {
			return group
		}
	}
	return nil
}

// Diagnose 返回第一条命中规则的分组，都未命中返回 nil
func (s StageRuleSet) Diagnose(snap *Snapshot) *Group {
	for _, rule := range s {
		if group := rule.Diagnose(snap); group != nil {
			return group
		}
	}
	return nil
}
