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

package publish

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("ListHistory", func() {
	const (
		appID   = "demo-app"
		envName = "test"
	)

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("converts records into display rows with human-readable file size", func() {
		cli.EXPECT().
			ListDevModePublishRecords(mock.Anything, appID, envName, "").
			Return([]client.DevModePublishRecord{
				{
					Instance:   "pod-1",
					BinaryName: "server",
					FileSize:   1024,
					MD5:        "m1",
					Status:     "success",
					Operator:   "admin",
					UpdatedAt:  "2026-09-14T00:00:00Z",
				},
			}, nil)

		rows, err := ListHistory(ctx, cli, appID, envName, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
		Expect(rows[0].Instance).To(Equal("pod-1"))
		Expect(rows[0].BinaryName).To(Equal("server"))
		Expect(rows[0].FileSize).To(Equal("1.0 KB"))
		Expect(rows[0].MD5).To(Equal("m1"))
		Expect(rows[0].Status).To(Equal("success"))
		Expect(rows[0].Operator).To(Equal("admin"))
	})
})

var _ = DescribeTable("formatFileSize",
	func(bytes int64, expected string) {
		Expect(formatFileSize(bytes)).To(Equal(expected))
	},
	Entry("bytes", int64(500), "500 B"),
	Entry("kilobytes", int64(1024), "1.0 KB"),
	Entry("megabytes", int64(3*1024*1024), "3.0 MB"),
	Entry("gigabytes", int64(5*1024*1024*1024), "5.0 GB"),
)
