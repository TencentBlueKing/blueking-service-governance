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

package appmodel

import (
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// resourceSnapshotCollectionName AppModel 部署资源清单快照集合
const resourceSnapshotCollectionName = "app_model_resource_snapshots"

// ErrResourceSnapshotNotFound 该部署记录下无任何资源清单快照
var ErrResourceSnapshotNotFound = errors.New("app model resource snapshot not found")

// ErrResourceSnapshotRowNotFound 指定快照行不存在或不属于该应用
var ErrResourceSnapshotRowNotFound = errors.New("app model resource snapshot row not found")

// ResourceSnapshot 资源清单快照
// 每一条记录对应一个资源清单快照，一次部署有多个资源清单快照
type ResourceSnapshot struct {
	// ID 快照 ID
	ID bson.ObjectID `bson:"_id,omitempty"`
	// DeployRecordID 部署记录 ID
	DeployRecordID bson.ObjectID `bson:"deployRecordId"`
	// AppID 应用 ID
	AppID string `bson:"appID"`
	// APIVersion 资源 API 版本
	APIVersion string `bson:"apiVersion,omitempty"`
	// Kind 资源类型
	Kind string `bson:"kind"`
	// Name 资源名称
	Name string `bson:"name"`
	// Manifest 资源清单
	Manifest string `bson:"manifest,omitempty"`
	// IsTruncated 是否被截断
	IsTruncated bool `bson:"isTruncated"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
}
