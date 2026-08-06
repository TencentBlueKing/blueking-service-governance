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

// Package bscpcfg 提供应用配置管理的对外入口（借助 BSCP 实现配置下发）
package bscpcfg

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"

// 类型别名 —— 方便外部包直接使用 bscpcfg.XxxType 而无需引入 model 子包
type (
	// Store 应用配置管理统一存储接口
	Store = model.Store
	// Snapshot 聚合快照（Metadata + EnvBinding）
	Snapshot = model.Snapshot
)

// NewStoreMongo 重新导出构造函数
var NewStoreMongo = model.NewStoreMongo
