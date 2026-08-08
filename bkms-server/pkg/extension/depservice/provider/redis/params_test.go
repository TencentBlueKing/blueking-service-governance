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

package redis_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/redis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
)

var _ = Describe("CreateParams", func() {
	baseCluster := func() *redis.CreateParams {
		return &redis.CreateParams{
			BkBizID:         1,
			DBAppAbbr:       "demo",
			ClusterType:     dbm.ClusterTypeTwemproxyRedis,
			ClusterName:     "c1",
			DBVersion:       "Redis-6",
			ProxyPort:       50000,
			ClusterShardNum: 3,
		}
	}

	baseIns := func() *redis.CreateParams {
		return &redis.CreateParams{
			BkBizID:     1,
			DBAppAbbr:   "demo",
			ClusterType: dbm.ClusterTypeRedisInstance,
			ClusterName: "ms1",
			DBVersion:   "Redis-6",
			Port:        6379,
			Databases:   2,
		}
	}

	Describe("Validate", func() {
		It("accepts a valid cluster deploy params", func() {
			Expect(baseCluster().Validate()).To(Succeed())
		})

		It("accepts a valid RedisInstance deploy params", func() {
			Expect(baseIns().Validate()).To(Succeed())
		})

		It("rejects RedisInstance without port/databases", func() {
			p := baseIns()
			p.Port = 0
			Expect(p.Validate()).To(MatchError(ContainSubstring("port")))

			p = baseIns()
			p.Databases = 0
			Expect(p.Validate()).To(MatchError(ContainSubstring("databases")))
		})

		It("rejects cluster deploy without proxyPort/clusterShardNum", func() {
			p := baseCluster()
			p.ProxyPort = 0
			Expect(p.Validate()).To(MatchError(ContainSubstring("proxyPort")))

			p = baseCluster()
			p.ClusterShardNum = 0
			Expect(p.Validate()).To(MatchError(ContainSubstring("clusterShardNum")))
		})

		It("rejects unsupported clusterType", func() {
			p := baseCluster()
			p.ClusterType = "UnknownType"
			Expect(p.Validate()).To(MatchError(ContainSubstring("unsupported clusterType")))
		})
	})

	Describe("ToCreateRedisParams", func() {
		It("maps cluster fields for REDIS_CLUSTER_APPLY", func() {
			params := baseCluster().ToCreateRedisParams()
			Expect(params.TicketType).To(Equal(dbm.TicketTypeRedisClusterApply))
			Expect(params.ClusterName).To(Equal("c1"))
			Expect(params.ProxyPort).To(Equal(50000))
			Expect(params.Infos).To(BeEmpty())
		})

		It("maps ClusterName/Databases into Infos for REDIS_INS_APPLY", func() {
			p := baseIns()
			p.RedisPwd = "secret"
			params := p.ToCreateRedisParams()
			Expect(params.TicketType).To(Equal(dbm.TicketTypeRedisInsApply))
			Expect(params.ClusterName).To(BeEmpty())
			Expect(params.Port).To(Equal(6379))
			Expect(params.RedisPwd).To(Equal("secret"))
			Expect(params.AppendApply).To(BeFalse())
			Expect(params.Infos).To(Equal([]dbm.RedisInsInfo{{
				ClusterName: "ms1",
				Databases:   2,
			}}))
		})
	})
})
