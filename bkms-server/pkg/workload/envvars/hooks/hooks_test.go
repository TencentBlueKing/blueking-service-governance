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

package hooks_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Env delete hooks from envvars", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var envSvc *bkmsenv.EnvService
	var db *mongo.Database

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsenv.FxModule,
			envvars.FxModule,
			depenvvars.FxModule,
			fx.Populate(
				&envSvc,
			),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(envvars.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		Expect(testutil.CleanupCollection("environments")).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	It("should clean env scoped vars", func() {
		workspaceID := "test-workspace-" + stringx.Random(6)
		environment := dbfactory.Env(ctx, envSvc, workspaceID)
		envName := environment.Name
		otherEnvName := "env-" + stringx.Random(6)

		var err error
		for _, item := range []envvars.ScopedEnvVar{
			{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  envName,
				Key:         "ENV_KEY_A",
				Value:       "env-value-a",
			},
			{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  envName,
				Key:         "ENV_KEY_B",
				Value:       "env-value-b",
			},
			{
				WorkspaceID: workspaceID,
				ScopeType:   envvartypes.ScopeTypeEnv,
				ScopeValue:  otherEnvName,
				Key:         "OTHER_ENV_KEY",
				Value:       "other-env-value",
			},
		} {
			err = store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &item)
			Expect(err).NotTo(HaveOccurred())
		}

		hook := envvarhooks.NewCleanupScopedEnvVarsByEnvHook(db)
		err = hook(ctx, *environment)
		Expect(err).NotTo(HaveOccurred())

		// Env-scoped variables belonging to the deleted environment should be cleaned by the hook.
		deletedEnvVars, err := envvars.List(
			ctx, db,
			workspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(envName)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedEnvVars).To(BeEmpty())

		// Variables in other env scopes must stay untouched.
		otherEnvVars, err := envvars.List(
			ctx, db,
			workspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(otherEnvName)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(otherEnvVars).To(HaveLen(1))
		Expect(otherEnvVars[0].Key).To(Equal("OTHER_ENV_KEY"))
	})
})
