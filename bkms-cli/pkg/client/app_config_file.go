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

// ConfigKindPlain 通用文本配置文件，按环境整份覆盖，用户可自定义挂载目录。
const ConfigKindPlain = "plain"

// ConfigKindFramework 框架配置文件（Helm values、tRPC/TAF 配置等）。
const ConfigKindFramework = "framework"

// AppConfigFileDef 配置文件定义（def 视角），附带默认文件或指定环境下生效文件的信息。
type AppConfigFileDef struct {
	// ID 配置文件定义 ID。
	ID string `json:"id" yaml:"id"`
	// FileID 展示文件的 ID：未指定环境时为默认文件，指定环境且存在独立实例时为环境实例。
	FileID string `json:"fileId,omitempty" yaml:"fileId,omitempty"`
	// Name 文件名称。
	Name string `json:"name" yaml:"name"`
	// ConfigKind 配置种类：framework 或 plain。
	ConfigKind string `json:"configKind" yaml:"configKind"`
	// MountDir 容器内挂载目录。
	MountDir string `json:"mountDir,omitempty" yaml:"mountDir,omitempty"`
	// IsUnifiedConfig 为 true 时所有环境共享默认内容，为 false 时可按环境独立配置。
	IsUnifiedConfig bool `json:"isUnifiedConfig" yaml:"isUnifiedConfig"`
	// MountedEnvNames 挂载环境范围，为空表示挂载到所有环境。
	MountedEnvNames []string `json:"mountedEnvNames,omitempty" yaml:"mountedEnvNames,omitempty"`
	// EnableEnvVarRender 是否启用环境变量渲染。
	EnableEnvVarRender bool `json:"enableEnvVarRender" yaml:"enableEnvVarRender"`
	// EnvContentStrategy 环境内容策略（仅 framework）：overlay 或 overwrite。
	EnvContentStrategy string `json:"envContentStrategy,omitempty" yaml:"envContentStrategy,omitempty"`
	// FileType 文件类型：normal 或 overlay。
	FileType string `json:"fileType,omitempty" yaml:"fileType,omitempty"`
	// ContentSourceType 内容来源：local 或 bscp。
	ContentSourceType string `json:"contentSourceType" yaml:"contentSourceType"`
	// BaseAppConfigFileID overlay 文件引用的基础文件 ID。
	BaseAppConfigFileID string `json:"baseAppConfigFileId,omitempty" yaml:"baseAppConfigFileId,omitempty"`
	// BSCPConfig BSCP 来源配置。
	BSCPConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty" yaml:"bscpConfig,omitempty"`
	// FileFormat 文件格式：yaml 或 taf。
	FileFormat string `json:"fileFormat" yaml:"fileFormat"`
	// CurrentVersion 展示文件的当前版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
	// Content 主内容。
	Content *string `json:"content,omitempty" yaml:"content,omitempty"`
	// OverlayContent 覆盖内容。
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty"`
	// HasEnvInstance 指定环境是否有独立实例，仅按环境查询详情时返回。
	HasEnvInstance *bool `json:"hasEnvInstance,omitempty" yaml:"hasEnvInstance,omitempty"`
	// EditableContentField 可编辑字段：content、overlayContent 或 none，仅查询详情时返回。
	EditableContentField string `json:"editableContentField,omitempty" yaml:"editableContentField,omitempty"`
	// BaseContentInfo overlay / BSCP 文件的基础内容信息。
	BaseContentInfo *BaseContentInfo `json:"baseContentInfo,omitempty" yaml:"baseContentInfo,omitempty"`
	// Updater 最近修改人。
	Updater string `json:"updater" yaml:"updater"`
	// UpdatedAt 最近修改时间，RFC3339 格式。
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
	// Creator 创建人。
	Creator string `json:"creator" yaml:"creator"`
	// CreatedAt 创建时间，RFC3339 格式。
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
}

