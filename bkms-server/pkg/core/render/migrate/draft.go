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

package migrate

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Kind 标识一条 Draft 对应的字段类型。
// apply 阶段按 Kind 分派到对应 handler，使得每种字段都能用自然键
// （比如组件 Name、_id）定位，避免 generate 与 apply 之间数组重排
// 导致 entry 用错下标——这是旧版用 dot-path FieldPath 时最大的脆弱点。
type Kind string

// 7 个 Kind 一一对应 generate 阶段的扫描器与 apply 阶段的 handler。
// 新增 Kind 时务必：
//  1. 在 DraftSet 里加一个对应的切片字段。
//  2. 在 cmd/migration/migrate_render_generate.go 里加扫描器并 append。
//  3. 在 cmd/migration/migrate_render_apply.go 里加 apply handler。
const (
	// KindComponentDefDefaultValue : component_defs.properties[name].defaultValue
	KindComponentDefDefaultValue Kind = "componentDefDefaultValue"
	// KindAppModelComponentProperty : app_models.components[name].properties[key]
	KindAppModelComponentProperty Kind = "appModelComponentProperty"
	// KindWorkspaceComponentProperty : workspace_components(_id).properties[key]
	KindWorkspaceComponentProperty Kind = "workspaceComponentProperty"
	// KindAppModelTafFileContent : app_models(appID).workload.tafConfig.fileContent
	KindAppModelTafFileContent Kind = "appModelTafFileContent"
	// KindAppConfigFileTaf : app_config_files(_id).content / overlayContent (format=taf)
	KindAppConfigFileTaf Kind = "appConfigFileTaf"
	// KindAppConfigFileVersionTaf : app_config_file_versions(_id).content / overlayContent (format=taf)
	// 不迁这张表会导致用户回滚旧版本时把 legacy 语法重新写回 app_config_files。
	KindAppConfigFileVersionTaf Kind = "appConfigFileVersionTaf"
	// KindPolarisConfigProperty : polaris_configs(appID, name).<PropertyKey>
	// PropertyKey 通常是 polaris.Properties 的顶层 bson 字段名（如 polarisToken、
	// polarisName）；对 serviceLabels 这个 map 字段，PropertyKey 固定取 "serviceLabels"，
	// 此时 Original/Converted 是整张 map 的 JSON 字符串。运行时由独立 Polaris
	// 工作负载构建链路渲染这些字段，因此仍需迁移。
	KindPolarisConfigProperty Kind = "polarisConfigProperty"
)

// Labels 仅用于 review 阶段给人看，不参与定位/写入。
// 字段保持 omitempty，避免在 component_defs 这种没有 app/workspace 上下文的场景
// 产生一堆空字段，污染 yaml 输出。
type Labels struct {
	AppName           string `yaml:"appName,omitempty"`
	WorkspaceName     string `yaml:"workspaceName,omitempty"`
	WorkspaceDisabled bool   `yaml:"workspaceDisabled,omitempty"`
	// ComponentName 是组件实例名（app_models.components[].name 或 workspace_components.name），
	// 仅在该 Draft 涉及具体组件实例时有值，方便 reviewer 在 yaml 里直接看到
	// "这条到底是哪个组件实例的 property"，不再需要回数组下标。
	ComponentName string `yaml:"componentName,omitempty"`
}

// Base 是所有 Draft 共享的字段。
// 不变量：
//   - Original 必须是 generate 阶段读到的 DB 原文；apply 阶段用它做 staleness 校验
//     （DB 当前值 != Original ⇒ 拒绝写入），所以一旦 Draft 落盘就**不应该再被人手动修改**。
//   - Converted / Error 互斥：转换成功填 Converted，失败填 Error；apply 跳过 Error 非空的 entry。
type Base struct {
	Original  string `yaml:"original"`
	Converted string `yaml:"converted,omitempty"`
	Error     string `yaml:"error,omitempty"`
	Labels    Labels `yaml:"labels,omitempty"`
}

// ComponentDefDefaultValueDraft 定位 component_defs 中
// properties[name=PropertyName].defaultValue。
//
// apply 时通过 arrayFilters（`p.name`）匹配，而不是数组下标——
// 如果 properties 顺序改变（增删/重排），自然键仍能稳定定位。
type ComponentDefDefaultValueDraft struct {
	Base         `yaml:",inline"`
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	PropertyName string `yaml:"propertyName"`
}

// AppModelComponentPropertyDraft 定位
// app_models(appID).components[name=ComponentName].properties[PropertyKey]。
//
// 当 comp.Name 为空时该 entry 不可定位，generate 阶段必须直接跳过；
// 否则 apply 会拿空 name 去匹配 arrayFilters，匹配到错的组件实例。
type AppModelComponentPropertyDraft struct {
	Base          `yaml:",inline"`
	AppID         string `yaml:"appID"`
	ComponentName string `yaml:"componentName"`
	PropertyKey   string `yaml:"propertyKey"`
}

// WorkspaceComponentPropertyDraft 定位 workspace_components(_id).properties[PropertyKey]。
// _id 直接命中文档，没有数组下标问题。
type WorkspaceComponentPropertyDraft struct {
	Base        `yaml:",inline"`
	ID          string `yaml:"id"` // _id 的 hex 字符串
	PropertyKey string `yaml:"propertyKey"`
}

