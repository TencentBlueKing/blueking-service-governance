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

// Package handler 包含部署相关 Gin API 的 handler
package handler

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// validateHelmDeployAppEnv 校验 appID、envName 和应用权限，并返回部署环境信息
func (h *Handler) validateHelmDeployAppEnv(
	ctx context.Context,
	appID string,
	envName string,
	permType ginperm.Type,
) (*bkmsapp.Application, *bkmsenv.Environment, error) {
	app, err := ginperm.ValidateAppByID(ctx, h.registry, appID, permType)
	if err != nil {
		return nil, nil, err
	}
	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return nil, nil, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "get workspace %s env %s", app.WorkspaceID, envName)
	}
	return app, env, nil
}

// validateAppModelDeployAppEnv 是 AppModel 部署相关 handler 的统一前置校验
func (h *Handler) validateAppModelDeployAppEnv(
	ctx context.Context,
	appID string,
	envName string,
	permType ginperm.Type,
	requireDeployEnvPerm bool,
) (*bkmsapp.Application, *bkmsenv.Environment, error) {
	app, env, err := ginperm.ValidateAppEnvByName(ctx, h.registry, appID, envName, permType)
	if err != nil {
		return nil, nil, err
	}
	if !bkmsapp.IsAppModelType(app.Type) {
		return nil, nil, bkerrs.New(bkerrs.ErrCodeInvalidRequest, "not appmodel app")
	}
	if requireDeployEnvPerm {
		if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, envName); err != nil {
			return nil, nil, bkerrs.WrapIAMNoPermission(err, app.WorkspaceID, "check env deploy perm")
		}
	}
	return app, env, nil
}

// bindOptionalJSON 兼容旧 PUT 接口可只依赖路径参数的调用方式；请求体存在时仍按 Gin JSON binding 校验
func (h *Handler) bindOptionalJSON(c *gin.Context, obj any) error {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return nil
	}
	return ginutils.BindJSON(c, obj)
}

// newSnapshotService 创建镜像快照服务，用于 Helm 部署前检查镜像晋级状态
func (h *Handler) newSnapshotService() *snapshot.Service {
	return snapshot.NewService(h.registry.SnapshotStore, h.registry.BuildConfigStore, h.registry.AppStore)
}

// genDeployInfo 生成部署信息（用于日志输出标识部署详情）
func genDeployInfo(workspaceID, appID, envName, trafficLaneName string) string {
	info := fmt.Sprintf("<deploy workspace: %s, app: %s, env: %s", workspaceID, appID, envName)
	if trafficLaneName != "" {
		info += fmt.Sprintf(", lane: %s", trafficLaneName)
	}
	return info + ">"
}
