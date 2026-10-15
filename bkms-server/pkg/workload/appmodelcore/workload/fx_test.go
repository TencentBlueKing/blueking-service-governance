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

package workload_test

import (
	. "github.com/onsi/ginkgo/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

func newWorkloadPluginDependencies() (
	appcfg.AppConfigFileStore,
	*mongo.Database,
	appcfg.AppConfigFileVersionStore,
	polaris.PolarisConfigStore,
) {
	var appConfigFileStore appcfg.AppConfigFileStore
	var db *mongo.Database
	var appConfigFileVersionStore appcfg.AppConfigFileVersionStore
	var polarisConfigStore polaris.PolarisConfigStore

	diApp := fxtest.New(
		GinkgoT(),
		appcfg.FxModule,
		polaris.FxModule,
		store.FxModule,
		fx.Populate(
			&appConfigFileStore,
			&db,
			&appConfigFileVersionStore,
			&polarisConfigStore,
		),
	)
	diApp.RequireStart()
	diApp.RequireStop()

	return appConfigFileStore, db, appConfigFileVersionStore, polarisConfigStore
}
