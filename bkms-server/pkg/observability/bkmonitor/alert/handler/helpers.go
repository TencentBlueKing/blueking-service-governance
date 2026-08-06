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

package handler

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

func (h *Handler) alertStrategyService() *alertstrategy.Service {
	return alertstrategy.NewService(
		h.registry.AlertStrategyStore, h.registry.EnvStore, h.registry.AppStore, h.registry.ResourceSnapshotStore,
	)
}

func (h *Handler) loadStrategyContext(
	ctx context.Context,
	workspaceID, appID, strategyID string,
	permType ginperm.Type,
) (*bkmsapp.Application, *workspace.Workspace, bson.ObjectID, error) {
	app, err := validateAppInWorkspace(ctx, h.registry, workspaceID, appID, permType)
	if err != nil {
		return nil, nil, bson.NilObjectID, err
	}
	ws, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		return nil, nil, bson.NilObjectID, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace")
	}
	strategyObjID, err := bson.ObjectIDFromHex(strategyID)
	if err != nil {
		return nil, nil, bson.NilObjectID, bkerrs.Errorf(bkerrs.ErrCodeInvalidRequest, "invalid strategy ID")
	}
	if _, err = getAlertStrategyInApp(ctx, h.registry, strategyObjID, workspaceID, appID); err != nil {
		return nil, nil, bson.NilObjectID, wrapAlertStrategyLookupErr(err)
	}
	return app, ws, strategyObjID, nil
}

func wrapAlertStrategyLookupErr(err error) error {
	if errors.Is(err, alertstrategy.ErrNotFound) {
		return bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "alert strategy not found")
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get alert strategy")
}

func getAlertStrategyInApp(
	ctx context.Context,
	registry *storereg.Registry,
	strategyID bson.ObjectID,
	workspaceID, appID string,
) (*alertstrategy.AlertStrategy, error) {
	rule, err := registry.AlertStrategyStore.Get(ctx, strategyID)
	if err != nil {
		return nil, err
	}
	if rule.WorkspaceID != workspaceID || rule.AppID != appID {
		return nil, alertstrategy.ErrNotFound
	}
	return rule, nil
}

func validateAppInWorkspace(
	ctx context.Context,
	registry *storereg.Registry,
	workspaceID, appID string,
	permType ginperm.Type,
) (*bkmsapp.Application, error) {
	app, err := ginperm.ValidateAppByID(ctx, registry, appID, permType)
	if err != nil {
		return nil, err
	}
	if app.WorkspaceID != workspaceID {
		return nil, bkerrs.Errorf(bkerrs.ErrCodeNotFound, "app %s not found in workspace %s", appID, workspaceID)
	}
	return app, nil
}
