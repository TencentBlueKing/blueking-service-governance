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

// AppConfigFileDef 配置文件的逻辑身份，存储在 app_config_file_defs 集合。
// 一条 def 对应一或多条 AppConfigFile（默认实例 + 环境实例）。
type AppConfigFileDef struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	AppID      string        `bson:"appID"`
	Name       string        `bson:"name"`
	ConfigKind ConfigKind    `bson:"configKind"`
	// MountDir 容器内挂载目录（如 /usr/local/trpc/bin）。
	// 容器内完整路径 = filepath.Join(MountDir, Name)。
	MountDir string `bson:"mountDir,omitempty"`
	// EnvConfigMode 环境配置模式（统一配置 / 按环境独立配置）。
	EnvConfigMode EnvConfigMode `bson:"envConfigMode"`
	Creator       string        `bson:"creator"`
	CreatedAt     time.Time     `bson:"createdAt"`
}

// AppConfigFile 应用配置文件，如 Helm values 文件、tRPC 配置文件等。
//
// 合法的字段组合：
//  1. Normal 类型文件，仅有 Content
//  2. Normal 类型文件（非 local 来源），同时有 Content 和 OverlayContent
//  3. Overlay 类型文件，仅有 OverlayContent
//
// TODO: 引入 plain ConfigKind 后需补充 Overwrite 类型的字段组合说明。
//
// 按应用类型的使用规则：
//   - Helm：EnvName 始终为空，一个 appID 可有多个配置文件
//   - tRPC/TAF：EnvName 为空表示默认配置，非空表示环境配置；同一 appID + envName 最多一条记录
type AppConfigFile struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// DefID 关联的 AppConfigFileDef 记录 ID。
	DefID bson.ObjectID `bson:"defID,omitempty"`
	// AppID 所属应用 ID。
	AppID string `bson:"appID"`
	// EnvName 所属环境名称。
	EnvName string `bson:"envName"`
	// Type 配置文件类型（normal / overlay）。
	Type AppConfigFileType `bson:"type"`
	// VersionedContent 可变内容字段。
	VersionedContent `bson:",inline"`

	// Creator 创建者。
	Creator string `bson:"creator"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `bson:"createdAt"`
	// CurrentVersion 当前生效版本号。
	CurrentVersion int64 `bson:"currentVersion"`
	// Updater 最近一次修改者。
	Updater string `bson:"updater,omitempty"`
	// UpdatedAt 最近一次更新时间。
	UpdatedAt time.Time `bson:"updatedAt"`
}

// AppConfigFileVersion 配置文件的不可变版本记录。
type AppConfigFileVersion struct {
	ID              bson.ObjectID `bson:"_id,omitempty"`
	AppConfigFileID bson.ObjectID `bson:"appConfigFileID"`
	// DefID 关联的 AppConfigFileDef 记录 ID。
	DefID bson.ObjectID `bson:"defID,omitempty"`
	// AppID 版本创建时从文件快照的应用 ID。
	AppID string `bson:"appID"`
	// EnvName 版本创建时从文件快照的环境名。
	EnvName string `bson:"envName"`
	// Name 版本创建时从文件快照的文件名。
	Name string `bson:"name"`
	// Type 版本创建时从文件快照的文件类型。
	Type AppConfigFileType `bson:"type"`
	// VersionedContent 版本创建时快照的可变内容字段。
	VersionedContent `bson:",inline"`

	Version             int64                             `bson:"version"`
	Description         string                            `bson:"description,omitempty"`
	BaseVersion         *int64                            `bson:"baseVersion,omitempty"`
	OperationType       AppConfigFileVersionOperationType `bson:"operationType"`
	RollbackFromVersion *int64                            `bson:"rollbackFromVersion,omitempty"`
	IsDeleted           bool                              `bson:"isDeleted"`
	Deleter             string                            `bson:"deleter,omitempty"`
	DeletedAt           *time.Time                        `bson:"deletedAt,omitempty"`

	// Creator 创建此版本记录的操作者。
	Creator string `bson:"creator"`
	// CreatedAt 版本创建时间。
	CreatedAt time.Time `bson:"createdAt"`
}

// EnvConfigMode 描述逻辑配置文件的环境差异化策略。
type EnvConfigMode struct {
	// IsUnifiedConfig 为 true 时所有环境共享同一配置，不产生环境实例。
	IsUnifiedConfig bool `bson:"isUnifiedConfig"`
	// MountedEnvNames 已显式配置的环境列表。
	MountedEnvNames []string `bson:"mountedEnvNames,omitempty"`
}

// VersionedContent 配置文件的可变内容字段，AppConfigFile 和 AppConfigFileVersion 共享。
type VersionedContent struct {
	// ContentSourceType 内容来源类型（local / bscp）。
	ContentSourceType ContentSourceType `bson:"contentSourceType"`
	// Format 配置文件格式（yaml / taf）。
	Format FileFormat `bson:"format,omitempty"`
	// BSCPConfig BSCP 资源引用，仅 ContentSourceType=bscp 时有值。
	BSCPConfig *BSCPConfig `bson:"bscpConfig,omitempty"`
	// Content 主内容，overlay 类型文件为 nil。
	Content *string `bson:"content,omitempty"`
	// OverlayContent overlay/patch 内容。
	OverlayContent *string `bson:"overlayContent,omitempty"`
	// BaseAppConfigFileID 基础文件 ID，仅 overlay 类型文件设置。
	BaseAppConfigFileID *bson.ObjectID `bson:"baseAppConfigFileID,omitempty"`
}

// GetConfigFormat 返回配置格式，未指定时默认 YAML。
func (vc *VersionedContent) GetConfigFormat() FileFormat {
	if vc.Format == "" {
		log.WarnNoContextf("versioned content has no format, default to YAML format")
		return FileFormatYAML
	}
	return vc.Format
}

// BSCPConfig BSCP 资源引用。
type BSCPConfig struct {
	// BizID BSCP 业务 ID。
	BizID string `bson:"bizID"`
	// ServiceID BSCP 服务 ID。
	ServiceID string `bson:"serviceID"`
	// VersionID BSCP 版本 ID，仅记录不使用，bkms 总是拉取最新全量发布版本的配置内容。
	VersionID string `bson:"versionID"`
	// ConfigID BSCP 配置项 ID。
	ConfigID string `bson:"configID"`
}

// FetchContent 获取 BSCP 配置对应的配置内容。
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

// initializeContentFields 根据文件类型和内容来源初始化 Content / OverlayContent 字段。
// refs: design_notes/multiple_values.md
func (acf *AppConfigFile) initializeContentFields(
	fileType AppConfigFileType, contentSourceType ContentSourceType,
) {
	emptyStr := ""

	// Normal 类型
	if fileType == AppConfigFileTypeNormal {
		switch contentSourceType {
		case ContentSourceTypeBSCP:
			// BSCP 来源的 normal 文件，初始化 overlay 用于本地覆盖
			acf.OverlayContent = &emptyStr
		case ContentSourceTypeLocal:
			// local 来源的 normal 文件，初始化主内容
			acf.Content = &emptyStr
		}
		return
	}

	// Overlay 类型：初始化 overlay 内容
	if fileType == AppConfigFileTypeOverlay && contentSourceType == ContentSourceTypeLocal {
		acf.OverlayContent = &emptyStr
	}
}

// AppValuesConfig 应用的 values 文件配置，当前包含默认 values 文件 ID，后续可扩展更多字段。
type AppValuesConfig struct {
	// ID 唯一标识。
	ID bson.ObjectID `bson:"_id,omitempty"`
	// AppID 所属应用 ID。
	AppID string `bson:"appID"`
	// DefaultAppConfigFileID 默认配置文件 ID。
	DefaultAppConfigFileID *bson.ObjectID `bson:"defaultAppConfigFileID,omitempty"`
}

// AppConfigFileWithDef 配置文件记录 + def 信息的组合视图，供 service/handler 层使用。
type AppConfigFileWithDef struct {
	AppConfigFile
	Def *AppConfigFileDef
}

// GetName 返回文件名，数据来源为 Def.Name。
func (v *AppConfigFileWithDef) GetName() string {
	return v.Def.Name
}

// GetDefID 返回 Def 的 ID。
func (v *AppConfigFileWithDef) GetDefID() bson.ObjectID {
	return v.Def.ID
}

// GetConfigKind 返回配置文件的 kind，未设置时默认 framework。
func (v *AppConfigFileWithDef) GetConfigKind() ConfigKind {
	if v.Def.ConfigKind == "" {
		return ConfigKindFramework
	}
	return v.Def.ConfigKind
}

// applyStaticDefFields 将 FileDefUpdate 中的静态字段应用到 def。
func applyStaticDefFields(def *AppConfigFileDef, update FileDefUpdate) {
	if update.Name != nil {
		def.Name = *update.Name
	}
	if update.MountDir != nil {
		def.MountDir = *update.MountDir
	}
}
