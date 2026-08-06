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

package admin

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// 工作空间临时管理员记录表，记录临时管理员授权及其是否已回收。
const collectionName = "workspace_temp_admin_records"

// RoleStatus 描述指定用户在某个空间内是否拥有目标角色。
type RoleStatus struct {
	HasRole bool
}

// WorkspaceTempAdmin 存储工作空间临时管理员授权记录
type WorkspaceTempAdmin struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	WorkspaceID string        `bson:"workspaceID"`
	Username    string        `bson:"username"`
	ExpiresAt   time.Time     `bson:"expiresAt"`
	IsRecycled  bool          `bson:"isRecycled"`
	Creator     string        `bson:"creator"`
	Updater     string        `bson:"updater"`
	CreatedAt   time.Time     `bson:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"`
}
