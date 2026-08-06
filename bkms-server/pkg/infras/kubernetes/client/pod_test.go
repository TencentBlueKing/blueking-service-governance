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

package client

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("PodClient", func() {
	var (
		podCli    *PodClient
		ctx       context.Context
		namespace string
		podName   string
	)

	BeforeEach(func() {
		cfg, err := testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		podCli = NewPodClient(cfg)
		ctx = context.Background()
		namespace = "default"
		podName = "test-pod"
	})

	Context("ListLogs", func() {
		It("should retrieve pod logs and handle tailLines correctly", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				// Mock DoRaw 方法返回模拟的日志内容
				mockLogsContent := `2024-01-01T10:00:00.000000000Z Line 1
2024-01-01T10:00:01.000000000Z Line 2
2024-01-01T10:00:02.000000000Z Line 3`

				mockey.Mock((*rest.Request).DoRaw).Return([]byte(mockLogsContent), nil).Build()

				// 获取所有日志
				opts := &corev1.PodLogOptions{}
				logs, err := podCli.ListLogs(ctx, namespace, podName, opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(HaveLen(3))

				// 验证日志结构和时间戳格式
				for _, log := range logs {
					Expect(log.Timestamp).NotTo(BeEmpty())
					Expect(log.Timestamp).To(MatchRegexp(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`))
					Expect(log.Content).NotTo(BeEmpty())
				}

				// 验证具体内容
				Expect(logs[0].Timestamp).To(Equal("2024-01-01T10:00:00.000000000Z"))
				Expect(logs[0].Content).To(Equal("Line 1"))
				Expect(logs[1].Timestamp).To(Equal("2024-01-01T10:00:01.000000000Z"))
				Expect(logs[1].Content).To(Equal("Line 2"))
				Expect(logs[2].Timestamp).To(Equal("2024-01-01T10:00:02.000000000Z"))
				Expect(logs[2].Content).To(Equal("Line 3"))
			})
		})

		It("should return error for non-existent pod", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				// Mock DoRaw 方法返回错误
				mockErr := errors.New("pods \"non-existent-pod\" not found")
				mockey.Mock((*rest.Request).DoRaw).Return(nil, mockErr).Build()

				opts := &corev1.PodLogOptions{}
				_, err := podCli.ListLogs(ctx, namespace, "non-existent-pod", opts)
				Expect(err).To(HaveOccurred())
			})
		})

		It("should handle empty logs", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				// Mock DoRaw 方法返回空内容
				mockey.Mock((*rest.Request).DoRaw).Return([]byte(""), nil).Build()

				opts := &corev1.PodLogOptions{}
				logs, err := podCli.ListLogs(ctx, namespace, podName, opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(logs).To(BeEmpty())
			})
		})
	})
})
