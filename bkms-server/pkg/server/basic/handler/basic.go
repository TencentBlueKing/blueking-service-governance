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

// Package handler 提供 basic 模块的 Gin 视图逻辑。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/version"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/basic/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler 是 basic 模块的 Gin handler。
type Handler struct {
	registry *storereg.Registry
}

// New 创建一个新的 basic Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// Ping 联通性测试接口
//
//	@Summary		联通性测试接口
//	@Description	用于检测服务联通性
//	@Tags			basic
//	@Produce		json
func (h *Handler) Ping(c *gin.Context) {
	ginutils.OK(c, &serializer.PingOutput{Data: "pong"})
}

// Version 提供服务版本信息
//
//	@Summary		服务版本信息接口
//	@Description	返回服务版本、Git Hash、构建时间、Go 版本等信息
//	@Tags			basic
//	@Produce		json
func (h *Handler) Version(c *gin.Context) {
	ginutils.OK(c, &serializer.VersionOutput{
		Data: &serializer.VersionData{
			Version:   version.Version,
			GitHash:   version.GitHash,
			BuildTime: version.BuildTime,
			GoVersion: version.GoVersion,
		},
	})
}
