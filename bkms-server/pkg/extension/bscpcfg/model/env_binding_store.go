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

// Package model 定义了应用配置管理相关的纯数据模型。
package model

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 错误定义
var (
	// ErrEnvBindingNotFound 配置管理环境绑定不存在
	ErrEnvBindingNotFound = errors.New("bscpcfg env binding not found")

	// ErrEnvBindingAlreadyExists 配置管理环境绑定已存在（同一 app+env 只能有一条记录）
	ErrEnvBindingAlreadyExists = errors.New("bscpcfg env binding already exists")
)

// EnvBindingStore EnvBinding 存储，一个 app+env 一条记录
type EnvBindingStore = store.GenericStore[EnvBinding]

// EnvBindingMeta EnvBinding 的存储元数据，写入完整文档前会校验必填字段。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + envName
var EnvBindingMeta = store.Meta[EnvBinding]{
	Collection: "bscpcfg_env_bindings",
	NotFound:   ErrEnvBindingNotFound,
	Duplicate:  ErrEnvBindingAlreadyExists,
	BeforeSave: []func(*EnvBinding) error{validateEnvBinding},
}

func validateEnvBinding(binding *EnvBinding) error {
	return errors.Wrap(validate.Struct(binding), "env binding validation failed")
}

// ByAppAndEnv 按应用 ID 与环境名称匹配 EnvBinding（至多一条）
func ByAppAndEnv(appID, envName string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("envName", envName))
}
