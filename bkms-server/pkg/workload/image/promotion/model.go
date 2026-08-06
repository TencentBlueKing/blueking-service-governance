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

package promotion

import "time"

// Image 镜像晋级记录（独立于 snapshot.Image，以 AppID + RepoKey + Tag 为唯一键）
type Image struct {
	// AppID 应用 ID
	AppID string `bson:"appID"`
	// RepoKey 仓库实例唯一标识，用于与 snapshot.Image 关联
	RepoKey string `bson:"repoKey"`
	// Tag 镜像标签名
	Tag string `bson:"tag"`
	// PromotedAt 晋级操作时间
	PromotedAt time.Time `bson:"promotedAt"`
	// PromotedBy 晋级操作人
	PromotedBy string `bson:"promotedBy"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
