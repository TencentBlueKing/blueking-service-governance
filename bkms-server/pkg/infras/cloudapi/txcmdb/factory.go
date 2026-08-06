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

// Package txcmdb provides api client to tx-cmdb
package txcmdb

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// Factory 构造 Client 的工厂函数
//
// 具体实现可通过 RegisterFactory 注册；未注册时 New() 会退化为返回 noopClient，
// 所有查询返回空值，保证主流程可运行。
type Factory func() (Client, error)

// factory 存放当前注册的工厂，仅在包加载期由具体实现的 init() 注册，
// 读取发生在 New() 中（晚于所有 init），因此无需并发保护
var factory Factory

// RegisterFactory 注册全局工厂，通常在具体实现包的 init() 中调用
func RegisterFactory(f Factory) {
	if f == nil {
		panic("txcmdb factory is nil")
	}
	if factory != nil {
		panic("txcmdb factory already registered")
	}
	factory = f
}

// New 创建 Tx CMDB 客户端
//
// 启用 useStubTxCMDB 时使用 noopClient，避免开发测试环境访问真实 Tx CMDB；
// 已注册工厂时调用注册的工厂；未注册时退化为返回 noopClient，所有查询返回空值。
func New() (Client, error) {
	if config.G.Development.UseStubTxCMDB {
		log.InfoNoContext("use noop tx cmdb client according to config")
		return newNoopClient(), nil
	}
	if factory == nil {
		log.InfoNoContext("tx cmdb factory not registered, fallback to noop client")
		return newNoopClient(), nil
	}
	return factory()
}
