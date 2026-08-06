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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// GetWorkspaceImageRegistry 获取工作区当前使用的镜像仓库
func GetWorkspaceImageRegistry(ctx context.Context, workspaceID string) (*bkmsreg.ImageRegistry, error) {
	// 先获取工作空间，以知晓当前使用的镜像仓库类型
	wsStore, err := NewWorkspaceStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, err
	}
	workspace, err := wsStore.Get(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s", workspaceID)
	}
	// 再通过工作空间 ID + 类型，获取镜像仓库
	regStore, err := bkmsreg.NewImageRegistryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, err
	}
	registry, err := regStore.GetByWorkspaceAndType(ctx, workspaceID, workspace.ImageRegistryType)
	if err != nil {
		return nil, errors.Wrap(err, "get image registry")
	}
	return registry, nil
}
