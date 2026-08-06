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
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
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

// AppConfigFileContentSpec contains the shared content-related fields
// between AppConfigFile and AppConfigFileVersion.
type AppConfigFileContentSpec struct {
	// AppID is the ID of the application which the app config file belongs to.
	AppID string `bson:"appID"`
	// EnvName is the name of the environment which the app config file belongs to.
	// Usage varies by application type:
	// - For Helm apps: always empty
	// - For tRPC apps: empty means app-level default, non-empty means env-specific config
	EnvName string `bson:"envName"`

	// Name is the name of the app config file.
	Name string `bson:"name"`
	// Type is the type of the app config file (normal or overlay).
	Type AppConfigFileType `bson:"type"`
	// ContentSourceType indicates where the content of the app config file comes from,
	// can be either from the local storage or external system like BSCP.
	ContentSourceType ContentSourceType `bson:"contentSourceType"`
	// Format indicates the format of the config file (yaml, taf).
	// Default is yaml if not specified.
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

	// BaseAppConfigFileID is the ID of the base app config file, it is only set when Type is overlay.
	BaseAppConfigFileID *bson.ObjectID `bson:"baseAppConfigFileID,omitempty"`

	// Creator is the creator of this logical file / version record.
	Creator string `bson:"creator"`

	// CreatedAt is the creation time
	CreatedAt time.Time `bson:"createdAt"`
}

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
	ID                       bson.ObjectID `bson:"_id,omitempty"`
	AppConfigFileContentSpec `bson:",inline"`
	// CurrentVersion is the current active version number of this file.
	CurrentVersion int64 `bson:"currentVersion"`

	// Updater is the last modifier of the current active version.
	Updater string `bson:"updater,omitempty"`

	// UpdatedAt is the last update time
	UpdatedAt time.Time `bson:"updatedAt"`
}

// AppConfigFileVersion stores immutable records of app config file changes
type AppConfigFileVersion struct {
	ID                       bson.ObjectID `bson:"_id,omitempty"`
	AppConfigFileID          bson.ObjectID `bson:"appConfigFileID"`
	AppConfigFileContentSpec `bson:",inline"`
	Version                  int64                             `bson:"version"`
	Description              string                            `bson:"description,omitempty"`
	BaseVersion              *int64                            `bson:"baseVersion,omitempty"`
	OperationType            AppConfigFileVersionOperationType `bson:"operationType"`
	RollbackFromVersion      *int64                            `bson:"rollbackFromVersion,omitempty"`
	IsDeleted                bool                              `bson:"isDeleted"`
	Deleter                  string                            `bson:"deleter,omitempty"`
	DeletedAt                *time.Time                        `bson:"deletedAt,omitempty"`
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

// CreateCfgFileParams defines how to create an app config file.
type CreateCfgFileParams struct {
	AppID               string
	EnvName             string
	Name                string
	Type                AppConfigFileType
	ContentSourceType   ContentSourceType
	Format              FileFormat
	BaseAppConfigFileID *bson.ObjectID
	BSCPConfig          *BSCPConfig
	Content             *string
	OverlayContent      *string
	Creator             string
	Description         string
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
