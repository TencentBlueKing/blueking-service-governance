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
