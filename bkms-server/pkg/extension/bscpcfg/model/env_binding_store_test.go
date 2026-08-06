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

	Describe("Create", func() {
		Context("when creating a valid env binding", func() {
			It("should create successfully", func() {
				binding := &model.EnvBinding{
					AppID:   testAppID,
					EnvName: "dev",
					Services: []model.ServiceRef{
						{ID: "svc-1", Name: "file-svc"},
					},
					DefaultServiceID: "svc-1",
					Operator:         "tester",
				}

				err := store.Create(ctx, binding)
				Expect(err).NotTo(HaveOccurred())

				// 验证写入
				stored, err := store.Get(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.Services).To(HaveLen(1))
				Expect(stored.Services[0].ID).To(Equal("svc-1"))
				Expect(stored.DefaultServiceID).To(Equal("svc-1"))
				Expect(stored.CreatedAt).NotTo(BeZero())
				Expect(stored.UpdatedAt).NotTo(BeZero())
			})
		})

		Context("when creating duplicate app+env binding", func() {
			It("should return ErrEnvBindingAlreadyExists", func() {
				binding := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  "dev",
					Services: []model.ServiceRef{{ID: "svc-1", Name: "file-svc"}},
				}
				err := store.Create(ctx, binding)
				Expect(err).NotTo(HaveOccurred())

				binding2 := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  "dev",
					Services: []model.ServiceRef{{ID: "svc-2", Name: "file-svc-2"}},
				}
				err = store.Create(ctx, binding2)
				Expect(err).To(MatchError(model.ErrEnvBindingAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				binding := &model.EnvBinding{
					EnvName:  "dev",
					Services: []model.ServiceRef{{ID: "svc-1", Name: "file-svc"}},
				}
				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})

			It("should return validation error for empty services", func() {
				binding := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  "dev",
					Services: []model.ServiceRef{},
				}
				err := store.Create(ctx, binding)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			binding := &model.EnvBinding{
				AppID:    testAppID,
				EnvName:  "staging",
				Services: []model.ServiceRef{{ID: "app-1", Name: "my-svc"}},
				Operator: "admin",
			}
			err := store.Create(ctx, binding)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when binding exists", func() {
			It("should return the binding", func() {
				binding, err := store.Get(ctx, testAppID, "staging")
				Expect(err).NotTo(HaveOccurred())
				Expect(binding.Operator).To(Equal("admin"))
				Expect(binding.Services[0].Name).To(Equal("my-svc"))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := store.Get(ctx, testAppID, "non-existent")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			binding := &model.EnvBinding{
				AppID:    testAppID,
				EnvName:  "prod",
				Services: []model.ServiceRef{{ID: "svc-1", Name: "original"}},
				Operator: "user1",
			}
			err := store.Create(ctx, binding)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating services", func() {
			It("should replace services entirely", func() {
				newServices := []model.ServiceRef{
					{ID: "svc-1", Name: "original"},
					{ID: "svc-2", Name: "added"},
				}
				err := store.Update(ctx, testAppID, "prod", &model.EnvBindingUpdate{
					Services: &newServices,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID, "prod")
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Services).To(HaveLen(2))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				newServices := []model.ServiceRef{{ID: "svc-1", Name: "any"}}
				err := store.Update(ctx, testAppID, "non-existent", &model.EnvBindingUpdate{
					Services: &newServices,
				})
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})

		Context("when updateData is nil", func() {
			It("should return nil without error", func() {
				err := store.Update(ctx, testAppID, "prod", nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			binding := &model.EnvBinding{
				AppID:    testAppID,
				EnvName:  "dev",
				Services: []model.ServiceRef{{ID: "s1", Name: "svc"}},
			}
			err := store.Create(ctx, binding)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when binding exists", func() {
			It("should delete successfully", func() {
				err := store.Delete(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(ctx, testAppID, "dev")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				err := store.Delete(ctx, testAppID, "non-existent")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("DeleteByApp", func() {
		BeforeEach(func() {
			for _, env := range []string{"dev", "staging", "prod"} {
				binding := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  env,
					Services: []model.ServiceRef{{ID: "s1", Name: "svc"}},
				}
				err := store.Create(ctx, binding)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should delete all bindings for the app", func() {
			err := store.DeleteByApp(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())

			bindings, err := store.ListByApp(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(bindings).To(BeEmpty())
		})
	})

	Describe("ListByApp", func() {
		Context("when no bindings exist", func() {
			It("should return empty list", func() {
				bindings, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})

		Context("when multiple bindings exist", func() {
			It("should return all bindings for the app", func() {
				for _, env := range []string{"dev", "prod"} {
					binding := &model.EnvBinding{
						AppID:    testAppID,
						EnvName:  env,
						Services: []model.ServiceRef{{ID: "s1", Name: "svc"}},
					}
					err := store.Create(ctx, binding)
					Expect(err).NotTo(HaveOccurred())
				}

				bindings, err := store.ListByApp(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(HaveLen(2))
			})
		})
	})
})
