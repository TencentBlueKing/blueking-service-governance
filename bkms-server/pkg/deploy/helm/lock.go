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

// Package helm deploy lock.go provides deploy lock related functions.
package helm

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/lock"
)

// NewDeployLock 新建部署锁实例，key 带 ctx 租户前缀（应用归属租户）。
// 加锁（部署请求）与释放（部署状态轮询任务）须使用同一租户的 ctx，否则释放不到锁。
//
// TODO(tenant-lock-compat): 升级过渡期 lock.NewTenantRedisLock 会同时持有不带租户前缀的旧 key，
// 详见其注释，后续版本移除兼容逻辑后这里无需改动。
func NewDeployLock(ctx context.Context, appID, envName, trafficLaneName string) (lock.Locker, error) {
	// 锁超时时间与轮询状态超时时间保持一致
	timeout := config.G.TaskPoller.DeployStatus.Timeout
	key := fmt.Sprintf("lock:helm-deploy:%s:%s:%s", appID, envName, trafficLaneName)
	deployLock, err := lock.NewTenantRedisLock(ctx, key, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "build deploy lock of app %s", appID)
	}
	return deployLock, nil
}
