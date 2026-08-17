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

package depservice_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/fake"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// fakeProvisionParams 满足 types.ProvisionParams，供 CreateServiceInstance 入参使用。
type fakeProvisionParams struct{}

func (fakeProvisionParams) Validate() error { return nil }

var _ = Describe("ServiceManager", func() {
	var (
		ctx          context.Context
		diApp        *fxtest.App
		svcStore     model.ServiceStore
		instStore    model.ServiceInstanceStore
		bindingStore model.ServiceBindingStore
		mgr          *depservice.ServiceManager

		planName string
	)

	BeforeEach(func() {
		ctx = context.Background()
		fake.Reset()

		diApp = fxtest.New(
			GinkgoT(),
			model.FxModule,
			fx.Populate(&svcStore, &instStore, &bindingStore),
		)
		diApp.RequireStart()
		mgr = depservice.New(svcStore, instStore, bindingStore, nil)

		// serviceName 使用 "fake"，provider.New 会走注册表返回 fake.Provider，无需 Mock
		planName = "default"
		Expect(svcStore.Create(ctx, &model.Service{
			Name:        "fake",
			DisplayName: "fake",
			Plans: []model.ServicePlan{{
				Name:         planName,
				ProviderType: model.ProviderTypeSystemAllocated,
				Config:       map[string]any{},
			}},
		})).To(Succeed())
	})

	AfterEach(func() {
		fake.Reset()
		Expect(instStore.DeleteAll(ctx)).To(Succeed())
		Expect(bindingStore.DeleteAll(ctx)).To(Succeed())
		Expect(svcStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	createInstance := func(status model.InstanceStatus) bson.ObjectID {
		id, err := instStore.Create(ctx, &model.ServiceInstance{
			Name:         "inst-" + stringx.Random(6),
			ServiceName:  "fake",
			PlanName:     planName,
			ProviderType: model.ProviderTypeSystemAllocated,
			ScopeType:    model.ScopeTypeWorkspace,
			WorkspaceID:  "ws-" + stringx.Random(6),
			Status:       status,
			Operator:     "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	Context("GetServiceInstance", func() {
		It("returns the instance from mongodb", func() {
			instID := createInstance(model.AvailableStatus)

			inst, err := mgr.GetServiceInstance(ctx, instID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.ID).To(Equal(instID))
			Expect(inst.ServiceName).To(Equal("fake"))
		})

		It("returns not found for a missing instance", func() {
			_, err := mgr.GetServiceInstance(ctx, bson.NewObjectID())
			Expect(model.AsNotFoundError(err)).To(BeTrue())
		})
	})

	Context("DeleteServiceInstance", func() {
		It("deletes an available instance through a sync provider", func() {
			instID := createInstance(model.AvailableStatus)

			Expect(mgr.DeleteServiceInstance(ctx, instID)).To(Succeed())
			_, err := instStore.Get(ctx, instID)
			Expect(model.AsNotFoundError(err)).To(BeTrue())
		})

		It("rejects delete while provisioning", func() {
			instID := createInstance(model.ProvisioningStatus)

			err := mgr.DeleteServiceInstance(ctx, instID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, depservice.ErrInvalidArgument)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("service instance is provisioning, cannot delete"))

			inst, getErr := instStore.Get(ctx, instID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.ProvisioningStatus))
		})

		It("keeps deleting status for an async provider", func() {
			fake.Use(&fake.Provider{DeleteAsync: true})
			instID := createInstance(model.AvailableStatus)

			Expect(mgr.DeleteServiceInstance(ctx, instID)).To(Succeed())

			inst, err := instStore.Get(ctx, instID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.DeletingStatus))
		})

		It("directly deletes a createFailed instance", func() {
			instID := createInstance(model.CreateFailedStatus)

			Expect(mgr.DeleteServiceInstance(ctx, instID)).To(Succeed())
			_, err := instStore.Get(ctx, instID)
			Expect(model.AsNotFoundError(err)).To(BeTrue())
		})

		It("marks deleteFailed when the provider fails", func() {
			fake.Use(&fake.Provider{DeleteErr: errors.New("delete failed")})
			instID := createInstance(model.AvailableStatus)

			err := mgr.DeleteServiceInstance(ctx, instID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete failed"))

			inst, getErr := instStore.Get(ctx, instID)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.DeleteFailedStatus))
			Expect(inst.Message).To(ContainSubstring("delete failed"))
		})
	})

	Context("CreateServiceInstance", func() {
		It("creates an available instance with a sync provider", func() {
			fake.Use(&fake.Provider{
				CreateResult: &types.CreateInstanceResult{
					InstConfig:  map[string]any{"key": "val"},
					Credentials: map[string]any{"token": "secret"},
				},
			})

			id, err := mgr.CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
				Name:        "inst-" + stringx.Random(6),
				ServiceName: "fake",
				PlanName:    planName,
				ScopeType:   model.ScopeTypeWorkspace,
				WorkspaceID: "ws-" + stringx.Random(6),
				Operator:    "tester",
				Params:      fakeProvisionParams{},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))

			inst, getErr := instStore.Get(ctx, id)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.AvailableStatus))
			Expect(inst.Config["key"]).To(Equal("val"))
			Expect(inst.Credentials["token"]).To(Equal("secret"))
		})

		It("keeps provisioning status for an async provider", func() {
			fake.Use(&fake.Provider{CreateAsync: true})

			id, err := mgr.CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
				Name:        "inst-" + stringx.Random(6),
				ServiceName: "fake",
				PlanName:    planName,
				ScopeType:   model.ScopeTypeWorkspace,
				WorkspaceID: "ws-" + stringx.Random(6),
				Operator:    "tester",
				Params:      fakeProvisionParams{},
			})
			Expect(err).NotTo(HaveOccurred())

			inst, getErr := instStore.Get(ctx, id)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.ProvisioningStatus))
		})

		It("fails before writing an instance when provider is unknown", func() {
			unknownName := "unknown-" + stringx.Random(6)
			Expect(svcStore.Create(ctx, &model.Service{
				Name:        unknownName,
				DisplayName: unknownName,
				Plans: []model.ServicePlan{{
					Name:         planName,
					ProviderType: model.ProviderTypeSystemAllocated,
					Config:       map[string]any{},
				}},
			})).To(Succeed())

			id, err := mgr.CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
				Name:        "inst-" + stringx.Random(6),
				ServiceName: unknownName,
				PlanName:    planName,
				ScopeType:   model.ScopeTypeWorkspace,
				WorkspaceID: "ws-" + stringx.Random(6),
				Operator:    "tester",
				Params:      fakeProvisionParams{},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("provider not found"))
			Expect(id).To(Equal(bson.NilObjectID))

			insts, listErr := instStore.List(ctx, nil)
			Expect(listErr).NotTo(HaveOccurred())
			Expect(insts).To(BeEmpty())
		})

		It("marks createFailed when the provider create fails", func() {
			fake.Use(&fake.Provider{CreateErr: errors.New("provider create failed")})

			id, err := mgr.CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
				Name:        "inst-" + stringx.Random(6),
				ServiceName: "fake",
				PlanName:    planName,
				ScopeType:   model.ScopeTypeWorkspace,
				WorkspaceID: "ws-" + stringx.Random(6),
				Operator:    "tester",
				Params:      fakeProvisionParams{},
			})
			Expect(err).To(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))

			inst, getErr := instStore.Get(ctx, id)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(model.CreateFailedStatus))
			Expect(inst.Message).To(ContainSubstring("provider create failed"))
		})
	})

	Context("ListServiceInstances", func() {
		It("filters by workspace", func() {
			wsID := "ws-" + stringx.Random(6)
			for i := 0; i < 3; i++ {
				_, err := instStore.Create(ctx, &model.ServiceInstance{
					Name:         "inst-" + stringx.Random(6),
					ServiceName:  "fake",
					PlanName:     planName,
					ProviderType: model.ProviderTypeSystemAllocated,
					ScopeType:    model.ScopeTypeWorkspace,
					WorkspaceID:  wsID,
					Status:       model.AvailableStatus,
					Operator:     "tester",
				})
				Expect(err).NotTo(HaveOccurred())
			}
			// other workspace
			_ = createInstance(model.AvailableStatus)

			insts, err := mgr.ListServiceInstances(ctx, &depservice.ListServiceInstancesParams{
				WorkspaceID: wsID,
				ServiceName: "fake",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(insts).To(HaveLen(3))
		})
	})

	Context("ServiceBinding", func() {
		It("creates an empty binding and lists it", func() {
			binding, err := mgr.CreateServiceBinding(ctx, &depservice.CreateServiceBindingParams{
				Name:        "session",
				AppID:       "app-1",
				WorkspaceID: "ws-" + stringx.Random(6),
				ServiceName: "redis",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(binding.EnvInstanceMap).To(BeEmpty())
			Expect(binding.EnvVars).To(BeEmpty())

			list, err := mgr.ListServiceBindings(ctx, "app-1", "redis")
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
		})

		It("maps an instance per env and rejects delete while referenced", func() {
			wsID := "ws-" + stringx.Random(6)
			instID, err := instStore.Create(ctx, &model.ServiceInstance{
				Name:         "redis-" + stringx.Random(6),
				ServiceName:  "redis",
				PlanName:     planName,
				ProviderType: model.ProviderTypeSystemAllocated,
				ScopeType:    model.ScopeTypeWorkspace,
				WorkspaceID:  wsID,
				Status:       model.AvailableStatus,
				Operator:     "tester",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = mgr.CreateServiceBinding(ctx, &depservice.CreateServiceBindingParams{
				Name:           "cache",
				AppID:          "app-1",
				WorkspaceID:    wsID,
				ServiceName:    "redis",
				EnvInstanceMap: map[string]bson.ObjectID{"prod": instID},
				EnvVars:        map[string]string{"REDIS_HOST": "${{env.REDIS_HOST}}"},
			})
			Expect(err).NotTo(HaveOccurred())

			err = mgr.DeleteServiceInstance(ctx, instID)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, depservice.ErrInvalidArgument)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("bindings"))
		})
	})
})
