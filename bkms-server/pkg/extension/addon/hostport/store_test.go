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

package hostport_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
)

var _ = Describe("HostPortStore", func() {
	var (
		ctx       context.Context
		diApp     *fxtest.App
		store     hostport.HostPortStore
		testAppID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			hostport.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		testAppID = "hostport-app-" + stringx.Random(6)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, testAppID)
		diApp.RequireStop()
	})

	Describe("ListPorts", func() {
		It("returns empty slice when config document does not exist", func() {
			ports, err := store.ListPorts(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{}))
		})
	})

	Describe("Get", func() {
		It("returns ErrConfigNotFound when config document does not exist", func() {
			_, err := store.Get(ctx, testAppID)
			Expect(err).To(MatchError(hostport.ErrConfigNotFound))
		})
	})

	Describe("ReplacePorts", func() {
		It("lazily creates config and stores sorted unique ports", func() {
			config, err := store.ReplacePorts(ctx, testAppID, []int32{8080, 80, 80})
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Ports).To(Equal([]int32{80, 8080}))

			ports, err := store.ListPorts(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{80, 8080}))
		})

		It("replaces the full ports list", func() {
			_, err := store.ReplacePorts(ctx, testAppID, []int32{80, 8080})
			Expect(err).NotTo(HaveOccurred())

			config, err := store.ReplacePorts(ctx, testAppID, []int32{443})
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Ports).To(Equal([]int32{443}))
		})

		It("deletes the config document when ports is empty", func() {
			_, err := store.ReplacePorts(ctx, testAppID, []int32{80})
			Expect(err).NotTo(HaveOccurred())

			config, err := store.ReplacePorts(ctx, testAppID, []int32{})
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Ports).To(Equal([]int32{}))

			_, err = store.Get(ctx, testAppID)
			Expect(err).To(MatchError(hostport.ErrConfigNotFound))
		})
	})

	Describe("UpsertEnvState / RemoveEnvState", func() {
		It("persists applied ports for an env and can clear them", func() {
			Expect(store.UpsertEnvState(ctx, testAppID, "gray", []int32{8080, 80})).To(Succeed())

			config, err := store.Get(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.EnvStates["gray"].AppliedPorts).To(Equal([]int32{80, 8080}))

			Expect(store.RemoveEnvState(ctx, testAppID, "gray")).To(Succeed())
			config, err = store.Get(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			_, ok := config.EnvStates["gray"]
			Expect(ok).To(BeFalse())
		})

		It("is a no-op when removing env state without a config document", func() {
			Expect(store.RemoveEnvState(ctx, testAppID, "gray")).To(Succeed())
		})
	})
})
