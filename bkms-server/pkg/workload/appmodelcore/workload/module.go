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

package workload

import (
	"go.uber.org/fx"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var FxModule = fx.Module("workload",
	database.PrivateFxModule,
	// Mark all as private so current FxModule can be composed into other FxModules without
	// conflict with other FxModules that also depend on these dependencies.
	fx.Provide(
		envvars.NewScopedEnvVarStoreMongo,
		workspace.NewWorkspaceCompsStoreMongo,
		polaris.NewPolarisConfigStoreMongo,
		bscpcfg.NewStoreMongo,
		fx.Annotate(build.NewConfigStoreMongo, fx.As(new(build.ConfigStore))),
		fx.Annotate(appspec.NewAppSpecStoreMongo, fx.As(new(appspec.AppSpecStore))),
		fx.Private,
	),
	fx.Provide(
		NewBuilderService,
	),
)
