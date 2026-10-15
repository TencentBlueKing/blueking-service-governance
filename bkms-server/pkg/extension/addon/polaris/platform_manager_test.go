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

package polaris_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisinfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	gstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("PolarisPlatformManager", func() {
	var (
		ctx         context.Context
		diApp       *fxtest.App
		appStore    *bkmsapp.ApplicationStore
		configStore *polaris.ConfigStore
		db          *mongo.Database
		manager     *polaris.PolarisPlatformManager
		testAppID   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			polaris.FxModule,
			gstore.FxModule,
			fx.Provide(polaris.NewPolarisPlatformManager),
			fx.Populate(&manager),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		configStore = gstore.Of(db, polaris.ConfigMeta)
		testAppID = dbfactory.Application(ctx, appStore).ID
	})

	AfterEach(func() {
		_, _ = configStore.DeleteMany(ctx, polaris.ByApp(testAppID))
		diApp.RequireStop()
	})

	Describe("ListPolarisServiceInstances", func() {
		It("should return instances for configs available in the environment", func() {
			mockey.PatchConvey("list Polaris service instances", GinkgoT(), func() {
				mockey.Mock(polarisinfra.GetInstances).To(func(
					_ context.Context,
					namespace, serviceName string,
				) ([]*polarisinfra.Instance, error) {
					Expect(namespace).To(Equal("Test"))
					Expect(serviceName).To(Equal("service-a"))
					return []*polarisinfra.Instance{{IP: "127.0.0.1", Port: 8080}}, nil
				}).Build()

				config := &polaris.PolarisConfig{
					AppID: testAppID,
					Properties: polaris.Properties{
						PolarisName:      "service-a",
						PolarisNamespace: "Test",
						ServicePort:      8080,
					},
					ScopeEnvNames: []string{"dev"},
				}
				Expect(configStore.Create(ctx, config)).To(Succeed())

				instances, err := manager.ListPolarisServiceInstances(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(1))
				Expect(instances[0].ServiceName).To(Equal("service-a"))
				Expect(instances[0].Instances).To(HaveLen(1))
			})
		})
	})
})
