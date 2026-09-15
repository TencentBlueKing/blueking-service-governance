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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("Store", func() {
	var store model.Store
	var ctx context.Context
	var testAppID string
	var diApp *fxtest.App

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			model.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_ = store.DeleteEnvBindingsByApp(ctx, testAppID)
		_ = store.DeleteMetadata(ctx, testAppID)
		diApp.RequireStop()
	})

	Describe("GetSnapshot", func() {
		Context("when metadata does not exist", func() {
			It("should return nil, nil", func() {
				snap, err := store.GetSnapshot(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(snap).To(BeNil())
			})
		})

		Context("when metadata exists but env binding does not", func() {
			It("should return nil, nil", func() {
				createTestMetadata(ctx, store, testAppID)

				snap, err := store.GetSnapshot(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(snap).To(BeNil())
			})
		})

		Context("when both metadata and env binding exist", func() {
			It("should return the aggregated snapshot", func() {
				createTestMetadata(ctx, store, testAppID)
				createTestEnvBinding(ctx, store, testAppID, "dev")

				snap, err := store.GetSnapshot(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(snap).NotTo(BeNil())
				Expect(snap.Metadata.AppID).To(Equal(testAppID))
				Expect(snap.EnvBinding.EnvName).To(Equal("dev"))
				Expect(snap.EnvBinding.BscpAppID).To(Equal("1001"))
			})
		})
	})

	Describe("FeatureFlag", func() {
		Context("when upserting an enabled flag", func() {
			It("should be retrievable and enabled", func() {
				err := store.UpsertFeatureFlag(ctx, &model.FeatureFlag{
					AppID:    testAppID,
					Enabled:  true,
					Operator: "tester",
				})
				Expect(err).NotTo(HaveOccurred())

				flag, err := store.GetFeatureFlag(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(flag.Enabled).To(BeTrue())
				Expect(flag.Operator).To(Equal("tester"))
			})
		})

		Context("when upserting the same app again", func() {
			It("should update the existing flag", func() {
				err := store.UpsertFeatureFlag(ctx, &model.FeatureFlag{
					AppID:    testAppID,
					Enabled:  true,
					Operator: "tester",
				})
				Expect(err).NotTo(HaveOccurred())

				err = store.UpsertFeatureFlag(ctx, &model.FeatureFlag{
					AppID:    testAppID,
					Enabled:  false,
					Operator: "admin",
				})
				Expect(err).NotTo(HaveOccurred())

				flag, err := store.GetFeatureFlag(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(flag.Enabled).To(BeFalse())
				Expect(flag.Operator).To(Equal("admin"))
			})
		})

		Context("when feature flag does not exist", func() {
			It("should return ErrFeatureFlagNotFound", func() {
				_, err := store.GetFeatureFlag(ctx, testAppID)
				Expect(err).To(MatchError(model.ErrFeatureFlagNotFound))
			})
		})
	})
})

// createTestMetadata 创建一个满足 MetadataStore 校验的 Metadata。
func createTestMetadata(ctx context.Context, store model.Store, appID string) {
	err := store.CreateMetadata(ctx, &model.Metadata{
		AppID:        appID,
		BscpBizID:    "12345",
		MountPath:    "/data/bscp",
		CredentialID: "cred-1",
		Token:        "test-token",
		FeedAddr:     "bscp-feed.example.com:9500",
		WorkloadName: "test-workload",
		Operator:     "tester",
	})
	Expect(err).NotTo(HaveOccurred())
}

// createTestEnvBinding 创建一个满足 EnvBindingStore 校验的 EnvBinding。
func createTestEnvBinding(ctx context.Context, store model.Store, appID, envName string) {
	err := store.CreateEnvBinding(ctx, &model.EnvBinding{
		AppID:       appID,
		EnvName:     envName,
		BscpEnvID:   "1",
		BscpEnvName: "dev",
		BscpAppID:   "1001",
		Operator:    "tester",
	})
	Expect(err).NotTo(HaveOccurred())
}
