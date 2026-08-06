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

import "time"

// RoleBinding defines a platform administrator role binding record.
type RoleBinding struct {
	// Username is the unique BlueKing username of the platform administrator.
	Username string `bson:"username" json:"username"`
	// RoleCode is the platform role assigned to the user.
	RoleCode RoleCode `bson:"roleCode" json:"roleCode"`
	// CreatedAt is when the administrator entry was created.
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	// Creator is the operator who added this administrator.
	Creator string `bson:"creator" json:"creator"`
	// UpdatedAt is when the administrator entry was last updated.
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	// Updater is the operator who last updated this administrator.
	Updater string `bson:"updater" json:"updater"`
}

// ListOptions controls platform administrator list behavior.
type ListOptions struct {
	// Keyword fuzzy matches username case-insensitively.
	Keyword string
}
