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

package snapshot

import "time"

// Image 镜像快照记录
type Image struct {
	// RepoKey 仓库实例唯一标识
	RepoKey string `bson:"repoKey"`
	// Tag 镜像标签名
	Tag string `bson:"tag"`
	// Digest 镜像摘要
	Digest string `bson:"digest,omitempty"`
	// Size 镜像大小（字节）
	Size int64 `bson:"size,omitempty"`
	// BuiltAt 镜像构建时间（可为空，由 detail syncer 补全）
	BuiltAt *time.Time `bson:"builtAt,omitempty"`
	// DetailSyncPending 标记该标签的详情需要重新拉取（如标签被同名构建覆盖），
	// 详情同步成功后由 UpdateDetail 清除
	//
	// TODO: 并发同 tag 构建时存在丢刷新风险——若构建 B 在构建 A 的详情同步完成前
	// MarkDetailSyncPending，随后 A 的 UpdateDetail 仍会无条件清除该标记，导致 B 的
	// 新镜像详情可能不会被再次拉取，直到下一次同 tag 构建或手动刷新；彻底方案是将
	// 标记改为世代号（Mark 递增、UpdateDetail 仅清除等值世代）
	DetailSyncPending bool `bson:"detailSyncPending,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// RepoSnapshotStatus 仓库快照状态
type RepoSnapshotStatus struct {
	// RepoKey 仓库实例唯一标识
	RepoKey string `bson:"repoKey"`
	// RepoName 仓库名称（用于前端展示）
	RepoName string `bson:"repoName"`
	// RefreshStatus 当前刷新状态
	RefreshStatus RefreshStatus `bson:"refreshStatus"`
	// LastRefreshedAt 最后成功刷新时间
	LastRefreshedAt *time.Time `bson:"lastRefreshedAt,omitempty"`
	// LastDetailSyncedAt 最后成功详情同步时间
	LastDetailSyncedAt *time.Time `bson:"lastDetailSyncedAt,omitempty"`
	// LastError 最后失败信息
	LastError string `bson:"lastError,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}
