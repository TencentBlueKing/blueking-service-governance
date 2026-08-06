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

package topology

import (
	"github.com/TencentBlueKing/gopkg/mapx"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ExtractConditions 从资源的 status.conditions 列表提取 Condition 切片
// 返回包含 type、status、reason、message、lastTransitionTime 的完整条件列表
func ExtractConditions(obj *unstructured.Unstructured) []Condition {
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found || len(conditions) == 0 {
		return nil
	}

	result := make([]Condition, 0, len(conditions))
	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}

		result = append(result, Condition{
			Type:               mapx.GetStr(cond, "type"),
			Status:             mapx.GetStr(cond, "status"),
			Reason:             mapx.GetStr(cond, "reason"),
			Message:            mapx.GetStr(cond, "message"),
			LastTransitionTime: mapx.GetStr(cond, "lastTransitionTime"),
		})
	}

	return result
}
