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

package appcfg

import (
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// AppConfigFile represents an app config file, e.g. Helm values file, tRPC config file, etc.
//
// Valid AppConfigFile combinations:
//
// 1. Normal app config file with only Content
// 2. Normal app config file with non-local source with both Content and OverlayContent
// 3. Overlay app config file with only OverlayContent
//
// # Usage Rules by Application Type:
//
// ## Helm Applications (AppType = "helm"):
//   - EnvName is always empty (Helm apps don't use environment-specific configs in AppConfigFile)
//   - One appID can have multiple app config files
//
// ## tRPC Applications (AppType = "trpc"):
//   - EnvName empty: represents the application's default configuration file
//   - EnvName not empty: represents environment-specific configuration file
//   - For a given appID and envName, there should be at most ONE app config file
type AppConfigFile struct {
	// ID is the unique identifier for the app config file.
	ID bson.ObjectID `bson:"_id,omitempty"`
	// AppConfigFileContentSpec 内联嵌入身份字段与可版本化内容。
	AppConfigFileContentSpec `bson:",inline"`

	// ── 环境配置策略（仅存在于 AppConfigFile，不写入版本记录）────────
	EnvConfigPolicy `bson:",inline"`

	// CurrentVersion is the current active version number of this file.
	CurrentVersion int64 `bson:"currentVersion"`

	// Updater is the last modifier of the current active version.
	Updater string `bson:"updater,omitempty"`

	// UpdatedAt is the last update time
	UpdatedAt time.Time `bson:"updatedAt"`
}

// AppConfigFileVersion 存储配置文件变更的不可变历史记录。
//
// 每次对 AppConfigFile 执行创建、更新或回滚操作时，都会追加一条版本记录。
// 版本记录通过内联嵌入 AppConfigFileContentSpec 保存身份字段与 VersionedContent 快照。
// EnvConfigPolicy（挂载路径、统一/独立模式等）不包含在版本记录中——
// 这些策略字段仅存在于 AppConfigFile，不参与版本 diff 与回滚。
type AppConfigFileVersion struct {
	// ID 是版本记录的唯一标识。
	ID bson.ObjectID `bson:"_id,omitempty"`
	// AppConfigFileID 指向该版本所属的配置文件。
	AppConfigFileID bson.ObjectID `bson:"appConfigFileID"`
	// AppConfigFileContentSpec 内联嵌入，存储该版本时刻的身份字段与可版本化内容。
	AppConfigFileContentSpec `bson:",inline"`
	// Version 是单调递增的版本号，同一文件内唯一。
	Version int64 `bson:"version"`
	// Description 是本次变更的描述信息。
	Description string `bson:"description,omitempty"`
	// BaseVersion 仅在 overlay 文件中使用，记录创建版本时 base 文件的当前版本号。
	BaseVersion *int64 `bson:"baseVersion,omitempty"`
	// OperationType 标识版本产生的方式：create、update 或 rollback。
	OperationType AppConfigFileVersionOperationType `bson:"operationType"`
	// RollbackFromVersion 仅在 OperationType 为 rollback 时设置，记录回滚的目标版本号。
	RollbackFromVersion *int64 `bson:"rollbackFromVersion,omitempty"`
	// IsDeleted 标识该版本是否已被软删除。软删除后默认不展示，但仍可通过选项查询。
	IsDeleted bool `bson:"isDeleted"`
	// Deleter 是执行软删除操作的用户。
	Deleter string `bson:"deleter,omitempty"`
	// DeletedAt 是软删除的时间戳。
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// VersionedContent 文件内容字段，版本管理内容
type VersionedContent struct {
	// ContentSourceType 表示应用配置文件内容来源，
	// 可以来自本地存储，也可以来自 BSCP 等外部系统。
	ContentSourceType ContentSourceType `bson:"contentSourceType"`
	// Format 表示配置文件格式（如 yaml、taf）。
	// 未指定时默认按 yaml 处理。
	Format FileFormat `bson:"format,omitempty"`
	// BSCPConfig is BSCP resource config reference.
	// This field is only set when ContentSourceType is bscp.
	BSCPConfig *BSCPConfig `bson:"bscpConfig,omitempty"`

	// Content is the content of the app config file, this field is nil if current file is an overlay.
	Content *string `bson:"content,omitempty"`
	// OverlayContent is the overlay content of the app config file, both normal and overlay types
	// can have overlay content.
	//
	// - For normal app config file, the OverlayContent is nil most of the time, but if
	//   the file's source is from external system like BSCP, the overlay content can be set
	//   in order to override some values in the normal app config file.
	// - For overlay app config file, this field MUST be set.
	//
	// The format of the overlay content is like this:
	//
	// ```
	// patches:
	// - replicas: 5
	// ```
	//
	// The patches field might contain multiple patch documents.
	OverlayContent *string `bson:"overlayContent,omitempty"`
}

// EnvConfigPolicy 包含环境维度的配置策略与关系元数据，不参与版本管理
type EnvConfigPolicy struct {
	// MountPath 是 plain 配置文件在容器内的完整挂载路径。
	// Framework 文件仍然沿用 workload 层的 FilePath + FileName，而不使用该字段。
	MountPath string `bson:"mountPath,omitempty"`
	// DefaultAppConfigFileID 在当前记录是环境级 plain 实例时，指向其所属的默认逻辑文件。
	// 默认逻辑文件自身不设置该字段，并使用自己的 ID 作为逻辑根。
	DefaultAppConfigFileID *bson.ObjectID `bson:"defaultAppConfigFileID,omitempty"`
	// IsUnifiedConfig 表示当前逻辑文件是否为统一配置模式。
	// true = 所有挂载环境共用同一份内容；false = 按环境独立配置。
	// 不能使用 omitempty：Update 走 $set，false 被省略后数据库会留下旧的 true。
	IsUnifiedConfig bool `bson:"isUnifiedConfig"`
	// MountedEnvNames 表示当前文件的挂载环境范围。
	// nil = 对所有环境生效；非 nil 空切片 = 不挂载到任何环境；非空 = 仅对列出的环境生效。
	MountedEnvNames []string `bson:"mountedEnvNames"`
}

// AppConfigFileContentSpec 包含 AppConfigFile 与 AppConfigFileVersion 共享的字段集。
//
// 内部按职责分为两组：
//   - 身份字段：AppID、EnvName、Name 等创建后不变的标识信息
//   - VersionedContent（内联）：文件内容及格式，参与版本快照与回滚
type AppConfigFileContentSpec struct {
	// -- 身份字段 --

	// AppID 是该应用配置文件所属应用的 ID。
	AppID string `bson:"appID"`
	// EnvName 是该应用配置文件所属环境的名称。
	// 其使用方式会随应用类型不同而变化：
	// - Helm 应用：始终为空
	// - tRPC 应用：为空表示应用级默认配置，非空表示环境级配置
	EnvName string `bson:"envName"`
	// Name 是应用配置文件名称。
	Name string `bson:"name"`
	// Type 表示应用配置文件类型（normal 或 overlay）。
	Type AppConfigFileType `bson:"type"`
	// ConfigKind 表示配置文件遵循 framework 语义还是 plain-file 语义。
	// 历史记录如果没有该字段，则按 framework 文件处理。
	ConfigKind ConfigKind `bson:"configKind,omitempty"`
	// BaseAppConfigFileID is the ID of the base app config file, it is only set when Type is overlay.
	BaseAppConfigFileID *bson.ObjectID `bson:"baseAppConfigFileID,omitempty"`
	// Creator is the creator of this logical file / version record.
	Creator string `bson:"creator"`
	// CreatedAt is the creation time
	CreatedAt time.Time `bson:"createdAt"`

	//  -- 可版本化内容（参与回滚）--

	VersionedContent `bson:",inline"`
}

// AppValuesConfig stores the configuration related with values file for an application,
// the configuration contains the ID of the default values file ATM, more fields might be added later.
type AppValuesConfig struct {
	// ID is the unique identifier for the app values config.
	ID bson.ObjectID `bson:"_id,omitempty"`
	// AppID is the ID of the application which the values config belongs to.
	AppID string `bson:"appID"`
	// DefaultAppConfigFileID is the ID of the default app config file.
	DefaultAppConfigFileID *bson.ObjectID `bson:"defaultAppConfigFileID,omitempty"`
}

// GetConfigFormat returns the config format of the app config file, default to YAML format if not specified.
func (s *AppConfigFileContentSpec) GetConfigFormat() FileFormat {
	// For backward compatibility
	if s.Format == "" {
		log.WarnNoContextf("app config file %s/%s has no format, default to YAML format", s.AppID, s.Name)
		return FileFormatYAML
	}
	return s.Format
}

// GetConfigKind returns the semantic kind of the app config file.
func (s *AppConfigFileContentSpec) GetConfigKind() ConfigKind {
	return s.ConfigKind
}

// HasIndependentEnvConfig 判断当前逻辑文件是否处于按环境独立配置模式。
// 对于旧数据（IsUnifiedConfig 零值 false、无 env instance），进入独立分支后
// 查不到 env instance 会回退到默认内容，行为与统一配置一致，无副作用。
func (p *EnvConfigPolicy) HasIndependentEnvConfig() bool {
	if p == nil {
		return false
	}
	return !p.IsUnifiedConfig
}

// IsMountedToEnv 判断该逻辑文件是否应挂载到指定环境。
// nil 表示对所有环境生效；非 nil 空切片表示不挂载到任何环境。
func (p *EnvConfigPolicy) IsMountedToEnv(envName string) bool {
	if p == nil {
		return false
	}
	if p.MountedEnvNames == nil {
		return true
	}
	return lo.Contains(p.MountedEnvNames, envName)
}

// GetLogicalRootID returns the logical-root config file ID when it is known.
func (p *EnvConfigPolicy) GetLogicalRootID(selfID bson.ObjectID) (bson.ObjectID, bool) {
	if p.DefaultAppConfigFileID != nil {
		return *p.DefaultAppConfigFileID, true
	}
	if selfID != bson.NilObjectID {
		return selfID, true
	}
	return bson.NilObjectID, false
}

// initializeContentFields init Content and OverlayContent
// fields based on file type and content source type
// refs: design_notes/multiple_values.md
func (acf *AppConfigFile) initializeContentFields(
	fileType AppConfigFileType, contentSourceType ContentSourceType,
) {
	emptyStr := ""

	// Normal file type
	if fileType == AppConfigFileTypeNormal {
		switch contentSourceType {
		case ContentSourceTypeBSCP:
			// For normal files with BSCP source, initialize overlay content for overrides
			acf.OverlayContent = &emptyStr
		case ContentSourceTypeLocal:
			// For normal files with local source, initialize content
			acf.Content = &emptyStr
		}
		return
	}

	// Overlay file type: always initialize overlay content
	if fileType == AppConfigFileTypeOverlay && contentSourceType == ContentSourceTypeLocal {
		acf.OverlayContent = &emptyStr
	}
}
