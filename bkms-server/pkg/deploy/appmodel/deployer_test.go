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

package appmodel

import (
	"context"
	"time"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

// newGameDeploy 创建用于测试的 GameDeployment，支持预设两个层级的 Annotations
func newGameDeploy(objAnnotations, tplAnnotations map[string]string) *tkex.GameDeployment {
	return &tkex.GameDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app", Annotations: objAnnotations},
		Spec: tkex.GameDeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: tplAnnotations},
			},
		},
	}
}

var _ = Describe("injectDeployID", func() {
	deployer := &Deployer{}

	newResult := func(objAnnotations, tplAnnotations map[string]string) (*workload.BuildResult, *tkex.GameDeployment) {
		gd := newGameDeploy(objAnnotations, tplAnnotations)
		return &workload.BuildResult{MainWorkload: gd}, gd
	}

	// expectBothLevels 断言两个层级的 Annotations 都包含预期的 deployID
	expectBothLevels := func(gd *tkex.GameDeployment, deployID string) {
		Expect(gd.Annotations[AnnotationKeyDeployID]).To(Equal(deployID))
		Expect(gd.Spec.Template.Annotations[AnnotationKeyDeployID]).To(Equal(deployID))
	}

	It("should initialize nil Annotations and inject deployID at both levels", func() {
		result, gd := newResult(nil, nil)
		deployer.injectDeployID(result, "deploy-001")
		expectBothLevels(gd, "deploy-001")
	})

	It("should append deployID without overwriting existing Annotations", func() {
		result, gd := newResult(
			map[string]string{"existing-key": "existing-value"},
			map[string]string{"pod-existing-key": "pod-existing-value"},
		)
		deployer.injectDeployID(result, "deploy-002")
		expectBothLevels(gd, "deploy-002")
		Expect(gd.Annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
		Expect(gd.Spec.Template.Annotations).To(HaveKeyWithValue("pod-existing-key", "pod-existing-value"))
	})

	It("should overwrite the previous deployID with the new one", func() {
		result, gd := newResult(nil, nil)
		deployer.injectDeployID(result, "deploy-first")
		deployer.injectDeployID(result, "deploy-second")
		expectBothLevels(gd, "deploy-second")
	})
})

var _ = Describe("syncBuildAutoDeployStatus", func() {
	It("should update linked build auto deploy record by deploy ID", func() {
		ctx := context.Background()
		db := testutil.NewDB(GinkgoT())
		records := store.Of(db, autodeploy.RecordMeta)
		appID := "app-" + stringx.Random(6)
		DeferCleanup(func() { _, _ = records.DeleteMany(ctx, store.Where("appID", appID)) })

		deployID := bson.NewObjectID()
		record := &autodeploy.Record{
			AppID:    appID,
			BuildID:  "build-" + stringx.Random(6),
			DeployID: deployID.Hex(),
			Stage:    autodeploy.StageDeploy,
			Status:   "deploying",
		}
		Expect(records.Create(ctx, record)).To(Succeed())
		deployer := &Deployer{
			db:  db,
			app: &bkmsapp.Application{ID: appID},
		}
		deployRecord := &Record{
			ID:      deployID,
			Status:  StatusUninstalled,
			Message: "done",
			EndedAt: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC),
		}

		err := deployer.syncBuildAutoDeployStatus(ctx, deployRecord)

		Expect(err).NotTo(HaveOccurred())
		got, err := records.Get(ctx, autodeploy.ByDeploy(appID, deployID.Hex()))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Status).To(Equal(string(StatusUninstalled)))
		Expect(got.Stage).To(Equal(autodeploy.StageDeploy))
		Expect(got.Message).To(Equal("done"))
		Expect(got.EndedAt).To(BeTemporally("==", deployRecord.EndedAt))
	})
})
