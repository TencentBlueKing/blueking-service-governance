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

package bkmonitor_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var _ = Describe("InstanceRank", func() {
	var query *bkmonitor.InstanceRankQuery

	BeforeEach(func() {
		query = &bkmonitor.InstanceRankQuery{
			BkBizID:      100,
			ClusterID:    "BCS-K8S-00000",
			Namespace:    "demo-ns",
			WorkloadKind: "GameDeployment",
			WorkloadName: "demo",
			Limit:        10,
			StartTime:    1781366400,
			EndTime:      1781370000,
			Interval:     60,
			Username:     "tester",
		}
	})

	Describe("BuildInstanceRankPromQL", func() {
		It("builds topk on average cpu usage filtered by workload", func() {
			Expect(bkmonitor.BuildInstanceRankPromQL(query)).To(Equal(
				`topk(10, avg_over_time((sum by(pod)(rate(container_cpu_usage_seconds_total{` +
					`bcs_cluster_id="BCS-K8S-00000",namespace=~"^(demo-ns)$",` +
					`workload_kind="GameDeployment",workload_name="demo",container_name!="POD"}[1m])))` +
					`[3600s:60s]))`,
			))
		})

		It("uses bottomk for bottom ranking", func() {
			query.RankType = bkmonitor.RankTypeBottom
			Expect(bkmonitor.BuildInstanceRankPromQL(query)).To(HavePrefix("bottomk(10, avg_over_time(("))
		})

		It("applies the aggregation method", func() {
			query.Method = bkmonitor.AggMethodMax
			Expect(bkmonitor.BuildInstanceRankPromQL(query)).To(ContainSubstring("((max by(pod)(rate("))
		})

		It("uses the interval as subquery step", func() {
			query.Interval = 300
			Expect(bkmonitor.BuildInstanceRankPromQL(query)).To(HaveSuffix("[3600s:300s]))"))
		})

		It("raises step to 60s and range to at least one step", func() {
			query.Interval = 0
			query.StartTime = query.EndTime
			Expect(bkmonitor.BuildInstanceRankPromQL(query)).To(HaveSuffix("[60s:60s]))"))
		})
	})

	Describe("RankInstances", func() {
		var (
			captured *bkmapi.TimeSeriesUnifyQueryReq
			resp     *bkmapi.TimeSeriesUnifyQueryResp
			respErr  error
			mockers  []*mockey.Mocker
		)

		series := func(dims map[string]string, value float64) bkmapi.TimeSeriesData {
			return bkmapi.TimeSeriesData{Dimensions: dims, Datapoints: [][2]float64{{value, 1781370000000}}}
		}

		BeforeEach(func() {
			captured, resp, respErr = nil, &bkmapi.TimeSeriesUnifyQueryResp{}, nil
			mockers = []*mockey.Mocker{
				mockey.Mock(bkmapi.NewMonitorClient).To(func(operator string) (bkmapi.MonitorClient, error) {
					return bkmapi.NewStub(operator), nil
				}).Build(),
				mockey.Mock((*bkmapi.StubClient).TimeSeriesUnifyQuery).To(
					func(_ *bkmapi.StubClient, _ context.Context, req *bkmapi.TimeSeriesUnifyQueryReq,
					) (*bkmapi.TimeSeriesUnifyQueryResp, error) {
						captured = req
						return resp, respErr
					},
				).Build(),
			}
		})

		AfterEach(func() {
			for _, m := range mockers {
				m.UnPatch()
			}
		})

		names := func(ranks []*bkmonitor.InstanceRank) []string {
			return lo.Map(ranks, func(r *bkmonitor.InstanceRank, _ int) string { return r.Instance })
		}

		It("sends a single instant query with the rank promql", func() {
			_, err := bkmonitor.NewMetricTimeSeriesService().RankInstances(context.Background(), query)
			Expect(err).NotTo(HaveOccurred())
			Expect(captured).NotTo(BeNil())
			Expect(captured.Type).To(Equal("instant"))
			Expect(captured.BkBizID).To(Equal(int64(100)))
			Expect(captured.StartTime).To(Equal(query.StartTime))
			Expect(captured.EndTime).To(Equal(query.EndTime))
			Expect(captured.QueryConfigs).To(HaveLen(1))
			Expect(captured.QueryConfigs[0].PromQL).To(Equal(bkmonitor.BuildInstanceRankPromQL(query)))
		})

		It("sorts by value descending for top and reads pod from any supported dimension", func() {
			resp.Series = []bkmapi.TimeSeriesData{
				series(map[string]string{"pod": "low"}, 0.1),
				series(map[string]string{"pod_name": "high"}, 0.9),
				series(map[string]string{"instance": "mid"}, 0.5),
			}
			ranks, err := bkmonitor.NewMetricTimeSeriesService().RankInstances(context.Background(), query)
			Expect(err).NotTo(HaveOccurred())
			Expect(names(ranks)).To(Equal([]string{"high", "mid", "low"}))
			Expect(ranks[0].Value).To(Equal(0.9))
		})

		It("sorts by value ascending for bottom and truncates to limit", func() {
			query.RankType = bkmonitor.RankTypeBottom
			query.Limit = 2
			resp.Series = []bkmapi.TimeSeriesData{
				series(map[string]string{"pod": "high"}, 0.9),
				series(map[string]string{"pod": "low"}, 0.1),
				series(map[string]string{"pod": "mid"}, 0.5),
			}
			ranks, err := bkmonitor.NewMetricTimeSeriesService().RankInstances(context.Background(), query)
			Expect(err).NotTo(HaveOccurred())
			Expect(names(ranks)).To(Equal([]string{"low", "mid"}))
		})

		It("skips series without pod dimension or datapoints", func() {
			resp.Series = []bkmapi.TimeSeriesData{
				series(map[string]string{"namespace": "demo-ns"}, 0.9),
				{Dimensions: map[string]string{"pod": "no-points"}},
				series(map[string]string{"pod": "ok"}, 0.3),
			}
			ranks, err := bkmonitor.NewMetricTimeSeriesService().RankInstances(context.Background(), query)
			Expect(err).NotTo(HaveOccurred())
			Expect(names(ranks)).To(Equal([]string{"ok"}))
		})

		It("returns an error when the query fails", func() {
			respErr = errors.New("bkmonitor unavailable")
			_, err := bkmonitor.NewMetricTimeSeriesService().RankInstances(context.Background(), query)
			Expect(err).To(MatchError(ContainSubstring("bkmonitor unavailable")))
			Expect(err).To(MatchError(ContainSubstring("GameDeployment/demo")))
		})
	})
})
