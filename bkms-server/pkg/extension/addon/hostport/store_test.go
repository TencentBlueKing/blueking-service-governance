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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("HostPortStore", func() {
	var (
		ctx       context.Context
		db        *mongo.Database
		configs   *hostport.HostPortStore
		testAppID string
	)

	mustGet := func() *hostport.HostPortConfig {
		config, err := configs.Get(ctx, hostport.ByApp(testAppID))
		Expect(err).NotTo(HaveOccurred())
		return config
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		configs = store.Of(db, hostport.HostPortConfigMeta)
		testAppID = "hostport-app-" + stringx.Random(6)
		DeferCleanup(func() { _ = configs.DeleteIfExists(ctx, hostport.ByApp(testAppID)) })
	})

	Describe("Get", func() {
		It("returns ErrConfigNotFound when config document does not exist", func() {
			_, err := configs.Get(ctx, hostport.ByApp(testAppID))
			Expect(err).To(MatchError(hostport.ErrConfigNotFound))
		})
	})

	Describe("ListPorts", func() {
		It("returns empty slice when config document does not exist", func() {
			ports, err := hostport.ListPorts(ctx, db, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{}))
		})
	})

	Describe("ReplacePorts", func() {
		It("lazily creates config and stores sorted unique ports", func() {
			ports, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{8080, 80, 80})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{80, 8080}))

			config := mustGet()
			Expect(config.AppID).To(Equal(testAppID))
			Expect(config.Ports).To(Equal([]int32{80, 8080}))
			Expect(config.EnvStates).To(BeEmpty())
		})

		It("replaces the full ports list", func() {
			_, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{80, 8080})
			Expect(err).NotTo(HaveOccurred())

			ports, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{443})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{443}))
			Expect(mustGet().Ports).To(Equal([]int32{443}))
		})

		It("clears ports with an empty list and keeps envStates", func() {
			_, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{80})
			Expect(err).NotTo(HaveOccurred())
			Expect(hostport.UpsertEnvState(ctx, db, testAppID, "gray", []int32{80})).To(Succeed())

			ports, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ports).To(Equal([]int32{}))
			config := mustGet()
			Expect(config.Ports).To(Equal([]int32{}))
			Expect(config.EnvStates["gray"].AppliedPorts).To(Equal([]int32{80}))
		})
	})

	Describe("UpsertEnvState / RemoveEnvState", func() {
		It("lazily creates config with empty ports when recording env state", func() {
			Expect(hostport.UpsertEnvState(ctx, db, testAppID, "gray", []int32{8080, 80})).To(Succeed())

			config := mustGet()
			Expect(config.AppID).To(Equal(testAppID))
			Expect(config.Ports).To(Equal([]int32{}))
			Expect(config.EnvStates).To(HaveKey("gray"))
			Expect(config.EnvStates["gray"].AppliedPorts).To(Equal([]int32{80, 8080}))
			Expect(config.EnvStates["gray"].UpdatedAt).NotTo(BeZero())
		})

		It("only touches the target env and can clear it", func() {
			Expect(hostport.UpsertEnvState(ctx, db, testAppID, "gray", []int32{80})).To(Succeed())
			Expect(hostport.UpsertEnvState(ctx, db, testAppID, "prod", []int32{443})).To(Succeed())

			Expect(hostport.RemoveEnvState(ctx, db, testAppID, "gray")).To(Succeed())
			config := mustGet()
			Expect(config.EnvStates).NotTo(HaveKey("gray"))
			Expect(config.EnvStates["prod"].AppliedPorts).To(Equal([]int32{443}))
		})

		It("is a no-op when removing env state without a config document", func() {
			Expect(hostport.RemoveEnvState(ctx, db, testAppID, "gray")).To(Succeed())
			_, err := configs.Get(ctx, hostport.ByApp(testAppID))
			Expect(err).To(MatchError(hostport.ErrConfigNotFound))
		})

		It("rejects env names that are not valid field names", func() {
			for _, name := range []string{"", "a.b", "$x"} {
				Expect(hostport.UpsertEnvState(ctx, db, testAppID, name, []int32{80})).
					To(MatchError(ContainSubstring("invalid env name")))
				Expect(hostport.RemoveEnvState(ctx, db, testAppID, name)).
					To(MatchError(ContainSubstring("invalid env name")))
			}
		})
	})
})
