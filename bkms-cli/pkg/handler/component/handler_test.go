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

package component

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

var _ = Describe("ListAppComponents", func() {
	const appID = "demo-app"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("marks ref and inst component instances and returns both by default", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeTrpc,
			AppModelSpec: &client.AppModelSpec{
				Components: []client.AppComponent{
					{Name: "shared-limits", RefWorkspaceCompName: "ws-limits", Type: "ResourceLimits"},
					{Name: "local-vol", Type: "VolumeSecret", Version: "v1.0.0"},
				},
			},
		}, nil)

		comps, err := ListAppComponents(ctx, cli, appID, "")

		Expect(err).NotTo(HaveOccurred())
		Expect(comps).To(HaveLen(2))
		Expect(comps[0].Kind).To(Equal(client.AppComponentKindRef))
		Expect(comps[1].Kind).To(Equal(client.AppComponentKindInst))
	})

	It("filters referenced instances when kind=ref", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeTaf,
			AppModelSpec: &client.AppModelSpec{
				Components: []client.AppComponent{
					{Name: "shared-limits", RefWorkspaceCompName: "ws-limits"},
					{Name: "local-vol", Type: "VolumeSecret"},
				},
			},
		}, nil)

		comps, err := ListAppComponents(ctx, cli, appID, client.AppComponentKindRef)

		Expect(err).NotTo(HaveOccurred())
		Expect(comps).To(HaveLen(1))
		Expect(comps[0].Name).To(Equal("shared-limits"))
		Expect(comps[0].Kind).To(Equal(client.AppComponentKindRef))
	})

	It("accepts bkmsApp applications", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type:      constant.AppTypeBKMSApp,
			Framework: constant.FrameworkBlank,
			AppModelSpec: &client.AppModelSpec{
				Components: []client.AppComponent{
					{Name: "local-vol", Type: "VolumeSecret", Version: "v1.0.0"},
				},
			},
		}, nil)

		comps, err := ListAppComponents(ctx, cli, appID, "")

		Expect(err).NotTo(HaveOccurred())
		Expect(comps).To(HaveLen(1))
		Expect(comps[0].Kind).To(Equal(client.AppComponentKindInst))
	})

	It("rejects unsupported app types", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeHelm,
		}, nil)

		comps, err := ListAppComponents(ctx, cli, appID, "")

		Expect(comps).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("only appmodel apps"))
	})

	It("rejects unknown kind filters", func() {
		_, err := ListAppComponents(ctx, cli, appID, "unknown")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported kind"))
	})
})

var _ = Describe("CreateAppComponentRef", func() {
	const appID = "demo-app"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("creates a workspace component reference and returns the generated name", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeTrpc,
		}, nil)
		cli.EXPECT().
			CreateAppComponent(mock.Anything, appID, mock.MatchedBy(func(body any) bool {
				input, ok := body.(CreateAppComponentRefInput)
				return ok && input.RefWorkspaceCompName == "ws-limits" && input.CompName == "my-limits"
			})).
			Return("my-limits", nil)

		name, err := CreateAppComponentRef(ctx, cli, appID, "ws-limits", "my-limits")

		Expect(err).NotTo(HaveOccurred())
		Expect(name).To(Equal("my-limits"))
	})

	It("does not call create when the app type is helm", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeHelm,
		}, nil)

		name, err := CreateAppComponentRef(ctx, cli, appID, "ws-limits", "")

		Expect(name).To(BeEmpty())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("only appmodel apps"))
	})
})

var _ = Describe("DeleteAppComponent", func() {
	const appID = "demo-app"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("deletes an app component after verifying app type", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(&client.AppDetail{
			Type: constant.AppTypeTaf,
		}, nil)
		cli.EXPECT().DeleteAppComponent(mock.Anything, appID, "shared-limits").Return(nil)

		Expect(DeleteAppComponent(ctx, cli, appID, "shared-limits")).To(Succeed())
	})

	It("returns get app detail errors", func() {
		cli.EXPECT().GetAppDetail(mock.Anything, appID).Return(nil, errors.New("not found"))

		err := DeleteAppComponent(ctx, cli, appID, "shared-limits")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not found"))
	})
})
