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

package bcs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/bytedance/mockey"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("BCS API Client", func() {
	var cli Client
	var ctx context.Context

	var originCfg *config.Config

	BeforeEach(func() {
		ctx = context.Background()
		originCfg = config.G
		config.G = &config.Config{
			BkApp: config.BkAppConfig{Code: "foo", Secret: "bar"},
		}

		cli, _ = New(auth.User{ID: "foo"})
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("list authorized projects", func() {
		PatchConvey("test", GinkgoT(), func() {
			projectID1 := uuid.New().String()
			projectCode1 := stringx.Random(6)
			projectName1 := stringx.Random(6)
			BizID1 := stringx.Random(6)
			Description1 := stringx.Random(6)

			projectID2 := uuid.New().String()
			projectName2 := stringx.Random(6)
			projectCode2 := stringx.Random(6)
			BizID2 := stringx.Random(6)

			resultJson := fmt.Sprintf(`
{"data": {"total": 2, "results": 
[{"projectID": "%s", "projectCode": "%s", "name": "%s", "businessID": "%s", "isOffline": false, "description": "%s"},
{"projectID": "%s", "projectCode": "%s", "name": "%s", "businessID": "%s", "isOffline": true}]
}}`, projectID1, projectCode1, projectName1, BizID1, Description1, projectID2, projectCode2, projectName2, BizID2)
			result := make(map[string]any)
			Expect(json.Unmarshal([]byte(resultJson), &result)).To(BeNil())
			Mock((*ApiClient).handleOperation).Return(result, nil).Build()

			projects, err := cli.ListAuthorizedProjects(ctx)
			Expect(err).To(BeNil())

			Expect(len(projects)).To(Equal(2))

			Expect(projects[0]).To(Equal(Project{
				ID:          projectID1,
				Code:        projectCode1,
				Name:        projectName1,
				BizID:       BizID1,
				Description: Description1,
				IsOffline:   false,
			}))

			Expect(projects[1]).To(Equal(Project{
				ID:          projectID2,
				Code:        projectCode2,
				Name:        projectName2,
				BizID:       BizID2,
				Description: "",
				IsOffline:   true,
			}))
		})
	})

	It("get project", func() {
		PatchConvey("test", GinkgoT(), func() {
			projectID := uuid.New().String()
			projectCode := stringx.Random(6)
			projectName := stringx.Random(6)
			BizID := stringx.Random(6)
			Description := stringx.Random(6)

			resultJson := fmt.Sprintf(
				`{"data": {"projectID": "%s","projectCode": "%s","name": "%s","businessID": "%s","isOffline": false,"description": "%s"}}`,
				projectID,
				projectCode,
				projectName,
				BizID,
				Description,
			)
			result := make(map[string]any)
			Expect(json.Unmarshal([]byte(resultJson), &result)).To(BeNil())
			Mock((*ApiClient).handleOperation).Return(result, nil).Build()

			project, err := cli.GetProject(ctx, projectCode)
			Expect(err).To(BeNil())

			Expect(*project).To(Equal(Project{
				ID:          projectID,
				Code:        projectCode,
				Name:        projectName,
				BizID:       BizID,
				Description: Description,
				IsOffline:   false,
			}))
		})
	})

	It("list clusters by project", func() {
		PatchConvey("test", GinkgoT(), func() {
			projectID := uuid.New().String()

			clusterID1 := stringx.Random(6)
			clusterName1 := stringx.Random(6)
			clusterType1 := stringx.Random(6)
			clusterDesc1 := stringx.Random(6)
			clusterStatus1 := stringx.Random(6)
			clusterEnv1 := stringx.Random(6)

			clusterID2 := stringx.Random(6)
			clusterName2 := stringx.Random(6)
			clusterType2 := stringx.Random(6)
			clusterDesc2 := stringx.Random(6)
			clusterStatus2 := stringx.Random(6)
			clusterEnv2 := stringx.Random(6)

			clusterID3 := stringx.Random(6)
			clusterName3 := stringx.Random(6)
			clusterType3 := stringx.Random(6)
			clusterDesc3 := stringx.Random(6)
			clusterStatus3 := stringx.Random(6)
			clusterEnv3 := stringx.Random(6)

			// 创建三个集群数据, 2个 k8s 集群, 1个 非 k8s 集群
			resultJson := fmt.Sprintf(`
{"data": 
[{"clusterID": "%s", "clusterName": "%s", "clusterType": "%s", "description": "%s", "status": "%s", "environment": "%s", "engineType": "k8s"}, 
{"clusterID": "%s", "clusterName": "%s", "clusterType": "%s", "description": "%s", "status": "%s", "environment": "%s", "engineType": "k8s"},
{"clusterID": "%s", "clusterName": "%s", "clusterType": "%s", "description": "%s", "status": "%s", "environment": "%s", "engineType": "mesos"}]
}`,
				clusterID1,
				clusterName1,
				clusterType1,
				clusterDesc1,
				clusterStatus1,
				clusterEnv1,
				clusterID2,
				clusterName2,
				clusterType2,
				clusterDesc2,
				clusterStatus2,
				clusterEnv2,
				clusterID3,
				clusterName3,
				clusterType3,
				clusterDesc3,
				clusterStatus3,
				clusterEnv3,
			)

			result := make(map[string]any)
			Expect(json.Unmarshal([]byte(resultJson), &result)).To(BeNil())
			Mock((*ApiClient).handleOperation).Return(result, nil).Build()

			clusters, err := cli.ListClustersByProject(ctx, projectID)
			Expect(err).To(BeNil())

			// 只有 engineType 为 k8s 的集群才会返回, 因此过滤掉非 k8s 的集群
			Expect(clusters).To(HaveLen(2))

			Expect(clusters[0]).To(Equal(Cluster{
				ID:          clusterID1,
				Name:        clusterName1,
				Type:        clusterType1,
				Description: clusterDesc1,
				Status:      clusterStatus1,
				Environment: clusterEnv1,
			}))

			Expect(clusters[1]).To(Equal(Cluster{
				ID:          clusterID2,
				Name:        clusterName2,
				Type:        clusterType2,
				Description: clusterDesc2,
				Status:      clusterStatus2,
				Environment: clusterEnv2,
			}))
		})
	})

	It("list namespaces by cluster", func() {
		namespace1 := stringx.Random(6)
		namespace2 := stringx.Random(6)

		resultJson := fmt.Sprintf(
			`{"data": [{"name": "%s", "status": "Active"}, {"name": "%s", "status": "Active"}]}`,
			namespace1,
			namespace2,
		)
		result := make(map[string]any)
		Expect(json.Unmarshal([]byte(resultJson), &result)).To(BeNil())
		Mock((*ApiClient).handleOperation).Return(result, nil).Build()

		namespaces, err := cli.ListNamespacesByCluster(ctx, uuid.New().String(), stringx.Random(6))
		Expect(err).To(BeNil())
		Expect(namespaces).To(HaveLen(2))
		Expect(namespaces[0].Name).To(Equal(namespace1))
		Expect(namespaces[1].Name).To(Equal(namespace2))
	})
})
