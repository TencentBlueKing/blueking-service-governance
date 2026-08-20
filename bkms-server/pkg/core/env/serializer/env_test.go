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

package serializer_test

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
)

// newObjectID creates a deterministic ObjectID for tests.
func newObjectID(hex string) bson.ObjectID {
	id, err := bson.ObjectIDFromHex(hex)
	Expect(err).NotTo(HaveOccurred())
	return id
}

var _ = Describe("Env output serializers", func() {
	var (
		fixedTime = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		envIDHex  = "507f1f77bcf86cd799439011"
		envID     = newObjectID(envIDHex)
	)

	Describe("EnvOutput.FromModel", func() {
		It("converts all fields from an environment model", func() {
			env := envmodel.Environment{
				ID:          envID,
				Name:        "dev",
				DisplayName: "开发环境",
				Type:        "development",
				WorkspaceID: "ws-1",
				Kind:        envmodel.EnvironmentKindFeature,
				OwnerAppID:  "app-1",
				SourceEnvID: newObjectID("507f1f77bcf86cd799439012"),
				Cluster: envmodel.BizCluster{
					ProjectCode: "proj-code",
					ClusterID:   "cls-1",
					ClusterType: "single",
					Namespace:   "default",
				},
				Description: "desc",
				Creator:     "admin",
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
				AppIDs:      []string{"app-1", "app-2"},
				Status:      envmodel.EnvStatusReady,
			}

			output := new(serializer.EnvOutput).FromModel(env)

			Expect(output.ID).To(Equal(envIDHex))
			Expect(output.Name).To(Equal("dev"))
			Expect(output.DisplayName).To(Equal("开发环境"))
			Expect(output.Type).To(Equal("development"))
			Expect(output.CreatedAt).To(Equal(fixedTime))
			Expect(output.UpdatedAt).To(Equal(fixedTime))
			Expect(output.Cluster).NotTo(BeNil())
			Expect(output.Cluster.ClusterID).To(Equal("cls-1"))
			Expect(output.Cluster.ClusterType).To(Equal("single"))
			Expect(output.Cluster.Namespace).To(Equal("default"))
			Expect(output.Cluster.ProjectCode).To(Equal("proj-code"))
			Expect(output.Cluster.IsFederation).To(BeFalse())
			Expect(output.Status).To(Equal("Ready"))
			Expect(output.AppIDs).To(Equal([]string{"app-1", "app-2"}))
			Expect(output.Kind).To(Equal("feature"))
			Expect(output.OwnerAppID).To(Equal("app-1"))
			Expect(output.SourceEnvID).To(Equal("507f1f77bcf86cd799439012"))
		})

		It("marshals to correct JSON", func() {
			env := envmodel.Environment{
				ID:          envID,
				Name:        "test",
				DisplayName: "测试环境",
				Type:        "test",
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
				Cluster: envmodel.BizCluster{
					ClusterID:   "cls-1",
					ClusterType: "single",
					Namespace:   "default",
					ProjectCode: "proj-code",
				},
				AppIDs: []string{},
				Status: envmodel.EnvStatusNotReady,
			}

			output := new(serializer.EnvOutput).FromModel(env)
			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{
				"id": "507f1f77bcf86cd799439011",
				"name": "test",
				"displayName": "测试环境",
				"type": "test",
				"createdAt": "2026-01-15T10:30:00Z",
				"updatedAt": "2026-01-15T10:30:00Z",
				"cluster": {
					"clusterID": "cls-1",
					"clusterType": "single",
					"namespace": "default",
					"projectCode": "proj-code",
					"isFederation": false
				},
				"status": "NotReady",
				"appIDs": [],
				"kind": "standard"
			}`))
		})
	})

	Describe("ListFeatureEnvsOutput builders", func() {
		var featureEnv envmodel.Environment

		BeforeEach(func() {
			featureEnv = envmodel.Environment{
				ID:          envID,
				Name:        "feat-app-1-1",
				DisplayName: "登录功能联调",
				Type:        "development",
				Kind:        envmodel.EnvironmentKindFeature,
				OwnerAppID:  "app-1",
				SourceEnvID: newObjectID("507f1f77bcf86cd799439012"),
				Cluster: envmodel.BizCluster{
					ProjectCode: "proj-code",
					ClusterID:   "cls-1",
					ClusterType: "single",
					Namespace:   "feat-app-1-1",
				},
				Creator:   "admin",
				CreatedAt: fixedTime,
				UpdatedAt: fixedTime,
				AppIDs:    []string{"app-1"},
				Status:    envmodel.EnvStatusReady,
			}
		})

		It("builds a display-ready management item without unrelated internal fields", func() {
			sourceEnv := &envmodel.Environment{
				ID:          featureEnv.SourceEnvID,
				Name:        "dev",
				DisplayName: "开发环境",
			}

			output := serializer.NewListFeatureEnvsOutput(
				[]envmodel.Environment{featureEnv},
				map[string]*envmodel.Environment{featureEnv.Name: sourceEnv},
				nil,
			)
			Expect(output.Data).To(HaveLen(1))
			payload, err := json.Marshal(output.Data[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{
				"id": "507f1f77bcf86cd799439011",
				"name": "feat-app-1-1",
				"displayName": "登录功能联调",
				"type": "development",
				"sourceEnv": {
					"id": "507f1f77bcf86cd799439012",
					"name": "dev",
					"displayName": "开发环境",
					"isDeleted": false
				},
				"cluster": {
					"clusterID": "cls-1",
					"namespace": "feat-app-1-1"
				},
				"status": "Ready",
				"deployStatuses": null,
				"creator": "admin",
				"createdAt": "2026-01-15T10:30:00Z"
			}`))
		})

		It("keeps the source ID and marks a deleted source explicitly", func() {
			output := serializer.NewListFeatureEnvsOutput(
				[]envmodel.Environment{featureEnv},
				map[string]*envmodel.Environment{},
				nil,
			)

			Expect(output.Data).To(HaveLen(1))
			Expect(output.Data[0].SourceEnv).To(Equal(&serializer.FeatureEnvSourceOutput{
				ID:        "507f1f77bcf86cd799439012",
				IsDeleted: true,
			}))
		})

		It("returns an empty JSON array when the app has no feature environments", func() {
			output := serializer.NewListFeatureEnvsOutput(nil, nil, nil)

			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{"data": []}`))
		})

		It("includes deploy status only when the caller asks for it", func() {
			output := serializer.NewListFeatureEnvsOutput(
				[]envmodel.Environment{featureEnv},
				map[string]*envmodel.Environment{},
				map[string][]status.AppDeployStatus{
					featureEnv.Name: {
						{
							TrafficLaneName: "lane-a",
							DeployStatus:    "Deployed",
							ImageTag:        "v1.2.3",
						},
					},
				},
			)

			payload, err := json.Marshal(output.Data[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{
				"id": "507f1f77bcf86cd799439011",
				"name": "feat-app-1-1",
				"displayName": "登录功能联调",
				"type": "development",
				"sourceEnv": {
					"id": "507f1f77bcf86cd799439012",
					"isDeleted": true
				},
				"cluster": {
					"clusterID": "cls-1",
					"namespace": "feat-app-1-1"
				},
				"status": "Ready",
				"deployStatuses": [
					{
						"trafficLaneName": "lane-a",
						"deployStatus": "Deployed",
						"imageTag": "v1.2.3"
					}
				],
				"creator": "admin",
				"createdAt": "2026-01-15T10:30:00Z"
			}`))
		})

		It("returns an empty deployStatuses array when status query is requested but nothing is deployed", func() {
			output := serializer.NewListFeatureEnvsOutput(
				[]envmodel.Environment{featureEnv},
				map[string]*envmodel.Environment{},
				map[string][]status.AppDeployStatus{},
			)

			payload, err := json.Marshal(output.Data[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{
				"id": "507f1f77bcf86cd799439011",
				"name": "feat-app-1-1",
				"displayName": "登录功能联调",
				"type": "development",
				"sourceEnv": {
					"id": "507f1f77bcf86cd799439012",
					"isDeleted": true
				},
				"cluster": {
					"clusterID": "cls-1",
					"namespace": "feat-app-1-1"
				},
				"status": "Ready",
				"deployStatuses": [],
				"creator": "admin",
				"createdAt": "2026-01-15T10:30:00Z"
			}`))
		})
	})

	Describe("EnvDetailOutput.FromModel", func() {
		It("converts environment with deploy statuses", func() {
			env := envmodel.Environment{
				ID:          envID,
				Name:        "prod",
				DisplayName: "生产环境",
				Type:        "production",
				Creator:     "admin",
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
				Cluster: envmodel.BizCluster{
					ClusterID:   "cls-1",
					ClusterType: "single",
					Namespace:   "prod-ns",
					ProjectCode: "proj-code",
				},
				Description: "生产描述",
				AppIDs:      []string{"app-1"},
				Status:      envmodel.EnvStatusReady,
			}

			deployStatuses := []status.AppDeployStatus{
				{
					AppID:           "app-1",
					AppName:         "my-app",
					AppType:         "trpc",
					TrafficLaneName: "lane-1",
					DeployStatus:    "Deployed",
					ImageTag:        "v1.2.3",
				},
			}

			output := new(serializer.EnvDetailOutput).FromModel(env, deployStatuses)

			Expect(output.ID).To(Equal(envIDHex))
			Expect(output.Creator).To(Equal("admin"))
			Expect(output.Description).To(Equal("生产描述"))
			Expect(output.AppDeployStatuses).To(HaveLen(1))
			Expect(output.AppDeployStatuses[0].AppID).To(Equal("app-1"))
			Expect(output.AppDeployStatuses[0].AppName).To(Equal("my-app"))
			Expect(output.AppDeployStatuses[0].AppType).To(Equal("trpc"))
			Expect(output.AppDeployStatuses[0].TrafficLaneName).To(Equal("lane-1"))
			Expect(output.AppDeployStatuses[0].DeployStatus).To(Equal("Deployed"))
			Expect(output.AppDeployStatuses[0].ImageTag).To(Equal("v1.2.3"))
		})

		It("handles empty deploy statuses", func() {
			env := envmodel.Environment{
				ID:          envID,
				Name:        "dev",
				DisplayName: "开发环境",
				Type:        "development",
				Creator:     "admin",
				CreatedAt:   fixedTime,
				UpdatedAt:   fixedTime,
				Cluster:     envmodel.BizCluster{ClusterID: "cls-1"},
				Status:      envmodel.EnvStatusNotReady,
				AppIDs:      []string{},
			}

			output := new(serializer.EnvDetailOutput).FromModel(env, nil)
			Expect(output.AppDeployStatuses).To(BeEmpty())
		})
	})

	Describe("TrafficLaneOutput.FromModel", func() {
		It("converts all fields from a traffic lane model", func() {
			lane := &trafficmanager.TrafficLane{
				LaneId:                   "lane-1",
				LaneName:                 "lane-1",
				LaneDesc:                 "test lane",
				LaneType:                 "baseline",
				LaneLabels:               map[string]string{"service": "svc-1"},
				LaneAnnotations:          map[string]string{"key": "val"},
				LaneServiceVersionLabels: map[string]string{"version": "v1"},
			}

			output := new(serializer.TrafficLaneOutput).FromModel(lane)

			Expect(output.ID).To(Equal("lane-1"))
			Expect(output.Name).To(Equal("lane-1"))
			Expect(output.Description).To(Equal("test lane"))
			Expect(output.Type).To(Equal("baseline"))
			Expect(output.Labels).To(Equal(map[string]string{"service": "svc-1"}))
			Expect(output.Annotations).To(Equal(map[string]string{"key": "val"}))
			Expect(output.ServiceVersionLabels).To(Equal(map[string]string{"version": "v1"}))
		})

		It("marshals to correct JSON", func() {
			lane := &trafficmanager.TrafficLane{
				LaneId:   "lane-2",
				LaneName: "lane-2",
				LaneDesc: "desc",
				LaneType: "gray",
			}

			output := new(serializer.TrafficLaneOutput).FromModel(lane)
			payload, err := json.Marshal(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload).To(MatchJSON(`{
				"id": "lane-2",
				"name": "lane-2",
				"description": "desc",
				"type": "gray",
				"labels": null,
				"annotations": null,
				"serviceVersionLabels": null,
				"createdAt": "0001-01-01T00:00:00Z",
				"updatedAt": "0001-01-01T00:00:00Z"
			}`))
		})
	})
})

var _ = Describe("Env validation", func() {
	DescribeTable(
		"CreateEnvInput env type validation",
		func(envType string, shouldPass bool) {
			input := serializer.CreateEnvInput{
				Name:        "stag",
				DisplayName: "预发布环境",
				Type:        envType,
				Cluster: serializer.CreateEnvClusterInput{
					ClusterID:   "cls-1",
					ClusterType: "single",
					Namespace:   "stag",
				},
			}

			err := binding.Validator.ValidateStruct(input)
			if shouldPass {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("accepts staging type", string(bkmsenv.TypeStaging), true),
		Entry("rejects unknown type", "stage", false),
	)

	It("UpdateEnvBasicInfoInput accepts staging type", func() {
		envType := string(bkmsenv.TypeStaging)
		input := serializer.UpdateEnvBasicInfoInput{Type: &envType}

		Expect(binding.Validator.ValidateStruct(input)).NotTo(HaveOccurred())
	})
})
