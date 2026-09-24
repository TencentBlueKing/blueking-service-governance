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

package env_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("FeatureEnvService", func() {
	var (
		ctx           context.Context
		appStore      bkmsapp.ApplicationStore
		envSvc        *bkmsenv.EnvService
		envStore      model.EnvironmentStore
		counterStore  model.FeatureEnvCounterStore
		variableStore envvars.ScopedEnvVarStore
		service       *bkmsenv.FeatureEnvService
		nsInitializer *MockFeatureEnvNamespaceInitializer
		diApp         *fxtest.App
	)

	BeforeEach(func() {
		bkmsenv.ResetHooksForTest()
		nsInitializer = NewMockFeatureEnvNamespaceInitializer(GinkgoT())
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			bkmsenv.FxModule,
			envvars.FxModule,
			fxtest.WithTestLogger(GinkgoT()),
			fx.Replace(fx.Annotate(
				nsInitializer,
				fx.As(new(bkmsenv.FeatureEnvNamespaceInitializer)),
			)),
			fx.Populate(&appStore, &envSvc, &envStore, &counterStore, &variableStore, &service),
		)
		diApp.RequireStart()

		ctx = context.Background()
		envvarhooks.RegisterDeleteHooks(variableStore)
	})

	AfterEach(func() {
		bkmsenv.ResetHooksForTest()
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		Expect(counterStore.DeleteAll(ctx)).To(Succeed())
		Expect(variableStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	It("lists app feature environments together with available source environments", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		deletedSourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		firstFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, app, sourceEnv)
		secondFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, app, deletedSourceEnv)

		otherApp := dbfactory.Application(ctx, appStore)
		otherSourceEnv := dbfactory.Env(ctx, envSvc, otherApp.WorkspaceID)
		_ = dbfactory.FeatEnv(ctx, envSvc, otherApp, otherSourceEnv)

		Expect(envStore.Delete(ctx, deletedSourceEnv.ID)).To(Succeed())

		featureEnvs, sourceEnvByFeatEnvName, err := bkmsenv.ListAppFeatEnvs(ctx, envStore, app)
		Expect(err).NotTo(HaveOccurred())
		Expect(featureEnvs).To(HaveLen(2))
		Expect([]string{featureEnvs[0].Name, featureEnvs[1].Name}).To(ConsistOf(
			firstFeatureEnv.Name,
			secondFeatureEnv.Name,
		))
		Expect(sourceEnvByFeatEnvName).To(HaveLen(1))
		Expect(sourceEnvByFeatEnvName).To(HaveKey(firstFeatureEnv.Name))
		Expect(sourceEnvByFeatEnvName[firstFeatureEnv.Name].ID).To(Equal(sourceEnv.ID))
		Expect(sourceEnvByFeatEnvName).NotTo(HaveKey(secondFeatureEnv.Name))
	})

	It("creates feature environments from a standard source environment", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		_, err := variableStore.CreateSimpleEnvScopeVar(ctx, *sourceEnv, "SOURCE_ONLY", "source-value", "")
		Expect(err).NotTo(HaveOccurred())
		sourceEnv.Cluster.IsFederation = true
		firstNamespace := fmt.Sprintf("feat-%s-1", app.ID)
		nsInitializer.EXPECT().Initialize(
			ctx, sourceEnv.Cluster.ClusterID, firstNamespace, expectedOwnerLabels(app, firstNamespace),
		).Return(nil).Once()

		featEnv, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "  登录联调  ",
			Creator:     "alice",
		})
		Expect(err).NotTo(HaveOccurred())

		// Create the first feat env and check
		expectedEnv := deriveFeatureEnv(sourceEnv, model.Environment{
			ID:          featEnv.ID,
			Name:        firstNamespace,
			DisplayName: "登录联调",
			OwnerAppID:  app.ID,
			Creator:     "alice",
		})
		expectEnvironmentsEqual(featEnv, expectedEnv)

		storedEnv, err := envStore.Get(ctx, featEnv.ID)
		Expect(err).NotTo(HaveOccurred())
		expectEnvironmentsEqual(storedEnv, expectedEnv)
		firstVars, err := variableStore.List(
			ctx,
			app.WorkspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(featEnv.Name)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstVars).To(BeEmpty())

		// Create another feat env and check
		secondNamespace := fmt.Sprintf("feat-%s-2", app.ID)
		nsInitializer.EXPECT().Initialize(
			ctx, sourceEnv.Cluster.ClusterID, secondNamespace, expectedOwnerLabels(app, secondNamespace),
		).Return(nil).Once()
		nextEnv, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "第二个",
			Creator:     "alice",
			CopyEnvVars: false,
		})
		Expect(err).NotTo(HaveOccurred())
		secondVars, err := variableStore.List(
			ctx,
			app.WorkspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(nextEnv.Name)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(secondVars).To(BeEmpty())
		expectEnvironmentsEqual(nextEnv, deriveFeatureEnv(sourceEnv, model.Environment{
			ID:          nextEnv.ID,
			Name:        secondNamespace,
			DisplayName: "第二个",
			OwnerAppID:  app.ID,
			Creator:     "alice",
		}))
	})

	It("copies custom variables as an independent snapshot including sensitive values", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		plainID, err := variableStore.CreateSimpleEnvScopeVar(
			ctx,
			*sourceEnv,
			"CONFIG_KEY",
			"source-value",
			"config desc",
		)
		Expect(err).NotTo(HaveOccurred())
		secretID, err := variableStore.Create(ctx, envvars.ScopedEnvVar{
			WorkspaceID: app.WorkspaceID, ScopeType: envvartypes.ScopeTypeEnv, ScopeValue: sourceEnv.Name,
			Key: "SECRET", Value: "source-secret", Description: "secret desc", IsSensitive: true,
		})
		Expect(err).NotTo(HaveOccurred())
		for _, variable := range []envvars.ScopedEnvVar{
			{ScopeType: envvartypes.ScopeTypeWorkspace, Key: "PUBLIC", Value: "public"},
			{ScopeType: envvartypes.ScopeTypeEnvType, ScopeValue: sourceEnv.Type, Key: "TYPE_VAR", Value: "type"},
			{ScopeType: envvartypes.ScopeTypeEnv, ScopeValue: sourceEnv.Name, Key: "BUILTIN", Value: "builtin", IsBuiltin: true},
		} {
			variable.WorkspaceID = app.WorkspaceID
			_, err = variableStore.Create(ctx, variable)
			Expect(err).NotTo(HaveOccurred())
		}
		namespace := fmt.Sprintf("feat-%s-1", app.ID)
		nsInitializer.EXPECT().Initialize(
			ctx, sourceEnv.Cluster.ClusterID, namespace, expectedOwnerLabels(app, namespace),
		).Return(nil).Once()

		featureEnv, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App: app, SourceEnv: sourceEnv, DisplayName: "copy variables", CopyEnvVars: true,
		})
		Expect(err).NotTo(HaveOccurred())
		copied, err := variableStore.List(
			ctx,
			app.WorkspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(featureEnv.Name)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(copied).To(HaveLen(2))
		Expect(copied[0].Key).To(Equal("CONFIG_KEY"))
		Expect(copied[0].Value).To(Equal("source-value"))
		Expect(copied[0].Description).To(Equal("config desc"))
		Expect(copied[0].IsSensitive).To(BeFalse())
		Expect(copied[0].ID).NotTo(Equal(plainID))
		Expect(copied[1].Key).To(Equal("SECRET"))
		Expect(copied[1].Value).To(Equal("source-secret"))
		Expect(copied[1].Description).To(Equal("secret desc"))
		Expect(copied[1].IsSensitive).To(BeTrue())
		Expect(copied[1].ID).NotTo(Equal(secretID))

		builtins, err := variableStore.List(ctx, app.WorkspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(featureEnv.Name)), envvars.WithOnlyBuiltin())
		Expect(err).NotTo(HaveOccurred())
		Expect(builtins).To(BeEmpty())

		Expect(variableStore.UpdateByID(ctx, app.WorkspaceID, plainID, envvars.ScopedEnvVarUpdateData{
			Key: "CONFIG_KEY", Value: lo.ToPtr("source-updated"),
		})).To(Succeed())
		featurePlain, err := variableStore.GetByID(ctx, app.WorkspaceID, copied[0].ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(featurePlain.Value).To(Equal("source-value"))
		Expect(variableStore.UpdateByID(ctx, app.WorkspaceID, copied[1].ID, envvars.ScopedEnvVarUpdateData{
			Key: "SECRET", Value: lo.ToPtr("feature-updated"),
		})).To(Succeed())
		sourceSecret, err := variableStore.GetByID(ctx, app.WorkspaceID, secretID)
		Expect(err).NotTo(HaveOccurred())
		Expect(sourceSecret.Value).To(Equal("source-secret"))
		Expect(variableStore.DeleteByID(ctx, app.WorkspaceID, copied[0].ID)).To(Succeed())
		sourcePlain, err := variableStore.GetByID(ctx, app.WorkspaceID, plainID)
		Expect(err).NotTo(HaveOccurred())
		Expect(sourcePlain.Value).To(Equal("source-updated"))
	})

	It("cleans the entire target scope through delete hooks before removing a failed environment", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		for _, key := range []string{"A_OK", "Z_FAIL"} {
			_, err := variableStore.CreateSimpleEnvScopeVar(ctx, *sourceEnv, key, "source", "")
			Expect(err).NotTo(HaveOccurred())
		}
		namespace := fmt.Sprintf("feat-%s-1", app.ID)
		// 目标作用域里先占住第二个 key，复制写完 A_OK 后在 Z_FAIL 上撞唯一索引。
		_, err := variableStore.CreateSimpleEnvScopeVar(
			ctx,
			model.Environment{WorkspaceID: app.WorkspaceID, Name: namespace},
			"Z_FAIL",
			"pre-existing",
			"",
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App: app, SourceEnv: sourceEnv, DisplayName: "partial copy failure", CopyEnvVars: true,
		})
		Expect(err).To(MatchError(envvars.ErrScopedEnvVarKeyConflict))
		_, err = envStore.GetByWorkspaceAndName(ctx, app.WorkspaceID, namespace)
		Expect(err).To(MatchError(model.ErrEnvNotFound))
		// 环境即将删除，目标作用域内的本次副本及并发写入记录都必须清理。
		copied, err := variableStore.List(ctx, app.WorkspaceID, envvars.WithScopes(envvartypes.ScopeEnv(namespace)))
		Expect(err).NotTo(HaveOccurred())
		Expect(copied).To(BeEmpty())
		sourceVars, err := variableStore.List(
			ctx,
			app.WorkspaceID,
			envvars.WithScopes(envvartypes.ScopeEnv(sourceEnv.Name)),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(sourceVars).To(HaveLen(2))
	})

	It("keeps a failed environment when cleanup fails and allows deletion to be retried", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		_, err := variableStore.Create(ctx, envvars.ScopedEnvVar{
			WorkspaceID: app.WorkspaceID, ScopeType: envvartypes.ScopeTypeEnv, ScopeValue: sourceEnv.Name,
			Key: "A_SECRET", Value: "source-secret", IsSensitive: true,
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = variableStore.CreateSimpleEnvScopeVar(ctx, *sourceEnv, "Z_FAIL", "source", "")
		Expect(err).NotTo(HaveOccurred())
		namespace := fmt.Sprintf("feat-%s-1", app.ID)
		_, err = variableStore.CreateSimpleEnvScopeVar(ctx,
			model.Environment{WorkspaceID: app.WorkspaceID, Name: namespace}, "Z_FAIL", "concurrent-value", "")
		Expect(err).NotTo(HaveOccurred())

		bkmsenv.ResetHooksForTest()
		allowCleanup := false
		cleanupErr := errors.New("cleanup failed")
		Expect(bkmsenv.RegisterDeleteHook("test.cleanup_failure", func(context.Context, model.Environment) error {
			if !allowCleanup {
				return cleanupErr
			}
			return nil
		})).To(BeTrue())
		envvarhooks.RegisterDeleteHooks(variableStore)

		_, err = service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App: app, SourceEnv: sourceEnv, DisplayName: "cleanup failure", CopyEnvVars: true,
		})
		Expect(err).To(MatchError(ContainSubstring("cleanup failed")))
		Expect(errors.Is(err, cleanupErr)).To(BeTrue())
		Expect(errors.Is(err, envvars.ErrScopedEnvVarKeyConflict)).To(BeFalse())
		Expect(err.Error()).To(ContainSubstring("delete this environment before retrying"))
		Expect(err.Error()).To(ContainSubstring(namespace))
		retainedEnv, getErr := envStore.GetByWorkspaceAndName(ctx, app.WorkspaceID, namespace)
		Expect(getErr).NotTo(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(retainedEnv.ID.Hex()))
		remaining, err := variableStore.List(ctx, app.WorkspaceID, envvars.WithScopes(envvartypes.ScopeEnv(namespace)))
		Expect(err).NotTo(HaveOccurred())
		Expect(remaining).To(HaveLen(2))
		Expect(remaining[0].Key).To(Equal("A_SECRET"))
		Expect(remaining[0].IsSensitive).To(BeTrue())

		allowCleanup = true
		Expect(envSvc.Delete(ctx, retainedEnv.ID)).To(Succeed())
		_, err = envStore.Get(ctx, retainedEnv.ID)
		Expect(err).To(MatchError(model.ErrEnvNotFound))
		remaining, err = variableStore.List(ctx, app.WorkspaceID, envvars.WithScopes(envvartypes.ScopeEnv(namespace)))
		Expect(err).NotTo(HaveOccurred())
		Expect(remaining).To(BeEmpty())
	})

	It("still persists the feature environment when namespace creation fails", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		_, err := variableStore.CreateSimpleEnvScopeVar(ctx, *sourceEnv, "CONFIG_KEY", "source-value", "")
		Expect(err).NotTo(HaveOccurred())
		namespace := fmt.Sprintf("feat-%s-1", app.ID)
		initErr := errors.New("namespace initialization failed")
		nsInitializer.EXPECT().Initialize(
			ctx, sourceEnv.Cluster.ClusterID, namespace, expectedOwnerLabels(app, namespace),
		).Return(initErr).Once()

		featEnv, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "登录联调",
			Creator:     "alice",
			CopyEnvVars: true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(featEnv.Name).To(Equal(namespace))

		storedEnv, err := envStore.GetByName(ctx, app.WorkspaceID, app.ID, namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(storedEnv.ID).To(Equal(featEnv.ID))
		copied, err := variableStore.List(ctx, app.WorkspaceID, envvars.WithScopes(envvartypes.ScopeEnv(featEnv.Name)))
		Expect(err).NotTo(HaveOccurred())
		Expect(copied).To(HaveLen(1))
		Expect(copied[0].Value).To(Equal("source-value"))
	})

	It("returns domain conflict error when creating a feature env with an occupied cluster namespace", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		occupiedNamespace := fmt.Sprintf("feat-%s-1", app.ID)
		occupiedEnvName := "occupied-" + occupiedNamespace
		occupiedWorkspaceID := "other-workspace-" + app.ID
		_, err := envStore.Create(ctx, &model.Environment{
			Name:        occupiedEnvName,
			DisplayName: occupiedEnvName,
			Type:        sourceEnv.Type,
			WorkspaceID: occupiedWorkspaceID,
			Cluster: model.BizCluster{
				ProjectCode: sourceEnv.Cluster.ProjectCode,
				ClusterID:   sourceEnv.Cluster.ClusterID,
				ClusterType: sourceEnv.Cluster.ClusterType,
				Namespace:   occupiedNamespace,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "联调",
			Creator:     "alice",
		})
		Expect(bkmsenv.IsErrEnvClusterNamespaceOccupied(err)).To(BeTrue())

		occupiedErr, ok := bkmsenv.GetEnvClusterNamespaceConflictInfo(err)
		Expect(ok).To(BeTrue())
		Expect(occupiedErr.ClusterID).To(Equal(sourceEnv.Cluster.ClusterID))
		Expect(occupiedErr.Namespace).To(Equal(occupiedNamespace))
		Expect(occupiedErr.OccupiedByEnvName).To(Equal(occupiedEnvName))
		Expect(occupiedErr.OccupiedByWorkspaceID).To(Equal(occupiedWorkspaceID))
	})

	It("rejects feature environments as source environments", func() {
		app := dbfactory.Application(ctx, appStore)
		standardEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
		sourceEnv := dbfactory.FeatEnv(ctx, envSvc, app, standardEnv)

		_, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "联调",
		})
		Expect(err).To(MatchError(ContainSubstring("SourceEnv must be a standard environment")))
	})

	It("rejects invalid display names and cross-workspace source environments", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, "other-workspace")

		_, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "联调",
		})
		Expect(err).To(MatchError(ContainSubstring("SourceEnv must belong to the same workspace as app")))

		sourceEnv.WorkspaceID = app.WorkspaceID
		_, err = service.Create(ctx, bkmsenv.CreateFeatureEnvInput{
			App:         app,
			SourceEnv:   sourceEnv,
			DisplayName: "   ",
		})
		Expect(err).To(MatchError(ContainSubstring("DisplayName must not be blank")))
	})

	It("reports required fields using their input field names", func() {
		_, err := service.Create(ctx, bkmsenv.CreateFeatureEnvInput{})
		Expect(errors.Is(err, bkmsenv.ErrInvalidFeatureEnvInput)).To(BeTrue())
		Expect(err).To(MatchError(And(
			ContainSubstring("App is required"),
			ContainSubstring("SourceEnv is required"),
		)))
	})
})