// BaseContentInfo 基础配置文件内容信息。
type BaseContentInfo struct {
	// HolderID 基础内容持有者 ID。
	HolderID string `json:"holderId" yaml:"holderId"`
	// HolderName 基础内容持有者名称。
	HolderName string `json:"holderName" yaml:"holderName"`
	// HolderContentSourceType 基础内容来源，例如 local 或 bscp。
	HolderContentSourceType string `json:"holderContentSourceType" yaml:"holderContentSourceType"`
	// Content 基础文件内容。
	Content string `json:"content" yaml:"content"`
	// IsFromAnotherFile 表示基础内容是否来自另一个应用配置文件。
	IsFromAnotherFile bool `json:"isFromAnotherFile" yaml:"isFromAnotherFile"`
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

// AppConfigFileDefItemRespData 返回单个配置文件定义的响应。
type AppConfigFileDefItemRespData struct {
	Item *AppConfigFileDef `json:"item"`
}

// ListAppConfigFileDefsRespData 配置文件定义列表响应。
type ListAppConfigFileDefsRespData struct {
	Items []AppConfigFileDef `json:"items"`
}

// CreateAppConfigFileDefBody 创建配置文件定义的请求体。
type CreateAppConfigFileDefBody struct {
	// Name 文件名称。
	Name string `json:"name"`
	// ConfigKind 配置种类：framework 或 plain。
	ConfigKind string `json:"configKind"`
	// MountDir 容器内挂载目录，plain 必填。
	MountDir string `json:"mountDir,omitempty"`
	// FileFormat 文件格式：yaml 或 taf。
	FileFormat string `json:"fileFormat"`
	// FileType 文件类型：normal 或 overlay。
	FileType string `json:"fileType"`
	// ContentSourceType 内容来源：local 或 bscp。
	ContentSourceType string `json:"contentSourceType"`
	// Content 初始内容。
	Content *string `json:"content,omitempty"`
	// MountedEnvNames 挂载环境范围，nil 表示挂载到所有环境。
	MountedEnvNames *[]string `json:"mountedEnvNames,omitempty"`
	// EnableEnvVarRender 是否启用环境变量渲染，nil 表示使用后端默认值。
	EnableEnvVarRender *bool `json:"enableEnvVarRender,omitempty"`
	// Description 版本描述。
	Description string `json:"description"`
}

// UpdateAppConfigFileDefBody 更新配置文件定义的请求体，nil 字段表示不修改。
type UpdateAppConfigFileDefBody struct {
	// Name 文件名称。
	Name *string `json:"name,omitempty"`
	// MountDir 容器内挂载目录。
	MountDir *string `json:"mountDir,omitempty"`
	// IsUnifiedConfig 是否统一配置。
	IsUnifiedConfig *bool `json:"isUnifiedConfig,omitempty"`
	// MountedEnvNames 挂载环境范围。
	MountedEnvNames *[]string `json:"mountedEnvNames,omitempty"`
	// EnableEnvVarRender 是否启用环境变量渲染。
	EnableEnvVarRender *bool `json:"enableEnvVarRender,omitempty"`
	// EnvContentStrategy 环境内容策略（仅 framework）：overlay 或 overwrite。
	EnvContentStrategy *string `json:"envContentStrategy,omitempty"`
}

// UpdateAppConfigFileDefContentBody 更新配置内容的请求体。
type UpdateAppConfigFileDefContentBody struct {
	// Content 要写入可编辑字段的内容。
	Content string `json:"content"`
	// Description 版本描述。
	Description string `json:"description"`
	// CurrentVersion 编辑开始时的当前版本号，用于乐观锁冲突检测。
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// AppConfigFileDefContentUpdateResult 更新配置内容后的返回结果。
type AppConfigFileDefContentUpdateResult struct {
	// FileID 被写入的文件 ID（默认文件或环境实例）。
	FileID string `json:"fileId" yaml:"fileId"`
	// CurrentVersion 写入后的版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
	// Content 编译后的完整内容。
	Content string `json:"content" yaml:"content"`
}

// AppConfigFileEnvInstance 配置文件定义下的一个文件实例（默认文件或环境实例）。
type AppConfigFileEnvInstance struct {
	// FileID 文件 ID。
	FileID string `json:"fileId" yaml:"fileId"`
	// EnvName 环境名称，空字符串表示默认文件。
	EnvName string `json:"envName" yaml:"envName"`
	// Type 文件类型：normal 或 overlay。
	Type string `json:"type" yaml:"type"`
	// CurrentVersion 当前版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
	// Content 主内容。
	Content *string `json:"content,omitempty" yaml:"content,omitempty"`
	// OverlayContent 覆盖内容。
	OverlayContent *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty"`
	// Updater 最近修改人。
	Updater string `json:"updater" yaml:"updater"`
	// UpdatedAt 最近修改时间，RFC3339 格式。
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// ListAppConfigFileEnvInstancesRespData 环境实例列表响应。
type ListAppConfigFileEnvInstancesRespData struct {
	Items []AppConfigFileEnvInstance `json:"items"`
}

// AppConfigFileMountPreviewItem 挂载预览中的单个文件。
type AppConfigFileMountPreviewItem struct {
	// DefID 配置文件定义 ID。
	DefID string `json:"defId" yaml:"defId"`
	// Name 文件名称。
	Name string `json:"name" yaml:"name"`
	// ConfigKind 配置种类。
	ConfigKind string `json:"configKind" yaml:"configKind"`
	// MountDir 容器内挂载目录。
	MountDir string `json:"mountDir,omitempty" yaml:"mountDir,omitempty"`
	// ContentSource 内容来源：normal、overlay 或 overwrite。
	ContentSource string `json:"contentSource" yaml:"contentSource"`
	// HasEnvFile 该环境是否有独立的文件实例。
	HasEnvFile bool `json:"hasEnvFile" yaml:"hasEnvFile"`
}

// AppConfigFileMountPreviewRespData 挂载预览响应。
type AppConfigFileMountPreviewRespData struct {
	Items []AppConfigFileMountPreviewItem `json:"items"`
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
	// EnvName 环境名，空字符串表示默认文件。
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
	// Version 版本号。
	Version *int64
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

// AppConfigFileVersionComparison 两个历史版本的对比结果。
type AppConfigFileVersionComparison struct {
	// Previous 旧版本。
	Previous *AppConfigFileVersion `json:"previous" yaml:"previous"`
	// Current 新版本。
	Current *AppConfigFileVersion `json:"current" yaml:"current"`
}

// RollbackAppConfigFileVersionOptions 回滚应用配置文件历史版本参数。
type RollbackAppConfigFileVersionOptions struct {
	// CurrentVersion 编辑开始时的当前版本号，用于乐观锁冲突检测。
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
	// Description 回滚版本描述，为空则由后端自动生成。
	Description string `json:"description,omitempty"`
}

// RollbackAppConfigFileVersionResult 回滚后被修改的文件信息。
type RollbackAppConfigFileVersionResult struct {
	// ID 文件 ID。
	ID string `json:"id" yaml:"id"`
	// EnvName 环境名，空字符串表示默认文件。
	EnvName string `json:"envName" yaml:"envName"`
	// CurrentVersion 回滚后的当前版本号。
	CurrentVersion int64 `json:"currentVersion" yaml:"currentVersion"`
}

// RollbackAppConfigFileVersionRespData 回滚应用配置文件历史版本返回数据。
type RollbackAppConfigFileVersionRespData struct {
	Data RollbackAppConfigFileVersionResult `json:"data"`
}
