// Package serializer defines Gin input and output serializers for app config file APIs.
package serializer

import (
	"regexp"
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
	}
}

func validateAppConfigFileName(fl validator.FieldLevel) bool {
	return appConfigFileNamePattern.MatchString(fl.Field().String())
}

var appConfigFileNamePattern = regexp.MustCompile("^[a-zA-Z0-9-_]+$")

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
	// 环境名称，可选。为空表示应用级别配置，非空表示特定环境的配置
	EnvName *string `json:"envName,omitempty"`
	// 文件格式，可选 yaml 或 taf
	FileFormat string `json:"fileFormat" binding:"required,oneof=yaml taf"`
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
