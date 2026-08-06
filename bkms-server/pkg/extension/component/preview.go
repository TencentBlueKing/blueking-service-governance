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

// Package component 组件输出「预览」（本文件中的 PreviewBuilder）用于在**不接入真实应用、环境** 的前提下，
// 对组件定义的 patcher/spec 模板做一次试运行：得到附加资源清单与基于固定 GameDeployment（后面可能拓展到其他资源）
// 底稿的 patch 合并结果。
//
// 与正式渲染（AppComponentApplier / AppPropertiesBuilder）相比的主要差异：
//
//   - **变量来源**：预览注入固定的占位 App/Env（见 previewAppName 等常量 与 buildBasicBuiltin）；
//
//     正式渲染会从库中加载 Environment，并通常以 envvars.BuildAppEnvVars().ToMap() 等构建完整变量表。
//
//   - **属性值中的占位符**：预览不会注入真实环境变量，因此属性里的 ${{env.KEY}} 会按当前渲染器规则处理。
//
//   - **patcher/spec 模板**：预览仅做 **单层 Go 模板**，模板数据为「属性名 → 已展开值」
//     的扁平 map（含 name、bkms* 内置键等）；字段名拼写错误或引用未注入的键会得到 **`<no value>`**，
//     不会像属性值那样先经 ${{}} 再经 legacy 的双层管线。
//
//   - **Patch 预览**：在本地固定的 sample GameDeployment 上顺序 strategic merge patch，用于参考
//
// 预览结果不能等价于集群中实际下发或运行时的最终 YAML，仅用于编辑期/试运行的近似展示。
package component

