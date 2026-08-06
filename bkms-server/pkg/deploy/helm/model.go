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

package helm

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

// Record 部署记录
type Record struct {
	// ID 部署记录 ID
	ID bson.ObjectID `bson:"_id,omitempty"`

	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// AppID 应用 ID
	AppID string `bson:"appID"`

	// EnvName 环境名称
	EnvName string `bson:"envName"`
	// TrafficLaneName 泳道名称
	TrafficLaneName string `bson:"trafficLaneName"`
	// Revision 版本信息
	Revision string `bson:"revision"`
	// ProjectCode 蓝盾项目 ID
	ProjectCode string `bson:"projectCode"`
	// ClusterID 集群 ID
	ClusterID string `bson:"clusterID"`
	// Namespace 命名空间
	Namespace string `bson:"namespace"`
	// ReleaseName 发布名称
	ReleaseName string `bson:"releaseName"`
	// ChartName Chart 名称
	ChartName string `bson:"chartName"`
	// ChartVersion Chart 版本
	ChartVersion string `bson:"chartVersion"`
	// ValuesFileID Values 文件 ID
	ValuesFileID string `bson:"valuesFileID"`
	// ImageTag 镜像版本
	ImageTag string `bson:"imageTag"`
	// Message 描述信息
	Message string `bson:"message"`
	// Status 构建状态
	Status helmrelease.Status `bson:"status"`
	// Operator 操作人
	Operator string `bson:"operator"`
	// Extras 额外信息
	Extras map[string]string `bson:"extras"`

	// StartedAt 开始时间
	StartedAt time.Time `bson:"startedAt"`
	// EndedAt 结束时间
	EndedAt time.Time `bson:"endedAt"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
