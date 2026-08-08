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
	// ConflictLevelWarn 软冲突：匹配范围重叠但版本号规则不同，允许保存但需告警。
	// 命中时每条策略各触发一次构建、产出多个镜像
	ConflictLevelWarn ConflictLevel = "warn"
	// ConflictLevelError 硬冲突：匹配范围重叠且版本号规则完全相同，禁止保存。
	// 同一次推送会产出同名镜像并互相覆盖
	ConflictLevelError ConflictLevel = "error"
)

// ConflictCheckInput is the JSON body for pre-checking policy overlap conflicts.
type ConflictCheckInput struct {
	// 待检测的策略表单
	Policy *PolicyFormInput `json:"policy" binding:"required"`
	// 排除的策略 ID，编辑场景下用于排除自身；新建场景留空
	ExcludeTriggerID string `json:"excludeTriggerID"`
}

// ConflictCheckOutputObj is the JSON representation of a conflict check result.
type ConflictCheckOutputObj struct {
	// 冲突级别：none 无冲突，warn 软冲突（可保存），error 硬冲突（禁止保存）
	Level string `json:"level"`
	// 发生冲突的已有策略名列表，无冲突时为空数组
	ConflictPolicyNames []string `json:"conflictPolicyNames"`
}

// ConflictCheckOutput is the JSON response for pre-checking policy overlap conflicts.
type ConflictCheckOutput struct {
	// 冲突检测结果
	Data *ConflictCheckOutputObj `json:"data"`
}
