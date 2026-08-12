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

package event

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildSearchConditions", func() {
	It("adds strategy and environment conditions together", func() {
		conditions := buildSearchConditions(SearchInput{
			StrategyName: "memory_limit_usage_high",
			ClusterID:    "BCS-K8S-100018",
			Namespace:    "ieg-bkms-pd-teamdev",
		}, []int64{11766, 11767})

		Expect(conditions).To(ContainElement(map[string]any{
			"key":   "strategy_id",
			"value": []int64{11766, 11767},
		}))
		Expect(conditions).To(ContainElement(map[string]any{
			"key":   "strategy_name",
			"value": []string{"memory_limit_usage_high"},
		}))
		Expect(conditions).To(ContainElement(map[string]any{
			"key":   "tags.bcs_cluster_id",
			"value": []string{"BCS-K8S-100018"},
		}))
		Expect(conditions).To(ContainElement(map[string]any{
			"key":   "tags.namespace",
			"value": []string{"ieg-bkms-pd-teamdev"},
		}))
	})
})
