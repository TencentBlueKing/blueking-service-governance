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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/deployment"
	gamedeploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/gamedeployment"
)

var _ = Describe("DeployStateGetter", func() {
	var (
		ctx    context.Context
		getter *DeployStateGetter
		record *Record
		mocker *mockey.Mocker
	)

	BeforeEach(func() {
		cfg, err := testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		// Set up the mocker to make the syncer always use the test config.
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()

		ctx = context.Background()
		record = &Record{
			WorkspaceID:     "test-workspace",
			AppID:           "test-app",
			EnvName:         "test-env",
			TrafficLaneName: "base",
			ClusterID:       "BCS-K8S-12345",
			Namespace:       "default",
			ImageTag:        "v1.0.0",
			Replicas:        3,
			ResourceKeys: ResourceKeys{
				{Kind: k8skind.GameDeploy, Name: "test-game-deploy"},
				{Kind: k8skind.SVC, Name: "test-service"},
				{Kind: k8skind.CM, Name: "test-configmap"},
			},
		}
		getter = NewDeployStateGetter(record)
	})

	AfterEach(func() {
		mocker.Release()
	})

	Describe("Get", func() {
		Context("when all resources are healthy", func() {
			It("should return StatusDeployed", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockGVR := &schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					}
					mockey.Mock(discovery.GetGroupVersionResource).Return(mockGVR, nil).Build()

					mockResource := &unstructured.Unstructured{}
					mockey.Mock((*k8sclient.Client).Get).Return(mockResource, nil).Build()

					healthyResult := &k8sstatus.Result{
						Code:    k8sstatus.Healthy,
						Message: "All replicas are ready",
					}
					mockey.Mock(gamedeploystatus.Parse).Return(healthyResult, nil).Build()

					state, err := getter.Get(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(state.Status).To(Equal(StatusDeployed))
				})
			})
		})

		Context("when the main workload is a Deployment", func() {
			It("should return StatusDeployed when the Deployment is available", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					record.ResourceKeys = ResourceKeys{{Kind: k8skind.Deploy, Name: "test-deploy"}}
					getter = NewDeployStateGetter(record)

					mockResource := &unstructured.Unstructured{}
					mockey.Mock((*k8sclient.Client).Get).Return(mockResource, nil).Build()
					mockey.Mock(deploystatus.Parse).Return(&k8sstatus.Result{
						Code:    k8sstatus.Available,
						Message: "Deployment is available",
					}).Build()

					state, err := getter.Get(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(state.Status).To(Equal(StatusDeployed))
				})
			})
		})

		Context("when dependent resources are missing", func() {
			It("should quickly return StatusFailed", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.SVC, nil).Build()
					mockey.Mock((*k8sclient.Client).Get).Return(nil, k8sclient.ErrResourceNotFound).Build()

					state, err := getter.Get(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(state.Status).To(Equal(StatusFailed))
				})
			})
		})

		Context("when encountering network errors", func() {
			It("should return error instead of StatusFailed", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.SVC, nil).Build()
					networkErr := errors.New("context deadline exceeded")
					mockey.Mock((*k8sclient.Client).Get).Return(nil, networkErr).Build()

					_, err := getter.Get(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when GameDeployment is in Degraded status", func() {
			It("should return StatusFailed", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.SVC, nil).Build()

					mockResource := &unstructured.Unstructured{}
					mockey.Mock((*k8sclient.Client).Get).Return(mockResource, nil).Build()

					degradedResult := &k8sstatus.Result{
						Code:    k8sstatus.Degraded,
						Message: "Some replicas are not ready",
					}
					mockey.Mock(gamedeploystatus.Parse).Return(degradedResult, nil).Build()

					state, err := getter.Get(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(state.Status).To(Equal(StatusFailed))
				})
			})
		})

		Context("when GameDeployment is in Progressing status", func() {
			It("should return StatusDeploying", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.SVC, nil).Build()

					mockResource := &unstructured.Unstructured{}
					mockey.Mock((*k8sclient.Client).Get).Return(mockResource, nil).Build()

					progressingResult := &k8sstatus.Result{
						Code:    k8sstatus.Progressing,
						Message: "Deployment is progressing",
					}
					mockey.Mock(gamedeploystatus.Parse).Return(progressingResult, nil).Build()

					state, err := getter.Get(ctx)
					Expect(err).NotTo(HaveOccurred())
					Expect(state.Status).To(Equal(StatusDeploying))
				})
			})
		})

		Context("when GVR retrieval fails", func() {
			It("should return error", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					gvrErr := errors.New("failed to discover GVR")
					mockey.Mock(discovery.GetGroupVersionResource).Return(nil, gvrErr).Build()

					_, err := getter.Get(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("getMainWorkload", func() {
		It("prefers Deployment over GameDeployment", func() {
			record.ResourceKeys = ResourceKeys{
				{Kind: k8skind.Deploy, Name: "test-deploy"},
				{Kind: k8skind.GameDeploy, Name: "test-game-deploy"},
			}
			getter = NewDeployStateGetter(record)
			kind, name := getter.getMainWorkload()
			Expect(kind).To(Equal(k8skind.Deploy))
			Expect(name).To(Equal("test-deploy"))
		})
	})

	Describe("Edge case tests", func() {
		Context("when there is no GameDeployment resource", func() {
			It("should return error", func() {
				mockey.PatchConvey("test", GinkgoT(), func() {
					record.ResourceKeys = ResourceKeys{{Kind: k8skind.SVC, Name: "test-service"}}
					getter = NewDeployStateGetter(record)

					mockey.Mock(discovery.GetGroupVersionResource).Return(&gvr.SVC, nil).Build()
					mockResource := &unstructured.Unstructured{}
					mockey.Mock((*k8sclient.Client).Get).Return(mockResource, nil).Build()

					_, err := getter.Get(ctx)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("main workload is not managed"))
				})
			})
		})
	})
})
