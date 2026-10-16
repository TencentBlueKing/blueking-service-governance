package appdefaults_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default rule store", func() {
	var ctx context.Context
	var store *appdefaults.RuleStore

	BeforeEach(func() {
		ctx = context.Background()
		store = testutil.NewStore(GinkgoT(), appdefaults.RuleMeta)
	})

	AfterEach(func() {
		// Clear documents without Drop so migration-managed unique indexes survive.
		Expect(testutil.CleanupMetaColl(appdefaults.RuleMeta)).To(Succeed())
	})

	It("scopes rules by workspace, section and ID", func() {
		resources := resourcesRule("workspace-a", "production")
		devMode := devModeRule("workspace-a", "production", true)
		otherWorkspace := resourcesRule("workspace-b", "production")
		Expect(store.Create(ctx, resources)).To(Succeed())
		Expect(store.Create(ctx, devMode)).To(Succeed())
		Expect(store.Create(ctx, otherWorkspace)).To(Succeed())

		rules, err := store.List(ctx, appdefaults.ByWorkspace("workspace-a"))
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(HaveLen(2))
		resourcesRules, err := store.List(
			ctx,
			appdefaults.ByConfigType("workspace-a", appspec.AppSpecSectionResources),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(resourcesRules).To(HaveLen(1))
		Expect(resourcesRules[0].ID).To(Equal(resources.ID))

		stored, err := store.Get(ctx, appdefaults.ByID("workspace-a", appspec.AppSpecSectionResources, resources.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ID).To(Equal(resources.ID))

		_, err = store.Get(ctx, appdefaults.ByID("workspace-b", appspec.AppSpecSectionResources, resources.ID))
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())
		_, err = store.Get(ctx, appdefaults.ByID("workspace-a", appspec.AppSpecSectionDevMode, resources.ID))
		Expect(errors.Is(err, appdefaults.ErrRuleNotFound)).To(BeTrue())
	})

	It("rejects duplicate workspace section and environment type rules", func() {
		Expect(store.Create(ctx, resourcesRule("workspace-conflict", "production"))).To(Succeed())
		err := store.Create(ctx, resourcesRule("workspace-conflict", "production"))
		Expect(errors.Is(err, appdefaults.ErrRuleConflict)).To(BeTrue())
	})

	It("rejects overlapping environment types in the same section", func() {
		Expect(store.Create(ctx, resourcesRule("workspace-overlap", "production", "staging"))).To(Succeed())
		err := store.Create(ctx, resourcesRule("workspace-overlap", "staging", "test"))
		Expect(errors.Is(err, appdefaults.ErrRuleConflict)).To(BeTrue())

		Expect(store.Create(ctx, resourcesRule("workspace-overlap", "test"))).To(Succeed())
		Expect(store.Create(ctx, devModeRule("workspace-overlap", "production", true))).To(Succeed())
	})
})
