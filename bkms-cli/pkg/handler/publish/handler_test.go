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

var _ = Describe("Publisher instance resolution", func() {
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

	It("lists all running instances", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				mock.MatchedBy(func(opts client.ListAppInstancesOptions) bool {
					return opts.Page == 1 && opts.PageSize == client.DefaultListAppInstancesPageSize
				}),
			).
			Return(&client.PaginatedInstances{
				Count: "3",
				Results: []client.Instance{
					{ID: "pod-1", Status: instanceStatusRunning},
					{ID: "pod-x", Status: "Pending"},
					{ID: "pod-2", Status: instanceStatusRunning},
				},
			}, nil)

		publisher := NewPublisher(ctx, cli, "", appID, envName)
		instanceIDs, err := publisher.GetAllRunningInstanceIDs()

		Expect(err).NotTo(HaveOccurred())
		Expect(instanceIDs).To(Equal([]string{"pod-1", "pod-2"}))
	})

	It("returns an error when no running instances are found", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				mock.MatchedBy(func(opts client.ListAppInstancesOptions) bool {
					return opts.Page == 1 && opts.PageSize == client.DefaultListAppInstancesPageSize
				}),
			).
			Return(&client.PaginatedInstances{
				Count: "2",
				Results: []client.Instance{
					{ID: "pod-1", Status: "Pending"},
					{ID: "pod-2", Status: "Failed"},
				},
			}, nil)

		publisher := NewPublisher(ctx, cli, "", appID, envName)
		instanceIDs, err := publisher.GetAllRunningInstanceIDs()

		Expect(instanceIDs).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no running instances found"))
	})

	It("returns specified instances when they all exist", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				mock.MatchedBy(func(opts client.ListAppInstancesOptions) bool {
					return opts.Page == 1 && opts.PageSize == client.DefaultListAppInstancesPageSize
				}),
			).
			Return(&client.PaginatedInstances{
				Count: "3",
				Results: []client.Instance{
					{ID: "pod-1", Status: instanceStatusRunning},
					{ID: "pod-2", Status: "Pending"},
					{ID: "pod-3", Status: instanceStatusRunning},
				},
			}, nil)

		publisher := NewPublisher(ctx, cli, "", appID, envName)
		instanceIDs, err := publisher.GetSpecifiedInstanceIDs([]string{"pod-1", "pod-3"})

		Expect(err).NotTo(HaveOccurred())
		Expect(instanceIDs).To(Equal([]string{"pod-1", "pod-3"}))
	})

	It("returns an error when specified instances are not found", func() {
		cli.EXPECT().
			ListAppInstances(
				mock.Anything,
				appID,
				envName,
				mock.MatchedBy(func(opts client.ListAppInstancesOptions) bool {
					return opts.Page == 1 && opts.PageSize == client.DefaultListAppInstancesPageSize
				}),
			).
			Return(&client.PaginatedInstances{
				Count: "2",
				Results: []client.Instance{
					{ID: "pod-1", Status: instanceStatusRunning},
					{ID: "pod-2", Status: "Pending"},
				},
			}, nil)

		publisher := NewPublisher(ctx, cli, "", appID, envName)
		instanceIDs, err := publisher.GetSpecifiedInstanceIDs([]string{"pod-1", "pod-3"})

		Expect(instanceIDs).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("instances not found"))
		Expect(err.Error()).To(ContainSubstring("pod-3"))
	})

	It("returns an error when publish is called before preCheck", func() {
		publisher := NewPublisher(ctx, cli, "", appID, envName)

		err := publisher.Publish("/tmp/demo", []string{"pod-1"}, "")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("preCheck must be called before publish"))
	})
})
