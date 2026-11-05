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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("NormalizeTags", func() {
	It("trims blanks and removes duplicates while keeping order", func() {
		Expect(NormalizeTags([]string{" v2 ", "v1", "", "v2", "  "})).To(Equal([]string{"v2", "v1"}))
	})

	It("returns empty for blank input", func() {
		Expect(NormalizeTags([]string{"", " "})).To(BeEmpty())
	})
})

var _ = Describe("PromoteImages", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("promotes every tag in order", func() {
		cli.EXPECT().PromoteAppImage(mock.Anything, appID, "v1").Return(nil).Once()
		cli.EXPECT().PromoteAppImage(mock.Anything, appID, "v2").Return(nil).Once()

		results := PromoteImages(ctx, cli, appID, []string{"v1", "v2"})

		Expect(results).To(HaveLen(2))
		Expect(results[0].Tag).To(Equal("v1"))
		Expect(results[1].Tag).To(Equal("v2"))
		Expect(FailedPromoteResults(results)).To(BeEmpty())
	})

	It("continues promoting remaining tags when one fails", func() {
		clientErr := errors.New("promote app image v1 failed: [404] -> not found")
		cli.EXPECT().PromoteAppImage(mock.Anything, appID, "v1").Return(clientErr).Once()
		cli.EXPECT().PromoteAppImage(mock.Anything, appID, "v2").Return(nil).Once()

		results := PromoteImages(ctx, cli, appID, []string{"v1", "v2"})

		failed := FailedPromoteResults(results)
		Expect(failed).To(HaveLen(1))
		Expect(failed[0].Tag).To(Equal("v1"))
		// client 错误已含操作名与 tag，原样透传，不再重复包装
		Expect(failed[0].Err).To(BeIdenticalTo(clientErr))
		Expect(results[1].Err).NotTo(HaveOccurred())
	})

	It("skips remaining tags after context is cancelled", func() {
		cancelCtx, cancel := context.WithCancel(ctx)
		cli.EXPECT().PromoteAppImage(mock.Anything, appID, "v1").
			RunAndReturn(func(context.Context, string, string) error {
				cancel()
				return nil
			}).Once()

		results := PromoteImages(cancelCtx, cli, appID, []string{"v1", "v2"})

		Expect(results).To(HaveLen(2))
		Expect(results[0].Err).NotTo(HaveOccurred())
		Expect(results[1].Err).To(MatchError(context.Canceled))
	})
})
