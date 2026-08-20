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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("FeatureEnvService", func() {
	var (
		ctx           context.Context
		appStore      bkmsapp.ApplicationStore
		envSvc        *bkmsenv.EnvService
		envStore      model.EnvironmentStore
		counterStore  model.FeatureEnvCounterStore
		service       *bkmsenv.FeatureEnvService
		nsInitializer *MockFeatureEnvNamespaceInitializer
		diApp         *fxtest.App
	)

	BeforeEach(func() {
		nsInitializer = NewMockFeatureEnvNamespaceInitializer(GinkgoT())
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			bkmsenv.FxModule,
			fxtest.WithTestLogger(GinkgoT()),
			fx.Replace(fx.Annotate(
				nsInitializer,
				fx.As(new(bkmsenv.FeatureEnvNamespaceInitializer)),
			)),
			fx.Populate(&appStore, &envSvc, &envStore, &counterStore, &service),
		)
		diApp.RequireStart()

		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		Expect(counterStore.DeleteAll(ctx)).To(Succeed())
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
		})
		Expect(err).NotTo(HaveOccurred())
		expectEnvironmentsEqual(nextEnv, deriveFeatureEnv(sourceEnv, model.Environment{
			ID:          nextEnv.ID,
			Name:        secondNamespace,
			DisplayName: "第二个",
			OwnerAppID:  app.ID,
			Creator:     "alice",
		}))
	})

	It("still persists the feature environment when namespace creation fails", func() {
		app := dbfactory.Application(ctx, appStore)
		sourceEnv := dbfactory.Env(ctx, envSvc, app.WorkspaceID)
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
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(featEnv.Name).To(Equal(namespace))

		storedEnv, err := envStore.GetByName(ctx, app.WorkspaceID, app.ID, namespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(storedEnv.ID).To(Equal(featEnv.ID))
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
