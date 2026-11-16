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

package perm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

var errIAMNoPermission = bkerrs.Define(
	bkerrs.ErrCodeIAMNoPermission,
	bkerrs.ErrDetailCodeIAMNoPermission,
	"no iam permission",
)

// WrapIAMNoPermission 包装为 IAM 无权限错误，详情中附带工作空间各角色对应的权限中心用户组 ID，便于用户自助申请
func WrapIAMNoPermission(err error, workspaceID, msg string) error {
	cause := errors.New(msg)
	if err != nil {
		cause = errors.Wrap(err, msg)
	}
	wrappedErr := errIAMNoPermission.WithCause(cause)

	ctx := context.Background()
	roles, err := NewManager().ListRoles(ctx, workspaceID)
	// 获取角色失败时不追加用户组信息
	if err != nil {
		log.Errorf(ctx, "failed to list roles for workspace %s: %v", workspaceID, err)
		return wrappedErr
	}

	// 用户组角色（如：admin）-> 权限中心用户组 ID
	roleCodeToUserGroupIDMap := map[string]string{}
	for _, role := range roles {
		roleCodeToUserGroupIDMap[role.RoleCode] = cast.ToString(role.UserGroupID)
	}
	return wrappedErr.SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeIAMNoPermission,
			"should apply iam user group for permissions",
			bkerrs.WithExtras(roleCodeToUserGroupIDMap),
		),
	)
}
