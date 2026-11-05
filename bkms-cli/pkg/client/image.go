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

// Package client image resp、options、data等数据结构
package client

// Image 描述 bkms-cli 与服务端交互所用的镜像元数据
type Image struct {
	// Repository 镜像仓库
	Repository string `json:"repository" yaml:"repository"`

	// Tag 镜像 TAG
	Tag string `json:"tag" yaml:"tag"`

	// Digest 摘要
	Digest string `json:"digest" yaml:"digest"`

	// Size 镜像大小
	Size string `json:"size" yaml:"size"`

	// BuiltAt 构建时间
	BuiltAt string `json:"builtAt" yaml:"builtAt"`

	// IsPromoted 是否已晋级
	IsPromoted bool `json:"isPromoted" yaml:"isPromoted"`

	// PromotedAt 晋级时间
	PromotedAt string `json:"promotedAt" yaml:"promotedAt"`

	// PromotedBy 晋级操作人
	PromotedBy string `json:"promotedBy" yaml:"promotedBy"`
}

// ListAppImagesRespData 列出应用镜像
type ListAppImagesRespData struct {
	// Count 数量
	Count string `json:"count"`

	// Results 结果
	Results []Image `json:"results"`
}

// ListAppImagesResp 列出应用镜像
type ListAppImagesResp struct {
	// Data ListAppImagesRespData
	Data ListAppImagesRespData `json:"data"`
}

// 刷新应用镜像快照的结果状态
const (
	// RefreshAppImagesStatusSuccess 本次刷新已完成
	RefreshAppImagesStatusSuccess = "success"
	// RefreshAppImagesStatusRefreshing 已有刷新在进行中，本次未重复发起
	RefreshAppImagesStatusRefreshing = "refreshing"
	// RefreshAppImagesStatusFailed 刷新失败，快照保留上一次成功的内容
	RefreshAppImagesStatusFailed = "failed"
)

// RefreshAppImagesResult 刷新应用镜像快照结果
type RefreshAppImagesResult struct {
	// Status 刷新状态：success / refreshing / failed
	Status string `json:"status" yaml:"status"`

	// Message 提示信息
	Message string `json:"message" yaml:"message"`

	// AddedTagCnt 本次新增 tag 数量
	AddedTagCnt string `json:"addedTagCnt" yaml:"addedTagCnt"`

	// RemovedTagCnt 本次删除 tag 数量
	RemovedTagCnt string `json:"removedTagCnt" yaml:"removedTagCnt"`
}

// RefreshAppImagesResp 刷新应用镜像快照
type RefreshAppImagesResp struct {
	// Data RefreshAppImagesResult
	Data RefreshAppImagesResult `json:"data"`
}
