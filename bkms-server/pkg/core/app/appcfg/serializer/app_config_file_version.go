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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// ListAppConfigFileVersionsQueryInput is the query input for listing versions.
type ListAppConfigFileVersionsQueryInput struct {
	// 应用配置文件 ID
	AppConfigFileID *string `form:"appConfigFileID"`
	// 环境名
	EnvName *string `form:"envName"`
	// 文件名
	Name *string `form:"name"`
	// 版本号
	Version *int64 `form:"version"`
	// 创建人
	Creator *string `form:"creator"`
	// 版本描述
	Description *string `form:"description"`
	// 页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量，仅支持固定枚举值
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// CompareAppConfigFileVersionsInput is the JSON body for comparing two versions.
type CompareAppConfigFileVersionsInput struct {
	// 原来版本的 ID
	PreviousVersionID string `json:"previousVersionID" binding:"required,mongodb"`
	// 当前版本的 ID
	CurrentVersionID string `json:"currentVersionID" binding:"required,mongodb"`
}

// VersionIDs returns the compare request IDs as ObjectIDs.
func (i CompareAppConfigFileVersionsInput) VersionIDs() (bson.ObjectID, bson.ObjectID, error) {
	previousID, err := bson.ObjectIDFromHex(i.PreviousVersionID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, errors.Wrap(err, "parse previousVersionID")
	}
	currentID, err := bson.ObjectIDFromHex(i.CurrentVersionID)
	if err != nil {
		return bson.ObjectID{}, bson.ObjectID{}, errors.Wrap(err, "parse currentVersionID")
	}
	return previousID, currentID, nil
}

// RollbackAppConfigFileVersionInput is the JSON body for rolling back a version.
type RollbackAppConfigFileVersionInput struct {
	// 回滚版本描述，为空则由后端自动生成
	Description *string `json:"description,omitempty"`
	// 编辑开始时的当前版本号，用于乐观锁冲突检测
	CurrentVersion *int64 `json:"currentVersion,omitempty"`
}

// AppConfigFileVersionOutputObj is the JSON representation of an app config file version.
type AppConfigFileVersionOutputObj struct {
	// 版本记录 ID
	ID string `json:"id"`
	// 所属应用配置文件 ID
	AppConfigFileID string `json:"appConfigFileID"`
	// 应用 ID
	AppID string `json:"appID"`
	// 环境名
	EnvName string `json:"envName"`
	// 文件名
	Name string `json:"name"`
	// 版本号
	Version int64 `json:"version"`
	// 版本描述
	Description string `json:"description"`
	// 文件类型
	Type string `json:"type"`
	// 内容来源
	ContentSourceType string `json:"contentSourceType"`
	// 文件格式
	FileFormat string `json:"fileFormat"`
	// 普通内容
	Content *string `json:"content,omitempty"`
	// 覆盖内容
	OverlayContent *string `json:"overlayContent,omitempty"`
	// base 文件 ID
	BaseAppConfigFileID string `json:"baseAppConfigFileID,omitempty"`
	// base 文件版本号
	BaseVersion *int64 `json:"baseVersion,omitempty"`
	// BSCP 配置引用
	BscpConfig *BSCPAppConfigFileConfig `json:"bscpConfig,omitempty"`
	// 版本操作类型
	OperationType string `json:"operationType"`
	// 若为 rollback，表示回滚来源版本号
	RollbackFromVersion *int64 `json:"rollbackFromVersion,omitempty"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间，RFC3339 格式
	CreatedAt string `json:"createdAt"`
	// 是否软删除
	IsDeleted bool `json:"isDeleted"`
	// 删除人
	Deleter string `json:"deleter,omitempty"`
	// 删除时间，RFC3339 格式
	DeletedAt string `json:"deletedAt,omitempty"`
}

// FromModel fills output fields from an app config file version model.
func (o *AppConfigFileVersionOutputObj) FromModel(obj appcfg.AppConfigFileVersion) *AppConfigFileVersionOutputObj {
	*o = AppConfigFileVersionOutputObj{
		ID:                  obj.ID.Hex(),
		AppConfigFileID:     obj.AppConfigFileID.Hex(),
		AppID:               obj.AppID,
		EnvName:             obj.EnvName,
		Name:                obj.Name,
		Version:             obj.Version,
		Description:         obj.Description,
		Type:                string(obj.Type),
		ContentSourceType:   string(obj.ContentSourceType),
		FileFormat:          string(obj.Format),
		Content:             obj.Content,
		OverlayContent:      obj.OverlayContent,
		BaseVersion:         obj.BaseVersion,
		OperationType:       string(obj.OperationType),
		RollbackFromVersion: obj.RollbackFromVersion,
		Creator:             obj.Creator,
		CreatedAt:           obj.CreatedAt.Format(time.RFC3339),
		IsDeleted:           obj.IsDeleted,
		Deleter:             obj.Deleter,
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
	if obj.DeletedAt != nil {
		o.DeletedAt = obj.DeletedAt.Format(time.RFC3339)
	}
	return o
}

// PaginatedAppConfigFileVersionOutputObjs is the paginated version list payload.
type PaginatedAppConfigFileVersionOutputObjs struct {
	// 结果数量
	Count int64 `json:"count"`
	// 查询结果
	Results []*AppConfigFileVersionOutputObj `json:"results"`
}

// ListAppConfigFileVersionsOutput is the JSON response for listing versions.
type ListAppConfigFileVersionsOutput struct {
	Data *PaginatedAppConfigFileVersionOutputObjs `json:"data"`
}

// GetAppConfigFileVersionOutput is the JSON response for one version detail.
type GetAppConfigFileVersionOutput struct {
	Data *AppConfigFileVersionOutputObj `json:"data"`
}

// CompareAppConfigFileVersionsOutput is the JSON response for comparing versions.
type CompareAppConfigFileVersionsOutput struct {
	// 原来版本
	Previous *AppConfigFileVersionOutputObj `json:"previous"`
	// 当前版本
	Current *AppConfigFileVersionOutputObj `json:"current"`
}

// RollbackAppConfigFileVersionOutput is the JSON response for rolling back a version.
type RollbackAppConfigFileVersionOutput struct {
	Data *AppConfigFileOutputObj `json:"data"`
}
