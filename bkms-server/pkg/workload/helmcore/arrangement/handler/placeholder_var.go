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

// Package handler 提供 arrangement 模块的 Gin 视图逻辑。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement/serializer"
)

// Handler 是 arrangement 模块的 Gin handler。
type Handler struct {
	registry *storereg.Registry
}

// New 创建一个新的 arrangement Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListPlaceholderVars 获取编排可用的应用占位符变量列表。
//
//	@ID			ListPlaceholderVars
//	@Summary	获取编排可用的应用占位符变量列表
//	@Tags		arrangement
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListPlaceholderVarsOutput
//	@Router		/placeholder-vars [get]
func (h *Handler) ListPlaceholderVars(c *gin.Context) {
	ginutils.OK(c, new(serializer.ListPlaceholderVarsOutput).FromModels(arrangement.ListPlaceholderVars()))
}
