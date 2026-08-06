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

package worker

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusOK  = "ok"
	statusErr = "err"
)

var (
	taskFinishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "worker",
			Name:      "task_finish_total",
			Help:      "Total number of async task executions.",
		},
		[]string{"task_name", "status"},
	)

	taskReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "worker",
			Name:      "task_received_total",
			Help:      "Total number of async task received.",
		},
		[]string{"task_name"},
	)

	taskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "bkms",
			Subsystem: "worker",
			Name:      "task_duration_seconds",
			Help:      "Async task execution duration in seconds.",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"task_name"},
	)

	taskEnqueueTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "bkms",
			Subsystem: "worker",
			Name:      "task_enqueue_total",
			Help:      "Total number of async task enqueue operations.",
		},
		[]string{"task_name", "status"},
	)
)

func reportTaskExecution(name taskName, status string, started time.Time) {
	n := string(name)
	taskFinishTotal.WithLabelValues(n, status).Inc()
	taskDuration.WithLabelValues(n).Observe(time.Since(started).Seconds())
}

func reportTaskReceived(name taskName) {
	taskReceivedTotal.WithLabelValues(string(name)).Inc()
}

func reportTaskEnqueue(name taskName, status string) {
	taskEnqueueTotal.WithLabelValues(string(name), status).Inc()
}
