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

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
)

// AppEnvURIInput contains app and env path parameters.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// EnvVarPreCheckOutput is the response body for a deployment env var pre-check.
type EnvVarPreCheckOutput struct {
	UndefinedVars []UndefinedEnvVarOutput `json:"undefinedVars"`
}

// UndefinedEnvVarOutput contains one referenced but undefined env var.
type UndefinedEnvVarOutput struct {
	Key     string                        `json:"key"`
	Sources []EnvVarReferenceSourceOutput `json:"sources"`
}

// EnvVarReferenceSourceOutput identifies one configuration source.
type EnvVarReferenceSourceOutput struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// FromModel converts a domain pre-check result to an API output.
func (o *EnvVarPreCheckOutput) FromModel(result *deploypkg.EnvVarPreCheckResult) *EnvVarPreCheckOutput {
	*o = EnvVarPreCheckOutput{
		UndefinedVars: lo.Map(
			result.UndefinedVars,
			func(item envvarrefs.UndefinedEnvVar, _ int) UndefinedEnvVarOutput {
				return UndefinedEnvVarOutput{
					Key: item.Key,
					Sources: lo.Map(
						item.Sources,
						func(source envvarrefs.Source, _ int) EnvVarReferenceSourceOutput {
							return EnvVarReferenceSourceOutput{
								Type: string(source.Type),
								Name: source.Name,
							}
						},
					),
				}
			},
		),
	}
	return o
}

// DeployURIInput contains app, env and deploy path parameters.
type DeployURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 部署记录 ID
	DeployID string `uri:"deployID" binding:"required"`
}

// ResourceSnapshotURIInput contains resource snapshot path parameters.
type ResourceSnapshotURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 部署记录 ID
	DeployID string `uri:"deployID" binding:"required"`
	// 快照 ID
	SnapshotID string `uri:"snapshotID" binding:"required"`
}

// ListAppModelDeployRecordsInput contains query parameters for listing deploy records.
type ListAppModelDeployRecordsInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 搜索关键字
	Keyword string `form:"keyword"`
	// 分页页码
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页大小
	PageSize int64 `form:"pageSize" binding:"required,oneof=1 5 10 20 50 100"`
}

// CreateAppModelDeployInput contains the request body for creating a deploy.
type CreateAppModelDeployInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
	// 部署的镜像版本
	ImageTag string `json:"imageTag" binding:"required"`
	// 副本数量
	Replicas int32 `json:"replicas" binding:"required,gte=1"`
}

// DeleteAppModelDeployInput contains query parameters for deleting a deploy.
type DeleteAppModelDeployInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// GetLatestAppModelDeployStatusInput contains query parameters for latest deploy status.
type GetLatestAppModelDeployStatusInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// ListAppModelResourceSnapshotsInput contains query parameters for listing resource snapshots.
type ListAppModelResourceSnapshotsInput struct {
	// 分页页码
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页大小
	PageSize int64 `form:"pageSize" binding:"required,oneof=1 5 10 20 50 100"`
}

// ListAppModelDeployRecordsOutput is the response body for deploy records.
type ListAppModelDeployRecordsOutput struct {
	Data PaginatedAppModelDeployRecordsOutputObjs `json:"data"`
}

// PaginatedAppModelDeployRecordsOutputObjs contains paginated deploy records.
type PaginatedAppModelDeployRecordsOutputObjs struct {
	Count   int64                            `json:"count,string"`
	Results []*AppModelDeployRecordOutputObj `json:"results"`
}

