package storereg_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
)

var _ = Describe("Store registry", func() {
	AfterEach(func() {
		storereg.Reset()
	})

	It("should register resource lifecycle hooks during initialization", func() {
		storereg.Init(context.Background())

		Expect(
			bkmsenv.IsDeleteHookRegistered(envvarhooks.CleanupScopedEnvVarsByEnvHookName),
		).To(BeTrue(), "envvars cleanup hook must be registered by store registry")
		Expect(
			bkmsworkspace.IsPreDeleteHookRegistered(appdefaultshooks.CleanupRulesByWorkspaceHookName),
		).To(BeTrue(), "workspace AppSpec rule cleanup hook must be registered by store registry")
	})
})
