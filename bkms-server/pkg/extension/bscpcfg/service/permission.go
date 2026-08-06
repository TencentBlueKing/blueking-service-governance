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

package service

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

// addBSCPPermissions 刷新 BSCP 权限范围到 workspace 的权限组合中。
// 注意：UpdateWorkspaceAdmin、UpdateWorkspaceScopeBuiltinRoles 都是仅新增 bscp fileSvc 到 iam 侧，不会剔除已有资源。
func addBSCPPermissions(
	ctx context.Context,
	ws *workspace.Workspace,
	fileSvc *bscpapi.Service,
) error {
	client, err := bkcc.New(auth.MustGetUser(ctx))
	if err != nil {
		return err
	}
	business, err := client.GetBusinessByID(ctx, cast.ToInt64(ws.BkSystems.BkCCBizID))
	if err != nil {
		return err
	}

	workspaceData := bkiam.WorkspaceData{
		WorkspaceID:   ws.ID,
		WorkspaceName: ws.DisplayName,
		BSCP: &bkiam.BSCPOptions{
			BizID:   ws.BkSystems.BkCCBizID,
			BizName: business.BizName,
			Services: []bkiam.BSCPService{
				{
					ID:   fileSvc.ID,
					Name: fileSvc.Name,
				},
			},
		},
	}

	// 新增管理员权限范围
	permMgr := perm.NewManager()
	if err = permMgr.UpdateWorkspaceAdmin(ctx, workspaceData); err != nil {
		return errors.Wrap(err, "update workspace admin")
	}

	// 新增内置角色权限范围（sre、developer、operator）
	if err = permMgr.UpdateWorkspaceScopeBuiltinRoles(ctx, workspaceData); err != nil {
		return errors.Wrap(err, "update workspace scope builtin roles")
	}

	return nil
}
