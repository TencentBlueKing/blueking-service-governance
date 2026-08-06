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

package deploy

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// mockTrafficManagerAndGetBaseline 封装 trafficmanager mock 逻辑
func mockTrafficManagerAndGetBaseline(baselineLaneName string) {
	baselineLane := &trafficmanager.TrafficLane{
		LaneName: baselineLaneName,
		LaneType: string(trafficmanager.LaneTypeBaseline),
	}
	mockClient := &trafficmanager.StubTrafficManager{}
	mockey.Mock(trafficmanager.New).Return(mockClient).Build()
	mockey.Mock((*trafficmanager.StubTrafficManager).GetBaselineTrafficLane).Return(baselineLane, nil).Build()
}

// newTestChecker 创建用于测试的 PreDeployChecker，注入空依赖
func newTestChecker() *PreDeployChecker {
	return NewPreDeployChecker(
		&envmodel.EnvironmentStoreMongo{},
		&promotion.PromotionStoreMongo{},
		&snapshot.Service{},
	)
}

var _ = Describe("PreDeployChecker", func() {
	var (
		ctx             context.Context
		workspaceID     string
		envName         string
		trafficLaneName string
		appType         string
		appID           string
		checker         *PreDeployChecker
		params          *PreDeployCheckParams
		devEnv          *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "test-workspace-" + stringx.Random(6)
		envName = "staging"
		trafficLaneName = "lane-01"
		appType = bkmsapp.AppTypeHelm
		appID = "test-app-" + stringx.Random(6)

		checker = newTestChecker()
		params = &PreDeployCheckParams{
			WorkspaceID:     workspaceID,
			EnvName:         envName,
			TrafficLaneName: trafficLaneName,
			AppType:         appType,
			AppID:           appID,
			ImageTag:        "v1.0.0",
		}
		devEnv = &envmodel.Environment{
			Name:        envName,
			WorkspaceID: workspaceID,
			Type:        "development",
		}
	})

	Describe("Do", func() {
		It("should pass when traffic lane name is empty", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(devEnv, nil).Build()

				params.TrafficLaneName = ""
				err := checker.Do(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should pass when deploying to baseline lane", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockTrafficManagerAndGetBaseline("baseline")
				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(devEnv, nil).Build()

				params.TrafficLaneName = "baseline"
				err := checker.Do(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should pass when baseline lane is deployed", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockTrafficManagerAndGetBaseline("baseline")
				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(devEnv, nil).Build()

				deployedRecord := &helmdeploy.Record{
					AppID:           appID,
					EnvName:         envName,
					TrafficLaneName: "baseline",
					Status:          helm.StatusDeployed,
				}
				mockey.Mock((*helmdeploy.RecordStoreMongo).GetLatest).Return(deployedRecord, nil).Build()

				err := checker.Do(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should return error when baseline lane is not deployed", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockTrafficManagerAndGetBaseline("base")

				notDeployedRecord := &helmdeploy.Record{
					AppID:           appID,
					EnvName:         envName,
					TrafficLaneName: "baseline",
					Status:          helm.StatusFailed,
				}
				mockey.Mock((*helmdeploy.RecordStoreMongo).GetLatest).Return(notDeployedRecord, nil).Build()

				err := checker.Do(ctx, params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("base not deployed"))
			})
		})
	})

	Describe("checkIfImagePromoted", func() {
		var (
			productionEnv *envmodel.Environment
			devEnv        *envmodel.Environment
		)

		BeforeEach(func() {
			productionEnv = &envmodel.Environment{
				Name:        envName,
				WorkspaceID: workspaceID,
				Type:        "production",
			}
			devEnv = &envmodel.Environment{
				Name:        envName,
				WorkspaceID: workspaceID,
				Type:        "development",
			}
		})

		It("should return error when imageTag is empty", func() {
			params.ImageTag = ""
			params.TrafficLaneName = ""
			err := checker.Do(ctx, params)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("image tag is required"))
		})

		It("should skip check when env type is not production", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				params.TrafficLaneName = ""
				params.ImageTag = "v1.0.0"
				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(devEnv, nil).Build()

				err := checker.Do(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should pass when imageTag is in promoted list", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				params.TrafficLaneName = ""
				params.ImageTag = "v1.0.0"

				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(productionEnv, nil).Build()
				mockey.Mock((*snapshot.Service).ResolveRepoKeyForApp).Return(
					&snapshot.RepoKeyInfo{RepoKey: "repo-key-abc"}, nil,
				).Build()
				mockey.Mock((*promotion.PromotionStoreMongo).IsTagPromoted).Return(true, nil).Build()

				err := checker.Do(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("should return error when imageTag is not in promoted list", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				params.TrafficLaneName = ""
				params.ImageTag = "v2.0.0-dev"

				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(productionEnv, nil).Build()
				mockey.Mock((*snapshot.Service).ResolveRepoKeyForApp).Return(
					&snapshot.RepoKeyInfo{RepoKey: "repo-key-abc"}, nil,
				).Build()
				mockey.Mock((*promotion.PromotionStoreMongo).IsTagPromoted).Return(false, nil).Build()

				err := checker.Do(ctx, params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("has not been promoted to production"))
			})
		})

		It("should return error when env query fails", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				params.TrafficLaneName = ""
				params.ImageTag = "v1.0.0"

				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(
					nil, errors.New("db connection error"),
				).Build()

				err := checker.Do(ctx, params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get env"))
			})
		})

		It("should return error when ResolveRepoKeyForApp fails", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				params.TrafficLaneName = ""
				params.ImageTag = "v1.0.0"

				mockey.Mock((*envmodel.EnvironmentStoreMongo).GetByName).Return(productionEnv, nil).Build()
				mockey.Mock((*snapshot.Service).ResolveRepoKeyForApp).Return(
					nil, errors.New("build config not found"),
				).Build()

				err := checker.Do(ctx, params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("resolve repo key"))
			})
		})
	})
})
