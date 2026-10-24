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

// Package serializer defines Gin input and output serializers for platform workspace image registry APIs.
package serializer

import bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"

// WorkspacePath is the URI input for a single platform workspace image registry API.
type WorkspacePath struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// UpdateImageRegistryInput is the JSON input for switching or updating a workspace image registry.
type UpdateImageRegistryInput struct {
	// 目标镜像仓库类型：builtin（切回平台镜像仓库，其余字段不生效）/ external（外部镜像仓库）
	Type string `json:"type" binding:"required,oneof=builtin external"`
	// 外部镜像仓库地址，形如 mirrors.tencent.com/my-namespace，不带协议前缀
	Registry string `json:"registry" binding:"required_if=Type external,max=256"`
	// 外部镜像仓库账号
	Username string `json:"username" binding:"required_if=Type external,max=128"`
	// 外部镜像仓库密码；留空表示沿用该空间已有外部仓库的密码
	Password string `json:"password" binding:"max=512"`
}

// ImageRegistryOutput is the effective image registry of a workspace.
type ImageRegistryOutput struct {
	// 镜像仓库类型：builtin（平台镜像仓库）/ external（外部镜像仓库）
	Type string `json:"type"`
	// 镜像仓库地址（含命名空间），配置缺失时为空
	Registry string `json:"registry"`
	// 镜像账号，配置缺失时为空
	Username string `json:"username"`
	// 镜像密码，配置缺失时为空
	Password string `json:"password"`
}

// NewImageRegistryOutput builds output from the image registry model.
func NewImageRegistryOutput(reg *bkmsreg.ImageRegistry) *ImageRegistryOutput {
	return &ImageRegistryOutput{
		Type:     string(reg.Type),
		Registry: reg.Registry,
		Username: reg.Username,
		Password: reg.Password,
	}
}

// ImageRegistryResponse is the JSON response carrying a workspace image registry.
type ImageRegistryResponse struct {
	Data *ImageRegistryOutput `json:"data"`
}
