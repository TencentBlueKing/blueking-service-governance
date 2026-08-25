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

// Package serializer defines Gin input and output serializers for app config file APIs.
package serializer

import (
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("app_config_file_name", validateAppConfigFileName); err != nil {
			panic("failed to register app_config_file_name validator: " + err.Error())
		}
		if err := v.RegisterValidation("app_config_file_format", validateAppConfigFileFormat); err != nil {
			panic("failed to register app_config_file_format validator: " + err.Error())
		}
		if err := v.RegisterValidation("plain_mount_path", validatePlainMountPath); err != nil {
			panic("failed to register plain_mount_path validator: " + err.Error())
		}
		v.RegisterStructValidation(validateCreateAppConfigFileInput, CreateAppConfigFileInput{})
	}
}

func validateAppConfigFileName(fl validator.FieldLevel) bool {
	return appConfigFileNamePattern.MatchString(fl.Field().String())
}

func validateAppConfigFileFormat(fl validator.FieldLevel) bool {
	return appConfigFileFormatPattern.MatchString(fl.Field().String())
}

var (
	appConfigFileNamePattern   = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
	appConfigFileFormatPattern = regexp.MustCompile("^[a-z0-9._-]+$")
)

// validatePlainMountPath 校验容器内绝对文件路径：以 / 开头、不等于 /、不以 / 结尾。
func validatePlainMountPath(fl validator.FieldLevel) bool {
	p := strings.TrimSpace(fl.Field().String())
	if p == "" {
		return true // 空值由 required 或 struct-level 校验处理
	}
	return strings.HasPrefix(p, "/") && p != "/" && !strings.HasSuffix(p, "/")
}

// validateCreateAppConfigFileInput 校验 CreateAppConfigFileInput 的跨字段约束。
func validateCreateAppConfigFileInput(sl validator.StructLevel) {
	input := sl.Current().Interface().(CreateAppConfigFileInput)
	kind := strings.ToLower(strings.TrimSpace(input.ConfigKind))
	isPlain := kind == string(appcfg.ConfigKindPlain)

	if !isPlain {
		if strings.TrimSpace(input.MountPath) != "" {
			sl.ReportError(input.MountPath, "MountPath", "MountPath", "excluded_for_framework", "")
		}
		if input.MountedEnvNames != nil {
			sl.ReportError(input.MountedEnvNames, "MountedEnvNames", "MountedEnvNames", "excluded_for_framework", "")
		}
		return
	}
	// plain 类型约束
	if input.Type != "normal" {
		sl.ReportError(input.Type, "Type", "Type", "plain_must_be_normal", "")
	}
	if input.ContentSourceType != "local" {
		sl.ReportError(input.ContentSourceType, "ContentSourceType", "ContentSourceType", "plain_must_be_local", "")
	}
	if input.BaseAppConfigFileID != "" {
		sl.ReportError(
			input.BaseAppConfigFileID,
			"BaseAppConfigFileID",
			"BaseAppConfigFileID",
			"excluded_for_plain",
			"",
		)
	}
	if strings.TrimSpace(input.MountPath) == "" {
		sl.ReportError(input.MountPath, "MountPath", "MountPath", "required_for_plain", "")
	}
}

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppConfigFileURIInput is the path input for one app config file.
type AppConfigFileURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 应用配置文件 ID
	ID string `uri:"id" binding:"required,mongodb"`
}

// AppConfigFileObjectID returns the bound app config file ID as ObjectID.
func (i AppConfigFileURIInput) AppConfigFileObjectID() (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(i.ID)
	if err != nil {
		return bson.ObjectID{}, errors.Wrap(err, "parse app config file ID")
	}
	return id, nil
}

// AppConfigFileVersionURIInput is the path input for one app config file version.
type AppConfigFileVersionURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 版本记录 ID
	ID string `uri:"id" binding:"required,mongodb"`
}

// VersionObjectID returns the bound version ID as ObjectID.
func (i AppConfigFileVersionURIInput) VersionObjectID() (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(i.ID)
	if err != nil {
		return bson.ObjectID{}, errors.Wrap(err, "parse version ID")
	}
	return id, nil
}

