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

package model_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("GetSnapshot", func() {
	var (
		ctx       context.Context
		db        *mongo.Database
		metas     *model.MetadataStore
		bindings  *model.EnvBindingStore
		testAppID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		metas = testutil.NewStore(GinkgoT(), model.MetadataMeta)
		bindings = testutil.NewStore(GinkgoT(), model.EnvBindingMeta)
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_, _ = bindings.DeleteMany(ctx, model.ByApp(testAppID))
		_ = metas.DeleteIfExists(ctx, model.ByApp(testAppID))
	})

	Context("when metadata does not exist", func() {
		It("should return nil, nil", func() {
			snap, err := model.GetSnapshot(ctx, db, testAppID, "dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(snap).To(BeNil())
		})
	})

	Context("when metadata exists but env binding does not", func() {
		It("should return nil, nil", func() {
			createTestMetadata(ctx, metas, testAppID)

			snap, err := model.GetSnapshot(ctx, db, testAppID, "dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(snap).To(BeNil())
		})
	})

	Context("when both metadata and env binding exist", func() {
		It("should return the aggregated snapshot", func() {
			createTestMetadata(ctx, metas, testAppID)
			createTestEnvBinding(ctx, bindings, testAppID, "dev")

			snap, err := model.GetSnapshot(ctx, db, testAppID, "dev")
			Expect(err).NotTo(HaveOccurred())
			Expect(snap).NotTo(BeNil())
			Expect(snap.Metadata.AppID).To(Equal(testAppID))
			Expect(snap.EnvBinding.EnvName).To(Equal("dev"))
			Expect(snap.EnvBinding.BscpAppID).To(Equal("1001"))
		})
	})
})

// createTestMetadata 创建一个满足 MetadataStore 校验的 Metadata。
func createTestMetadata(ctx context.Context, metas *model.MetadataStore, appID string) {
	err := metas.Create(ctx, &model.Metadata{
		AppID:        appID,
		Enable:       true,
		MountPath:    "/data/bscp",
		WorkloadName: "test-workload",
		Operator:     "tester",
	})
	Expect(err).NotTo(HaveOccurred())
}

// createTestEnvBinding 创建一个满足 EnvBindingStore 校验的 EnvBinding。
func createTestEnvBinding(ctx context.Context, bindings *model.EnvBindingStore, appID, envName string) {
	err := bindings.Create(ctx, &model.EnvBinding{
		AppID:       appID,
		EnvName:     envName,
		BscpEnvID:   "1",
		BscpEnvName: "dev",
		BscpAppID:   "1001",
		Operator:    "tester",
	})
	Expect(err).NotTo(HaveOccurred())
}
