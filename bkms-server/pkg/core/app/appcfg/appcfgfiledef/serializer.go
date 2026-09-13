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

// Package appcfgfiledef 提供新版 def 视角的 API 序列化类型，适用于 framework 和 plain 配置文件。
package appcfgfiledef

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// --- URI / Query 输入 ---

// DefURIInput def 操作的路径参数。
type DefURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
	ID    string `uri:"id" binding:"required,mongodb"`
}

// DefEnvURIInput def + env 操作的路径参数。
type DefEnvURIInput struct {
	DefURIInput
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// AppURIInput 应用级操作的路径参数。
type AppURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// EnvNameQuery 环境名查询参数。
type EnvNameQuery struct {
	EnvName string `form:"envName" binding:"omitempty"`
}

// --- 创建 ---

// CreateDefInput 创建配置文件 def + 默认文件的请求体。
type CreateDefInput struct {
	// 文件名称
	Name string `json:"name" binding:"required,min=1,max=64,app_config_file_name"`
	// 配置种类：framework / plain
	ConfigKind string `json:"configKind" binding:"required,oneof=framework plain"`
	// 容器内挂载目录（plain 必填）
	MountDir string `json:"mountDir,omitempty" binding:"omitempty,max=255,mount_dir"`
	// 文件格式
	FileFormat string `json:"fileFormat" binding:"required,oneof=yaml taf"`
	// 文件类型：normal / overlay
	FileType string `json:"fileType" binding:"required,oneof=normal overlay"`
	// 内容来源：local / bscp
	ContentSourceType string `json:"contentSourceType" binding:"required,oneof=local bscp"`
	// overlay 文件引用的基础文件 ID（fileType=overlay 时必填）
	BaseAppConfigFileID string `json:"baseAppConfigFileId,omitempty" binding:"omitempty,mongodb"`
	// BSCP 来源配置（contentSourceType=bscp 时必填）
	BSCPConfig *BSCPConfigInput `json:"bscpConfig,omitempty"`
	// 初始内容（可选，仅 local 来源）
	Content *string `json:"content,omitempty"`
	// 版本描述
	Description string `json:"description"`
}

// BSCPConfigInput BSCP 来源配置输入。
type BSCPConfigInput struct {
	BizID     string `json:"bizID" binding:"required"`
	ServiceID string `json:"serviceID" binding:"required"`
	ID        string `json:"id" binding:"required"`
}

// CreateDefOutput 创建成功响应。
type CreateDefOutput struct {
	Item *DefDetailObj `json:"item"`
}

// --- 列表 ---

// ListDefsOutput 列表响应。
type ListDefsOutput struct {
	Items []*DefDetailObj `json:"items"`
}

// DefSummaryObj def 列表中的摘要信息。
type DefSummaryObj struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	ConfigKind      string   `json:"configKind"`
	MountDir        string   `json:"mountDir,omitempty"`
	IsUnifiedConfig bool     `json:"isUnifiedConfig"`
	MountedEnvNames []string `json:"mountedEnvNames,omitempty"`
	Creator         string   `json:"creator"`
	CreatedAt       string   `json:"createdAt"`
}

// FromDef 从 def model 填充。
func (o *DefSummaryObj) FromDef(def appcfg.AppConfigFileDef) *DefSummaryObj {
	*o = DefSummaryObj{
		ID:              def.ID.Hex(),
		Name:            def.Name,
		ConfigKind:      string(def.ConfigKind),
		MountDir:        def.MountDir,
		IsUnifiedConfig: def.EnvConfigMode.IsUnifiedConfig,
		MountedEnvNames: def.EnvConfigMode.MountedEnvNames,
		Creator:         def.Creator,
		CreatedAt:       def.CreatedAt.Format(time.RFC3339),
	}
	return o
}

// --- 详情 ---

// GetDefDetailOutput def 详情响应（含默认文件信息）。
type GetDefDetailOutput struct {
	Item *DefDetailObj `json:"item"`
}

