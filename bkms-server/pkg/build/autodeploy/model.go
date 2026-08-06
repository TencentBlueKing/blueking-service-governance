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

package autodeploy

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Stage 表示一键构建部署当前所处阶段
type Stage string

const (
	// StageBuild 表示当前处于构建阶段。
	StageBuild Stage = "build"
	// StageDeploy 表示当前处于部署阶段。
	StageDeploy Stage = "deploy"
)

// Record 表示一次 build auto deploy 执行记录
type Record struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	WorkspaceID     string `bson:"workspaceID"`
	AppID           string `bson:"appID"`
	AppType         string `bson:"appType"`
	EnvName         string `bson:"envName"`
	TrafficLaneName string `bson:"trafficLaneName"`

	BuildID    string `bson:"buildID"`
	DeployID   string `bson:"deployID,omitempty"`
	Branch     string `bson:"branch,omitempty"`
	ImageTag   string `bson:"imageTag,omitempty"`
	PipelineID string `bson:"pipelineID,omitempty"`

	Stage   Stage  `bson:"stage"`
	Status  string `bson:"status"`
	Message string `bson:"message"`

	Operator string `bson:"operator"`

	StartedAt time.Time `bson:"startedAt"`
	EndedAt   time.Time `bson:"endedAt"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}
