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

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot/tagdeletion"
)

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
}

// AppEnvURIInput is the path input for APIs scoped by application and environment.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,min=1"`
}

// AppImageTagURIInput is the path input for APIs scoped by application and image tag.
type AppImageTagURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
	// 镜像标签
	Tag string `uri:"tag" binding:"required,min=1,max=128"`
}

// ListAppImagesQueryInput is the query input for listing app images.
type ListAppImagesQueryInput struct {
	// 搜索关键字
	Keyword string `form:"keyword"`
	// 分页参数
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// ListImageTagDeployRecordsQueryInput is the query input for listing image tag deploy records.
type ListImageTagDeployRecordsQueryInput struct {
	// 分页参数
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// ListDeployableImageTagsQueryInput is the query input for listing deployable image tags.
type ListDeployableImageTagsQueryInput struct {
	// 搜索关键字（可选，按 TAG 名称模糊搜索）
	Keyword string `form:"keyword"`
	// 分页参数
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// RuntimeImageURIInput is the path input for APIs scoped by runtime image.
type RuntimeImageURIInput struct {
	// 运行时镜像记录 ID
	ImageID string `uri:"imageID" binding:"required"`
}

// ListRuntimeImagesQueryInput is the query input for listing runtime images.
type ListRuntimeImagesQueryInput struct {
	// 镜像类型
	Type string `form:"type" binding:"required,oneof=builder runner"`
	// 搜索关键字（可选，按名称或描述模糊搜索）
	Keyword string `form:"keyword"`
}

// ListRuntimeImageTagsQueryInput is the query input for listing tags of one runtime image.
type ListRuntimeImageTagsQueryInput struct {
	// 搜索关键字（可选，按 TAG 名称模糊搜索）
	Keyword string `form:"keyword"`
	// 分页参数：页码
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数：每页数量
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// ImageEmptyOutput is the JSON response for APIs that return no data.
type ImageEmptyOutput struct{}

// DeployedEnvInfoOutputObj is the JSON representation of one deployed environment.
type DeployedEnvInfoOutputObj struct {
	// 环境名称
	EnvName string `json:"envName"`
	// 环境类型（development/test/staging/production）
	EnvType string `json:"envType"`
}

// SnapshotStatusInfoOutputObj is the JSON representation of snapshot status.
type SnapshotStatusInfoOutputObj struct {
	// 仓库实例唯一标识，便于排查问题
	RepoKey string `json:"repoKey"`
	// 当前刷新状态：idle / refreshing / detail_syncing
	RefreshStatus string `json:"refreshStatus"`
	// 最后成功刷新时间
	LastRefreshedAt *time.Time `json:"lastRefreshedAt,omitempty"`
	// 最后成功详情同步时间
	LastDetailSyncedAt *time.Time `json:"lastDetailSyncedAt,omitempty"`
	// 最后失败信息（为空表示无失败）
	LastError string `json:"lastError"`
}

// FromModel fills output fields from snapshot status model.
func (o *SnapshotStatusInfoOutputObj) FromModel(status *snapshot.RepoSnapshotStatus) *SnapshotStatusInfoOutputObj {
	if status == nil {
		*o = SnapshotStatusInfoOutputObj{RefreshStatus: string(snapshot.RefreshStatusIdle)}
		return o
	}

	*o = SnapshotStatusInfoOutputObj{
		RepoKey:            status.RepoKey,
		RefreshStatus:      string(status.RefreshStatus),
		LastRefreshedAt:    status.LastRefreshedAt,
		LastDetailSyncedAt: status.LastDetailSyncedAt,
		LastError:          status.LastError,
	}
	return o
}

// AppImageOutputObj is the JSON representation of one app image.
type AppImageOutputObj struct {
	// 镜像仓库
	Repository string `json:"repository"`
	// 镜像 TAG
	Tag string `json:"tag"`
	// 摘要
	Digest string `json:"digest"`
	// 镜像大小
	Size int64 `json:"size,string"`
	// 镜像构建时间
	BuiltAt *time.Time `json:"builtAt,omitempty"`
	// 是否已晋级
	IsPromoted bool `json:"isPromoted"`
	// 晋级时间
	PromotedAt *time.Time `json:"promotedAt,omitempty"`
	// 晋级操作人
	PromotedBy string `json:"promotedBy"`
	// 已部署环境列表
	DeployedEnvs []*DeployedEnvInfoOutputObj `json:"deployedEnvs"`
}

// FromModel fills output fields from image snapshot and related model data.
func (o *AppImageOutputObj) FromModel(
	snap snapshot.Image,
	repository string,
	promo *promotion.Image,
	deployed []deploytypes.ImageTagEnvPair,
	envTypeMap map[string]string,
) *AppImageOutputObj {
	*o = AppImageOutputObj{
		Repository:   repository,
		Tag:          snap.Tag,
		Digest:       snap.Digest,
		Size:         snap.Size,
		BuiltAt:      snap.BuiltAt,
		DeployedEnvs: deployedEnvOutputs(deployed, envTypeMap),
	}
	if promo != nil {
		o.IsPromoted = true
		o.PromotedAt = &promo.PromotedAt
		o.PromotedBy = promo.PromotedBy
	}
	return o
}

// PaginatedAppImagesOutputObjs is the paginated app images payload.
type PaginatedAppImagesOutputObjs struct {
	Count int64 `json:"count,string"`
	// 当前页镜像结果
	Results []*AppImageOutputObj `json:"results"`
	// 快照状态信息
	SnapshotStatus *SnapshotStatusInfoOutputObj `json:"snapshotStatus"`
	// 当前工作空间中所有生产类型环境的名称列表
	ProductionEnvNames []string `json:"productionEnvNames"`
}

// ListAppImagesOutput is the JSON response for listing app images.
type ListAppImagesOutput struct {
	Data *PaginatedAppImagesOutputObjs `json:"data"`
}

// RuntimeImageOutputObj is the JSON representation of one runtime image.
type RuntimeImageOutputObj struct {
	// 记录 ID
	ID string `json:"id"`
	// 镜像类型
	Type string `json:"type"`
	// 镜像仓库名称，不包含 tag
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from runtime image model.
func (o *RuntimeImageOutputObj) FromModel(image workloadruntime.Image) *RuntimeImageOutputObj {
	*o = RuntimeImageOutputObj{
		ID:          image.ID,
		Type:        string(image.Type),
		Name:        image.Name,
		Description: image.Description,
		CreatedAt:   image.CreatedAt,
		UpdatedAt:   image.UpdatedAt,
	}
	return o
}

// RuntimeImagesOutputObjs is the runtime image list payload.
type RuntimeImagesOutputObjs struct {
	Results []*RuntimeImageOutputObj `json:"results"`
}

// ListRuntimeImagesOutput is the JSON response for listing runtime images.
type ListRuntimeImagesOutput struct {
	Data *RuntimeImagesOutputObjs `json:"data"`
}

// RuntimeImageTagOutputObj is the JSON representation of one available runtime image tag.
type RuntimeImageTagOutputObj struct {
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
func (o *RuntimeImageTagOutputObj) FromModel(snap snapshot.Image) *RuntimeImageTagOutputObj {
	return &RuntimeImageTagOutputObj{
		Tag:     snap.Tag,
		Digest:  snap.Digest,
		Size:    snap.Size,
		BuiltAt: snap.BuiltAt,
	}
}

// PaginatedRuntimeImageTagOutputObjs is the paginated runtime image tag payload.
type PaginatedRuntimeImageTagOutputObjs struct {
	// 满足条件的总记录数
	Count int64 `json:"count,string"`
	// 当前页的镜像 TAG 列表
	Results []*RuntimeImageTagOutputObj `json:"results"`
	// 快照状态信息
	SnapshotStatus *SnapshotStatusInfoOutputObj `json:"snapshotStatus"`
}

// ListRuntimeImageTagsOutput is the JSON response for listing runtime image tags.
type ListRuntimeImageTagsOutput struct {
	Data *PaginatedRuntimeImageTagOutputObjs `json:"data"`
}

// RefreshResultInfoOutputObj is the JSON representation of a refresh result.
type RefreshResultInfoOutputObj struct {
	// 刷新状态：success / refreshing（已有刷新在进行中）/ failed
	Status string `json:"status"`
	// 提示信息
	Message string `json:"message"`
	// 本次新增标签数量
	AddedTagCnt int64 `json:"addedTagCnt,string"`
	// 本次删除标签数量
	RemovedTagCnt int64 `json:"removedTagCnt,string"`
}

// FromModel fills output fields from refresh result model.
func (o *RefreshResultInfoOutputObj) FromModel(result *snapshot.RefreshResult) *RefreshResultInfoOutputObj {
	if result == nil {
		return nil
	}
	*o = RefreshResultInfoOutputObj{
		Status:        result.Status,
		Message:       result.Message,
		AddedTagCnt:   result.AddedTagCnt,
		RemovedTagCnt: result.RemovedTagCnt,
	}
	return o
}

// RefreshAppImagesOutput is the JSON response for refreshing app images.
type RefreshAppImagesOutput struct {
	Data *RefreshResultInfoOutputObj `json:"data"`
}

// ImageTagUsageOutputObj is one current usage hit for the target image tag.
type ImageTagUsageOutputObj struct {
	// 环境名称
	EnvName string `json:"envName"`
	// 泳道名称，基线泳道为空字符串
	LaneName string `json:"laneName"`
	// 当前命中的部署记录对应的 workload 名称
	WorkloadName string `json:"workloadName"`
	// 当前命中的部署记录原始状态值
	Status string `json:"status"`
}

// FromModel fills output fields from one usage model.
func (o *ImageTagUsageOutputObj) FromModel(usage tagdeletion.ImageUsage) *ImageTagUsageOutputObj {
	*o = ImageTagUsageOutputObj{
		EnvName:      usage.EnvName,
		LaneName:     usage.LaneName,
		WorkloadName: usage.WorkloadName,
		Status:       usage.Status,
	}
	return o
}

// ImageTagUsagesOutputObj is the JSON representation of an image-usage result.
type ImageTagUsagesOutputObj struct {
	// 当前镜像 tag 是否仍可能被使用
	InUse bool `json:"inUse"`
	// 命中的 env/lane/workload 列表
	Usages []*ImageTagUsageOutputObj `json:"usages"`
}

// FromModels fills output fields from usage models.
func (o *ImageTagUsagesOutputObj) FromModels(usages []tagdeletion.ImageUsage) *ImageTagUsagesOutputObj {
	results := make([]*ImageTagUsageOutputObj, 0, len(usages))
	for _, usage := range usages {
		results = append(results, new(ImageTagUsageOutputObj).FromModel(usage))
	}
	*o = ImageTagUsagesOutputObj{
		InUse:  len(results) > 0,
		Usages: results,
	}
	return o
}

// ListAppImageUsagesOutput is the JSON response for listing image usages.
type ListAppImageUsagesOutput struct {
	Data *ImageTagUsagesOutputObj `json:"data"`
}

// ImageTagDeployRecordOutputObj is the JSON representation of one image-tag deploy record.
type ImageTagDeployRecordOutputObj struct {
	// 部署人
	Operator string `json:"operator"`
	// 记录创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 环境名称
	EnvName string `json:"envName"`
	// 部署状态
	Status string `json:"status"`
}

// FromAppModelRecord fills output fields from one app-model deploy record.
func (o *ImageTagDeployRecordOutputObj) FromAppModelRecord(
	record appmodeldeploy.Record,
) *ImageTagDeployRecordOutputObj {
	*o = ImageTagDeployRecordOutputObj{
		Operator:  record.Creator,
		CreatedAt: record.CreatedAt,
		EnvName:   record.EnvName,
		Status:    string(record.Status),
	}
	return o
}

// FromHelmRecord fills output fields from one helm deploy record.
func (o *ImageTagDeployRecordOutputObj) FromHelmRecord(record helmdeploy.Record) *ImageTagDeployRecordOutputObj {
	*o = ImageTagDeployRecordOutputObj{
		Operator:  record.Operator,
		CreatedAt: record.CreatedAt,
		EnvName:   record.EnvName,
		Status:    string(record.Status),
	}
	return o
}

// PaginatedImageTagDeployRecordOutputObjs is the paginated deploy-record payload.
type PaginatedImageTagDeployRecordOutputObjs struct {
	Count int64 `json:"count,string"`
	// 当前页部署记录结果
	Results []*ImageTagDeployRecordOutputObj `json:"results"`
}

// ListImageTagDeployRecordsOutput is the JSON response for listing image-tag deploy records.
type ListImageTagDeployRecordsOutput struct {
	Data *PaginatedImageTagDeployRecordOutputObjs `json:"data"`
}

// DeployableImageTagOutputObj is the JSON representation of one deployable image tag.
type DeployableImageTagOutputObj struct {
	// 镜像标签名
	Tag string `json:"tag"`
	// 镜像构建时间
	BuiltAt *time.Time `json:"builtAt,omitempty"`
}

// FromModel fills output fields from snapshot model.
func (o *DeployableImageTagOutputObj) FromModel(snap snapshot.Image) *DeployableImageTagOutputObj {
	*o = DeployableImageTagOutputObj{
		Tag:     snap.Tag,
		BuiltAt: snap.BuiltAt,
	}
	return o
}

// PaginatedDeployableImageTagOutputObjs is the paginated deployable-tag payload.
type PaginatedDeployableImageTagOutputObjs struct {
	// 满足条件的总记录数
	Count int64 `json:"count,string"`
	// 当前页的镜像 TAG 列表
	Results []*DeployableImageTagOutputObj `json:"results"`
}

// ListDeployableImageTagsOutput is the JSON response for listing deployable image tags.
type ListDeployableImageTagsOutput struct {
	Data *PaginatedDeployableImageTagOutputObjs `json:"data"`
}

func deployedEnvOutputs(
	pairs []deploytypes.ImageTagEnvPair,
	envTypeMap map[string]string,
) []*DeployedEnvInfoOutputObj {
	results := make([]*DeployedEnvInfoOutputObj, 0, len(pairs))
	for _, pair := range pairs {
		envType, exists := envTypeMap[pair.EnvName]
		if !exists {
			continue
		}
		results = append(results, &DeployedEnvInfoOutputObj{
			EnvName: pair.EnvName,
			EnvType: envType,
		})
	}
	return results
}
