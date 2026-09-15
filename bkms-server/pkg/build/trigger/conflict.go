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

package trigger

import (
	"fmt"
	"strings"
)

// OverlapType 分支匹配范围重叠类型，同时作为预检响应里的 overlapType 取值
type OverlapType string

const (
	// OverlapTypeAll 与「全部」重叠
	OverlapTypeAll OverlapType = "all"
	// OverlapTypeEqEq 等于值相同
	OverlapTypeEqEq OverlapType = "eq_eq"
	// OverlapTypePrefixPrefix 前缀互相包含
	OverlapTypePrefixPrefix OverlapType = "prefix_prefix"
	// OverlapTypeEqHitsPrefix 等于值命中前缀
	OverlapTypeEqHitsPrefix OverlapType = "eq_hits_prefix"
)

// conflictConsequence 各类冲突共用的后果说明：tag 规则统一取应用 buildConfig.tagConfig，
// 范围一旦重叠，两条策略必然算出相同镜像 tag
const conflictConsequence = "同一次推送会产出同名镜像并互相覆盖"

// ConflictHit 与某条已有策略的一次硬冲突命中
type ConflictHit struct {
	// PolicyName 冲突的已有策略名，直接展示给用户，因此不带策略 ID
	PolicyName string
	// OverlapType 重叠类型，供前端按类型做差异化提示
	OverlapType OverlapType
	// Message 已拼好的可读原因，含策略名与后果
	Message string
}

// PolicyConflictError 策略分支匹配硬冲突，创建 / 更新提交时由业务层返回，映射为 400
type PolicyConflictError struct {
	Hits []ConflictHit
}

// Error 用分号连接全部命中原因；Hits 为空时退回通用文案，避免出现空错误信息
func (e *PolicyConflictError) Error() string {
	if e == nil || len(e.Hits) == 0 {
		return "触发策略分支匹配范围重叠"
	}
	msgs := make([]string, 0, len(e.Hits))
	for _, hit := range e.Hits {
		msgs = append(msgs, hit.Message)
	}
	return strings.Join(msgs, "; ")
}

// detectOverlap 判断待保存表单与某条已有策略的分支匹配范围是否重叠，无重叠返回 nil。
//
// 判定口径：
//   - 只比较相同 event；不同 event 不会由同一次推送同时命中，天然不冲突
//   - pathFilter 不参与：一次推送可以同时改动两条策略各自关心的路径，收窄路径并不能避免同名镜像
//   - 匹配值大小写敏感，比较前已按英文逗号拆分并去掉空白
//   - 不看 status，已停用策略同样占冲突空间，由调用方 collectConflicts 决定比较范围
//
// 两侧匹配方式两两组合共 9 种，按下列顺序收敛为 4 类重叠：任一方为 all 直接命中（all 覆盖全部分支，
// 必然与对侧重叠，无需再看匹配值）；其余按 eq×eq、prefix×prefix、eq×prefix 判定，
// eq 与 prefix 的两个方向结论相同，只是实参顺序相反
func (m *PolicyManager) detectOverlap(candidate PolicyForm, existing Policy) *ConflictHit {
	if candidate.Event != existing.Event {
		return nil
	}

	candMode := candidate.BranchMatchMode
	existMode := existing.BranchMatchMode
	candVals := parseMatchValues(candMode, candidate.BranchMatchValue)
	existVals := parseMatchValues(existMode, existing.BranchMatchValue)

	if candMode == BranchMatchModeAll || existMode == BranchMatchModeAll {
		return overlapHit(existing.Name, OverlapTypeAll)
	}
	if candMode == BranchMatchModeEq && existMode == BranchMatchModeEq {
		if valuesIntersect(candVals, existVals) {
			return overlapHit(existing.Name, OverlapTypeEqEq)
		}
		return nil
	}
	if candMode == BranchMatchModePrefix && existMode == BranchMatchModePrefix {
		if prefixesOverlap(candVals, existVals) {
			return overlapHit(existing.Name, OverlapTypePrefixPrefix)
		}
		return nil
	}
	if candMode == BranchMatchModeEq && existMode == BranchMatchModePrefix {
		if eqHitsPrefix(candVals, existVals) {
			return overlapHit(existing.Name, OverlapTypeEqHitsPrefix)
		}
		return nil
	}
	if candMode == BranchMatchModePrefix && existMode == BranchMatchModeEq {
		if eqHitsPrefix(existVals, candVals) {
			return overlapHit(existing.Name, OverlapTypeEqHitsPrefix)
		}
		return nil
	}
	// 兜底未知匹配方式组合：表单校验已拦掉非法取值，这里不再判为冲突
	return nil
}

// overlapHit 按 overlapType 生成带策略名与固定后果说明的硬冲突命中
func overlapHit(policyName string, overlapType OverlapType) *ConflictHit {
	return &ConflictHit{
		PolicyName:  policyName,
		OverlapType: overlapType,
		Message:     formatConflictMessage(policyName, overlapType),
	}
}

// formatConflictMessage 按重叠类型拼可读原因，文案统一带策略名与 conflictConsequence 后果；
// 该文案会直接透出到接口响应与前端提示
func formatConflictMessage(policyName string, overlapType OverlapType) string {
	switch overlapType {
	case OverlapTypeAll:
		return fmt.Sprintf("与策略「%s」冲突：已有策略匹配全部分支，%s", policyName, conflictConsequence)
	case OverlapTypeEqEq:
		return fmt.Sprintf("与策略「%s」冲突：分支等于值相同，%s", policyName, conflictConsequence)
	case OverlapTypePrefixPrefix:
		return fmt.Sprintf("与策略「%s」冲突：分支前缀互相包含，%s", policyName, conflictConsequence)
	case OverlapTypeEqHitsPrefix:
		return fmt.Sprintf("与策略「%s」冲突：分支等于值命中已有前缀，%s", policyName, conflictConsequence)
	default:
		return fmt.Sprintf("与策略「%s」冲突：分支匹配范围重叠，%s", policyName, conflictConsequence)
	}
}

// parseMatchValues 按英文逗号拆分匹配值，去掉首尾空白并忽略空段；all 模式没有匹配值，返回 nil
func parseMatchValues(mode BranchMatchMode, raw string) []string {
	if mode == BranchMatchModeAll {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		values = append(values, v)
	}
	return values
}

// joinMatchValues 把已规范化的匹配值用英文逗号拼回，中间不保留空白
func joinMatchValues(values []string) string {
	return strings.Join(values, ",")
}

// valuesIntersect 判断两侧等于值是否有交集，大小写敏感；只要有一个分支名相同，同一次推送就会双双命中
func valuesIntersect(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// prefixesOverlap 任一对前缀互相包含即视为重叠，双向判断，例如 feature/ 与 feature/foo：
// 较长前缀能命中的分支必然也被较短前缀命中
func prefixesOverlap(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if strings.HasPrefix(left, right) || strings.HasPrefix(right, left) {
				return true
			}
		}
	}
	return false
}

// eqHitsPrefix 任一等于值以某前缀开头即视为重叠，例如 feature/login 命中 feature/
// 参数顺序不可互换：eqValues 必须来自 eq 一侧，prefixValues 必须来自 prefix 一侧
func eqHitsPrefix(eqValues, prefixValues []string) bool {
	for _, eq := range eqValues {
		for _, prefix := range prefixValues {
			if strings.HasPrefix(eq, prefix) {
				return true
			}
		}
	}
	return false
}
