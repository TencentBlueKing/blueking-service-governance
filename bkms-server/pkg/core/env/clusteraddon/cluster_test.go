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

package clusteraddon_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

var _ = Describe("Workspace clusters", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "tester"})
	})

	// mockStubBCS 不依赖配置中的 useStubBCS，固定使用 stub BCS 客户端
	mockStubBCS := func() {
		mockey.Mock(bcs.New).Return(bcs.Client(bcs.NewStub(auth.User{})), nil).Build()
	}

	// mockProjectClusters 让 stub BCS 客户端返回一个独立集群与一个共享集群，并记录查询的项目
	mockProjectClusters := func(queried *[]string) {
		mockStubBCS()
		mockey.Mock((*bcs.StubApiClient).ListClustersByProject).To(
			func(_ *bcs.StubApiClient, _ context.Context, projectID string) ([]bcs.Cluster, error) {
				*queried = append(*queried, projectID)
				return []bcs.Cluster{
					{ID: "BCS-K8S-00001", Name: "dedicated"},
					{ID: "BCS-K8S-00002", Name: "shared", IsShared: true},
				}, nil
			},
		).Build()
	}

	Describe("ListWorkspaceClusters", func() {
		It("returns only dedicated clusters of the bcs project", func() {
			mockey.PatchConvey("dedicated clusters", GinkgoT(), func() {
				var queried []string
				mockProjectClusters(&queried)

				clusters, err := clusteraddon.ListWorkspaceClusters(ctx, "project-id")
				Expect(err).NotTo(HaveOccurred())
				Expect(clusters).To(HaveLen(1))
				Expect(clusters[0].ID).To(Equal("BCS-K8S-00001"))
				Expect(queried).To(Equal([]string{"project-id"}))
			})
		})

		It("returns an empty list without querying bcs when the workspace has no bcs project", func() {
			mockey.PatchConvey("no bcs project", GinkgoT(), func() {
				var queried []string
				mockProjectClusters(&queried)

				clusters, err := clusteraddon.ListWorkspaceClusters(ctx, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(clusters).To(BeEmpty())
				Expect(queried).To(BeEmpty())
			})
		})

		It("wraps bcs failures with the project id", func() {
			mockey.PatchConvey("bcs failure", GinkgoT(), func() {
				cause := errors.New("bcs unavailable")
				mockStubBCS()
				mockey.Mock((*bcs.StubApiClient).ListClustersByProject).Return(nil, cause).Build()

				_, err := clusteraddon.ListWorkspaceClusters(ctx, "project-id")
				Expect(errors.Is(err, cause)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("project-id"))
			})
		})
	})

	Describe("GetWorkspaceCluster", func() {
		DescribeTable("finds dedicated clusters of the bcs project only",
			func(projectID, clusterID string, found bool) {
				mockey.PatchConvey("get workspace cluster", GinkgoT(), func() {
					var queried []string
					mockProjectClusters(&queried)

					cluster, err := clusteraddon.GetWorkspaceCluster(ctx, projectID, clusterID)
					if !found {
						Expect(errors.Is(err, clusteraddon.ErrClusterNotInWorkspace)).To(BeTrue())
						return
					}
					Expect(err).NotTo(HaveOccurred())
					Expect(cluster.ID).To(Equal(clusterID))
				})
			},
			Entry("dedicated cluster", "project-id", "BCS-K8S-00001", true),
			Entry("shared cluster", "project-id", "BCS-K8S-00002", false),
			Entry("cluster outside the project", "project-id", "BCS-K8S-00003", false),
			Entry("workspace without bcs project", "", "BCS-K8S-00001", false),
		)
	})

	Describe("TargetOfCluster", func() {
		var origin []string

		BeforeEach(func() {
			origin = config.G.BCS.FederationClusterIDs
			config.G.BCS.FederationClusterIDs = []string{"BCS-K8S-FED01"}
		})

		AfterEach(func() {
			config.G.BCS.FederationClusterIDs = origin
		})

		DescribeTable("detects federation clusters from config",
			func(clusterID string, isFederation bool) {
				Expect(clusteraddon.TargetOfCluster(clusterID)).To(Equal(clusteraddon.ClusterTarget{
					ClusterID: clusterID, IsFederation: isFederation,
				}))
			},
			Entry("federation host", "BCS-K8S-FED01", true),
			Entry("regular cluster", "BCS-K8S-00001", false),
		)
	})
})
