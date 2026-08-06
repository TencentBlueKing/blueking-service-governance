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

// Package serializer defines Gin input and output serializers for env APIs.
package serializer

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// envNamePattern 匹配以小写字母开头，且只包含小写字母、数字、连字符的字符串
var envNamePattern = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("env_name", validateEnvName); err != nil {
			panic("failed to register env_name validator: " + err.Error())
		}
		if err := v.RegisterValidation("env_type", validateEnvType); err != nil {
			panic("failed to register env_type validator: " + err.Error())
		}
	}
}

func validateEnvName(fl validator.FieldLevel) bool {
	return envNamePattern.MatchString(fl.Field().String())
}

func validateEnvType(fl validator.FieldLevel) bool {
	return bkmsenv.IsValidEnvType(fl.Field().String())
}

// -----------------------------------------------------------------------------
// Path inputs
// -----------------------------------------------------------------------------

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// EnvURIInput is the path input for APIs scoped by environment.
type EnvURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,mongodb"`
}

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// WorkspaceEnvNameURIInput is the path input for APIs scoped by workspace and env name.
type WorkspaceEnvNameURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// -----------------------------------------------------------------------------
// Create env API serializers
// -----------------------------------------------------------------------------

// CreateEnvClusterInput is the JSON body for cluster info when creating an env.
type CreateEnvClusterInput struct {
	// 集群 ID
	ClusterID string `json:"clusterID" binding:"required,min=1"`
	// 集群类型
	ClusterType string `json:"clusterType" binding:"required,min=1"`
	// 集群命名空间
	Namespace string `json:"namespace" binding:"required,min=1"`
}

// CreateEnvInput is the JSON body for creating an environment.
type CreateEnvInput struct {
	// 环境名称
	Name string `json:"name" binding:"required,min=1,max=20,env_name"`
	// 环境显示名称
	DisplayName string `json:"displayName" binding:"required,min=1"`
	// 环境类型, 可选值 development、test、staging 或 production
	Type string `json:"type" binding:"required,env_type"`
	// 环境描述
	Description string `json:"description"`
	// 环境关联的业务集群信息
	Cluster CreateEnvClusterInput `json:"cluster" binding:"required"`
	// 绑定的 APM ID（可选，为空则创建同名 APM，不为空则使用共享 APM）
	ApmID *int64 `json:"apmID"`
}

// EnvIDOutput is the JSON response for creating an environment.
type EnvIDOutput struct {
	// 环境 ID
	ID string `json:"id"`
}

// CreateEnvOutput is the JSON response for creating an environment.
type CreateEnvOutput struct {
	Data *EnvIDOutput `json:"data"`
}

// CreateFeatureEnvInput is the JSON body for creating a feature environment.
type CreateFeatureEnvInput struct {
	// 特性环境展示名称
	DisplayName string `json:"displayName" binding:"required"`
	// 来源标准环境 ID
	SourceEnvID string `json:"sourceEnvID" binding:"required,mongodb"`
}

// CreateFeatureEnvOutput is the JSON response for creating a feature environment.
type CreateFeatureEnvOutput struct {
	Data *EnvOutput `json:"data"`
}

// -----------------------------------------------------------------------------
// List envs API serializers
// -----------------------------------------------------------------------------

// ListFeatureEnvsQueryInput is the query input for listing feature environments.
type ListFeatureEnvsQueryInput struct {
	// 是否附带当前应用在每个特性环境下的部署状态；默认不返回
	WithDeployStatus bool `form:"with_deploy_status"`
}

// EnvClusterOutput is the JSON representation of an env cluster.
type EnvClusterOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 集群类型
	ClusterType string `json:"clusterType"`
	// 集群命名空间
	Namespace string `json:"namespace"`
	// 项目 code
	ProjectCode string `json:"projectCode"`
}

