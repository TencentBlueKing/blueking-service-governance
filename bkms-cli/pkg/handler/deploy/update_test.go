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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

var _ = Describe("appModelUpdateSpec.handler", func() {
	ctx := context.Background()

	newSpec := func() *appModelUpdateSpec {
		return &appModelUpdateSpec{
			UpdateMode: imageUpdateMode,
			ImageTag:   "v2",
			Strategy:   inplaceUpdateStrategy,
			appType:    constant.AppTypeTrpc,
		}
	}

	It("skips env without deploy record", func() {
		cli := mocks.NewMockClient(GinkgoT())
		cli.EXPECT().ListTrpcDeployRecords(ctx, "app-1", "test", "", "").
			Return([]client.AppModelDeployRecord{}, nil).Once()

		err := newSpec().handler(ctx, cli, "app-1", "test")
		Expect(err).To(MatchError(errNotDeployed))
	})

	It("skips env whose latest deploy is uninstalled", func() {
		cli := mocks.NewMockClient(GinkgoT())
		cli.EXPECT().ListTrpcDeployRecords(ctx, "app-1", "test", "", "").
			Return([]client.AppModelDeployRecord{{Status: constant.DeployStatusUninstalled}}, nil).Once()

		err := newSpec().handler(ctx, cli, "app-1", "test")
		Expect(err).To(MatchError(errNotDeployed))
	})

	It("updates env that has been deployed", func() {
		cli := mocks.NewMockClient(GinkgoT())
		cli.EXPECT().ListTrpcDeployRecords(ctx, "app-1", "test", "", "").
			Return([]client.AppModelDeployRecord{{Status: "deployed", ImageTag: "v1", Replicas: 2}}, nil).Once()
		cli.EXPECT().BatchUpdateInstance(ctx, "app-1", "test", "v2", string(inplaceUpdateStrategy)).
			Return(nil).Once()

		Expect(newSpec().handler(ctx, cli, "app-1", "test")).To(Succeed())
	})
})
