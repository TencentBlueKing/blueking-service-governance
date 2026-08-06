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

package task

import "time"

// TotalFailureRetryCount 任务失败重试次数
const TotalFailureRetryCount = 10

// saveStatusTimeout 任务最终状态保存超时时间
const saveStatusTimeout = 10 * time.Second

// buildPollingTimeout 构建轮询总超时时间（24 小时）
// 部分应用构建耗时较长，因此采用非固定（梯度增长）的轮询间隔 + 长超时策略
const buildPollingTimeout = 24 * time.Hour

// HelmChart 构建相对轻量 & 快速，暂时不开放动态配置
const (
	// helmChartBuildPollingInterval HelmChart 构建轮询固定间隔（单位：秒）
	helmChartBuildPollingInterval = 5
	// helmChartBuildPollingTimeout HelmChart 构建轮询总超时时间（单位：秒）
	helmChartBuildPollingTimeout = 600
)

// pollingTier 定义单个轮询梯度：当已运行时长 >= Threshold 时，使用 Interval 作为轮询间隔
type pollingTier struct {
	Threshold time.Duration
	Interval  time.Duration
}

// buildPollingTiers 构建轮询梯度表（按阈值降序排列）
// 匹配规则：从上往下找到第一个 elapsed >= Threshold 的梯度
var buildPollingTiers = []pollingTier{
	{Threshold: 3 * time.Hour, Interval: 300 * time.Second},   // 3 ~ 24 小时：5 分钟
	{Threshold: 1 * time.Hour, Interval: 60 * time.Second},    // 1 ~ 3 小时：60 秒
	{Threshold: 30 * time.Minute, Interval: 30 * time.Second}, // 30 ~ 60 分钟：30 秒
	{Threshold: 15 * time.Minute, Interval: 15 * time.Second}, // 15 ~ 30 分钟：15 秒
	{Threshold: 0, Interval: 10 * time.Second},                // 0 ~ 15 分钟：10 秒
}

func calcBuildPollingInterval(elapsed time.Duration) time.Duration {
	for _, tier := range buildPollingTiers {
		if elapsed >= tier.Threshold {
			return tier.Interval
		}
	}
	// 理论上不会走到这里，因为最后一个 Threshold 是 0
	return buildPollingTiers[len(buildPollingTiers)-1].Interval
}
