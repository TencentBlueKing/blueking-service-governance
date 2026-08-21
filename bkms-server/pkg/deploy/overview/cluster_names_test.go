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

package overview

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

var _ = Describe("queryClusterNames", func() {
	It("returns empty map when there is no authenticated user", func() {
		names := queryClusterNames(context.Background(), []EnvRow{{
			Cluster: ClusterInfo{ProjectCode: "proj", ClusterID: "BCS-K8S-1"},
		}})
		Expect(names).To(BeEmpty())
	})

	It("maps cluster IDs to BCS display names", func() {
		ctx := auth.WithUser(context.Background(), auth.User{ID: "tester"})
		rows := []EnvRow{
			{Cluster: ClusterInfo{ProjectCode: "proj-a", ClusterID: "BCS-K8S-1"}},
			{Cluster: ClusterInfo{ProjectCode: "proj-a", ClusterID: "BCS-K8S-2"}},
		}

		mockey.PatchConvey("bcs returns cluster names", GinkgoT(), func() {
			stub := bcs.NewStub(auth.User{ID: "tester"})
			mockey.Mock(bcs.New).Return(stub, nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:   "proj-id-a",
				Code: "proj-a",
			}, nil).Build()
			mockey.Mock((*bcs.StubApiClient).ListClustersByProject).Return([]bcs.Cluster{
				{ID: "BCS-K8S-1", Name: "深圳-预发布集群"},
				{ID: "BCS-K8S-2", Name: "深圳-开发集群"},
			}, nil).Build()

			names := queryClusterNames(ctx, rows)
			Expect(names).To(Equal(clusterNameByID{
				"BCS-K8S-1": "深圳-预发布集群",
				"BCS-K8S-2": "深圳-开发集群",
			}))
		})
	})
})
