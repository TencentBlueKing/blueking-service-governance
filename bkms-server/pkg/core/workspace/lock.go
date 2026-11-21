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
	"fmt"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/lock"
)

// stateLockTimeoutSeconds 需覆盖持锁期间查询与启停关联项目的外部调用耗时
const stateLockTimeoutSeconds = 60

// NewStateLock 新建工作空间启停锁，key 带 ctx 租户前缀。
//
// 启停空间与补停关联项目须持有该锁，避免空间状态与关联项目状态被并发修改得不一致。
func NewStateLock(ctx context.Context, workspaceID string) (lock.Locker, error) {
	key := fmt.Sprintf("lock:workspace-state:%s", workspaceID)
	stateLock, err := lock.NewTenantRedisLock(ctx, key, stateLockTimeoutSeconds)
	if err != nil {
		return nil, errors.Wrapf(err, "build state lock of workspace %s", workspaceID)
	}
	return stateLock, nil
}
