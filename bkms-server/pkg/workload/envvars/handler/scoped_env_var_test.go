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

package handler

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("validateScopedEnvVarScopeValue", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var envSvc *bkmsenv.EnvService
	var envStore envmodel.EnvironmentStore
	var handler *Handler
	var standardEnv, featureEnv *envmodel.Environment
	var workspaceID string

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(GinkgoT(), bkmsenv.FxModule, fx.Populate(&envSvc, &envStore))
		diApp.RequireStart()
		DeferCleanup(diApp.RequireStop)
		workspaceID = "envvars-test-" + stringx.Random(8)
		standardEnv = dbfactory.Env(ctx, envSvc, workspaceID)
		DeferCleanup(func() { Expect(envStore.Delete(ctx, standardEnv.ID)).To(Succeed()) })
		featureEnv = dbfactory.FeatEnv(ctx, envSvc, &bkmsapp.Application{
			ID: "owner-app", WorkspaceID: workspaceID,
		}, standardEnv)
		DeferCleanup(func() { Expect(envStore.Delete(ctx, featureEnv.ID)).To(Succeed()) })
		handler = &Handler{registry: &storereg.Registry{EnvStore: envStore}}
	})

	It("accepts standard and feature environments in the workspace", func() {
		for _, scope := range []envvartypes.ScopedEnvVarScope{
			envvartypes.ScopeEnv(standardEnv.Name),
			envvartypes.ScopeEnv(featureEnv.Name),
		} {
			Expect(handler.validateScopedEnvVarScopeValue(ctx, workspaceID, scope)).To(Succeed())
		}
	})

	It("rejects environments from another workspace", func() {
		for _, environment := range []*envmodel.Environment{standardEnv, featureEnv} {
			err := handler.validateScopedEnvVarScopeValue(ctx, "other-workspace",
				envvartypes.ScopeEnv(environment.Name))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		}
	})

	It("rejects a missing environment", func() {
		err := handler.validateScopedEnvVarScopeValue(ctx, workspaceID, envvartypes.ScopeEnv("missing-env"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})
})

var _ = Describe("ensureAppSupportsDefinedEnvVars", func() {
	It("allows app model types", func() {
		err := ensureAppSupportsDefinedEnvVars(&bkmsapp.Application{Type: bkmsapp.AppTypeTRPC})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects non app model types", func() {
		err := ensureAppSupportsDefinedEnvVars(&bkmsapp.Application{Type: bkmsapp.AppTypeHelm})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("does not support app-defined env vars"))
	})
})
