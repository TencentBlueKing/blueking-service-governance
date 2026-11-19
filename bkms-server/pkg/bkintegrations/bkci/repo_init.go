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

package bkci

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

// RepositoryInitSpec carries the repository URL/alias pair required to
// initialize a BKCI repository binding.
type RepositoryInitSpec struct {
	URL   string
	Alias string
}

// EnsureWorkspaceRepositories initializes each distinct repository binding for
// a workspace at most once per request.
func EnsureWorkspaceRepositories(
	ctx context.Context, db *mongo.Database, workspaceID string, specs []RepositoryInitSpec,
) error {
	initializer := NewRepositoryManager(db, workspaceID)
	seen := make(map[RepositoryInitSpec]struct{}, len(specs))
	for _, spec := range specs {
		if spec.URL == "" || spec.Alias == "" {
			continue
		}
		if _, ok := seen[spec]; ok {
			continue
		}
		if _, err := initializer.Initialize(ctx, spec.URL, spec.Alias); err != nil {
			return err
		}
		seen[spec] = struct{}{}
	}
	return nil
}

// WrapRepositoryErr 将关联 / 访问蓝盾代码库的错误转换为接口错误：
// 蓝盾拒绝时返回无权限并附带引导授权的详情码，其余情况视为内部错误
func WrapRepositoryErr(err error, msg string) *bkerrs.Error {
	if errors.Is(err, bkci.PermissionDenied) {
		return bkerrs.WrapBkCIRepositoryPermissionDenied(err, msg)
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, msg)
}