// BSCPAppConfigFileConfig is the JSON representation of a BSCP config reference.
type BSCPAppConfigFileConfig struct {
	// BSCP 业务 ID
	BizID string `json:"bizID" binding:"required"`
	// BSCP 服务 ID
	ServiceID string `json:"serviceID" binding:"required"`
	// BSCP 配置 ID
	ID string `json:"id" binding:"required"`
}

// CreateAppConfigFileInput is the JSON body for creating an app config file.
type CreateAppConfigFileInput struct {
	// 应用配置文件名称，包含大小写字母、数字和符号（_-），长度 1-20 之间
	Name string `json:"name" binding:"required,min=1,max=20,app_config_file_name"`
	// 应用配置文件类型，普通或覆盖层
	Type string `json:"type" binding:"required,oneof=normal overlay"`
	// 基础应用配置文件 ID，仅当 type 是 overlay 时为必填，在 handler 中做业务校验
	BaseAppConfigFileID string `json:"baseAppConfigFileID"`
	// 内容来源，可选本地（local）或 bscp
	ContentSourceType string `json:"contentSourceType" binding:"required,oneof=local bscp"`
	// 当 contentSourceType 为 bscp 时，bscpConfig 为必填
	BscpConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty"`
	// 配置文件语义类型，不传时默认按 framework 处理
	ConfigKind string `json:"configKind,omitempty" binding:"omitempty,oneof=framework plain"`
	// plain 配置文件的容器内完整挂载路径
	MountPath string `json:"mountPath,omitempty" binding:"omitempty,min=1,max=255,plain_mount_path"`
	// 挂载环境范围。不传/nil = 对所有环境生效；传空数组 = 不挂载到任何环境；
	// 非空 = 仅对列出的环境生效。创建时始终按统一配置处理，此字段仅控制挂载范围。
	MountedEnvNames []string `json:"mountedEnvNames,omitempty" binding:"omitempty,dive,min=1,max=64"`
	// 环境名称，可选。为空表示应用级别配置，非空表示特定环境的配置
	EnvName *string `json:"envName,omitempty"`
	// 文件格式标识，framework 可选 yaml 或 taf，plain 不校验文件格式
	FileFormat string `json:"fileFormat" binding:"required,min=1,max=32,app_config_file_format"`
	// 版本描述
	Description string `json:"description"`
}

