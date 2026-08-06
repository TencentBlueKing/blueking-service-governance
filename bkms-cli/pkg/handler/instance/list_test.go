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

package instance

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("ListInstances", func() {
	const (
		appID   = "demo"
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

	It("lists all instances across pages", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				matchListInstancesOptions(1, client.DefaultListAppInstancesPageSize),
			).
			Return(&client.PaginatedInstances{
				Count: "2",
				Results: []client.Instance{{
					ID: "pod-1",
				}},
			}, nil)
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				matchListInstancesOptions(2, client.DefaultListAppInstancesPageSize),
			).
			Return(&client.PaginatedInstances{
				Count: "2",
				Results: []client.Instance{{
					ID: "pod-2",
				}},
			}, nil)

		instances, err := ListInstances(ctx, cli, appID, envName, ListInstancesOptions{})

		Expect(err).NotTo(HaveOccurred())
		Expect(instances).To(HaveLen(2))
		Expect(instances[0].ID).To(Equal("pod-1"))
		Expect(instances[1].ID).To(Equal("pod-2"))
	})

	It("filters instances by status in handler", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				matchListInstancesOptions(1, client.DefaultListAppInstancesPageSize),
			).
			Return(&client.PaginatedInstances{
				Count: "2",
				Results: []client.Instance{
					{
						ID:     "pod-1",
						Status: "Running",
					},
					{
						ID:     "pod-2",
						Status: "Failed",
					},
				},
			}, nil)

		instances, err := ListInstances(ctx, cli, appID, envName, ListInstancesOptions{
			Status: "Running",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(instances).To(HaveLen(1))
		Expect(instances[0].ID).To(Equal("pod-1"))
		Expect(instances[0].Status).To(Equal("Running"))
	})

	It("returns an error when response data is empty", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				matchListInstancesOptions(1, client.DefaultListAppInstancesPageSize),
			).
			Return(nil, nil)

		instances, err := ListInstances(ctx, cli, appID, envName, ListInstancesOptions{})

		Expect(instances).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty app instances"))
	})
})

func matchListInstancesOptions(page, pageSize int) interface{} {
	return mock.MatchedBy(func(opts client.ListAppInstancesOptions) bool {
		return opts.Page == page && opts.PageSize == pageSize
	})
}
