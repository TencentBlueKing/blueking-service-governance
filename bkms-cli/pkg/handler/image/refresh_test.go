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

package image

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("RefreshImages", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	DescribeTable("accepts normal statuses",
		func(status string) {
			cli.EXPECT().RefreshAppImages(mock.Anything, appID).
				Return(&client.RefreshAppImagesResult{Status: status, AddedTagCnt: "1"}, nil).Once()

			result, err := RefreshImages(ctx, cli, appID)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal(status))
		},
		Entry("success", client.RefreshAppImagesStatusSuccess),
		Entry("refreshing", client.RefreshAppImagesStatusRefreshing),
	)

	DescribeTable("rejects failed and unexpected statuses",
		func(status, message, expectedErr string) {
			cli.EXPECT().RefreshAppImages(mock.Anything, appID).
				Return(&client.RefreshAppImagesResult{Status: status, Message: message}, nil).Once()

			result, err := RefreshImages(ctx, cli, appID)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError(ContainSubstring(expectedErr)))
		},
		Entry("failed", client.RefreshAppImagesStatusFailed, "registry timeout",
			"refresh app images failed: registry timeout"),
		Entry("empty response", "", "", `unexpected status ""`),
		Entry("unknown status", "weird", "", `unexpected status "weird"`),
	)

	It("returns client error as is", func() {
		cli.EXPECT().RefreshAppImages(mock.Anything, appID).Return(nil, errors.New("boom")).Once()

		_, err := RefreshImages(ctx, cli, appID)

		Expect(err).To(MatchError("boom"))
	})
})
