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

package client

// AppConfigFile 应用配置文件摘要信息。
type AppConfigFile struct {
	// ID 应用配置文件 ID。
	ID string `json:"id" yaml:"id"`
	// Name 文件名称。
	Name string `json:"name" yaml:"name"`
	// Type 文件类型，例如 normal 或 overlay。
	Type string `json:"type" yaml:"type"`
	// ContentSourceType 文件内容来源，例如 local 或 bscp。
	ContentSourceType string `json:"contentSourceType" yaml:"contentSourceType"`
	// BaseAppConfigFileID overlay 文件关联的基础文件 ID。
	BaseAppConfigFileID string `json:"baseAppConfigFileID,omitempty" yaml:"baseAppConfigFileID,omitempty"`
	// EnvName 环境名称，空字符串表示默认应用级配置。
	EnvName string `json:"envName" yaml:"envName"`
	// FileFormat 配置文件格式，例如 yaml 或 taf。
	FileFormat string `json:"fileFormat" yaml:"fileFormat"`
	// CurrentVersion 当前生效版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
	// Updater 当前版本最后修改人。
	Updater string `json:"updater" yaml:"updater"`
	// UpdatedAt 当前版本最后修改时间，RFC3339 格式。
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// ListAppConfigFilesRespData 获取应用配置文件列表返回数据。
type ListAppConfigFilesRespData struct {
	// Items 应用配置文件列表。
	Items []AppConfigFile `json:"items"`
}

// AppConfigFileDetails 应用配置文件详情。
type AppConfigFileDetails struct {
	// EditableContentField 可编辑字段：none、content 或 overlayContent。
	EditableContentField string `json:"editableContentField" yaml:"editableContentField"`
	// Content 默认或 normal 配置文件内容。
	Content *string `json:"content,omitempty" yaml:"content,omitempty"`
	// OverlayContent 环境覆盖层内容。
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty"`
	// BaseContentInfo overlay 文件对应的基础内容信息。
	BaseContentInfo *BaseContentInfo `json:"baseContentInfo,omitempty" yaml:"baseContentInfo,omitempty"`
	// CurrentVersion 当前生效版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
	// Updater 当前版本最后修改人。
	Updater string `json:"updater" yaml:"updater"`
	// UpdatedAt 当前版本最后修改时间，RFC3339 格式。
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// BaseContentInfo 基础配置文件内容信息。
type BaseContentInfo struct {
	// HolderID 基础内容持有者 ID。
	HolderID string `json:"holderID" yaml:"holderID"`
	// HolderName 基础内容持有者名称。
	HolderName string `json:"holderName" yaml:"holderName"`
	// HolderContentSourceType 基础内容来源，例如 local 或 bscp。
	HolderContentSourceType string `json:"holderContentSourceType" yaml:"holderContentSourceType"`
	// Content 基础文件内容。
	Content string `json:"content" yaml:"content"`
	// IsFromAnotherFile 表示基础内容是否来自另一个应用配置文件。
	IsFromAnotherFile bool `json:"isFromAnotherFile" yaml:"isFromAnotherFile"`
}

// AppConfigFileContentOptions 更新应用配置文件内容的参数。
type AppConfigFileContentOptions struct {
	// Content 要写入 content 或 overlayContent 的配置文件内容。
	Content string
	// Description 本次配置文件版本描述。
	Description string
	// CurrentVersion 编辑开始时的当前版本号，用于乐观锁冲突检测。
	CurrentVersion *int64
}

// AppConfigFileContentUpdateResult 更新应用配置文件内容后的返回结果。
type AppConfigFileContentUpdateResult struct {
	// CompiledContent 应用配置文件合并后的完整内容。
	CompiledContent string `json:"compiledContent" yaml:"compiledContent"`
	// ArrgData 基于新的完整内容产生的编排校验结果。
	// INFO：目前非 Helm 应用暂时不使用该字段
	ArrgData any `json:"arrgData,omitempty" yaml:"arrgData,omitempty"`
}

// BSCPAppConfigFileConfig BSCP 配置引用信息。
type BSCPAppConfigFileConfig struct {
	// BizID BSCP 业务 ID。
	BizID string `json:"bizID" yaml:"bizID"`
	// ID BSCP 配置 ID。
	ID string `json:"id" yaml:"id"`
	// ServiceID BSCP 服务 ID。
	ServiceID string `json:"serviceID" yaml:"serviceID"`
}

// AppConfigFileVersion 应用配置文件某个历史版本。
type AppConfigFileVersion struct {
	// AppConfigFileID 所属应用配置文件 ID。
	AppConfigFileID string `json:"appConfigFileID" yaml:"appConfigFileID"`
	// AppID 应用 ID。
	AppID string `json:"appID" yaml:"appID"`
	// BaseAppConfigFileID base 文件 ID。
	BaseAppConfigFileID string `json:"baseAppConfigFileID,omitempty" yaml:"baseAppConfigFileID,omitempty"`
	// BaseVersion base 文件版本号。
	BaseVersion *int64 `json:"baseVersion,omitempty" yaml:"baseVersion,omitempty"`
	// BSCPConfig BSCP 配置引用。
	BSCPConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty" yaml:"bscpConfig,omitempty"`
	// Content 普通内容。
	Content *string `json:"content,omitempty" yaml:"content,omitempty"`
	// ContentSourceType 内容来源。
	ContentSourceType string `json:"contentSourceType" yaml:"contentSourceType"`
	// CreatedAt 创建时间，RFC3339 格式。
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
	// Creator 创建人。
	Creator string `json:"creator" yaml:"creator"`
	// DeletedAt 删除时间，RFC3339 格式。
	DeletedAt *string `json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	// Deleter 删除人。
	Deleter *string `json:"deleter,omitempty" yaml:"deleter,omitempty"`
	// Description 版本描述。
	Description string `json:"description" yaml:"description"`
	// EnvName 环境名，空字符串表示默认应用级配置。
	EnvName string `json:"envName" yaml:"envName"`
	// FileFormat 文件格式。
	FileFormat string `json:"fileFormat" yaml:"fileFormat"`
	// ID 版本记录 ID。
	ID string `json:"id" yaml:"id"`
	// IsDeleted 是否软删除。
	IsDeleted bool `json:"isDeleted" yaml:"isDeleted"`
	// Name 文件名。
	Name string `json:"name" yaml:"name"`
	// OperationType 版本操作类型。
	OperationType string `json:"operationType" yaml:"operationType"`
	// OverlayContent 覆盖内容。
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty"`
	// RollbackFromVersion 若为 rollback，表示回滚来源版本号。
	RollbackFromVersion *int64 `json:"rollbackFromVersion,omitempty" yaml:"rollbackFromVersion,omitempty"`
	// Type 文件类型。
	Type string `json:"type" yaml:"type"`
	// Version 版本号。
	Version int64 `json:"version" yaml:"version"`
}

// ListAppConfigFileVersionsOptions 查询应用配置文件版本列表参数。
type ListAppConfigFileVersionsOptions struct {
	// AppConfigFileID 应用配置文件 ID。
	AppConfigFileID string
	// EnvName 环境名。
	EnvName string
	// Name 文件名。
	Name string
	// Version 版本号。
	Version *int64
	// Creator 创建人。
	Creator string
	// Description 版本描述。
	Description string
	// Page 页码，从 1 开始。
	Page int
	// PageSize 每页数量。
	PageSize int
}

// ListAppConfigFileVersionsRespData 获取应用配置文件历史版本列表返回数据。
type ListAppConfigFileVersionsRespData struct {
	Data PaginatedAppConfigFileVersions `json:"data"`
}

// PaginatedAppConfigFileVersions 分页应用配置文件历史版本。
type PaginatedAppConfigFileVersions struct {
	// Count 总记录数。
	Count int `json:"count" yaml:"count"`
	// Results 当前页历史版本列表。
	Results []AppConfigFileVersion `json:"results" yaml:"results"`
}

// GetAppConfigFileVersionRespData 获取应用配置文件某个历史版本返回数据。
type GetAppConfigFileVersionRespData struct {
	Data AppConfigFileVersion `json:"data"`
}

// RollbackAppConfigFileVersionOptions 回滚应用配置文件历史版本参数。
type RollbackAppConfigFileVersionOptions struct {
	// CurrentVersion 编辑开始时的当前版本号，用于乐观锁冲突检测。
	CurrentVersion *int64
	// Description 回滚版本描述，为空则由后端自动生成。
	Description string
}

// RollbackAppConfigFileVersionRespData 回滚应用配置文件历史版本返回数据。
type RollbackAppConfigFileVersionRespData struct {
	Data AppConfigFile `json:"data"`
}
