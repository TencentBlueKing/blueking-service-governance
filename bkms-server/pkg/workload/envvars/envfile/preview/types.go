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

package preview

// ImportAction 单条导入环境变量的处理动作。
type ImportAction string

const (
	// ImportActionNew 表示该 key 在目标作用域中不存在，将作为新增处理。
	ImportActionNew ImportAction = "new"
	// ImportActionOverwrite 表示该 key 在目标作用域中直接定义已存在，将覆盖原值
	ImportActionOverwrite ImportAction = "overwrite"
)

// ImportEffectScope 描述 `.env` 文件中可选的 scope 在本次预览中的处理方式。
type ImportEffectScope string

const (
	// ImportEffectScopeNone 表示未提供 scope 指令
	ImportEffectScopeNone ImportEffectScope = "none"
	// ImportEffectScopeApplied 表示 scope 指令合法且已生效
	ImportEffectScopeApplied ImportEffectScope = "applied"
)

// ImportPreviewScope 表示单条记录上的 scope 信息。
type ImportPreviewScope struct {
	// Type 为 scope 类型（workspace / envType / env）。
	Type string
	// Value 为 scope 值；当 Type 为 workspace 时为空字符串。
	Value string
}

// ImportPreviewItem 表示单条导入环境变量的预览结果
type ImportPreviewItem struct {
	// Key 环境变量名，始终有值。
	Key string
	// Value 导入的变量值，始终有值。
	Value string
	// OriginalValue 被覆盖变量的原值。
	// 仅当 Action 为 overwrite 时有值，否则为空字符串。
	OriginalValue string
	// Description 从 `# desc:` 注释中解析出的描述信息。
	// 仅当 .env 文件中该条记录包含 # desc: 注释时有值，否则为空字符串。
	Description string
	// DeclaredScope 为输入中显式声明的原始 scope 信息；未声明时为 nil。
	DeclaredScope *ImportPreviewScope
	// EffectiveScope 为预览后实际生效的 scope 信息；不适用时为 nil。
	EffectiveScope *ImportPreviewScope
	// Action 导入动作：new（新增）或 overwrite（覆盖），始终有值。
	Action ImportAction
	// EffectScope scope 指令的生效状态，始终有值。
	EffectScope ImportEffectScope
	// Messages 额外提示信息。
	// 仅当存在附加提示信息时有值，否则为 nil。
	Messages []string
}

// ImportPreviewSummary 导入预览的汇总统计。
type ImportPreviewSummary struct {
	// Total 导入条目总数。
	Total int
	// New 新增条目数。
	New int
	// Overwrite 覆盖条目数。
	Overwrite int
}

// ImportPreview 导入预览的完整结果，由所有 import-preview API 共享。
type ImportPreview struct {
	// Items 逐条预览结果。
	Items []ImportPreviewItem
	// Summary 汇总统计。
	Summary ImportPreviewSummary
}
