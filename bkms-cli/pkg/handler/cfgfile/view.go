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

package cfgfile

import (
	"context"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ViewResult 是配置文件在指定环境下的查看结果。
type ViewResult struct {
	target *fileTarget
}

// ViewOutput 是配置文件查看命令的结构化输出。
type ViewOutput struct {
	Name                 string  `json:"name" yaml:"name"`
	ConfigKind           string  `json:"configKind" yaml:"configKind"`
	MountDir             string  `json:"mountDir,omitempty" yaml:"mountDir,omitempty"`
	MountEnvs            string  `json:"mountEnvs" yaml:"mountEnvs"`
	EnvMode              string  `json:"envMode" yaml:"envMode"`
	EnvContentStrategy   string  `json:"envContentStrategy,omitempty" yaml:"envContentStrategy,omitempty"`
	EnableEnvVarRender   bool    `json:"enableEnvVarRender" yaml:"enableEnvVarRender"`
	EnvName              string  `json:"envName" yaml:"envName"`
	SharesDefault        bool    `json:"sharesDefault" yaml:"sharesDefault"`
	FileType             string  `json:"fileType" yaml:"fileType"`
	FileFormat           string  `json:"fileFormat" yaml:"fileFormat"`
	ContentSourceType    string  `json:"contentSourceType" yaml:"contentSourceType"`
	EditableContentField string  `json:"editableContentField" yaml:"editableContentField"`
	CurrentVersion       int64   `json:"currentVersion" yaml:"currentVersion"`
	Updater              string  `json:"updater" yaml:"updater"`
	UpdatedAt            string  `json:"updatedAt" yaml:"updatedAt"`
	ID                   string  `json:"id" yaml:"id"`
	FileID               string  `json:"fileId" yaml:"fileId"`
	BaseContent          *string `json:"baseContent,omitempty" yaml:"baseContent,omitempty" table:"-"`
	Content              *string `json:"content,omitempty" yaml:"content,omitempty" table:"-"`
	OverlayContent       *string `json:"overlayContent,omitempty" yaml:"overlayContent,omitempty" table:"-"`
}

// View 查看配置文件在指定环境下实际生效的内容，envName 为空时查看默认文件。
func View(ctx context.Context, cli client.Client, appID string, ref FileRef, envName string) (*ViewResult, error) {
	target, err := loadTarget(ctx, cli, appID, ref, envName)
	if err != nil {
		return nil, err
	}
	return &ViewResult{target: target}, nil
}

// SharesDefault 表示指定了环境，但该环境没有独立实例，展示的是默认文件。
func (r *ViewResult) SharesDefault() bool {
	return r.target.SharesDefault()
}

// IsUnifiedConfig 表示文件为统一配置，所有环境共享默认内容。
func (r *ViewResult) IsUnifiedConfig() bool {
	return r.target.Def.IsUnifiedConfig
}

// EditableContent 返回可编辑字段的内容，与 edit 命令写入的字段一致，便于「查看 → 修改 → 写回」。
func (r *ViewResult) EditableContent() string {
	detail := r.target.Detail
	if detail.EditableContentField == editableContentFieldOverlayContent {
		return lo.FromPtr(detail.OverlayContent)
	}
	return lo.FromPtr(detail.Content)
}

// Output 返回用于格式化输出的结构化数据。
func (r *ViewResult) Output() *ViewOutput {
	def, detail := r.target.Def, r.target.Detail
	out := &ViewOutput{
		Name:                 def.Name,
		ConfigKind:           def.ConfigKind,
		MountDir:             def.MountDir,
		MountEnvs:            formatMountEnvs(def),
		EnvMode:              formatEnvMode(def),
		EnvContentStrategy:   def.EnvContentStrategy,
		EnableEnvVarRender:   def.EnableEnvVarRender,
		EnvName:              formatEnvName(r.target.EnvName),
		SharesDefault:        r.target.SharesDefault(),
		FileType:             detail.FileType,
		FileFormat:           detail.FileFormat,
		ContentSourceType:    detail.ContentSourceType,
		EditableContentField: detail.EditableContentField,
		CurrentVersion:       detail.CurrentVersion,
		Updater:              detail.Updater,
		UpdatedAt:            detail.UpdatedAt,
		ID:                   def.ID,
		FileID:               detail.FileID,
		Content:              detail.Content,
		OverlayContent:       detail.OverlayContent,
	}
	if detail.BaseContentInfo != nil {
		out.BaseContent = &detail.BaseContentInfo.Content
	}
	return out
}
