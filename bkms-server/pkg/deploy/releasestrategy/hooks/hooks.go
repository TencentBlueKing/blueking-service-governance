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

// Package hooks registers release strategy lifecycle hooks.
package hooks

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CleanupStrategiesByWorkspaceHookName identifies the callback that removes a
// workspace's release strategies during workspace deletion.
const CleanupStrategiesByWorkspaceHookName = "releasestrategy.cleanup_strategies_by_workspace"

// RegisterPreDeleteHooks registers release strategy cleanup before workspace deletion.
func RegisterPreDeleteHooks(db *mongo.Database) {
	bkmsworkspace.RegisterPreDeleteHook(
		CleanupStrategiesByWorkspaceHookName,
		NewCleanupStrategiesByWorkspaceHook(db),
	)
}

// NewCleanupStrategiesByWorkspaceHook removes all release strategies before deleting a workspace.
func NewCleanupStrategiesByWorkspaceHook(db *mongo.Database) bkmsworkspace.PreDeleteHook {
	return func(ctx context.Context, workspaceID string) error {
		strategies := store.Of(db, releasestrategy.Meta)
		if _, err := strategies.DeleteMany(ctx, releasestrategy.ByWorkspace(workspaceID)); err != nil {
			return errors.Wrapf(err, "delete release strategies of workspace %s", workspaceID)
		}
		return nil
	}
}
