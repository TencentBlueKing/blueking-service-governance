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
	"context"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	svc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	cfgswitchmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/cfgswitch/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
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
	mgr, err := svc.NewManager(user, h.registry.DB)
	if err != nil {
		return nil, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create bscp cfg service manager")
	}
	return mgr, nil
}

// validateApp 校验应用权限，并要求应用所在空间已开启配置管理（BSCP）。
// 本包所有接口都须经过它或 validateAppEnv，开关关闭时读写接口一律不可用
func (h *Handler) validateApp(
	ctx context.Context, appID string, permType perm.Type,
) (*bkmsapp.Application, error) {
	app, err := perm.ValidateAppByID(ctx, h.registry, appID, permType)
	if err != nil {
		return nil, err
	}
	if err = h.requireBscpEnabled(ctx, app.WorkspaceID); err != nil {
		return nil, err
	}
	return app, nil
}

// validateAppEnv 校验应用与环境权限，并要求应用所在空间已开启配置管理（BSCP）
func (h *Handler) validateAppEnv(
	ctx context.Context, appID, envName string, permType perm.Type,
) (*bkmsapp.Application, *envmodel.Environment, error) {
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, appID, envName, permType)
	if err != nil {
		return nil, nil, err
	}
	if err = h.requireBscpEnabled(ctx, app.WorkspaceID); err != nil {
		return nil, nil, err
	}
	return app, env, nil
}

// requireBscpEnabled 空间未开启配置管理（BSCP）时返回无权限错误。开关由平台管理员在平台管理中设置
func (h *Handler) requireBscpEnabled(ctx context.Context, workspaceID string) error {
	enabled, err := cfgswitchmodel.IsEnabled(ctx, h.registry.DB, workspaceID, cfgswitchmodel.SwitchBscp)
	if err != nil {
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "check workspace bscp feature")
	}
	if !enabled {
		return bkerrs.Errorf(
			bkerrs.ErrCodeNoPermission,
			"config management (bscp) is not enabled for workspace %s, contact the platform administrator",
			workspaceID,
		)
	}
	return nil
}
