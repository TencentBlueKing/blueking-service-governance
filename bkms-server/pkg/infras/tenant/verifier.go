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

package tenant

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/bkuser"
)

type bkUserVerifier struct {
	userClient bkuser.UserClient
}

// NewBKUserVerifier creates a verifier backed by bk-user APIs.
func NewBKUserVerifier(userClient bkuser.UserClient) Verifier {
	return &bkUserVerifier{userClient: userClient}
}

// Verifier validates that the authenticated user can access the requested tenant.
type Verifier interface {
	Verify(ctx context.Context, bkUsername, tenantID string) error
}

// Verify confirms the user belongs to the requested tenant and is enabled.
func (v *bkUserVerifier) Verify(ctx context.Context, bkUsername, tenantID string) error {
	user, err := v.userClient.GetUser(ctx, bkUsername)
	if err != nil {
		return errors.Wrapf(err, "get bk-user user %s", bkUsername)
	}
	if user.TenantID != tenantID {
		return ErrTenantAccessDenied
	}
	if user.Status != bkuser.UserStatusEnabled {
		return ErrTenantUserDisabled
	}
	return nil
}
