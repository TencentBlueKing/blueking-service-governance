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

// Package dashboard 提供应用与蓝鲸监控仪表盘的绑定管理能力。
package dashboard

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AppDashboard 记录应用与蓝鲸监控仪表盘的绑定关系。
// 应用可以绑定某个仪表盘（uid），并自定义展示标题（title）。
type AppDashboard struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`

	// AppID 应用 ID
	AppID string `bson:"appID" validate:"required"`
	// UID 仪表盘 uid
	UID string `bson:"uid" validate:"required"`
	// Title 仪表盘标题（bkms 自定义）
	Title string `bson:"title" validate:"required"`

	// Creator 创建人
	Creator string `bson:"creator"`
	// Updater 更新人
	Updater string `bson:"updater"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// AppDashboardUpdateData 应用仪表盘绑定更新数据。
type AppDashboardUpdateData struct {
	// UID 仪表盘 uid
	UID *string
	// Title 仪表盘标题
	Title *string
}

// ToBSON converts AppDashboardUpdateData to bson.M for update operations.
func (d *AppDashboardUpdateData) ToBSON() (bson.M, bool) {
	data := bson.M{}

	if d.UID != nil {
		data["uid"] = *d.UID
	}
	if d.Title != nil {
		data["title"] = *d.Title
	}

	return data, len(data) == 0
}