// DefDetailObj def 详情，包含默认文件的内容信息。
type DefDetailObj struct {
	ID                  string   `json:"id"`
	FileID              string   `json:"fileId,omitempty"`
	Name                string   `json:"name"`
	ConfigKind          string   `json:"configKind"`
	MountDir            string   `json:"mountDir,omitempty"`
	IsUnifiedConfig     bool     `json:"isUnifiedConfig"`
	MountedEnvNames     []string `json:"mountedEnvNames,omitempty"`
	FileType            string   `json:"fileType,omitempty"`
	ContentSourceType   string   `json:"contentSourceType"`
	BaseAppConfigFileID string   `json:"baseAppConfigFileId,omitempty"`
	FileFormat          string   `json:"fileFormat"`
	CurrentVersion      int64    `json:"currentVersion"`
	Content             *string  `json:"content,omitempty"`
	OverlayContent      *string  `json:"overlayContent,omitempty"`
	// HasEnvInstance 指定环境是否有独立实例（仅按环境查询时有意义）。
	HasEnvInstance *bool `json:"hasEnvInstance,omitempty"`
	// EditableContentField 前端可编辑的字段（"content" / "overlayContent" / "none"）。
	EditableContentField string              `json:"editableContentField,omitempty"`
	BaseContentInfo      *BaseContentInfoObj `json:"baseContentInfo,omitempty"`
	Updater              string              `json:"updater"`
	UpdatedAt            string              `json:"updatedAt"`
	Creator              string              `json:"creator"`
	CreatedAt            string              `json:"createdAt"`
}

// BaseContentInfoObj base 内容信息（overlay / BSCP 场景下有值）。
type BaseContentInfoObj struct {
	HolderID                string `json:"holderId"`
	HolderName              string `json:"holderName"`
	HolderContentSourceType string `json:"holderContentSourceType"`
	Content                 string `json:"content"`
	IsFromAnotherFile       bool   `json:"isFromAnotherFile"`
}

// FromDefAndFile 从 def + 文件填充（列表场景使用，填充默认文件信息）。
func (o *DefDetailObj) FromDefAndFile(def appcfg.AppConfigFileDef, file *appcfg.AppConfigFile) *DefDetailObj {
	o.fillDef(def)
	if file != nil {
		o.fillFile(file)
	}
	return o
}

// FromEnvFileDetail 从环境文件详情结果填充（按环境查询场景）。
// 展示逻辑由 service 层通过 DisplayFile 决定，serializer 不关心 ConfigKind。
func (o *DefDetailObj) FromEnvFileDetail(result *appcfg.EnvFileDetailResult) *DefDetailObj {
	o.fillDef(*result.Def)

	// 始终填充默认文件的基础元信息（格式、来源类型等）
	if result.DefaultFile != nil {
		o.FileID = result.DefaultFile.ID.Hex()
		o.ContentSourceType = string(result.DefaultFile.ContentSourceType)
		o.FileFormat = string(result.DefaultFile.GetConfigFormat())
		o.CurrentVersion = result.DefaultFile.CurrentVersion
		o.Updater = result.DefaultFile.Updater
		o.UpdatedAt = result.DefaultFile.UpdatedAt.Format(time.RFC3339)
	}

	o.HasEnvInstance = &result.HasEnvInstance
	o.EditableContentField = result.EditableContentField

	if result.BaseContentInfo != nil {
		o.BaseContentInfo = &BaseContentInfoObj{
			HolderID:                result.BaseContentInfo.HolderID.Hex(),
			HolderName:              result.BaseContentInfo.HolderName,
			HolderContentSourceType: result.BaseContentInfo.HolderContentSourceType,
			Content:                 result.BaseContentInfo.Content,
			IsFromAnotherFile:       result.BaseContentInfo.IsFromAnotherFile,
		}
	}

	// DisplayFile 由 service 按策略决定：有值则展示其内容，nil 表示无可展示内容
	if result.DisplayFile != nil {
		o.fillFile(result.DisplayFile)
	}
	return o
}

func (o *DefDetailObj) fillDef(def appcfg.AppConfigFileDef) {
	o.ID = def.ID.Hex()
	o.Name = def.Name
	o.ConfigKind = string(def.ConfigKind)
	o.MountDir = def.MountDir
	o.IsUnifiedConfig = def.EnvConfigMode.IsUnifiedConfig
	o.MountedEnvNames = def.EnvConfigMode.MountedEnvNames
	o.Creator = def.Creator
	o.CreatedAt = def.CreatedAt.Format(time.RFC3339)
}

