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

package workspace

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// GetWorkspaceImageRegistry 获取工作区当前使用的镜像仓库
func GetWorkspaceImageRegistry(
	ctx context.Context,
	db *mongo.Database,
	workspaceID string,
) (*bkmsreg.ImageRegistry, error) {
	// 先获取工作空间，以知晓当前使用的镜像仓库类型
	workspace, err := store.Of(db, Meta).Get(ctx, ByWorkspaceID(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s", workspaceID)
	}
	// 再通过工作空间 ID + 类型，获取镜像仓库
	registry, err := store.Of(db, bkmsreg.ImageRegistryMeta).Get(
		ctx, bkmsreg.ByWorkspaceAndType(workspaceID, workspace.ImageRegistryType),
	)
	if err != nil {
		return nil, errors.Wrap(err, "get image registry")
	}
	return registry, nil
}
