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

// Package handler 提供应用仪表盘绑定相关的 HTTP 接口处理逻辑。
package handler

import (
	"errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmdashboard "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/dashboard"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	errDashboardNotExist = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"OBSERVABILITY_BKMONITOR_DASHBOARD_NOT_EXIST",
		"specified dashboard does not exist in bkmonitor",
	)
	errDashboardDuplicate = bkerrs.Define(
		bkerrs.ErrCodeAlreadyExists,
		"OBSERVABILITY_BKMONITOR_DASHBOARD_DUPLICATE",
		"dashboard is already bound to app",
	)
	errDashboardBindingNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"OBSERVABILITY_BKMONITOR_DASHBOARD_NOT_FOUND",
		"app dashboard binding not found",
	)
	errDashboardUIDRequired = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"OBSERVABILITY_BKMONITOR_DASHBOARD_UID_REQUIRED",
		"dashboard uid is required",
	)
)

// Handler handles app dashboard binding HTTP requests.
type Handler struct {
	registry *storereg.Registry
	service  *bkmdashboard.Service
}

// New creates an app dashboard binding HTTP handler.
func New(registry *storereg.Registry, service *bkmdashboard.Service) *Handler {
	return &Handler{registry: registry, service: service}
}

func (h *Handler) workspaces() *workspace.WorkspaceStore {
	return store.Of(h.registry.DB, workspace.Meta)
}

// wrapError 将底层错误转换为统一的 bkerrs 错误码。
func (h *Handler) wrapError(err error, action string) error {
	switch {
	case errors.Is(err, bkmdashboard.ErrDashboardNotExist):
		return errDashboardNotExist.WithCause(err)
	case errors.Is(err, bkmdashboard.ErrDuplicate):
		return errDashboardDuplicate.WithCause(err)
	case errors.Is(err, bkmdashboard.ErrNotFound):
		return errDashboardBindingNotFound.WithCause(err)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, action)
	}
}
