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

// Package serializer defines Gin input and output serializers for component APIs.
package serializer

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// PreviewComponentDefInput POST /component-defs/preview 请求体
type PreviewComponentDefInput struct {
	// 组件名称，用于预览 name 等内置变量
	CompDefName string `json:"compDefName" binding:"required"`
	// 属性定义列表
	Properties []PropertyDefInput `json:"properties"`
	// 根节点 YAML Patch 模板列表
	Patchers []string `json:"patchers" binding:"omitempty,dive,component_fragment"`
	// 额外 Kubernetes 资源 YAML 模板列表
	Specs []string `json:"specs" binding:"omitempty,dive,component_fragment"`
}

// PropertiesToModel 将属性请求结构转换为领域属性定义。
func (i PreviewComponentDefInput) PropertiesToModel() []component.Property {
	return lo.Map(i.Properties, func(p PropertyDefInput, _ int) component.Property {
		return p.ToModel()
	})
}

// PreviewComponentInstInput POST /component-insts/preview 请求体
type PreviewComponentInstInput struct {
	// 组件类型，即组件在市场中的名字，等同于 ComponentDef 的 name
	Type string `json:"type" binding:"required"`
	// 组件属性值
	Properties map[string]any `json:"properties" binding:"required"`
}

// PreviewOutput 预览响应（两个预览 API 共用）
type PreviewOutput struct {
	// 渲染后的附加资源列表
	Resources []PreviewResourceOutput `json:"resources"`
	// patch 预览列表
	PatchPreview []PreviewPatchOutput `json:"patchPreview"`
}

// PreviewResourceOutput 附加资源预览
type PreviewResourceOutput struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	// 渲染后的完整资源 YAML
	Manifest string `json:"manifest"`
}

// PreviewPatchOutput patch 预览
type PreviewPatchOutput struct {
	// 被 patch 的目标资源类型；当前固定 GameDeployment
	TargetKind string `json:"targetKind"`
	// 预置底稿 YAML
	BaseManifest string `json:"baseManifest"`
	// 应用全部 patcher 后的 YAML
	PatchedManifest string `json:"patchedManifest"`
}

// FromModel 从领域模型构建预览响应。
func (o *PreviewOutput) FromModel(result *component.PreviewResult) *PreviewOutput {
	*o = PreviewOutput{
		Resources: lo.Map(result.Resources, func(r component.PreviewResource, _ int) PreviewResourceOutput {
			return *new(PreviewResourceOutput).FromModel(r)
		}),
		PatchPreview: lo.Map(result.Patches, func(p component.PreviewPatch, _ int) PreviewPatchOutput {
			return *new(PreviewPatchOutput).FromModel(p)
		}),
	}
	return o
}

// FromModel 从领域模型构建附加资源预览响应。
func (o *PreviewResourceOutput) FromModel(r component.PreviewResource) *PreviewResourceOutput {
	*o = PreviewResourceOutput{
		APIVersion: r.APIVersion,
		Kind:       r.Kind,
		Name:       r.Name,
		Manifest:   r.Manifest,
	}
	return o
}

// FromModel 从领域模型构建 patch 预览响应。
func (o *PreviewPatchOutput) FromModel(p component.PreviewPatch) *PreviewPatchOutput {
	*o = PreviewPatchOutput{
		TargetKind:      p.TargetKind,
		BaseManifest:    p.BaseManifest,
		PatchedManifest: p.PatchedManifest,
	}
	return o
}