// AppModelTafFileContentDraft 对应 app_models(appID).workload.tafConfig.fileContent。
//
// 该字段是历史遗留兜底数据：TAF 配置内容已经迁到 app_config_files 存储
// （见 pkg/workload/appmodelcore/appmodel/entities.go: TafConfig），但存量库里 fileContent
// 可能还残留旧值，本期一并扫描迁移。
type AppModelTafFileContentDraft struct {
	Base  `yaml:",inline"`
	AppID string `yaml:"appID"`
}

// AppConfigFileTafDraft 对应 app_config_files(_id) 的 content / overlayContent
// 字段（仅 format=taf 文件）。
//
// Overlay=true 表示写 overlayContent，false 表示写 content。
// 两个字段在 schema 里都是 *string；apply 时如果当前值是 nil，视为 STALE
// （表示文件结构变了，让用户去重跑 generate）。
type AppConfigFileTafDraft struct {
	Base    `yaml:",inline"`
	ID      string `yaml:"id"`
	Overlay bool   `yaml:"overlay"`
}

// PolarisConfigPropertyDraft 定位 polaris_configs 中
// (appID, name) 文档的顶层 string 字段（PropertyKey 即 bson tag 名）。
//
// 与 AppModel/WorkspaceComponent 的 properties[K] 不同：这里 PropertyKey
// 写的是文档顶层字段（如 polarisToken），$set 路径就是裸字段名，
// 不走 properties 子文档。
type PolarisConfigPropertyDraft struct {
	Base        `yaml:",inline"`
	AppID       string `yaml:"appID"`
	Name        string `yaml:"name"`
	PropertyKey string `yaml:"propertyKey"`
}

// AppConfigFileVersionTafDraft 与 AppConfigFileTafDraft 同形，
// 但对应 app_config_file_versions（版本历史表）。
//
// 这张表的 Content / OverlayContent 通过内嵌的 AppConfigFileContentSpec 提供，
// 见 pkg/core/app/appcfg/types.go: AppConfigFileVersion。
// 必须迁这张表，否则用户回滚到一个旧版本时，
// 旧版本里的 legacy {{...}} 会被重新写回 app_config_files。
type AppConfigFileVersionTafDraft struct {
	Base    `yaml:",inline"`
	ID      string `yaml:"id"`
	Overlay bool   `yaml:"overlay"`
}

// DraftSet 按 Kind 分桶。
//
// 这样做的理由：
//   - apply 用 range 直接拿到强类型，不用类型断言。
//   - yaml 序列化/反序列化用默认行为即可，无需自定义 Unmarshaler 来识别 kind 字段。
//   - 新增 Kind 时只需要加切片字段 + 扫描器 + handler，三处都很显眼。
type DraftSet struct {
	ComponentDefDefaultValues    []ComponentDefDefaultValueDraft   `yaml:"componentDefDefaultValues,omitempty"`
	AppModelComponentProperties  []AppModelComponentPropertyDraft  `yaml:"appModelComponentProperties,omitempty"`
	WorkspaceComponentProperties []WorkspaceComponentPropertyDraft `yaml:"workspaceComponentProperties,omitempty"`
	AppModelTafFileContents      []AppModelTafFileContentDraft     `yaml:"appModelTafFileContents,omitempty"`
	AppConfigFileTafs            []AppConfigFileTafDraft           `yaml:"appConfigFileTafs,omitempty"`
	AppConfigFileVersionTafs     []AppConfigFileVersionTafDraft    `yaml:"appConfigFileVersionTafs,omitempty"`
	PolarisConfigProperties      []PolarisConfigPropertyDraft      `yaml:"polarisConfigProperties,omitempty"`
}

// LoadDrafts 从 path 读取 DraftSet。
// 文件不存在时返回空 DraftSet，方便首次 apply 走 happy path。
func LoadDrafts(path string) (*DraftSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DraftSet{}, nil
		}
		return nil, err
	}
	var ds DraftSet
	if err := yaml.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("unmarshal drafts %q: %w", path, err)
	}
	return &ds, nil
}

// SaveDrafts 将 DraftSet 写入 path，覆盖原文件。
func SaveDrafts(path string, ds *DraftSet) error {
	data, err := yaml.Marshal(ds)
	if err != nil {
		return fmt.Errorf("marshal drafts: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// ConvertResult 是 runConvert 的返回类型，避免裸三返回值的位置歧义。
type ConvertResult struct {
	Converted string
	Err       error
	Skip      bool // HasTemplate=false 或 converted==original
}

// RunConvert 跑 Convert 并归一化结果，供 generate 阶段各扫描器复用。
//
// 调用方约定：
//   - Skip=true：跳过，不落 Draft（也不算 failed）。
//   - Err != nil：转换失败；调用方仍应落一条 Draft 并把 Err.Error() 写入 Base.Error，
//     方便 reviewer 在 yaml 里看到失败原因。Converted 此时为空。
//   - Err == nil && Skip == false：成功；用 Converted 构造 Draft。
func RunConvert(text string) ConvertResult {
	if !HasTemplate(text) {
		return ConvertResult{Skip: true}
	}
	converted, err := Convert(text)
	if err != nil {
		return ConvertResult{Err: err}
	}
	if converted == text {
		return ConvertResult{Skip: true}
	}
	return ConvertResult{Converted: converted}
}
