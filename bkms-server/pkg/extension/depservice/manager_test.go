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

package depservice

import (
	"context"
	"time"

	. "github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

var _ = Describe("Test ServiceManager on service instance operations", func() {
	var mgr *ServiceManager

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		mgr = &ServiceManager{
			svcStore:  &model.ServiceStoreMongo{},
			instStore: &model.ServiceInstanceStoreMongo{},
		}
	})

	Context("test GetServiceInstance", func() {
		It("get successfully", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{ID: instID}, nil).
					Build()

				inst, err := mgr.GetServiceInstance(ctx, instID)
				Expect(err).NotTo(HaveOccurred())
				Expect(inst.ID).To(Equal(instID))
			})
		})
		It("service instance not found", func() {
			PatchConvey("test", GinkgoT(), func() {
				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(nil, model.NewNotFoundError("service instance")).
					Build()
				_, err := mgr.GetServiceInstance(ctx, bson.NewObjectID())
				Expect(model.AsNotFoundError(err)).To(Equal(true))
			})
		})
	})

	Context("test DeleteServiceInstance", func() {
		It("delete successfully", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				serviceName := "test-service"
				planName := "test-plan"

				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:           instID,
					ServiceName:  serviceName,
					PlanName:     planName,
					Status:       model.AvailableStatus,
					AttachedApps: []string{},
				}, nil).
					Build()

				Mock((*model.ServiceStoreMongo).Get).Return(&model.Service{
					Name:  serviceName,
					Plans: []model.ServicePlan{{Name: planName, ProviderType: model.ProviderTypeSystemAllocated}},
				}, nil).Build()

				Mock(provider.New).Return(&polaris.Provider{}, nil).Build()
				Mock((*polaris.Provider).DeleteInstance).Return(nil).Build()
				Mock((*model.ServiceInstanceStoreMongo).Delete).Return(nil).Build()

				err := mgr.DeleteServiceInstance(ctx, instID)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("delete failed because instance is used by apps", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:           instID,
					AttachedApps: []string{"app1"},
				}, nil).
					Build()
				err := mgr.DeleteServiceInstance(ctx, instID)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("is used by apps"))
			})
		})

		It("delete successfully when instance status is provisioning", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:           instID,
					Status:       model.ProvisioningStatus,
					AttachedApps: []string{},
				}, nil).
					Build()
				Mock((*model.ServiceInstanceStoreMongo).Delete).Return(nil).Build()

				err := mgr.DeleteServiceInstance(ctx, instID)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("delete successfully when instance status is createFailed", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:           instID,
					Status:       model.CreateFailedStatus,
					AttachedApps: []string{},
				}, nil).
					Build()
				Mock((*model.ServiceInstanceStoreMongo).Delete).Return(nil).Build()

				err := mgr.DeleteServiceInstance(ctx, instID)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("delete failed because provider failed", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				serviceName := "test-service"
				planName := "test-plan"

				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:           instID,
					ServiceName:  serviceName,
					PlanName:     planName,
					Status:       model.AvailableStatus,
					AttachedApps: []string{},
				}, nil).
					Build()

				Mock((*model.ServiceStoreMongo).Get).Return(&model.Service{
					Name:  serviceName,
					Plans: []model.ServicePlan{{Name: planName, ProviderType: model.ProviderTypeSystemAllocated}},
				}, nil).Build()

				Mock(provider.New).Return(&polaris.Provider{}, nil).Build()
				Mock((*polaris.Provider).DeleteInstance).Return(errors.New("delete failed")).Build()
				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.DeleteServiceInstance(ctx, instID)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("test CreateServiceInstance", func() {
		It("test create successfully", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				serviceName := "test-service"
				planName := "test-plan"

				Mock(
					(*model.ServiceStoreMongo).Get,
				).Return(&model.Service{
					Name: serviceName,
					Plans: []model.ServicePlan{{
						Name:         planName,
						ProviderType: model.ProviderTypeSystemAllocated,
						Config:       map[string]any{},
					}},
				}, nil).
					Build()

				Mock(provider.New).Return(&polaris.Provider{}, nil).Build()
				Mock((*model.ServiceInstanceStoreMongo).Create).Return(instID, nil).Build()
				Mock((*polaris.Provider).CreateInstance).Return(&types.CreateInstanceResult{
					InstConfig:  map[string]any{},
					Credentials: map[string]any{},
				}, nil).Build()
				Mock((*model.ServiceInstanceStoreMongo).UpdateConfig).Return(nil).Build()
				Mock((*model.ServiceInstanceStoreMongo).UpdateCredentials).Return(nil).Build()
				Mock((*ServiceManager).startPollingInstance).Return(nil).Build()

				id, err := mgr.CreateServiceInstance(ctx, &CreateServiceInstanceParams{
					Name:        "test-instance",
					ServiceName: serviceName,
					PlanName:    planName,
					ScopeType:   model.ScopeTypeWorkspace,
					WorkspaceID: "test-workspace",
					Operator:    "test-operator",
					Params: &polaris.CreateParams{
						PolarisName:      "test-service",
						PolarisNamespace: "test-namespace",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(instID))

				// 等待一小段时间，确保 goroutine 在 Mock 作用域内执行完毕, 否则 startPollingInstance mock 会失效
				// TODO: 将 startPollingInstance 改成异步执行, 从而避免此问题
				time.Sleep(1 * time.Second)
			})
		})
	})

	Context("test startPollingInstance", func() {
		It("test success", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()

				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:          instID,
					Config:      map[string]any{},
					Credentials: map[string]any{},
				}, nil).
					Build()

				Mock(
					(*polaris.Provider).QueryInstance,
				).Return(&types.QueryInstanceResult{Status: types.AvailableStatus}, nil).
					Build()

				Mock((*ServiceManager).updateInstanceByResult).Return(nil).Build()

				err := mgr.startPollingInstance(
					ctx,
					instID,
					&polaris.Provider{},
					make(map[string]any),
					60*time.Second,
				)

				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("test timeout", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()

				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:          instID,
					Config:      map[string]any{},
					Credentials: map[string]any{},
				}, nil).
					Build()

				Mock(
					(*polaris.Provider).QueryInstance,
				).Return(&types.QueryInstanceResult{Status: types.ProvisioningStatus}, nil).
					Build()

				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.startPollingInstance(
					ctx,
					instID,
					&polaris.Provider{},
					make(map[string]any),
					5*time.Second,
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("polling service instance by provider exceeded 5s"))
			})
		})

		It("test query instance failed", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()

				Mock(
					(*model.ServiceInstanceStoreMongo).Get,
				).Return(&model.ServiceInstance{
					ID:          instID,
					Config:      map[string]any{},
					Credentials: map[string]any{},
				}, nil).
					Build()

				Mock(
					(*polaris.Provider).QueryInstance,
				).Return(nil, errors.New("query failed")).
					Build()

				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.startPollingInstance(
					ctx,
					instID,
					&polaris.Provider{},
					make(map[string]any),
					60*time.Second,
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("test updateInstanceByResult", func() {
		It("update successfully when status is available", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				credentials := map[string]any{"username": "test", "password": "test123"}

				Mock((*model.ServiceInstanceStoreMongo).UpdateCredentials).Return(nil).Build()
				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.updateInstanceByResult(
					ctx,
					&types.QueryInstanceResult{Status: types.AvailableStatus},
					instID,
					credentials,
				)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("update failed when status is unavailable", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				credentials := map[string]any{"username": "test", "password": "test123"}

				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.updateInstanceByResult(
					ctx,
					&types.QueryInstanceResult{Status: types.UnavailableStatus},
					instID,
					credentials,
				)
				Expect(err).To(HaveOccurred())
			})
		})

		It("update credentials failed", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				credentials := map[string]any{"username": "test", "password": "test123"}

				Mock(
					(*model.ServiceInstanceStoreMongo).UpdateCredentials,
				).Return(errors.New("update failed")).
					Build()
				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.updateInstanceByResult(
					ctx,
					&types.QueryInstanceResult{Status: types.AvailableStatus},
					instID,
					credentials,
				)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("test updateInstanceStatus", func() {
		It("update status successfully", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()

				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.updateInstanceStatus(ctx, instID, model.AvailableStatus, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("update status with error message", func() {
			PatchConvey("test", GinkgoT(), func() {
				instID := bson.NewObjectID()
				statusError := errors.New("some error")

				Mock((*model.ServiceInstanceStoreMongo).UpdateStatus).Return(nil).Build()

				err := mgr.updateInstanceStatus(ctx, instID, model.CreateFailedStatus, statusError)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(statusError))
			})
		})
	})
})
