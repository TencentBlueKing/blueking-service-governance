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

package clusterresources

import (
	"context"
	"time"

	. "github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("ClusterResources API Client", func() {
	var cli *ApiClient
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

	It("list events", func() {
		PatchConvey("test", GinkgoT(), func() {
			Mock((*ApiClient).handleOperation).Return(mockedListEventsApiResult, nil).Build()

			paginatedEvents, err := cli.ListEvents(ctx, "test-project", "BCS-K8S-26666", ListEventParams{
				Namespace:     "bcs-system",
				ResourceKinds: []string{"ReplicaSet", "PersistentVolumeClaim"},
				ResourceNames: []string{"blueking-nginx-ingress-58877d456b", "data-etcd-0"},
				Page:          1,
				PageSize:      2,
			})
			Expect(err).To(BeNil())

			// 校验总数
			Expect(paginatedEvents.Count).To(Equal(int64(7035)))
			// 校验事件条目数量
			Expect(paginatedEvents.Data).To(HaveLen(2))

			// 校验第一个事件条目
			Expect(paginatedEvents.Data[0]).To(Equal(EventEntry{
				ClusterID: "BCS-K8S-26666",
				Namespace: "bcs-system",
				Level:     "Warning",
				Content: `Error creating: pods "blueking-nginx-ingress-58877d456b-" is forbidden: ` +
					`error looking up service account bcs-system/default-backend: ` +
					`serviceaccount "default-backend" not found`,
				Type:          "FailedCreate",
				ComponentName: "replicaset-controller",
				ResourceKind:  "ReplicaSet",
				ResourcesName: "blueking-nginx-ingress-58877d456b",
				CreatedAt:     time.Date(2026, 2, 9, 12, 11, 16, 0, time.UTC),
			}))

			// 校验第二个事件条目
			Expect(paginatedEvents.Data[1]).To(Equal(EventEntry{
				ClusterID: "BCS-K8S-26666",
				Namespace: "observability-practice-guide",
				Level:     "Normal",
				Content: `waiting for a volume to be created, either by external provisioner ` +
					`"com.example.csi.cbs" or manually created by system administrator`,
				Type:          "ExternalProvisioning",
				ComponentName: "persistentvolume-controller",
				ResourceKind:  "PersistentVolumeClaim",
				ResourcesName: "data-etcd-0",
				CreatedAt:     time.Date(2026, 2, 9, 12, 10, 37, 0, time.UTC),
			}))
		})
	})
})

// mockedListEventsApiResult mock 的 ListEvents API 接口返回结果
var mockedListEventsApiResult = map[string]any{
	"result":  true,
	"code":    0,
	"message": "success",
	"data": []any{
		map[string]any{
			"type": "FailedCreate",
			"data": map[string]any{
				"firstTimestamp": "2020-07-01T13:30:01Z",
				"source": map[string]any{
					"component": "replicaset-controller",
				},
				"involvedObject": map[string]any{
					"uid":             "1966e5e8-f97f-4f76-a929-d4f52fca05b1",
					"apiVersion":      "apps/v1",
					"kind":            "ReplicaSet",
					"name":            "blueking-nginx-ingress-58877d456b",
					"namespace":       "bcs-system",
					"resourceVersion": "236329731",
				},
				"apiVersion":        "v1",
				"type":              "Warning",
				"reportingInstance": "",
				"lastTimestamp":     "2026-02-09T12:11:16Z",
				"metadata": map[string]any{
					"uid":               "2599bf81-6283-4934-9e55-aeee1b3ecca2",
					"creationTimestamp": "2020-07-01T13:30:01Z",
					"name":              "blueking-nginx-ingress-58877d456b.176ead4029a73946",
					"namespace":         "bcs-system",
					"resourceVersion":   "795091919",
					"selfLink": "/api/v1/namespaces/bcs-system/events/blueking-nginx-ingress-" +
						"58877d456b.176ead4029a73946",
				},
				"message": `Error creating: pods "blueking-nginx-ingress-58877d456b-" is forbidden: ` +
					`error looking up service account bcs-system/default-backend: ` +
					`serviceaccount "default-backend" not found`,
				"eventTime":          nil,
				"count":              float64(83097),
				"reportingComponent": "",
				"reason":             "FailedCreate",
				"kind":               "Event",
			},
			"extraInfo": map[string]any{
				"namespace": "bcs-system",
				"name":      "blueking-nginx-ingress-58877d456b",
				"kind":      "ReplicaSet",
			},
			"kind": "ReplicaSet",
			"describe": `Error creating: pods "blueking-nginx-ingress-58877d456b-" is forbidden: ` +
				`error looking up service account bcs-system/default-backend: ` +
				`serviceaccount "default-backend" not found`,
			"level":      "Warning",
			"clusterId":  "BCS-K8S-26666",
			"createTime": "2026-02-09T12:11:16.942Z",
			"component":  "replicaset-controller",
			"env":        "k8s",
			"eventTime":  "2026-02-09T12:11:16Z",
		},
		map[string]any{
			"env":        "k8s",
			"clusterId":  "BCS-K8S-26666",
			"eventTime":  "2026-02-09T12:10:37Z",
			"createTime": "2026-02-09T12:10:37.099Z",
			"kind":       "PersistentVolumeClaim",
			"extraInfo": map[string]any{
				"kind":      "PersistentVolumeClaim",
				"namespace": "observability-practice-guide",
				"name":      "data-etcd-0",
			},
			"type": "ExternalProvisioning",
			"describe": `waiting for a volume to be created, either by external provisioner ` +
				`"com.example.csi.cbs" or manually created by system administrator`,
			"level": "Normal",
			"data": map[string]any{
				"apiVersion": "v1",
				"type":       "Normal",
				"involvedObject": map[string]any{
					"namespace":       "observability-practice-guide",
					"resourceVersion": "68755520",
					"uid":             "14dd970a-032b-4292-8909-104be1c244d2",
					"apiVersion":      "v1",
					"kind":            "PersistentVolumeClaim",
					"name":            "data-etcd-0",
				},
				"firstTimestamp": "2022-10-28T07:00:36Z",
				"metadata": map[string]any{
					"selfLink": "/api/v1/namespaces/observability-practice-guide/" +
						"events/data-etcd-0.1722297ebfc91aaf",
					"uid":               "32cfae6c-1f15-4f01-81ff-0f371bf072ec",
					"creationTimestamp": "2022-10-28T07:00:36Z",
					"name":              "data-etcd-0.1722297ebfc91aaf",
					"namespace":         "observability-practice-guide",
					"resourceVersion":   "795091663",
				},
				"count":              float64(6914536),
				"reason":             "ExternalProvisioning",
				"eventTime":          nil,
				"reportingInstance":  "",
				"reportingComponent": "",
				"source": map[string]any{
					"component": "persistentvolume-controller",
				},
				"message": `waiting for a volume to be created, either by external provisioner ` +
					`"com.example.csi.cbs" or manually created by system administrator`,
				"lastTimestamp": "2026-02-09T12:10:37Z",
				"kind":          "Event",
			},
			"component": "persistentvolume-controller",
		},
	},
	"total": float64(7035),
}
