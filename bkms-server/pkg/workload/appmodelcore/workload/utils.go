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

package workload

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AppendAsUnstructured appends any kubernetes resources to the given unstructured objects slice.
func AppendAsUnstructured(
	items []unstructured.Unstructured,
	inputObjs ...client.Object,
) ([]unstructured.Unstructured, error) {
	for _, input := range inputObjs {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(input)
		if err != nil {
			return nil, errors.Wrapf(err, "converting %s to unstructured", input.GetName())
		}
		items = append(items, unstructured.Unstructured{Object: obj})
	}
	return items, nil
}
