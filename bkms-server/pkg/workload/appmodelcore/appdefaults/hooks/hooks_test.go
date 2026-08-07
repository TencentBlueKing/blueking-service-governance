package hooks_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default lifecycle hooks", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var ruleStore appdefaults.RuleStore
	var workspaceStore bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			appdefaults.FxModule,
			bkmsworkspace.FxModule,
			fx.Populate(&ruleStore, &workspaceStore),
		)
		diApp.RequireStart()
		appdefaultshooks.RegisterPreDeleteHooks(ruleStore)
	})

	AfterEach(func() {
		if workspace != nil {
			Expect(workspaceStore.Delete(ctx, workspace.ID)).To(Succeed())
		}
		Expect(ruleStore.Drop(ctx)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
		diApp.RequireStop()
	})

	It("removes rules when their workspace is deleted", func() {
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		workspaceID := workspace.ID
		enabled := true
		Expect(ruleStore.Create(ctx, &appdefaults.Rule{
			WorkspaceID: workspaceID,
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvType:     "production",
			Spec: &appspec.AppSpec{
				DevMode: &appspec.DevModeSpec{Enabled: &enabled},
			},
		})).To(Succeed())

		Expect(workspaceStore.Delete(ctx, workspaceID)).To(Succeed())
		workspace = nil

		rules, err := ruleStore.List(ctx, workspaceID)
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})
})
