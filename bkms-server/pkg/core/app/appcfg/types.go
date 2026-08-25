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
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
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

// ConfigKind indicates the semantic kind of an app config file.
type ConfigKind string

const (
	// ConfigKindFramework represents the existing framework-managed config file semantics.
	ConfigKindFramework ConfigKind = "framework"
	// ConfigKindPlain represents a plain text-like config file mounted as-is.
	ConfigKindPlain ConfigKind = "plain"
)

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
)

// BSCPConfig is the reference to BSCP resource.
type BSCPConfig struct {
	// BizID is BSCP business ID
	BizID string `bson:"bizID"`
	// ServiceID is BSCP service ID
	ServiceID string `bson:"serviceID"`
	// VersionID is BSCP Version ID which only record but not used,
	// bkms always fetch "latest fully releases version" config content
	VersionID string `bson:"versionID"`
	// ConfigID is BSCP configuration ID
	ConfigID string `bson:"configID"`
}

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

// FetchContent 获取 BSCP 配置对应的配置内容
func (c *BSCPConfig) FetchContent(ctx context.Context) (string, error) {
	if c == nil {
		return "", errors.New("bscp config cannot be nil")
	}

	// 1. 初始化客户端
	client, err := bscp.New(auth.MustGetUser(ctx))
	if err != nil {
		return "", errors.Wrap(err, "initial bscp client")
	}

	// 2. 获取服务下的版本列表
	versions, err := client.ListServiceVersions(ctx, c.BizID, c.ServiceID)
	if err != nil {
		return "", errors.Wrap(err, "list bscp service versions")
	}

	// 3. 找到最新的全量发布的已上线版本
	ver := versions.LatestFullyReleased()
	if ver == nil {
		return "", errors.New("no fully released version")
	}

	// 4. 获取服务版本下的配置 & 提取配置内容
	cfg, err := client.GetServiceConfig(ctx, c.BizID, c.ServiceID, ver.ID, c.ConfigID)
	if err != nil {
		return "", errors.Wrap(err, "get bscp service config")
	}

	content, err := cfg.Content(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get bscp config content")
	}

	return content, nil
}

// CreateCfgFileParams defines how to create an app config file.
type CreateCfgFileParams struct {
	AppID                  string
	EnvName                string
	Name                   string
	Type                   AppConfigFileType
	ContentSourceType      ContentSourceType
	Format                 FileFormat
	ConfigKind             ConfigKind
	MountPath              string
	DefaultAppConfigFileID *bson.ObjectID
	IsUnifiedConfig        bool
	MountedEnvNames        []string
	BaseAppConfigFileID    *bson.ObjectID
	BSCPConfig             *BSCPConfig
	Content                *string
	OverlayContent         *string
	Creator                string
	Description            string
}

// UpdateCfgFileOptions describes how to persist a file change.
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

// UpdateEnvConfigParams 描述更新默认逻辑文件按环境配置策略时所需的参数。
type UpdateEnvConfigParams struct {
	// IsUnifiedConfig 目标配置模式，true = 统一配置，false = 按环境独立配置。
	IsUnifiedConfig bool
	// MountedEnvNames 挂载环境范围。nil = 全部环境；非 nil 空切片 = 不挂载到任何环境；非空 = 仅指定环境。
	MountedEnvNames []string
	// FallbackConfigEnv 回退为共用配置的环境名称。仅在 IsUnifiedConfig=false 时有效。
	// 该环境的独立实例（若存在）将被删除，恢复为引用默认记录内容的状态。
	FallbackConfigEnv string
	// Operator 当前操作人，用于记录版本与审计信息。
	Operator string
	// Description 本次变更说明。
	Description string
	// ExpectedCurrentVersion 默认逻辑文件期望的当前版本号，用于乐观锁校验。
	ExpectedCurrentVersion *int64
}
