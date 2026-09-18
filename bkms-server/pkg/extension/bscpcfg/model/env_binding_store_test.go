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

var _ = Describe("EnvBindingStore", func() {
	var store model.EnvBindingStore
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
		_ = store.DeleteByApp(ctx, testAppID)
		diApp.RequireStop()
	})

	// newBinding 构造一个满足校验的 EnvBinding。
	newBinding := func(envName string) *model.EnvBinding {
		return &model.EnvBinding{
			AppID:       testAppID,
			EnvName:     envName,
			BscpEnvID:   "12",
			BscpEnvName: envName,
			BscpAppID:   "541",
			Operator:    "tester",
		}
	}

	Describe("Create", func() {
		Context("when creating a valid binding", func() {
			It("should create successfully", func() {
				err := store.Create(ctx, newBinding("dev"))
				Expect(err).NotTo(HaveOccurred())

				stored, err := store.Get(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.BscpEnvID).To(Equal("12"))
				Expect(stored.BscpEnvName).To(Equal("dev"))
				Expect(stored.BscpAppID).To(Equal("541"))
				Expect(stored.Operator).To(Equal("tester"))
				Expect(stored.CreatedAt).NotTo(BeZero())
				Expect(stored.UpdatedAt).NotTo(BeZero())
			})
		})

		Context("when creating a duplicate app+env binding", func() {
			It("should return ErrEnvBindingAlreadyExists", func() {
				err := store.Create(ctx, newBinding("dev"))
				Expect(err).NotTo(HaveOccurred())

				err = store.Create(ctx, newBinding("dev"))
				Expect(err).To(MatchError(model.ErrEnvBindingAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				binding := newBinding("dev")
				binding.AppID = ""

				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})

			It("should return validation error for missing envName", func() {
				binding := newBinding("")
				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			Expect(store.Create(ctx, newBinding("dev"))).NotTo(HaveOccurred())
			Expect(store.Create(ctx, newBinding("prod"))).NotTo(HaveOccurred())
		})

		Context("when binding exists", func() {
			It("should return the binding", func() {
				stored, err := store.Get(ctx, testAppID, "prod")
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.EnvName).To(Equal("prod"))
				Expect(stored.BscpAppID).To(Equal("541"))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := store.Get(ctx, testAppID, "staging")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("ListByApp", func() {
		Context("when the app has multiple bindings", func() {
			It("should return all of them", func() {
				Expect(store.Create(ctx, newBinding("dev"))).NotTo(HaveOccurred())
				Expect(store.Create(ctx, newBinding("prod"))).NotTo(HaveOccurred())

				bindings, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(HaveLen(2))

				names := []string{bindings[0].EnvName, bindings[1].EnvName}
				Expect(names).To(ConsistOf("dev", "prod"))
			})
		})

		Context("when the app has no bindings", func() {
			It("should return an empty slice", func() {
				bindings, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			Expect(store.Create(ctx, newBinding("dev"))).NotTo(HaveOccurred())
		})

		Context("when binding exists", func() {
			It("should delete it", func() {
				err := store.Delete(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(ctx, testAppID, "dev")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				err := store.Delete(ctx, testAppID, "staging")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("DeleteByApp", func() {
		Context("when the app has bindings", func() {
			It("should delete all of them", func() {
				Expect(store.Create(ctx, newBinding("dev"))).NotTo(HaveOccurred())
				Expect(store.Create(ctx, newBinding("prod"))).NotTo(HaveOccurred())

				err := store.DeleteByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())

				bindings, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})
	})
})