func (o *DefDetailObj) fillFile(file *appcfg.AppConfigFile) {
	o.FileID = file.ID.Hex()
	o.FileType = string(file.Type)
	o.ContentSourceType = string(file.ContentSourceType)
	o.FileFormat = string(file.GetConfigFormat())
	o.CurrentVersion = file.CurrentVersion
	o.Content = file.Content
	o.OverlayContent = file.OverlayContent
	if file.BaseAppConfigFileID != nil {
		o.BaseAppConfigFileID = file.BaseAppConfigFileID.Hex()
	}
	o.Updater = file.Updater
	o.UpdatedAt = file.UpdatedAt.Format(time.RFC3339)
}

// --- 更新 def ---

// UpdateDefInput 更新 def 信息请求体。
type UpdateDefInput struct {
	Name            *string   `json:"name,omitempty" binding:"omitempty,min=1,max=64,app_config_file_name"`
	MountDir        *string   `json:"mountDir,omitempty" binding:"omitempty,max=255,mount_dir"`
	IsUnifiedConfig *bool     `json:"isUnifiedConfig,omitempty"`
	MountedEnvNames *[]string `json:"mountedEnvNames,omitempty"`
}

// ToFileDefUpdate 转换为 service 层参数。
func (i UpdateDefInput) ToFileDefUpdate(operator string) appcfg.FileDefUpdate {
	return appcfg.FileDefUpdate{
		Name:            i.Name,
		MountDir:        i.MountDir,
		IsUnifiedConfig: i.IsUnifiedConfig,
		MountedEnvNames: i.MountedEnvNames,
		Operator:        operator,
	}
}

// UpdateDefOutput 更新 def 响应。
type UpdateDefOutput struct {
	Item *DefSummaryObj `json:"item"`
}

// --- 环境实例 ---

// EnvInstanceObj 环境实例信息。
type EnvInstanceObj struct {
	FileID         string  `json:"fileId"`
	EnvName        string  `json:"envName"`
	Type           string  `json:"type"`
	CurrentVersion int64   `json:"currentVersion"`
	Content        *string `json:"content,omitempty"`
	OverlayContent *string `json:"overlayContent,omitempty"`
	Updater        string  `json:"updater"`
	UpdatedAt      string  `json:"updatedAt"`
}

// FromModel 从文件记录填充。
func (o *EnvInstanceObj) FromModel(f appcfg.AppConfigFile) *EnvInstanceObj {
	*o = EnvInstanceObj{
		FileID:         f.ID.Hex(),
		EnvName:        f.EnvName,
		Type:           string(f.Type),
		CurrentVersion: f.CurrentVersion,
		Content:        f.Content,
		OverlayContent: f.OverlayContent,
		Updater:        f.Updater,
		UpdatedAt:      f.UpdatedAt.Format(time.RFC3339),
	}
	return o
}

// ListEnvInstancesOutput 环境实例列表响应。
type ListEnvInstancesOutput struct {
	Items []*EnvInstanceObj `json:"items"`
}

// --- 环境详情 ---

// GetEnvDetailOutput 某环境下的 def 详情响应。
type GetEnvDetailOutput struct {
	Item *DefDetailObj `json:"item"`
}

// --- 内容更新 ---

// UpdateContentInput 更新配置内容请求体。
type UpdateContentInput struct {
	Content        string `json:"content"`
	Description    string `json:"description"`
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// UpdateContentOutput 更新内容响应。
type UpdateContentOutput struct {
	FileID         string `json:"fileId"`
	CurrentVersion int64  `json:"currentVersion"`
	Content        string `json:"content"`
}

// --- 重置 ---

// ResetEnvOutput 重置环境到默认响应。
type ResetEnvOutput struct{}

// --- 删除 ---

// DeleteDefOutput 删除 def 响应。
type DeleteDefOutput struct{}

// --- 挂载预览 ---

// MountPreviewItemObj 挂载预览单条记录。
type MountPreviewItemObj struct {
	DefID         string `json:"defId"`
	Name          string `json:"name"`
	ConfigKind    string `json:"configKind"`
	MountDir      string `json:"mountDir,omitempty"`
	ContentSource string `json:"contentSource"`
	HasEnvFile    bool   `json:"hasEnvFile"`
}

// MountPreviewOutput 挂载预览响应。
type MountPreviewOutput struct {
	Items []*MountPreviewItemObj `json:"items"`
}
