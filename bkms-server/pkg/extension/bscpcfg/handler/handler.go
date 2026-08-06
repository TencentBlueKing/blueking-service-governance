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

// Package handler 包含应用配置管理 API 的 Handler 实现。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// Handler 处理应用配置管理 API 请求。
type Handler struct {
	registry *storereg.Registry
}

// New 创建 Handler。
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// newManager 从 gin.Context 中获取当前用户并创建 Manager。
func (h *Handler) newManager(c *gin.Context) (*svc.Manager, error) {
	ctx := c.Request.Context()
	user := auth.MustGetUser(ctx)
	mgr, err := svc.NewManager(user, h.registry.BscpCfgStore)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create bscp cfg service manager")
	}
	return mgr, nil
}
