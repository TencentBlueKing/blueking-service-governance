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

package installtask_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/installtask"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Meta", func() {
	var (
		ctx       context.Context
		taskStore *installtask.Store
		clusterID string
	)

	newTask := func(namespace, addonName string) *installtask.InstallTask {
		return &installtask.InstallTask{
			ClusterID:    clusterID,
			Namespace:    namespace,
			AddonName:    addonName,
			ChartVersion: "1.0.0",
			Status:       installtask.StatusRunning,
			StartedAt:    time.Now(),
		}
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		taskStore = testutil.NewStore(GinkgoT(), installtask.Meta)
		clusterID = "BCS-K8S-" + stringx.Random(5)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(installtask.Meta)).To(Succeed())
	})

	It("should allow only one running task per addon in a cluster namespace", func() {
		first := newTask("bcs-system", "bcs-polaris-operator")
		Expect(taskStore.Create(ctx, first)).To(Succeed())

		err := taskStore.Create(ctx, newTask("bcs-system", "bcs-polaris-operator"))
		Expect(err).To(MatchError(installtask.ErrRunning))

		// 其他 Addon、其他命名空间不受影响
		Expect(taskStore.Create(ctx, newTask("bcs-system", "general-pod-autoscaler"))).To(Succeed())
		Expect(taskStore.Create(ctx, newTask("other-ns", "bcs-polaris-operator"))).To(Succeed())

		// 第一个任务结束后可以再发起
		Expect(taskStore.Update(ctx,
			map[string]any{"status": installtask.StatusSucceeded, "finishedAt": time.Now()},
			store.ByID(first.ID),
		)).To(Succeed())
		Expect(taskStore.Create(ctx, newTask("bcs-system", "bcs-polaris-operator"))).To(Succeed())
	})

	It("should reject running tasks on the same addon across tenants", func() {
		Expect(taskStore.Create(ctx, newTask("bcs-system", "bcs-polaris-operator"))).To(Succeed())

		ctxOther := tenant.WithTenantID(context.Background(), "tenant-"+stringx.Random(6))
		err := taskStore.Create(ctxOther, newTask("bcs-system", "bcs-polaris-operator"))
		Expect(err).To(MatchError(installtask.ErrRunning))
	})

	It("should list the latest task of an addon first", func() {
		older := newTask("bcs-system", "bcs-polaris-operator")
		older.Status = installtask.StatusFailed
		older.CreatedAt = time.Now().Add(-time.Minute)
		Expect(taskStore.Create(ctx, older)).To(Succeed())
		newer := newTask("bcs-system", "bcs-polaris-operator")
		newer.CreatedAt = time.Now()
		Expect(taskStore.Create(ctx, newer)).To(Succeed())

		latest, err := taskStore.Get(ctx,
			installtask.ByTarget(clusterID, "bcs-system", "bcs-polaris-operator"),
			installtask.LatestFirst(),
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(latest.ID).To(Equal(newer.ID))
	})
})
