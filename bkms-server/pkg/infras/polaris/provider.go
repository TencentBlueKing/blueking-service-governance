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

// Package polaris 提供北极星 SDK 的封装，用于查询北极星服务实例状态。
//
// 本包作为 facade 层，对外暴露与具体 SDK 无关的接口和数据类型。
// 具体实现由 libs/bkms-adapter/polaris 包提供，可通过 module replace
// 在编译期切换为不同的 SDK 实现。
package polaris

import (
	"context"
	"sync"

	polarisadapter "github.com/TencentBlueKing/blueking-service-governance/libs/bkms-adapter/polaris"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// Instance 北极星实例信息（类型别名，保持调用方兼容）。
type Instance = polarisadapter.Instance

// initOnce 保证底层 SDK 只初始化一次。
var initOnce sync.Once

// provider 为当前生效的 Provider 实现。
//
// 不同的 SDK 实现通过 libs/bkms-adapter 的 module replace 在编译期切换，
// 因此这里无需运行时替换入口，直接构造单例即可。
var provider = polarisadapter.NewDefaultProvider(
	polarisadapter.WithWarnFunc(log.Warnf),
	polarisadapter.WithInfoFunc(log.Infof),
)

// MustInitClient 初始化北极星消费端（单例）。初始化失败将直接终止进程。
func MustInitClient(ctx context.Context, joinPoint, sdkAddress string) {
	initOnce.Do(func() {
		if err := provider.Init(ctx, joinPoint, sdkAddress); err != nil {
			log.Fatalf("failed to init polaris client: %v", err)
		}
	})
}

// GetInstances 查询北极星服务的所有实例。
//
// 注意：SDK 内部有最多 15 秒的缓存延迟，可能出现实际存在的服务/实例查询不到等情况。
func GetInstances(ctx context.Context, namespace, serviceName string) ([]*Instance, error) {
	if namespace == "" || serviceName == "" {
		return nil, errors.New("namespace and serviceName must not be empty")
	}
	return provider.GetInstances(ctx, namespace, serviceName)
}
