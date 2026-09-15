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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("Publisher", func() {
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

	It("returns an error when publish is called before preCheck", func() {
		publisher := NewPublisher(ctx, cli, appID, envName)

		err := publisher.Publish("/tmp/demo", []string{"pod-1"})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("preCheck must be called before publish"))
	})

	Describe("PreCheck", func() {
		It("succeeds when preflight returns valid data with specified instances", func() {
			cli.EXPECT().
				DevModePublishPreflight(mock.Anything, appID, envName, mock.Anything, mock.Anything).
				Return(&client.DevModePreflightData{
					Token:       "test-token",
					Address:     "https://bcs-api.example.com/clusters/BCS-K8S-12345/",
					Namespace:   "bkms-test",
					InstanceIDs: []string{"pod-1"},
					DevMode: &client.DevModeConfig{
						WorkPath:  "/data/bkms/dev-mode/trpc",
						MountPath: "/data/bkms/dev-mode/trpc/configmap-scripts",
					},
				}, nil)

			publisher := NewPublisher(ctx, cli, appID, envName)
			err := publisher.PreCheck([]string{"pod-1"}, false)

			Expect(err).NotTo(HaveOccurred())
			Expect(publisher.GetInstanceIDs()).To(Equal([]string{"pod-1"}))
		})

		It("succeeds when preflight returns valid data with publishAll", func() {
			cli.EXPECT().
				DevModePublishPreflight(mock.Anything, appID, envName, mock.Anything, mock.Anything).
				Return(&client.DevModePreflightData{
					Token:       "test-token",
					Address:     "https://bcs-api.example.com/clusters/BCS-K8S-12345/",
					Namespace:   "bkms-test",
					InstanceIDs: []string{"pod-1", "pod-2"},
					DevMode: &client.DevModeConfig{
						WorkPath:  "/data/bkms/dev-mode/trpc",
						MountPath: "/data/bkms/dev-mode/trpc/configmap-scripts",
					},
				}, nil)

			publisher := NewPublisher(ctx, cli, appID, envName)
			err := publisher.PreCheck(nil, true)

			Expect(err).NotTo(HaveOccurred())
			Expect(publisher.GetInstanceIDs()).To(Equal([]string{"pod-1", "pod-2"}))
		})

		It("returns an error when preflight fails", func() {
			cli.EXPECT().
				DevModePublishPreflight(mock.Anything, appID, envName, mock.Anything, mock.Anything).
				Return(nil, errors.New("devmode publish preflight failed: [400] -> dev mode is not enabled"))

			publisher := NewPublisher(ctx, cli, appID, envName)
			err := publisher.PreCheck([]string{"pod-1"}, false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("publish preflight failed"))
		})

		It("returns an error when publish is not supported", func() {
			cli.EXPECT().
				DevModePublishPreflight(mock.Anything, appID, envName, mock.Anything, mock.Anything).
				Return(nil, errors.New("devmode publish preflight failed: [400] -> publish is not supported"))

			publisher := NewPublisher(ctx, cli, appID, envName)
			err := publisher.PreCheck([]string{"pod-1"}, false)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("publish is not supported"))
		})

		It("returns an error when no running instances found", func() {
			cli.EXPECT().
				DevModePublishPreflight(mock.Anything, appID, envName, mock.Anything, mock.Anything).
				Return(nil, errors.New("devmode publish preflight failed: [404] -> no running instances found"))

			publisher := NewPublisher(ctx, cli, appID, envName)
			err := publisher.PreCheck(nil, true)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no running instances found"))
		})
	})

	Describe("report", func() {
		It("reports publish result to server", func() {
			cli.EXPECT().
				ReportDevModePublish(mock.Anything, appID, envName, mock.Anything).
				Return(nil)

			publisher := NewPublisher(ctx, cli, appID, envName)
			publisher.report("server", 1024, "abc", []client.DevModePublishResult{
				{Instance: "pod-1", Status: publishStatusSuccess},
			})
		})

		It("skips reporting when there is no result", func() {
			publisher := NewPublisher(ctx, cli, appID, envName)
			publisher.report("server", 1024, "abc", nil)
		})
	})
})
