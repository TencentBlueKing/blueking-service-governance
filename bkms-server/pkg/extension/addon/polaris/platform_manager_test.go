package polaris_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	polarisinfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

var _ = Describe("PolarisPlatformManager", func() {
	var (
		ctx       context.Context
		diApp     *fxtest.App
		appStore  bkmsapp.ApplicationStore
		store     polaris.PolarisConfigStore
		manager   *polaris.PolarisPlatformManager
		testAppID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			polaris.FxModule,
			depsvcmodel.FxModule,
			fx.Provide(polaris.NewPolarisPlatformManager),
			fx.Populate(&appStore, &store, &manager),
		)
		diApp.RequireStart()
		testAppID = dbfactory.Application(ctx, appStore).ID
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, testAppID)
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
				Expect(store.Create(ctx, config)).To(Succeed())

				instances, err := manager.ListPolarisServiceInstances(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(1))
				Expect(instances[0].ServiceName).To(Equal("service-a"))
				Expect(instances[0].Instances).To(HaveLen(1))
			})
		})
	})
})
