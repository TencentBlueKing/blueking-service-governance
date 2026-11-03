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

package status_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	helmrelease "helm.sh/helm/v3/pkg/release"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	buildimage "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
)

var _ = Describe("DeployStatusService.GetLatestDeploy", func() {
	var (
		ctx      context.Context
		diApp    *fxtest.App
		db       *mongo.Database
		appStore *bkmsapp.ApplicationStore
		svc      *DeployStatusService
		trpcApp  *bkmsapp.Application
		testEnv  *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		var envSvc *bkmsenv.EnvService
		diApp = fxtest.New(GinkgoT(), appcfg.FxModule, bkmsenv.FxModule, fx.Populate(&envSvc))
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		svc = NewDeployStatusService(db)

		trpcApp, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{AppStore: appStore}, nil)
		testEnv = dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	It("returns ErrDeployRecordNotFound when nothing has been deployed", func() {
		_, err := svc.GetLatestDeploy(ctx, trpcApp, testEnv.Name, "")
		Expect(errors.Is(err, ErrDeployRecordNotFound)).To(BeTrue())
	})

	It("returns the app model deploy and its pod location", func() {
		dbfactory.AppModelDeployRecord(ctx, db, trpcApp, testEnv, &dbfactory.AppModelDeployRecordOpts{
			Status:        appmodeldeploy.StatusPollingTimeout,
			ClusterID:     "BCS-K8S-00001",
			Namespace:     "ns-a",
			LabelSelector: map[string]string{"app": trpcApp.ID},
		})

		latest, err := svc.GetLatestDeploy(ctx, trpcApp, testEnv.Name, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(latest.Source).To(Equal(DeploySourceAppModel))
		Expect(latest.Phase).To(Equal(DeployPhaseFailed))
		Expect(latest.Target).NotTo(BeNil())
		Expect(latest.Target.DeployID).NotTo(BeEmpty())
		Expect(latest.Target.ClusterID).To(Equal("BCS-K8S-00001"))
		Expect(latest.Target.Namespace).To(Equal("ns-a"))
		Expect(latest.Target.LabelSelector).To(Equal(map[string]string{"app": trpcApp.ID}))
	})

	It("prefers a newer build-stage failure but keeps the previous deploy as target", func() {
		dbfactory.AppModelDeployRecord(ctx, db, trpcApp, testEnv, &dbfactory.AppModelDeployRecordOpts{
			Status:    appmodeldeploy.StatusDeployed,
			Namespace: "ns-a",
		})
		dbfactory.BuildAutoDeployRecord(ctx, db, trpcApp, testEnv, &dbfactory.BuildAutoDeployRecordOpts{
			BuildID: "build-1",
			Stage:   autodeploy.StageBuild,
			Status:  string(buildimage.StatusPollingTimeout),
		})

		latest, err := svc.GetLatestDeploy(ctx, trpcApp, testEnv.Name, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(latest.Source).To(Equal(DeploySourceBuildAutoDeploy))
		Expect(latest.Stage).To(Equal(autodeploy.StageBuild))
		Expect(latest.BuildID).To(Equal("build-1"))
		Expect(latest.Phase).To(Equal(DeployPhaseFailed))
		Expect(latest.Target).NotTo(BeNil())
		Expect(latest.Target.Namespace).To(Equal("ns-a"))
	})

	It("has no target when the first auto deploy is still building", func() {
		dbfactory.BuildAutoDeployRecord(ctx, db, trpcApp, testEnv, &dbfactory.BuildAutoDeployRecordOpts{
			Stage:  autodeploy.StageBuild,
			Status: string(buildimage.StatusRunning),
		})

		latest, err := svc.GetLatestDeploy(ctx, trpcApp, testEnv.Name, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(latest.Phase).To(Equal(DeployPhaseInProgress))
		Expect(latest.Target).To(BeNil())
	})

	It("follows the app model status once the auto deploy enters the deploy stage", func() {
		dbfactory.BuildAutoDeployRecord(ctx, db, trpcApp, testEnv, &dbfactory.BuildAutoDeployRecordOpts{
			Stage:  autodeploy.StageDeploy,
			Status: string(appmodeldeploy.StatusDeployed),
		})

		latest, err := svc.GetLatestDeploy(ctx, trpcApp, testEnv.Name, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(latest.Phase).To(Equal(DeployPhaseSucceeded))
	})

	DescribeTable("maps helm release status to a phase",
		func(status helmrelease.Status, want DeployPhase) {
			helmApp := dbfactory.HelmApplication(ctx, &dbfactory.HelmApplicationStores{AppStore: appStore}, nil)
			dbfactory.HelmDeployRecord(ctx, db, helmApp, testEnv, &dbfactory.HelmDeployRecordOpts{Status: status})

			latest, err := svc.GetLatestDeploy(ctx, helmApp, testEnv.Name, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(latest.Source).To(Equal(DeploySourceHelm))
			Expect(latest.Phase).To(Equal(want))
			Expect(latest.Target).NotTo(BeNil())
			Expect(latest.Target.LabelSelector).To(BeEmpty())
		},
		Entry("pending upgrade", helmrelease.StatusPendingUpgrade, DeployPhaseInProgress),
		Entry("deployed", helmrelease.StatusDeployed, DeployPhaseSucceeded),
		Entry("failed", helmrelease.StatusFailed, DeployPhaseFailed),
		Entry("uninstalled", helmrelease.StatusUninstalled, DeployPhaseOther),
	)
})
