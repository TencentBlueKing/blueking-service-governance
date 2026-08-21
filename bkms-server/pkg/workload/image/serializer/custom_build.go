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

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("custom_image_repo", validateCustomImageRepo); err != nil {
			panic("failed to register custom_image_repo validator: " + err.Error())
		}
	}
}

// validateCustomImageRepo 与 customruntime.ValidateRepositoryName 共用同一套仓库名前缀口径
func validateCustomImageRepo(fl validator.FieldLevel) bool {
	return customruntime.ValidateRepositoryName(fl.Field().String()) == nil
}

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// ListCustomRuntimeImagesQueryInput is the query input for listing custom runtime images.
type ListCustomRuntimeImagesQueryInput struct {
	// 镜像类型
	Type string `form:"type" binding:"required,oneof=builder runner"`
	// 搜索关键字（可选，按名称模糊搜索）
	Keyword string `form:"keyword"`
}

// ListCustomRuntimeImageTagsQueryInput is the query input for listing tags of one custom runtime image.
//
// 镜像以完整名称而非记录 ID 传入：用户手动输入、尚未落库的镜像没有记录 ID，只有用
// 名称才能让「已落库走快照」与「手动输入走实时拉取」两条来源共用同一套出入参。
type ListCustomRuntimeImageTagsQueryInput struct {
	// 镜像完整仓库名称，含仓库前缀且不包含 tag 或 digest
	Name string `form:"name" binding:"required,custom_image_repo"`
	// 搜索关键字（可选，按 TAG 名称模糊搜索）
	Keyword string `form:"keyword"`
	// 分页参数：页码
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数：每页数量
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// RefreshCustomRuntimeImageTagsInput is the body input for refreshing tags of one custom runtime image.
type RefreshCustomRuntimeImageTagsInput struct {
	// 镜像完整仓库名称，含仓库前缀且不包含 tag 或 digest
	Name string `json:"name" binding:"required,custom_image_repo"`
}

// CustomRuntimeImageOutputObj is the JSON representation of one custom runtime image.
type CustomRuntimeImageOutputObj struct {
	// 记录 ID
	ID string `json:"id"`
	// 镜像类型
	Type string `json:"type"`
	// 镜像仓库名称，含仓库前缀，不包含 tag
	Name string `json:"name"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from custom runtime image model.
func (o *CustomRuntimeImageOutputObj) FromModel(image customruntime.Image) *CustomRuntimeImageOutputObj {
	// Type 以字符串输出，与平台通用构建镜像列表的 type 字段口径一致
	*o = CustomRuntimeImageOutputObj{
		ID:        image.ID,
		Type:      string(image.Type),
		Name:      image.Name,
		CreatedAt: image.CreatedAt,
		UpdatedAt: image.UpdatedAt,
	}
	return o
}

// CustomRuntimeImagesOutputObjs is the custom runtime image list payload.
//
// 候选数量预期在百条以内，与平台通用构建镜像列表一致不做分页，因此没有 count 字段。
type CustomRuntimeImagesOutputObjs struct {
	Results []*CustomRuntimeImageOutputObj `json:"results"`
}

// ListCustomRuntimeImagesOutput is the JSON response for listing custom runtime images.
type ListCustomRuntimeImagesOutput struct {
	Data *CustomRuntimeImagesOutputObjs `json:"data"`
}

// CustomRuntimeImageTagOutputObj is the JSON representation of one available custom runtime image tag.
type CustomRuntimeImageTagOutputObj struct {
	// 镜像标签名
	Tag string `json:"tag"`
	// 摘要
	Digest string `json:"digest"`
	// 镜像大小
	Size int64 `json:"size,string"`
	// 镜像构建时间
	BuiltAt *time.Time `json:"builtAt,omitempty"`
}

// FromModel fills output fields from snapshot model.
func (o *CustomRuntimeImageTagOutputObj) FromModel(snap snapshot.Image) *CustomRuntimeImageTagOutputObj {
	// 快照与实时拉取共用此出参；BuiltAt 在详情尚未补全时为 nil，由 omitempty 省略
	*o = CustomRuntimeImageTagOutputObj{
		Tag:     snap.Tag,
		Digest:  snap.Digest,
		Size:    snap.Size,
		BuiltAt: snap.BuiltAt,
	}
	return o
}

// PaginatedCustomRuntimeImageTagOutputObjs is the paginated custom runtime image tag payload.
//
// 无论 TAG 来自本地快照还是实时远程拉取，出参结构、分页语义与总数口径完全一致，
// 调用方无需按来源分支处理，也不需要传递来源标识。
type PaginatedCustomRuntimeImageTagOutputObjs struct {
	// 满足条件的总记录数
	Count int64 `json:"count,string"`
	// 当前页的镜像 TAG 列表
	Results []*CustomRuntimeImageTagOutputObj `json:"results"`
	// 快照状态信息，手动输入且尚无快照记录时 refreshStatus 为 idle
	SnapshotStatus *SnapshotStatusInfoOutputObj `json:"snapshotStatus"`
}

// ListCustomRuntimeImageTagsOutput is the JSON response for listing custom runtime image tags.
type ListCustomRuntimeImageTagsOutput struct {
	Data *PaginatedCustomRuntimeImageTagOutputObjs `json:"data"`
}

// RefreshCustomRuntimeImageTagsOutput is the JSON response for refreshing custom runtime image tags.
type RefreshCustomRuntimeImageTagsOutput struct {
	Data *RefreshResultInfoOutputObj `json:"data"`
}