func expectEnvironmentsEqual(actual, expected *model.Environment) {
	GinkgoHelper()

	actualCopy, expectedCopy := *actual, *expected
	for _, env := range []*model.Environment{&actualCopy, &expectedCopy} {
		env.CreatedAt = time.Time{}
		env.UpdatedAt = time.Time{}
		env.AppIDs = nil
		env.Status = ""
	}
	Expect(actualCopy).To(Equal(expectedCopy))
}

func deriveFeatureEnv(sourceEnv *model.Environment, overrides model.Environment) *model.Environment {
	env := &model.Environment{
		ID:          overrides.ID,
		Name:        overrides.Name,
		DisplayName: overrides.DisplayName,
		Type:        sourceEnv.Type,
		WorkspaceID: sourceEnv.WorkspaceID,
		Kind:        model.EnvironmentKindFeature,
		OwnerAppID:  overrides.OwnerAppID,
		SourceEnvID: sourceEnv.ID,
		Cluster:     sourceEnv.Cluster,
		Description: overrides.Description,
		Creator:     overrides.Creator,
	}
	env.Cluster.Namespace = env.Name
	return env
}

func expectedOwnerLabels(app *bkmsapp.Application, envName string) map[string]string {
	return map[string]string{
		bkmsenv.FeatureEnvNSLabelWorkspaceID: app.WorkspaceID,
		bkmsenv.FeatureEnvNSLabelEnvName:     envName,
		bkmsenv.FeatureEnvNSLabelAppID:       app.ID,
		bkmsenv.FeatureEnvNSLabelController:  bkmsenv.FeatureEnvNSControllerValue,
	}
}