// AppModelDeployRecordOutputObj contains one AppModel deploy record.
type AppModelDeployRecordOutputObj struct {
	ID        string    `json:"id"`
	ClusterID string    `json:"clusterID"`
	Namespace string    `json:"namespace"`
	ImageTag  string    `json:"imageTag"`
	Replicas  int32     `json:"replicas"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel converts an AppModel deploy record to output.
func (o *AppModelDeployRecordOutputObj) FromModel(record appmodeldeploy.Record) *AppModelDeployRecordOutputObj {
	o.ID = record.ID.Hex()
	o.ClusterID = record.ClusterID
	o.Namespace = record.Namespace
	o.ImageTag = record.ImageTag
	o.Replicas = record.Replicas
	o.Message = record.Message
	o.Status = string(record.Status)
	o.Operator = record.Updater
	o.CreatedAt = record.CreatedAt
	o.UpdatedAt = record.UpdatedAt
	return o
}

// GetLatestAppModelDeployStatusOutput is the response body for latest deploy status.
type GetLatestAppModelDeployStatusOutput struct {
	Data *LatestDeployStatus `json:"data"`
}

// LatestDeployStatus contains the latest AppModel deploy attempt status.
type LatestDeployStatus struct {
	Stage             string    `json:"stage"`
	Status            string    `json:"status"`
	Message           string    `json:"message"`
	BuildID           string    `json:"buildID"`
	DeployID          string    `json:"deployID"`
	IsBuildAutoDeploy bool      `json:"isBuildAutoDeploy"`
	StartedAt         time.Time `json:"startedAt"`
	EndedAt           time.Time `json:"endedAt"`
	Branch            string    `json:"branch"`
	ImageTag          string    `json:"imageTag"`
	Operator          string    `json:"operator"`
	PipelineID        string    `json:"pipelineID"`
	HasDeployRecord   bool      `json:"hasDeployRecord"`
}

// FromBuildAutoDeployRecord converts a build auto deploy record to latest status output.
func (o *LatestDeployStatus) FromBuildAutoDeployRecord(record *autodeploy.Record) *LatestDeployStatus {
	o.Stage = string(record.Stage)
	o.Status = record.Status
	o.Message = record.Message
	o.BuildID = record.BuildID
	o.DeployID = record.DeployID
	o.IsBuildAutoDeploy = true
	o.StartedAt = record.StartedAt
	o.EndedAt = record.EndedAt
	o.Branch = record.Branch
	o.ImageTag = record.ImageTag
	o.Operator = record.Operator
	o.PipelineID = record.PipelineID
	return o
}

// FromDeployRecord converts an AppModel deploy record to latest status output.
func (o *LatestDeployStatus) FromDeployRecord(record *appmodeldeploy.Record) *LatestDeployStatus {
	o.Stage = string(autodeploy.StageDeploy)
	o.Status = string(record.Status)
	o.Message = record.Message
	o.DeployID = record.ID.Hex()
	o.IsBuildAutoDeploy = false
	o.StartedAt = record.StartedAt
	o.EndedAt = record.EndedAt
	return o
}

// ListAppModelResourceSnapshotsOutput is the response body for resource snapshots.
type ListAppModelResourceSnapshotsOutput struct {
	Data PaginatedAppModelResourceSnapshotsOutputObjs `json:"data"`
}

// PaginatedAppModelResourceSnapshotsOutputObjs contains paginated resource snapshots.
type PaginatedAppModelResourceSnapshotsOutputObjs struct {
	Count   int64                       `json:"count,string"`
	Results []*AppModelResourceSnapshot `json:"results"`
}

// GetAppModelResourceSnapshotOutput is the response body for one resource snapshot.
type GetAppModelResourceSnapshotOutput struct {
	Snapshot *AppModelResourceSnapshot `json:"snapshot"`
}

// AppModelResourceSnapshot contains one resource manifest snapshot.
type AppModelResourceSnapshot struct {
	ID          string    `json:"id"`
	APIVersion  string    `json:"apiVersion"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	IsTruncated bool      `json:"isTruncated"`
	CreatedAt   time.Time `json:"createdAt"`
	Manifest    *string   `json:"manifest,omitempty"`
}

// FromModel converts a resource snapshot to output.
func (o *AppModelResourceSnapshot) FromModel(
	snapshot appmodeldeploy.ResourceSnapshot,
	includeManifest bool,
) *AppModelResourceSnapshot {
	o.ID = snapshot.ID.Hex()
	o.APIVersion = snapshot.APIVersion
	o.Kind = snapshot.Kind
	o.Name = snapshot.Name
	o.IsTruncated = snapshot.IsTruncated
	o.CreatedAt = snapshot.CreatedAt
	if includeManifest {
		o.Manifest = &snapshot.Manifest
	}
	return o
}
