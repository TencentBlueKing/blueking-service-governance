package appdefaults_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default rule store", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var store appdefaults.RuleStore

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			appdefaults.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(store.Drop(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	It("supports scoped CRUD operations", func() {
		resources := resourcesRule("workspace-a", "production")
		devMode := devModeRule("workspace-a", "production", true)
		otherWorkspace := resourcesRule("workspace-b", "production")
		Expect(store.Create(ctx, resources)).To(Succeed())
		Expect(store.Create(ctx, devMode)).To(Succeed())
		Expect(store.Create(ctx, otherWorkspace)).To(Succeed())

		rules, err := store.List(ctx, "workspace-a")
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(HaveLen(2))
		resourcesRules, err := store.ListByConfigType(
			ctx,
			"workspace-a",
			appspec.AppSpecSectionResources,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(resourcesRules).To(HaveLen(1))
		Expect(resourcesRules[0].ID).To(Equal(resources.ID))

		stored, err := store.Get(ctx, "workspace-a", appspec.AppSpecSectionResources, resources.ID)
		Expect(err).NotTo(HaveOccurred())
		updated := *stored
		updated.EnvType = "test"
		updated.Spec = &appspec.AppSpec{
			Resources: resourcesSpec(3, "1", "2", "2Gi", "4Gi"),
		}
		Expect(store.Update(ctx, "workspace-a", appspec.AppSpecSectionResources, &updated)).To(Succeed())

		stored, err = store.Get(ctx, "workspace-a", appspec.AppSpecSectionResources, resources.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.EnvType).To(Equal("test"))
		Expect(*stored.Spec.Resources.Replicas).To(Equal(int32(3)))

		_, err = store.Get(ctx, "workspace-b", appspec.AppSpecSectionResources, resources.ID)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())
		_, err = store.Get(ctx, "workspace-a", appspec.AppSpecSectionDevMode, resources.ID)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())

		deleted, err := store.Delete(ctx, "workspace-a", appspec.AppSpecSectionResources, resources.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(deleted.ID).To(Equal(resources.ID))
		_, err = store.Get(ctx, "workspace-a", appspec.AppSpecSectionResources, resources.ID)
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())
	})

	It("rejects duplicate workspace section and environment type rules", func() {
		Expect(store.Create(ctx, resourcesRule("workspace-conflict", "production"))).To(Succeed())
		err := store.Create(ctx, resourcesRule("workspace-conflict", "production"))
		Expect(errors.Is(err, appdefaults.ErrRuleConflict)).To(BeTrue())
	})
})
