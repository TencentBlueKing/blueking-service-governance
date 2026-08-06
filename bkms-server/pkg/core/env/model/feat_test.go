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

package model

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("FeatureEnvCounterStoreMongo", func() {
	var (
		ctx   context.Context
		store FeatureEnvCounterStore
		diApp *fxtest.App
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			database.PrivateFxModule,
			fx.Provide(NewFeatureEnvCounterStoreMongo),
			fx.Populate(&store),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	It("allocates independent sequential indexes per app", func() {
		appID1 := stringx.Random(10)
		appID2 := stringx.Random(10)

		index, err := store.Next(ctx, appID1)
		Expect(err).NotTo(HaveOccurred())
		Expect(index).To(Equal(int64(1)))

		index, err = store.Next(ctx, appID1)
		Expect(err).NotTo(HaveOccurred())
		Expect(index).To(Equal(int64(2)))

		index, err = store.Next(ctx, appID2)
		Expect(err).NotTo(HaveOccurred())
		Expect(index).To(Equal(int64(1)))
	})
})
