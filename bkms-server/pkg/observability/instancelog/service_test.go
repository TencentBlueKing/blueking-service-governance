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

package instancelog

import (
	"context"
	stderrors "errors"
	"io"
	"strings"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
)

var _ = Describe("Instance log service helpers", func() {
	DescribeTable("sanitizeAttachmentFilenamePart",
		func(input, expected string) {
			Expect(sanitizeAttachmentFilenamePart(input)).To(Equal(expected))
		},
		Entry("returns fallback for empty string", "", "unknown"),
		Entry("returns fallback for whitespace", "   \t  ", "unknown"),
		Entry("preserves allowed characters", "pod.main-1_ok", "pod.main-1_ok"),
		Entry("replaces special characters", "pod/name:1", "pod_name_1"),
	)

	It("builds a sanitized instance log download filename", func() {
		filename := buildInstanceLogDownloadFilename("pod/name", "main container")
		Expect(filename).To(MatchRegexp(`^pod_name-main_container-\d{14}\.log$`))
	})
})

var _ = Describe("NewLogManager", func() {
	var oldConfig *config.Config

	BeforeEach(func() {
		oldConfig = config.G
		config.G = &config.Config{
			BCS: config.BCSConfig{
				BaseUrl: "http://example.invalid",
				Token:   "token",
			},
			Development: config.DevConfig{
				UseKubeConfigCluster: false,
			},
		}
	})

	AfterEach(func() {
		config.G = oldConfig
	})

	It("returns deploy record not found errors from the store", func() {
		mockey.PatchConvey("test GetLatest returns not found error", GinkgoT(), func() {
			mockey.Mock((*appmodeldeploy.RecordStoreMongo).GetLatest).
				Return(nil, appmodeldeploy.ErrDeployRecordNotFound).
				Build()

			_, err := NewLogManager(
				context.Background(),
				&appmodeldeploy.RecordStoreMongo{},
				&bkmsapp.Application{ID: "app-1", Type: bkmsapp.AppTypeTRPC},
				&envmodel.Environment{
					Name: "t",
					Cluster: envmodel.BizCluster{
						ClusterID: "cluster-1",
						Namespace: "default",
					},
				},
				"",
				"pod-1",
			)

			Expect(err).To(HaveOccurred())
			Expect(stderrors.Is(err, appmodeldeploy.ErrDeployRecordNotFound)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("get latest deploy record for app app-1"))
		})
	})

	It("rejects appmodel deploy records without label selectors", func() {
		mockey.PatchConvey("test GetLatest returns record with empty label selector", GinkgoT(), func() {
			mockey.Mock((*appmodeldeploy.RecordStoreMongo).GetLatest).
				Return(&appmodeldeploy.Record{
					ID:            bson.NewObjectID(),
					AppID:         "app-1",
					EnvName:       "test",
					Namespace:     "record-ns",
					LabelSelector: map[string]string{},
				}, nil).Build()

			_, err := NewLogManager(
				context.Background(),
				&appmodeldeploy.RecordStoreMongo{},
				&bkmsapp.Application{ID: "app-1", Type: bkmsapp.AppTypeTRPC},
				&envmodel.Environment{
					Name: "test",
					Cluster: envmodel.BizCluster{
						ClusterID: "cluster-1",
						Namespace: "default",
					},
				},
				"",
				"pod-1",
			)

			Expect(err).To(MatchError("deploy record label selector is empty"))
		})
	})
})