import (
	"fmt"
	"strings"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// 用于试运行内置变量的渲染
const (
	previewAppName      = "preview-app-name"
	previewEnvName      = "preview-env-name"
	previewEnvNamespace = "preview-env-namespace"
	previewEnvCluster   = "preview-env-cluster"
)

// PreviewBuilder 组件定义或实例试运行预览构建器
type PreviewBuilder struct {
	// Name 组件名称
	Name string
	// Properties 组件属性定义
	Properties []Property
	// Patchers 根节点 Patch 模板列表
	Patchers []string
	// Specs 额外资源模板列表
	Specs []string
	// PropertyValues 组件属性值
	PropertyValues map[string]any
}

// PreviewResult 预览结果
type PreviewResult struct {
	// Resources 附加资源预览
	Resources []PreviewResource
	// Patches patch 结果预览
	Patches []PreviewPatch
}

// PreviewResource 附加资源预览
type PreviewResource struct {
	// APIVersion API 版本
	APIVersion string
	// Kind 资源类型
	Kind string
	// Name 资源名称
	Name string
	// Manifest 资源 YAML
	Manifest string
}

// PreviewPatch patch 预览结果
type PreviewPatch struct {
	// TargetKind 被 patch 的目标资源类型
	TargetKind string
	// BaseManifest 预置底稿 YAML
	BaseManifest string
	// PatchedManifest 应用全部 patcher 后的 YAML
	PatchedManifest string
}

// NewPreviewBuilder creates a builder for previewing rendered component resources.
func NewPreviewBuilder(name string, properties []Property, patchers, specs []string) *PreviewBuilder {
	return &PreviewBuilder{
		Name:           name,
		Properties:     properties,
		Patchers:       patchers,
		Specs:          specs,
		PropertyValues: nil,
	}
}

// WithPropertyValues sets property overrides used when rendering the component preview.
func (b *PreviewBuilder) WithPropertyValues(propertyValues map[string]any) *PreviewBuilder {
	b.PropertyValues = propertyValues
	return b
}

// Build 使用内置变量与属性默认值渲染组件（试运行预览）。
func (b *PreviewBuilder) Build() (*PreviewResult, error) {
	previewDef := &ComponentDef{
		Name:       b.Name,
		Version:    DefaultComponentDefVersion,
		Properties: b.Properties,
		Patchers:   b.Patchers,
		Specs:      b.Specs,
	}
	if err := ValidateComponentDef(previewDef); err != nil {
		return nil, errors.Wrap(err, "validating component definition")
	}

	// 组装模板变量：属性默认值、实例覆盖与内置变量。
	props, err := b.buildProperties()
	if err != nil {
		return nil, errors.Wrap(err, "building preview properties")
	}

	evaluated, err := evaluateComponentTemplates(b.Patchers, b.Specs, props)
	if err != nil {
		return nil, err
	}

	resources, err := b.buildResources(evaluated.Specs)
	if err != nil {
		return nil, errors.Wrap(err, "building preview resources")
	}

	// 基于固定 GameDeployment 底稿构造 patch 预览
	patches, err := b.buildPatch(evaluated.Patchers)
	if err != nil {
		return nil, errors.Wrap(err, "building patch preview")
	}
	return &PreviewResult{Resources: resources, Patches: patches}, nil
}

// buildProperties 合并属性默认值、实例填写值和内置变量
func (b *PreviewBuilder) buildProperties() (map[string]any, error) {
	props := make(map[string]PropValue)
	for _, propDef := range b.Properties {
		props[propDef.Name] = PropValue{
			Ty:    propDef.Type,
			Value: propDef.NormalizedDefaultValue(),
		}
	}
	// 实例预览仅覆盖已声明属性，忽略未知 key
	for propName, propValue := range b.PropertyValues {
		prop, exists := props[propName]
		if !exists {
			continue
		}
		prop.Value = propValue
		props[propName] = prop
	}

	// 注入固定 App/Environment，复用与正式渲染一致的内置变量生成逻辑
	app := bkmsapp.Application{Name: previewAppName}
	env := envmodel.Environment{
		Name: previewEnvName,
		Cluster: envmodel.BizCluster{
			Namespace: previewEnvNamespace,
			ClusterID: previewEnvCluster,
		},
	}
	for key, value := range buildBasicBuiltin(app, env) {
		props[key] = PropValue{Ty: PropTypeString, Value: value}
	}
	// name 与真实实例命名规则一致：{appName}-{componentName}
	props["name"] = PropValue{
		Ty:    PropTypeString,
		Value: strings.ToLower(fmt.Sprintf("%s-%s", previewAppName, b.Name)),
	}

	renderedProps, err := renderProps(props, nil, nil, "")
	if err != nil {
		return nil, err
	}

	// 转为 rich value 便于后续渲染
	return lo.MapEntries(renderedProps, func(key string, value PropValue) (string, any) {
		return key, value.ToRichValue()
	}), nil
}

// buildResources 将渲染后的 spec 列表转换为带 manifest 的附加资源预览。
func (b *PreviewBuilder) buildResources(specs []map[string]any) ([]PreviewResource, error) {
	resources := make([]PreviewResource, 0, len(specs))
	for i, spec := range specs {
		manifest, err := yaml.Marshal(spec)
		if err != nil {
			return nil, errors.Wrapf(err, "marshaling spec[%d]", i)
		}

		obj := unstructured.Unstructured{Object: spec}
		resources = append(resources, PreviewResource{
			APIVersion: obj.GetAPIVersion(),
			Kind:       obj.GetKind(),
			Name:       obj.GetName(),
			Manifest:   string(manifest),
		})
	}
	return resources, nil
}

// buildPatch 在 GameDeployment 底稿上依次跑 strategic merge patch，并返回 patch 前后的 manifest。
func (b *PreviewBuilder) buildPatch(patchers []map[string]any) ([]PreviewPatch, error) {
	baseGD := sampleGameDeployment()
	baseManifest, err := yaml.Marshal(&baseGD)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling base manifest")
	}

	if len(patchers) == 0 {
		return []PreviewPatch{
			{
				TargetKind:      kind.GameDeploy,
				BaseManifest:    string(baseManifest),
				PatchedManifest: string(baseManifest),
			},
		}, nil
	}

	patchedGD, err := ApplyGameDeploymentPatchers(baseGD, patchers)
	if err != nil {
		return nil, err
	}
	patchedManifest, err := yaml.Marshal(&patchedGD)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling patched manifest")
	}

	return []PreviewPatch{
		{
			TargetKind:      kind.GameDeploy,
			BaseManifest:    string(baseManifest),
			PatchedManifest: string(patchedManifest),
		},
	}, nil
}

// sampleGameDeployment 构造 patch 预览用的固定 GameDeployment 底稿。
func sampleGameDeployment() tkex.GameDeployment {
	return tkex.GameDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tkex.tencent.com/v1alpha1",
			Kind:       kind.GameDeploy,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      previewAppName,
			Namespace: previewEnvNamespace,
		},
		Spec: tkex.GameDeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    defaults.WorkloadMainContainerName,
							Image:   "preview-image:latest",
							Command: []string{"./preview-app"},
							Args:    []string{"--port=8080"},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "preview-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}
