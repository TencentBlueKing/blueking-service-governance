package hooks_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var _ = Describe("Application default lifecycle hooks", func() {
	var ctx context.Context
	var ruleStore *appdefaults.RuleStore
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = context.Background()
		bkmsworkspace.ResetLifecycleHooksForTest()

		db := testutil.NewDB(GinkgoT())
		ruleStore = store.Of(db, appdefaults.RuleMeta)
		workspaceStore = store.Of(db, bkmsworkspace.Meta)
		appdefaultshooks.RegisterPreDeleteHooks(db)
	})

	AfterEach(func() {
		if workspace != nil {
			Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspace.ID)).To(Succeed())
		}
		Expect(testutil.CleanupMetaColl(appdefaults.RuleMeta)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
	})

	It("removes rules when their workspace is deleted", func() {
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		workspaceID := workspace.ID
		enabled := true
		Expect(ruleStore.Create(ctx, &appdefaults.Rule{
			WorkspaceID: workspaceID,
			ConfigType:  appspec.AppSpecSectionDevMode,
			EnvTypes:    []string{"production"},
			Spec: &appspec.AppSpec{
				DevMode: &appspec.DevModeSpec{Enabled: &enabled},
			},
		})).To(Succeed())

		Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspaceID)).To(Succeed())
		workspace = nil

		rules, err := ruleStore.List(ctx, appdefaults.ByWorkspace(workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})
})