var _ = Describe("LogManager logs", func() {
	const testLogTailLines = int64(100)
	const (
		testNamespace     = "default"
		testInstanceID    = "pod-1"
		testContainerName = "main"
	)

	var (
		ctx     context.Context
		manager *LogManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = &LogManager{
			namespace:     testNamespace,
			containerName: testContainerName,
			podClient:     &k8sclient.PodClient{},
		}
	})

	newPodWithRestartCount := func(restartCount int64) *unstructured.Unstructured {
		return &unstructured.Unstructured{
			Object: map[string]any{
				"metadata": map[string]any{
					"name":      testInstanceID,
					"namespace": testNamespace,
				},
				"status": map[string]any{
					"containerStatuses": []any{
						map[string]any{
							"name":         testContainerName,
							"restartCount": restartCount,
						},
					},
				},
			},
		}
	}

	It("should skip previous log listing when container has not restarted", func() {
		mockey.PatchConvey("previous logs absent", GinkgoT(), func() {
			listLogsCalled := 0

			mockey.Mock((*k8sclient.Client).Get).Return(newPodWithRestartCount(0), nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).To(
				func(
					_ *k8sclient.PodClient, _ context.Context, _, _ string, _ *corev1.PodLogOptions,
				) ([]k8sclient.LogEntry, error) {
					listLogsCalled++
					return nil, nil
				},
			).Build()

			logs, err := manager.ListLogs(ctx, testInstanceID, true, testLogTailLines)

			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(BeNil())
			Expect(listLogsCalled).To(Equal(0))
		})
	})

	It("should list previous logs when container has restarted", func() {
		mockey.PatchConvey("previous logs present", GinkgoT(), func() {
			listLogsCalled := 0
			var gotOpts *corev1.PodLogOptions
			expectedLogs := []k8sclient.LogEntry{
				{Timestamp: "2024-01-01T10:00:00.000000000Z", Content: "line 1"},
				{Timestamp: "2024-01-01T10:00:01.000000000Z", Content: "line 2"},
			}

			mockey.Mock((*k8sclient.Client).Get).Return(newPodWithRestartCount(1), nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).To(
				func(
					_ *k8sclient.PodClient,
					_ context.Context,
					namespace, podName string,
					opts *corev1.PodLogOptions,
				) ([]k8sclient.LogEntry, error) {
					listLogsCalled++
					Expect(namespace).To(Equal(testNamespace))
					Expect(podName).To(Equal(testInstanceID))
					gotOpts = opts
					return expectedLogs, nil
				},
			).Build()

			logs, err := manager.ListLogs(ctx, testInstanceID, true, testLogTailLines)

			Expect(err).NotTo(HaveOccurred())
			Expect(listLogsCalled).To(Equal(1))
			Expect(gotOpts).NotTo(BeNil())
			Expect(gotOpts.Container).To(Equal(testContainerName))
			Expect(gotOpts.Previous).To(BeTrue())
			Expect(gotOpts.TailLines).NotTo(BeNil())
			Expect(*gotOpts.TailLines).To(Equal(testLogTailLines))
			Expect(logs).To(HaveLen(2))
			Expect(logs[0].Timestamp).To(Equal(expectedLogs[0].Timestamp))
			Expect(logs[0].Content).To(Equal(expectedLogs[0].Content))
			Expect(logs[1].Timestamp).To(Equal(expectedLogs[1].Timestamp))
			Expect(logs[1].Content).To(Equal(expectedLogs[1].Content))
		})
	})

	It("should skip previous log download when container has not restarted", func() {
		mockey.PatchConvey("previous download absent", GinkgoT(), func() {
			openLogsStreamCalled := 0

			mockey.Mock((*k8sclient.Client).Get).Return(newPodWithRestartCount(0), nil).Build()
			mockey.Mock((*k8sclient.PodClient).OpenLogsStream).To(
				func(
					_ *k8sclient.PodClient, _ context.Context, _, _ string, _ *corev1.PodLogOptions,
				) (io.ReadCloser, error) {
					openLogsStreamCalled++
					return io.NopCloser(strings.NewReader("unexpected")), nil
				},
			).Build()

			result, err := manager.PrepareDownload(ctx, testInstanceID, true)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			defer result.Reader.Close()
			content, err := io.ReadAll(result.Reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(BeEmpty())
			Expect(openLogsStreamCalled).To(Equal(0))
			Expect(result.Filename).To(ContainSubstring(testInstanceID))
			Expect(result.Filename).To(ContainSubstring(testContainerName))
		})
	})
})