// EnvOutput is the JSON representation of an environment.
//
// [bkms-cli 使用] 避免破坏性修改
type EnvOutput struct {
	// 环境 ID
	ID string `json:"id"`
	// 环境名称
	Name string `json:"name"`
	// 环境显示名称
	DisplayName string `json:"displayName"`
	// 环境类型
	Type string `json:"type"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 业务集群信息
	Cluster *EnvClusterOutput `json:"cluster"`
	// 环境状态, 取值: Ready(就绪), NotReady(未就绪)
	Status string `json:"status"`
	// 已部署的应用 ID 列表
	AppIDs []string `json:"appIDs"`
	// 环境类别，standard 或 feature
	Kind string `json:"kind"`
	// 特性环境所属应用 ID，仅特性环境返回
	OwnerAppID string `json:"ownerAppID,omitempty"`
	// 特性环境来源环境 ID，仅特性环境返回
	SourceEnvID string `json:"sourceEnvID,omitempty"`
}

// FromModel fills output fields from an environment model.
func (o *EnvOutput) FromModel(env envmodel.Environment) *EnvOutput {
	sourceEnvID := lo.Ternary(env.SourceEnvID.IsZero(), "", env.SourceEnvID.Hex())

	*o = EnvOutput{
		ID:          env.ID.Hex(),
		Name:        env.Name,
		DisplayName: env.DisplayName,
		Type:        env.Type,
		CreatedAt:   env.CreatedAt,
		UpdatedAt:   env.UpdatedAt,
		Cluster: &EnvClusterOutput{
			ClusterID:   env.Cluster.ClusterID,
			ClusterType: env.Cluster.ClusterType,
			Namespace:   env.Cluster.Namespace,
			ProjectCode: env.Cluster.ProjectCode,
		},
		Status: string(env.Status),
		AppIDs: env.AppIDs,

		// 实现特性环境后新增字段
		Kind:        string(env.GetKind()),
		OwnerAppID:  env.OwnerAppID,
		SourceEnvID: sourceEnvID,
	}
	return o
}

// ListEnvsOutput is the JSON response for listing environments.
//
// [bkms-cli 使用] 避免破坏性修改
type ListEnvsOutput struct {
	Data []*EnvOutput `json:"data"`
}

// FeatureEnvSourceOutput is the JSON representation of a feature env's source environment.
//
// The source environment may be deleted independently. In that case, ID remains
// available for audit purposes while IsDeleted tells clients to render a clear fallback.
type FeatureEnvSourceOutput struct {
	// 来源环境 ID
	ID string `json:"id"`
	// 来源环境名称，来源环境已删除时为空
	Name string `json:"name,omitempty"`
	// 来源环境展示名称，来源环境已删除时为空
	DisplayName string `json:"displayName,omitempty"`
	// 来源环境是否已删除
	IsDeleted bool `json:"isDeleted"`
}

// FeatureEnvClusterOutput contains cluster fields needed by the feature env management list.
type FeatureEnvClusterOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 特性环境独占的命名空间
	Namespace string `json:"namespace"`
}

// FeatureEnvOutput is the JSON representation used by the feature env management list.
//
// This intentionally differs from EnvOutput: ownership is already established by
// the app-scoped route. the response includes creator details and a display-ready
// source environment.
type FeatureEnvOutput struct {
	// 特性环境 ID
	ID string `json:"id"`
	// 特性环境内部名称
	Name string `json:"name"`
	// 特性环境展示名称
	DisplayName string `json:"displayName"`
	// 环境类型，可选值 development、test、staging 或 production
	Type string `json:"type"`
	// 来源标准环境
	SourceEnv *FeatureEnvSourceOutput `json:"sourceEnv"`
	// 部署位置
	Cluster *FeatureEnvClusterOutput `json:"cluster"`
	// 环境状态，取值 Ready（就绪）或 NotReady（未就绪）
	Status string `json:"status"`
	// 当前应用在该特性环境下各泳道的部署状态；未显式请求时为 null，显式请求时为空数组或状态列表
	DeployStatuses []*FeatureEnvDeployStatusOutput `json:"deployStatuses"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// FeatureEnvDeployStatusOutput is the JSON representation of current app deploy status in a feature env lane.
type FeatureEnvDeployStatusOutput struct {
	// 泳道名称
	TrafficLaneName string `json:"trafficLaneName"`
	// 部署状态
	DeployStatus string `json:"deployStatus"`
	// 部署的镜像 Tag
	ImageTag string `json:"imageTag"`
}

// FromModel fills output fields from a feature environment and its optional source environment.
// A nil deployStatuses means the caller did not request deployment statuses and leaves the JSON field null;
// a non-nil slice means the caller requested them and produces an array, including [] when it is empty.
func (o *FeatureEnvOutput) FromModel(
	env envmodel.Environment,
	sourceEnv *envmodel.Environment,
	deployStatuses []status.AppDeployStatus,
) *FeatureEnvOutput {
	source := &FeatureEnvSourceOutput{IsDeleted: sourceEnv == nil}
	if !env.SourceEnvID.IsZero() {
		source.ID = env.SourceEnvID.Hex()
	}
	if sourceEnv != nil {
		source.Name = sourceEnv.Name
		source.DisplayName = sourceEnv.DisplayName
	}

	*o = FeatureEnvOutput{
		ID:          env.ID.Hex(),
		Name:        env.Name,
		DisplayName: env.DisplayName,
		Type:        env.Type,
		SourceEnv:   source,
		Cluster: &FeatureEnvClusterOutput{
			ClusterID: env.Cluster.ClusterID,
			Namespace: env.Cluster.Namespace,
		},
		Status:    string(env.Status),
		Creator:   env.Creator,
		CreatedAt: env.CreatedAt,
	}
	if deployStatuses != nil {
		rows := make([]*FeatureEnvDeployStatusOutput, 0, len(deployStatuses))
		for _, row := range deployStatuses {
			rows = append(rows, &FeatureEnvDeployStatusOutput{
				TrafficLaneName: row.TrafficLaneName,
				DeployStatus:    row.DeployStatus,
				ImageTag:        row.ImageTag,
			})
		}
		o.DeployStatuses = rows
	}
	return o
}

// ListFeatureEnvsOutput is the JSON response for listing an app's feature environments.
type ListFeatureEnvsOutput struct {
	Data []*FeatureEnvOutput `json:"data"`
}

// NewListFeatureEnvsOutput builds the list output from feature environments and
// source environments keyed by feature env name. A nil deployStatusByFeatEnvName
// means deployment statuses were not requested; a non-nil map means they were
// requested, so missing or nil values are represented as empty arrays.
func NewListFeatureEnvsOutput(
	featureEnvs []envmodel.Environment,
	sourceEnvByFeatEnvName map[string]*envmodel.Environment,
	deployStatusByFeatEnvName map[string][]status.AppDeployStatus,
) *ListFeatureEnvsOutput {
	data := make([]*FeatureEnvOutput, 0, len(featureEnvs))
	for _, env := range featureEnvs {
		var deployStatuses []status.AppDeployStatus
		if deployStatusByFeatEnvName != nil {
			deployStatuses = lo.CoalesceSliceOrEmpty(deployStatusByFeatEnvName[env.Name])
		}
		data = append(data, new(FeatureEnvOutput).FromModel(
			env,
			sourceEnvByFeatEnvName[env.Name],
			deployStatuses,
		))
	}

	return &ListFeatureEnvsOutput{Data: data}
}

// -----------------------------------------------------------------------------
// Get env API serializers
// -----------------------------------------------------------------------------

// EnvAppDeployStatusOutput is the JSON representation of app deploy status in an env.
type EnvAppDeployStatusOutput struct {
	// 应用 ID
	AppID string `json:"appID"`
	// 应用名称
	AppName string `json:"appName"`
	// 应用类型
	AppType string `json:"appType"`
	// 泳道名称
	TrafficLaneName string `json:"trafficLaneName"`
	// 部署状态
	DeployStatus string `json:"deployStatus"`
	// 部署的镜像 Tag
	ImageTag string `json:"imageTag"`
}

// EnvDetailOutput is the JSON representation of environment details.
type EnvDetailOutput struct {
	// 环境 ID
	ID string `json:"id"`
	// 环境名称
	Name string `json:"name"`
	// 环境显示名称
	DisplayName string `json:"displayName"`
	// 环境类型
	Type string `json:"type"`
	// 创建者
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 业务集群信息
	Cluster *EnvClusterOutput `json:"cluster"`
	// 环境状态, 取值: Ready(就绪), NotReady(未就绪)
	Status string `json:"status"`
	// 环境描述
	Description string `json:"description"`
	// 当前环境已部署应用及其部署状态
	AppDeployStatuses []*EnvAppDeployStatusOutput `json:"appDeployStatuses"`
}

// FromModel fills output fields from an environment model and its deploy statuses.
func (o *EnvDetailOutput) FromModel(
	env envmodel.Environment,
	deployStatuses []status.AppDeployStatus,
) *EnvDetailOutput {
	*o = EnvDetailOutput{
		ID:          env.ID.Hex(),
		Name:        env.Name,
		DisplayName: env.DisplayName,
		Type:        env.Type,
		Creator:     env.Creator,
		CreatedAt:   env.CreatedAt,
		UpdatedAt:   env.UpdatedAt,
		Cluster: &EnvClusterOutput{
			ClusterID:   env.Cluster.ClusterID,
			ClusterType: env.Cluster.ClusterType,
			Namespace:   env.Cluster.Namespace,
			ProjectCode: env.Cluster.ProjectCode,
		},
		Status:      string(env.Status),
		Description: env.Description,
		AppDeployStatuses: lo.Map(
			deployStatuses,
			func(row status.AppDeployStatus, _ int) *EnvAppDeployStatusOutput {
				return &EnvAppDeployStatusOutput{
					AppID:           row.AppID,
					AppName:         row.AppName,
					AppType:         row.AppType,
					TrafficLaneName: row.TrafficLaneName,
					DeployStatus:    row.DeployStatus,
					ImageTag:        row.ImageTag,
				}
			},
		),
	}
	return o
}

// GetEnvOutput is the JSON response for getting an environment.
type GetEnvOutput struct {
	Data *EnvDetailOutput `json:"data"`
}

// -----------------------------------------------------------------------------
// Update env API serializers
// -----------------------------------------------------------------------------

// UpdateEnvBasicInfoInput is the JSON body for updating env basic info.
type UpdateEnvBasicInfoInput struct {
	// 环境显示名称
	DisplayName *string `json:"displayName"`
	// 环境类型, 可选值 development、test、staging 或 production
	Type *string `json:"type" binding:"omitempty,env_type"`
}

// UpdateEnvClusterInput is the JSON body for updating env cluster info.
type UpdateEnvClusterInput struct {
	// 集群 ID
	ClusterID string `json:"clusterID" binding:"required,min=1"`
	// 集群类型
	ClusterType string `json:"clusterType" binding:"required,min=1"`
	// 集群命名空间
	Namespace string `json:"namespace" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// Empty output
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// -----------------------------------------------------------------------------
// List env traffic lanes API serializers
// -----------------------------------------------------------------------------

// TrafficLaneOutput is the JSON representation of a traffic lane.
type TrafficLaneOutput struct {
	// 泳道 ID
	ID string `json:"id"`
	// 泳道名称
	Name string `json:"name"`
	// 泳道描述
	Description string `json:"description"`
	// 泳道类型
	Type string `json:"type"`
	// 标签中包含泳道所属的微服务信息，类似于泳道组的概念
	Labels map[string]string `json:"labels"`
	// 注解是泳道扩展字段, 针对不同产品通过不同的 k-v 扩展
	Annotations map[string]string `json:"annotations"`
	// 泳道服务版本标签, 平台注入或者用户自定义
	ServiceVersionLabels map[string]string `json:"serviceVersionLabels"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a traffic lane model.
func (o *TrafficLaneOutput) FromModel(l *trafficmanager.TrafficLane) *TrafficLaneOutput {
	*o = TrafficLaneOutput{
		ID:                   l.LaneId,
		Name:                 l.LaneName,
		Description:          l.LaneDesc,
		Type:                 l.LaneType,
		Labels:               l.LaneLabels,
		Annotations:          l.LaneAnnotations,
		ServiceVersionLabels: l.LaneServiceVersionLabels,
	}
	return o
}

// ListEnvTrafficLanesOutput is the JSON response for listing env traffic lanes.
type ListEnvTrafficLanesOutput struct {
	Data []*TrafficLaneOutput `json:"data"`
}
