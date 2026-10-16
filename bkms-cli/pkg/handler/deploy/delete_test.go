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

package deploy

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
)

var _ = Describe("DeleteDeployBatch", func() {
	ctx := context.Background()

	It("deletes deployment for each environment", func() {
		cli := mocks.NewMockClient(GinkgoT())
		cli.EXPECT().DeleteTrpcDeploy(ctx, "app-1", "test").Return(nil).Once()
		cli.EXPECT().DeleteTrpcDeploy(ctx, "app-1", "staging").Return(nil).Once()

		err := DeleteDeployBatch(ctx, cli, constant.AppTypeTrpc, "app-1", []string{"test", "staging"}, "")
		Expect(err).NotTo(HaveOccurred())
	})

	It("aggregates failures across environments", func() {
		cli := mocks.NewMockClient(GinkgoT())
		cli.EXPECT().DeleteTrpcDeploy(ctx, "app-1", "test").Return(context.DeadlineExceeded).Once()
		cli.EXPECT().DeleteTrpcDeploy(ctx, "app-1", "staging").Return(nil).Once()

		err := DeleteDeployBatch(ctx, cli, constant.AppTypeTrpc, "app-1", []string{"test", "staging"}, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("env test"))
		// 失败明细已逐环境输出，统一出口不再重复打印
		var reported *clierr.ReportedError
		Expect(errors.As(err, &reported)).To(BeTrue())
	})

	It("requires deploy-id for helm applications", func() {
		cli := mocks.NewMockClient(GinkgoT())

		err := DeleteDeployBatch(ctx, cli, constant.AppTypeHelm, "app-1", []string{"test"}, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("deploy-id"))
	})
})
