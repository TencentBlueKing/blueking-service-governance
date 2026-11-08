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

const (
	// MaxEvidenceLines 单条规则最多保留的证据行数
	MaxEvidenceLines = 3
	// maxRelatedEvents 单个实例或分组最多挂上的事件条数
	maxRelatedEvents = 6
)

// spec 一条规则的固定文案和按钮，文案本身定义在 messages.go
type spec struct {
	// id 规则标识
	id string
	// name 标签文案
	name string
	// categoryID 分类标识，展示文案由 categoryNames 查得
	categoryID string
	// confidence 置信度
	confidence string
	// cause 出错原因主句
	cause string
	// causeDetail 主句下的说明；按分支变化的规则在命中时覆盖
	causeDetail string
	// solutions 处理步骤；按分支变化的规则在命中时覆盖
	solutions []string
	// actions 解决方案按钮
	actions []Action
}

// withDetail 返回替换了说明的副本，用于同一规则在不同分支下的文案差异
func (s spec) withDetail(causeDetail string) spec {
	s.causeDetail = causeDetail
	return s
}

// withSolutions 返回替换了处理步骤的副本
func (s spec) withSolutions(solutions []string) spec {
	s.solutions = solutions
	return s
}

// newGroup 按 spec 填好分组头部；切片都复制一份，避免调用方修改到 messages.go 里的共享变量
func newGroup(s spec, events []Event) *Group {
	return &Group{
		RuleID:        s.id,
		RuleName:      s.name,
		CategoryID:    s.categoryID,
		Category:      categoryNames[s.categoryID],
		Confidence:    s.confidence,
		Cause:         s.cause,
		CauseDetail:   s.causeDetail,
		Solutions:     append([]string{}, s.solutions...),
		Actions:       append([]Action{}, s.actions...),
		Instances:     []*GroupInstance{},
		RelatedEvents: pickWorkloadEvents(events),
	}
}

// newInstanceGroup 构造只含一个实例的分组，同规则的多个实例由上层合并
func newInstanceGroup(s spec, snap *Snapshot, inst *Instance, evidence []string, configs []ConfigRow) *Group {
	group := newGroup(s, snap.Events)
	group.Instances = []*GroupInstance{{
		ID:             inst.ID,
		Status:         inst.Status,
		RestartCount:   inst.RestartCount,
		Evidence:       append([]string{}, evidence...),
		RelatedConfigs: append([]ConfigRow{}, configs...),
		LogSource:      inst.LogSource,
		LogHint:        inst.LogHint,
		LogLines:       append([]string{}, inst.LogLines...),
		RelatedEvents:  pickInstanceEvents(snap.Events, inst.ID),
	}}
	return group
}

// pickInstanceEvents 取挂在该实例上的事件
func pickInstanceEvents(events []Event, instanceID string) []Event {
	return pickEvents(events, func(ev Event) bool { return ev.InstanceID == instanceID })
}

// pickWorkloadEvents 取不归属任何实例的工作负载事件
func pickWorkloadEvents(events []Event) []Event {
	return pickEvents(events, func(ev Event) bool { return ev.InstanceID == "" })
}

// pickEvents 按采集顺序取满 maxRelatedEvents 条即停
func pickEvents(events []Event, keep func(Event) bool) []Event {
	out := make([]Event, 0)
	for _, ev := range events {
		if !keep(ev) {
			continue
		}
		out = append(out, ev)
		if len(out) >= maxRelatedEvents {
			break
		}
	}
	return out
}
