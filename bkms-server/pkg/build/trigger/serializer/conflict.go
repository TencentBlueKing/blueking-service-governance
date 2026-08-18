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

package serializer

// ConflictLevel 策略重叠冲突级别
type ConflictLevel string

const (
	// ConflictLevelNone 无冲突
	ConflictLevelNone ConflictLevel = "none"
	// ConflictLevelWarn 软冲突枚举，仅作前向兼容；镜像 tag 已上收到应用配置后本期不返回
	ConflictLevelWarn ConflictLevel = "warn"
	// ConflictLevelError 硬冲突：匹配范围重叠且版本号规则完全相同，禁止保存。
	// 同一次推送会产出同名镜像并互相覆盖
	ConflictLevelError ConflictLevel = "error"
)

// ConflictCheckInput is the JSON body for pre-checking policy overlap conflicts.
type ConflictCheckInput struct {
	// 待检测的策略表单
	Policy *PolicyFormInput `json:"policy" binding:"required"`
	// 编辑时排除自身的策略 ID，新建留空；JSON 字段名为 excludeTriggerID，值不是蓝盾 triggerID
	ExcludeTriggerID string `json:"excludeTriggerID"`
}

// ConflictCheckOutputObj is the JSON representation of a conflict check result.
type ConflictCheckOutputObj struct {
	// 冲突级别：none 无冲突，error 硬冲突（禁止保存）；本期不返回 warn
	Level string `json:"level"`
	// 发生冲突的已有策略名列表，无冲突时为空数组
	ConflictPolicyNames []string `json:"conflictPolicyNames"`
	// 每条冲突的策略名、重叠类型与原因，无冲突时为空数组
	ConflictReasons []ConflictReasonObj `json:"conflictReasons"`
}

// ConflictReasonObj 单条硬冲突原因
type ConflictReasonObj struct {
	// 冲突的已有策略名
	PolicyName string `json:"policyName"`
	// 重叠类型：all / eq_eq / prefix_prefix / eq_hits_prefix
	OverlapType string `json:"overlapType"`
	// 人类可读冲突原因
	Message string `json:"message"`
}

// ConflictCheckOutput is the JSON response for pre-checking policy overlap conflicts.
type ConflictCheckOutput struct {
	// 冲突检测结果
	Data *ConflictCheckOutputObj `json:"data"`
}
