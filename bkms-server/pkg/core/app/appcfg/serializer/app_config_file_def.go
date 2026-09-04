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

package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// AppConfigFileDefUpdateInput 更新配置文件 def 信息的请求体。
// 所有字段均为可选（nil = 不修改）。
type AppConfigFileDefUpdateInput struct {
	// 应用配置文件名称；不传表示不修改，传时不能为空。
	Name *string `json:"name,omitempty" binding:"omitempty,min=1,max=64,app_config_file_name"`
	// 容器内挂载目录；不传表示不修改。
	MountDir *string `json:"mountDir,omitempty" binding:"omitempty,max=255,mount_dir"`
	// 是否统一配置；不传表示不修改。true = 统一配置；false = 按环境独立配置。
	IsUnifiedConfig *bool `json:"isUnifiedConfig,omitempty"`
}

// ToFileDefUpdate 将 HTTP 请求体转换为 service 层的 FileDefUpdate。
func (i AppConfigFileDefUpdateInput) ToFileDefUpdate(operator string) appcfg.FileDefUpdate {
	return appcfg.FileDefUpdate{
		Name:            i.Name,
		MountDir:        i.MountDir,
		IsUnifiedConfig: i.IsUnifiedConfig,
		Operator:        operator,
	}
}

// AppConfigFileDefOutputObj 新接口返回的配置文件 def 信息。
type AppConfigFileDefOutputObj struct {
	// Def ID
	ID string `json:"id"`
	// 文件名称
	Name string `json:"name"`
	// 容器内挂载目录
	MountDir string `json:"mountDir,omitempty"`
	// 文件类型
	Type string `json:"type"`
	// 配置种类
	ConfigKind string `json:"configKind"`
	// 是否统一配置
	IsUnifiedConfig bool `json:"isUnifiedConfig"`
	// 文件内容来源
	ContentSourceType string `json:"contentSourceType"`
	// 基础应用配置文件 ID
	BaseAppConfigFileID string `json:"baseAppConfigFileID,omitempty"`
	// 环境名称
	EnvName string `json:"envName"`
	// 文件格式
	FileFormat string `json:"fileFormat"`
	// 当前生效版本号
	CurrentVersion int64 `json:"currentVersion"`
	// 最后修改人
	Updater string `json:"updater"`
	// 最后修改时间
	UpdatedAt string `json:"updatedAt"`
}

// FromModel 从组合视图填充输出字段。
func (o *AppConfigFileDefOutputObj) FromModel(obj appcfg.AppConfigFileWithDef) *AppConfigFileDefOutputObj {
	*o = AppConfigFileDefOutputObj{
		ID:                obj.ID.Hex(),
		Name:              obj.GetName(),
		Type:              string(obj.Type),
		ContentSourceType: string(obj.ContentSourceType),
		EnvName:           obj.EnvName,
		FileFormat:        string(obj.GetConfigFormat()),
		CurrentVersion:    obj.CurrentVersion,
		Updater:           obj.Updater,
		UpdatedAt:         obj.UpdatedAt.Format(time.RFC3339),
	}
	if obj.Def != nil {
		o.ID = obj.Def.ID.Hex()
		o.MountDir = obj.Def.MountDir
		o.ConfigKind = string(obj.Def.ConfigKind)
		o.IsUnifiedConfig = obj.Def.EnvConfigMode.IsUnifiedConfig
	}
	if obj.BaseAppConfigFileID != nil {
		o.BaseAppConfigFileID = obj.BaseAppConfigFileID.Hex()
	}
	return o
}

// FromDef 仅从 def 填充输出字段（不依赖 file 记录）。
func (o *AppConfigFileDefOutputObj) FromDef(def appcfg.AppConfigFileDef) *AppConfigFileDefOutputObj {
	*o = AppConfigFileDefOutputObj{
		ID:              def.ID.Hex(),
		Name:            def.Name,
		MountDir:        def.MountDir,
		ConfigKind:      string(def.ConfigKind),
		IsUnifiedConfig: def.EnvConfigMode.IsUnifiedConfig,
	}
	return o
}

// AppConfigFileDefUpdateOutput 更新 def 后的响应。
type AppConfigFileDefUpdateOutput struct {
	Item *AppConfigFileDefOutputObj `json:"item"`
}
