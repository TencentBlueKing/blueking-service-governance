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

package service_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("Manager", func() {
	var store model.Store
	var mgr *service.Manager
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

		var err error
		mgr, err = service.NewManager(auth.User{ID: "test-user"}, store)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = store.DeleteEnvBindingsByApp(ctx, testAppID)
		_ = store.DeleteMetadata(ctx, testAppID)
		diApp.RequireStop()
	})

	// newInitParams 构造一个 InitMetadata 入参。
	newInitParams := func() *service.InitMetadataParams {
		return &service.InitMetadataParams{
			AppID:          testAppID,
			BscpBizID:      "12345",
			BscpProjectID:  "12",
			BscpProjectKey: "BK-BSCP-00012",
			Operator:       "tester",
		}
	}

	Describe("InitMetadata", func() {
		Context("when called for the first time", func() {
			It("should create metadata with credential and post hook", func() {
				meta, err := mgr.InitMetadata(ctx, newInitParams())
				Expect(err).NotTo(HaveOccurred())
				Expect(meta).NotTo(BeNil())
				Expect(meta.AppID).To(Equal(testAppID))
				Expect(meta.BscpBizID).To(Equal("12345"))
				Expect(meta.CredentialID).NotTo(BeEmpty())
				Expect(meta.CredentialName).To(Equal("bkms-credential"))
				Expect(meta.PostHookID).NotTo(BeEmpty())
				Expect(meta.Operator).To(Equal("tester"))
			})
		})

		Context("when called twice", func() {
			It("should be idempotent", func() {
				first, err := mgr.InitMetadata(ctx, newInitParams())
				Expect(err).NotTo(HaveOccurred())

				second, err := mgr.InitMetadata(ctx, newInitParams())
				Expect(err).NotTo(HaveOccurred())

				Expect(second.CredentialID).To(Equal(first.CredentialID))
				Expect(second.PostHookID).To(Equal(first.PostHookID))
			})
		})
	})

	Describe("GetOrCreateCredential", func() {
		Context("when the credential already exists", func() {
			It("should return it", func() {
				cred, err := mgr.GetOrCreateCredential(ctx, "12345", 12)
				Expect(err).NotTo(HaveOccurred())
				Expect(cred).NotTo(BeNil())
				Expect(cred.Name).To(Equal("bkms-credential"))
			})
		})

		Context("when called twice", func() {
			It("should be idempotent", func() {
				first, err := mgr.GetOrCreateCredential(ctx, "12345", 12)
				Expect(err).NotTo(HaveOccurred())

				second, err := mgr.GetOrCreateCredential(ctx, "12345", 12)
				Expect(err).NotTo(HaveOccurred())

				Expect(second.ID).To(Equal(first.ID))
			})
		})
	})

	Describe("GetOrCreatePostHook", func() {
		Context("when called for the first time", func() {
			It("should create a post hook and return a non-zero id", func() {
				hookID, err := mgr.GetOrCreatePostHook(ctx, "12345", 12, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(hookID).NotTo(BeZero())
			})
		})

		Context("when called twice", func() {
			It("should be idempotent", func() {
				first, err := mgr.GetOrCreatePostHook(ctx, "12345", 12, testAppID)
				Expect(err).NotTo(HaveOccurred())

				second, err := mgr.GetOrCreatePostHook(ctx, "12345", 12, testAppID)
				Expect(err).NotTo(HaveOccurred())

				Expect(second).To(Equal(first))
			})
		})
	})

	Describe("GetSnapshot", func() {
		BeforeEach(func() {
			createTestMetadata(ctx, store, testAppID)
			createTestEnvBinding(ctx, store, testAppID, "dev")
		})

		Context("when both metadata and env binding exist", func() {
			It("should return the aggregated snapshot", func() {
				snap, err := mgr.GetSnapshot(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(snap).NotTo(BeNil())
				Expect(snap.Metadata.AppID).To(Equal(testAppID))
				Expect(snap.EnvBinding.EnvName).To(Equal("dev"))
			})
		})
	})

	Describe("ListSnapshots", func() {
		BeforeEach(func() {
			createTestMetadata(ctx, store, testAppID)
			createTestEnvBinding(ctx, store, testAppID, "dev")
			createTestEnvBinding(ctx, store, testAppID, "prod")
		})

		Context("when the app has multiple env bindings", func() {
			It("should return all snapshots", func() {
				snaps, err := mgr.ListSnapshots(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(snaps).To(HaveLen(2))

				envNames := []string{snaps[0].EnvBinding.EnvName, snaps[1].EnvBinding.EnvName}
				Expect(envNames).To(ConsistOf("dev", "prod"))
			})
		})

		Context("when metadata does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := mgr.ListSnapshots(ctx, "non-existent-app")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("DeleteEnvBinding", func() {
		BeforeEach(func() {
			createTestMetadata(ctx, store, testAppID)
			createTestEnvBinding(ctx, store, testAppID, "dev")
		})

		Context("when the env binding exists", func() {
			It("should delete it", func() {
				err := mgr.DeleteEnvBinding(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())

				_, err = store.GetEnvBinding(ctx, testAppID, "dev")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("DeleteByApp", func() {
		BeforeEach(func() {
			createTestMetadata(ctx, store, testAppID)
			createTestEnvBinding(ctx, store, testAppID, "dev")
		})

		Context("when the app has metadata and env bindings", func() {
			It("should delete them all", func() {
				err := mgr.DeleteByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.GetMetadata(ctx, testAppID)
				Expect(err).To(MatchError(model.ErrMetadataNotFound))

				bindings, err := store.ListEnvBindingsByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
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
		CredentialID: "1",
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
		BscpEnvName: envName,
		BscpAppID:   "541",
		Operator:    "tester",
	})
	Expect(err).NotTo(HaveOccurred())
}
