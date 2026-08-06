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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"go.uber.org/fx"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/event"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// FxModule provides bkmonitor related dependencies via uber fx.
var FxModule = fx.Module("bkmonitor",
	database.PrivateFxModule,
	// 来自其他包的依赖，仅当前 module 内部可见
	fx.Provide(
		envvars.NewScopedEnvVarStoreMongo,
		fx.Private,
	),
	// 本包对外暴露的组件
	fx.Provide(
		NewApmInstConfigStoreMongo,
		NewApmService,
		NewUserGroupService,
		NewMetricTimeSeriesService,
	),
	strategy.FxModule,
	event.FxModule,
)
