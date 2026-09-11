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
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"
)

// ContentSourceType indicates the source type of the content.
type ContentSourceType string

const (
	// ContentSourceTypeLocal indicates that the content is from the local storage.
	ContentSourceTypeLocal ContentSourceType = "local"
	// ContentSourceTypeBSCP indicates that the content is from the BSCP.
	ContentSourceTypeBSCP ContentSourceType = "bscp"
)

// EnvNameDefault represents the default (app-level) configuration.
// In database, this is stored as empty string for efficiency and simplicity.
// Use this constant instead of empty string literal to make the code more explicit.
const EnvNameDefault = ""

const (
	// CfgSystemUser is the user who created the system original config.
	CfgSystemUser = "system"
	// CfgSystemVersionDescription is the description of the system config version.
	CfgSystemVersionDescription = "初始配置（由系统创建）"
)

// AppConfigFileType indicates the type of the app config file.
type AppConfigFileType string

// DefaultAppConfigFileName is the name for the default app config file of a Helm application.
// The file is created when the Helm application is created.
const DefaultAppConfigFileName = "default"

const (
	// AppConfigFileTypeNormal means the app config file is normal, it's content can be obtained directly
	// from the file object itself.
	//
	// An important note is that even if the app config file is normal, it can still have overlay content.
	AppConfigFileTypeNormal AppConfigFileType = "normal"

	// AppConfigFileTypeOverlay means the app config file is an overlay, to obtain its content, one must
	// merge it's content with the base app config file.
	AppConfigFileTypeOverlay AppConfigFileType = "overlay"
)

// AllowedAppConfigFileTypes is the list of allowed app config file types.
var AllowedAppConfigFileTypes = []AppConfigFileType{AppConfigFileTypeNormal, AppConfigFileTypeOverlay}

// FileFormat indicates the format of the config file content.
type FileFormat string

const (
	// FileFormatYAML indicates YAML format
	FileFormatYAML FileFormat = "yaml"
	// FileFormatTAF indicates TAF config file format
	FileFormatTAF FileFormat = "taf"
)

var (
	// ErrAppCfgFileVersionNotFound indicates the requested version record does not exist.
	ErrAppCfgFileVersionNotFound = errors.New("app config file version not found")
	// ErrComparedVersionsBelongToDifferentFiles indicates compare inputs are from different files.
	ErrComparedVersionsBelongToDifferentFiles = errors.New("two versions must belong to same app config file")
	// ErrUsingVersionCannotBeDeleted indicates the live version cannot be soft deleted.
	ErrUsingVersionCannotBeDeleted = errors.New("current active version cannot be deleted")
	// ErrAppConfigFileReferenced 文件被其他 overlay 文件引用，不允许删除。
	ErrAppConfigFileReferenced = errors.New("file is referenced by other files")
	// ErrEnvConfigRequiresDefaultFile 环境配置变更仅允许在默认实例上执行。
	ErrEnvConfigRequiresDefaultFile = errors.New("env config changes require default file")
	// ErrInvalidConfigSpec 配置规格不合法。
	ErrInvalidConfigSpec = errors.New("invalid config spec")
	// ErrPlainEnvInstanceDeleteNotAllowed plain 环境实例必须通过 env-config 策略或 reset 接口管理，
	// 不允许直接删除。
	ErrPlainEnvInstanceDeleteNotAllowed = errors.New(
		"plain env instance must be deleted via env-config-policy on the default file",
	)
	// ErrResetToDefaultRequiresIndependentConfig 恢复默认操作要求文件处于独立配置模式。
	ErrResetToDefaultRequiresIndependentConfig = errors.New(
		"reset to default requires independent env config mode",
	)
)

// AppConfigFileVersionOperationType indicates how a version was generated.
type AppConfigFileVersionOperationType string

const (
	// AppConfigFileVersionOperationTypeCreate means the version was created
	AppConfigFileVersionOperationTypeCreate AppConfigFileVersionOperationType = "create"
	// AppConfigFileVersionOperationTypeUpdate means the version was updated
	AppConfigFileVersionOperationTypeUpdate AppConfigFileVersionOperationType = "update"
	// AppConfigFileVersionOperationTypeRollback means the version was rolled back
	AppConfigFileVersionOperationTypeRollback AppConfigFileVersionOperationType = "rollback"
)

