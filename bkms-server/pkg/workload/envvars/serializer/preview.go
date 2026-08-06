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

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"

// EnvVarImportPreviewScopeOutputObj 单条预览中的 scope 输出。
type EnvVarImportPreviewScopeOutputObj struct {
	// scope 类型（workspace / envType / env）
	Type string `json:"type"`
	// scope 值；workspace 时省略
	Value string `json:"value,omitempty"`
}

// EnvVarImportPreviewItemOutputObj 单条导入环境变量预览结果的 JSON 输出。
type EnvVarImportPreviewItemOutputObj struct {
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量 Value（导入值）
	Value string `json:"value"`
	// 被覆盖变量的原值，仅当 action 为 overwrite 时返回；其他场景省略该字段
	OriginalValue string `json:"originalValue,omitempty"`
	// 描述信息
	Description string `json:"description"`
	// 输入中显式声明的原始 scope 信息；未声明时省略该字段
	DeclaredScope *EnvVarImportPreviewScopeOutputObj `json:"declaredScope,omitempty"`
	// 预览后实际生效的 scope 信息；不适用时省略该字段
	EffectiveScope *EnvVarImportPreviewScopeOutputObj `json:"effectiveScope,omitempty"`
	// 导入动作：new（新增）/ overwrite（覆盖）
	Action string `json:"action"`
	// scope 生效状态：none / applied
	EffectScope string `json:"effectScope"`
	// 额外提示信息；无提示时省略该字段
	Messages []string `json:"messages,omitempty"`
}

// FromModel 从领域模型填充输出字段。
func (o *EnvVarImportPreviewItemOutputObj) FromModel(
	item preview.ImportPreviewItem,
) *EnvVarImportPreviewItemOutputObj {
	*o = EnvVarImportPreviewItemOutputObj{
		Key:           item.Key,
		Value:         item.Value,
		OriginalValue: item.OriginalValue,
		Description:   item.Description,
		Action:        string(item.Action),
		EffectScope:   string(item.EffectScope),
		Messages:      append([]string(nil), item.Messages...),
	}
	if item.DeclaredScope != nil {
		o.DeclaredScope = &EnvVarImportPreviewScopeOutputObj{
			Type:  item.DeclaredScope.Type,
			Value: item.DeclaredScope.Value,
		}
	}
	if item.EffectiveScope != nil {
		o.EffectiveScope = &EnvVarImportPreviewScopeOutputObj{
			Type:  item.EffectiveScope.Type,
			Value: item.EffectiveScope.Value,
		}
	}
	return o
}

// EnvVarImportPreviewSummaryOutputObj 导入预览汇总统计的 JSON 输出。
type EnvVarImportPreviewSummaryOutputObj struct {
	// 导入条目总数
	Total int `json:"total"`
	// 新增条数
	New int `json:"new"`
	// 覆盖条数
	Overwrite int `json:"overwrite"`
}

// FromModel 从领域模型填充输出字段。
func (o *EnvVarImportPreviewSummaryOutputObj) FromModel(
	item preview.ImportPreviewSummary,
) *EnvVarImportPreviewSummaryOutputObj {
	*o = EnvVarImportPreviewSummaryOutputObj{
		Total:     item.Total,
		New:       item.New,
		Overwrite: item.Overwrite,
	}
	return o
}

// EnvVarImportPreviewOutputObj 导入预览完整结果的 JSON 输出。
type EnvVarImportPreviewOutputObj struct {
	// 逐条预览结果
	Items []*EnvVarImportPreviewItemOutputObj `json:"items"`
	// 汇总统计
	Summary *EnvVarImportPreviewSummaryOutputObj `json:"summary"`
}

// FromModel 从领域模型填充输出字段。
func (o *EnvVarImportPreviewOutputObj) FromModel(item *preview.ImportPreview) *EnvVarImportPreviewOutputObj {
	output := &EnvVarImportPreviewOutputObj{
		Items:   make([]*EnvVarImportPreviewItemOutputObj, 0, len(item.Items)),
		Summary: new(EnvVarImportPreviewSummaryOutputObj).FromModel(item.Summary),
	}
	for _, previewItem := range item.Items {
		output.Items = append(output.Items, new(EnvVarImportPreviewItemOutputObj).FromModel(previewItem))
	}
	*o = *output
	return o
}

// PreviewEnvVarOutput 导入预览接口的 JSON 响应。
type PreviewEnvVarOutput struct {
	// 预览结果
	Data *EnvVarImportPreviewOutputObj `json:"data"`
}