// UpdateAppConfigFileInput is the JSON body for updating app config file metadata.
type UpdateAppConfigFileInput struct {
	// 应用配置文件名称，包含大小写字母、数字和符号（_-），长度 1-20 之间
	Name string `json:"name" binding:"required,min=1,max=20,app_config_file_name"`
	// 基础应用配置文件 ID，仅当 type 是 overlay 时生效
	BaseAppConfigFileID string `json:"baseAppConfigFileID"`
	// 当 contentSourceType 为 bscp 时，bscpConfig 为必填
	BscpConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty"`
	// plain 配置文件的容器内完整挂载路径
	MountPath string `json:"mountPath,omitempty" binding:"omitempty,min=1,max=255,plain_mount_path"`
	// 文件格式标识，plain 仅做弱校验
	FileFormat string `json:"fileFormat,omitempty" binding:"omitempty,min=1,max=32,app_config_file_format"`
	// 版本描述
	Description string `json:"description"`
	// 编辑开始时的当前版本号，用于乐观锁冲突检测
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// UpdateAppConfigFileEnvConfigInput 是切换按环境配置模式时使用的 JSON 请求体。
type UpdateAppConfigFileEnvConfigInput struct {
	// 是否统一配置。true = 统一配置；false = 按环境独立配置。
	IsUnifiedConfig bool `json:"isUnifiedConfig"`
	// 挂载环境范围。不传/nil = 对所有环境生效；传空数组 = 不挂载到任何环境；
	// 非空 = 仅对列出的环境生效。
	MountedEnvNames []string `json:"mountedEnvNames,omitempty" binding:"omitempty,dive,min=1,max=64"`
	// 回退为共用配置的环境名称。仅在按环境独立配置模式下有效，
	// 该环境的独立实例将被删除，恢复为引用默认记录内容的状态。
	FallbackConfigEnv string `json:"fallbackConfigEnv,omitempty" binding:"omitempty,min=1,max=64"`
	// 版本描述
	Description string `json:"description"`
	// 编辑开始时的当前版本号，用于乐观锁冲突检测
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// ListAppConfigFilesQueryInput is the query input for listing app config files.
type ListAppConfigFilesQueryInput struct {
	// 按文件类型过滤，仅展示指定类型（normal/overlay)
	Type *string `form:"type" binding:"omitempty,oneof=normal overlay"`
	// 按环境名称过滤，可选。为空表示不过滤
	EnvName *string `form:"envName"`
}

// UpdateAppConfigFileContentInput is the JSON body for updating content.
type UpdateAppConfigFileContentInput struct {
	// 应用配置文件 content
	Content string `json:"content"`
	// 目标环境名（仅 plain 独立配置模式下使用）。当指定的环境处于引用状态（无独立实例）时，
	// 以当前请求内容为初始值创建独立实例，使该环境脱离对默认配置的引用。不传时直接更新 id 对应的文件。
	EnvName string `json:"envName,omitempty"`
	// 版本描述
	Description string `json:"description"`
	// 编辑开始时的当前版本号，用于乐观锁冲突检测
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// UpdateAppConfigFileOverlayContentInput is the JSON body for updating overlay content.
type UpdateAppConfigFileOverlayContentInput struct {
	// 应用配置文件 overlayContent
	OverlayContent string `json:"overlayContent"`
	// 版本描述
	Description string `json:"description"`
	// 编辑开始时的当前版本号，用于乐观锁冲突检测
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// PreviewOverlayMergeInput is the JSON body for previewing overlay merge result.
type PreviewOverlayMergeInput struct {
	// 覆盖内容（YAML 格式）
	OverlayContent string `json:"overlayContent"`
}

// AppConfigFileEmptyOutput is the JSON response for APIs that return no data.
type AppConfigFileEmptyOutput struct{}

// AppConfigFileOutputObj is the JSON representation of an app config file.
type AppConfigFileOutputObj struct {
	// 应用配置文件 ID
	ID string `json:"id"`
	// 文件名称
	Name string `json:"name"`
	// 文件类型
	Type string `json:"type"`
	// 文件内容来源
	ContentSourceType string `json:"contentSourceType"`
	// 配置文件语义类型
	ConfigKind string `json:"configKind"`
	// plain 配置文件的容器内完整挂载路径
	MountPath string `json:"mountPath,omitempty"`
	// 是否统一配置
	IsUnifiedConfig bool `json:"isUnifiedConfig"`
	// 挂载环境范围
	MountedEnvNames []string `json:"mountedEnvNames"`
	// 基础应用配置文件 ID，可能为空
	BaseAppConfigFileID string `json:"baseAppConfigFileID,omitempty"`
	// 仅当 contentSourceType 为 bscp 时，bscpConfig 才有值
	BscpConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty"`
	// 环境名称，为空表示应用级别配置，非空表示特定环境的配置
	EnvName string `json:"envName"`
	// 文件格式
	FileFormat string `json:"fileFormat"`
	// 当前生效版本号
	CurrentVersion int64 `json:"currentVersion"`
	// 当前生效版本最后修改人
	Updater string `json:"updater"`
	// 当前生效版本最后修改时间，RFC3339 格式
	UpdatedAt string `json:"updatedAt"`
}

// FromModel fills output fields from an app config file model.
func (o *AppConfigFileOutputObj) FromModel(obj appcfg.AppConfigFile) *AppConfigFileOutputObj {
	*o = AppConfigFileOutputObj{
		ID:                obj.ID.Hex(),
		Name:              obj.Name,
		Type:              string(obj.Type),
		ContentSourceType: string(obj.ContentSourceType),
		ConfigKind:        string(obj.GetConfigKind()),
		MountPath:         obj.MountPath,
		IsUnifiedConfig:   obj.IsUnifiedConfig,
		MountedEnvNames:   obj.MountedEnvNames,
		EnvName:           obj.EnvName,
		FileFormat:        string(obj.GetConfigFormat()),
		CurrentVersion:    obj.CurrentVersion,
		Updater:           obj.Updater,
		UpdatedAt:         obj.UpdatedAt.Format(time.RFC3339),
	}
	if obj.BaseAppConfigFileID != nil {
		o.BaseAppConfigFileID = obj.BaseAppConfigFileID.Hex()
	}
	if obj.BSCPConfig != nil {
		o.BscpConfig = &BSCPAppConfigFileConfig{
			BizID:     obj.BSCPConfig.BizID,
			ServiceID: obj.BSCPConfig.ServiceID,
			ID:        obj.BSCPConfig.ConfigID,
		}
	}
	return o
}

// CreateAppConfigFileOutput is the JSON response for creating an app config file.
type CreateAppConfigFileOutput struct {
	// 所创建的对象详情
	Item *AppConfigFileOutputObj `json:"item"`
}

// UpdateAppConfigFileOutput is the JSON response for updating app config file metadata.
type UpdateAppConfigFileOutput struct {
	// 所修改的对象详情
	Item *AppConfigFileOutputObj `json:"item"`
}

// ListAppConfigFilesOutput is the JSON response for listing app config files.
type ListAppConfigFilesOutput struct {
	// 应用配置文件列表
	Items []*AppConfigFileOutputObj `json:"items"`
}

// BaseContentInfoOutputObj provides base content information for overlay files.
type BaseContentInfoOutputObj struct {
	// 基础文件 ID
	HolderID string `json:"holderID"`
	// 基础文件名称，仅供展示用
	HolderName string `json:"holderName"`
	// 基础文件的内容来源
	HolderContentSourceType string `json:"holderContentSourceType"`
	// 基础文件内容
	Content string `json:"content"`
	// 基础文件是否是另一个文件
	IsFromAnotherFile bool `json:"isFromAnotherFile"`
}

// GetAppConfigFileDetailsOutput is the JSON response for app config file details.
type GetAppConfigFileDetailsOutput struct {
	// 可编辑字段，有效值为 "none"、"content"、"overlayContent"
	EditableContentField string `json:"editableContentField"`
	// 内容，通常在文件类型为 normal 时有值
	Content *string `json:"content,omitempty"`
	// 覆盖层内容，通常在文件类型为 overlay 时有值
	OverlayContent *string `json:"overlayContent,omitempty"`
	// 基础文件信息
	BaseContentInfo *BaseContentInfoOutputObj `json:"baseContentInfo,omitempty"`
	// 当前生效版本号
	CurrentVersion int64 `json:"currentVersion"`
	// 当前生效版本最后修改人
	Updater string `json:"updater"`
	// 当前生效版本最后修改时间，RFC3339 格式
	UpdatedAt string `json:"updatedAt"`
}

// ArrgResultItemOutputObj stores one arrangement validation result.
type ArrgResultItemOutputObj struct {
	// 编排状态，可能是 configured 或 skipped
	Status string `json:"status"`
	// 如果状态为 skipped，本字段将提供具体原因
	SkippedReason string `json:"skippedReason"`
}

// ValidateArrgValuesYAMLOutputObj is the JSON representation of arrangement results.
type ValidateArrgValuesYAMLOutputObj struct {
	WorkloadImage *ArrgResultItemOutputObj `json:"workloadImage"`
	IngressDomain *ArrgResultItemOutputObj `json:"ingressDomain"`
}

// UpdateAppConfigFileContentOutput is the JSON response for updating content or overlay content.
type UpdateAppConfigFileContentOutput struct {
	// 完整 values 内容
	CompiledContent string `json:"compiledContent"`
	// 基于新的 values 内容产生的编排结果
	ArrgData *ValidateArrgValuesYAMLOutputObj `json:"arrgData"`
}

// PreviewOverlayMergeOutput is the JSON response for preview overlay merge.
type PreviewOverlayMergeOutput struct {
	// 合并后的完整配置内容
	Data string `json:"data"`
}

// ToVersionListOptions converts query input into version list options.
func (i ListAppConfigFileVersionsQueryInput) ToVersionListOptions(
	appID string,
) (appcfg.AppConfigFileVersionListOptions, error) {
	opts := appcfg.AppConfigFileVersionListOptions{
		AppID:       appID,
		EnvName:     i.EnvName,
		Name:        i.Name,
		Version:     i.Version,
		Creator:     i.Creator,
		Description: i.Description,
		Page:        i.Page,
		PageSize:    i.PageSize,
	}
	if i.AppConfigFileID != nil && *i.AppConfigFileID != "" {
		id, err := bson.ObjectIDFromHex(*i.AppConfigFileID)
		if err != nil {
			return appcfg.AppConfigFileVersionListOptions{}, errors.Wrap(err, "parse appConfigFileID")
		}
		opts.AppConfigFileID = &id
	}
	return opts, nil
}