// ConfigKind 配置文件种类：framework（框架配置）或 plain（通用文本配置）。
type ConfigKind string

const (
	// ConfigKindFramework 框架管理的配置文件（Helm values、tRPC 配置等）。
	ConfigKindFramework ConfigKind = "framework"
	// ConfigKindPlain 通用文本配置文件，以 overwrite 方式按环境独立管理。
	ConfigKindPlain ConfigKind = "plain"
)

// CreateCfgFileParams 创建配置文件的参数。
type CreateCfgFileParams struct {
	AppID               string
	EnvName             string
	Name                string
	MountDir            string
	Type                AppConfigFileType
	ContentSourceType   ContentSourceType
	Format              FileFormat
	BaseAppConfigFileID *bson.ObjectID
	BSCPConfig          *BSCPConfig
	Content             *string
	OverlayContent      *string
	Creator             string
	Description         string
	// ConfigKind 决定适用的策略集，默认 ConfigKindFramework。
	ConfigKind ConfigKind
}

// FileDefUpdate 描述应用配置文件 def 级字段的更新请求。
// 指针字段为可选更新项，nil 表示不修改。
type FileDefUpdate struct {
	Name            *string
	MountDir        *string
	IsUnifiedConfig *bool
	MountedEnvNames *[]string
	Operator        string
}

// CreateEnvInstanceParams 创建环境级配置实例的参数。
type CreateEnvInstanceParams struct {
	EnvName        string
	Content        *string
	OverlayContent *string
	Operator       string
	Description    string
}

// UpdateCfgFileOptions 文件变更持久化选项。
type UpdateCfgFileOptions struct {
	OperationType       AppConfigFileVersionOperationType
	Description         string
	RollbackFromVersion *int64
	// ExpectedCurrentVersion 是编辑开始时的当前版本号，用于乐观锁冲突检测。
	// 传入时后端会校验该版本号是否与数据库中当前版本号一致，不一致则返回冲突错误。
	// 为 nil 时使用从数据库读取的当前版本号（兼容helm逻辑）。
	// todo 待helm前端适配版本管理后移除兼容，改为必填内容
	ExpectedCurrentVersion *int64
}

// UpsertEnvContentParams 描述一次按环境更新配置内容的场景参数。
type UpsertEnvContentParams struct {
	EnvName                 string
	Content                 string
	Operator                string
	Description             string
	ExpectedCurrentVersion  *int64
	ValidateCompiledContent func(targetFile *AppConfigFile, compiledContent string) error
}

// UpsertEnvContentResult 描述一次按环境更新配置内容后的结果。
type UpsertEnvContentResult struct {
	File            *AppConfigFile
	CompiledContent string
}

// EnvFileDetailResult 环境文件详情查询结果。
type EnvFileDetailResult struct {
	Def         *AppConfigFileDef
	DefaultFile *AppConfigFile
	// DisplayFile 应展示内容的文件，由策略决定：
	//   - 有环境实例 → 环境实例
	//   - 无实例 + overwrite 策略 → 默认文件（默认即生效内容）
	//   - 无实例 + overlay 策略 → nil（无定制内容）
	DisplayFile *AppConfigFile
	// HasEnvInstance 该环境是否存在独立实例。
	HasEnvInstance bool
	// EditableContentField 前端可编辑的字段名（"content" / "overlayContent" / "none"）。
	EditableContentField string
	// BaseContentInfo overlay / BSCP 文件的 base 内容信息，无 base 时为 nil。
	BaseContentInfo *BaseContentInfo
}

// ValidateFrameworkFileContent 校验 framework 配置文件内容语法（必须为合法 YAML）。
func ValidateFrameworkFileContent(content string) error {
	if content == "" {
		return nil
	}
	var out any
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		return errors.Wrap(err, "content is not valid YAML")
	}
	return nil
}
