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
	"strings"

	"github.com/hashicorp/go-set/v3"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// buildResourceRequirements constructs Kubernetes ResourceRequirements from a map of resource specifications.
// It support a simple syntax to define both requests and limits in a single string using a hyphen as separator.
// For example:
//
//	"cpu": "100m-200m"  // requests: 100m, limits: 200m
//	"memory": "256Mi"    // requests: 256Mi, limits: 256Mi
func buildResourceRequirements(resources map[string]string) (*corev1.ResourceRequirements, error) {
	if len(resources) == 0 {
		return nil, nil
	}

	validNames := set.From([]string{
		string(corev1.ResourceCPU),
		string(corev1.ResourceMemory),
		string(corev1.ResourceStorage),
		string(corev1.ResourceEphemeralStorage),
	})
	requests := make(corev1.ResourceList, len(resources))
	limits := make(corev1.ResourceList, len(resources))

	for name, rawValue := range resources {
		if !validNames.Contains(name) {
			return nil, errors.Errorf("invalid resource name %q", name)
		}
		parts := strings.Split(strings.TrimSpace(rawValue), "-")
		if len(parts) == 0 || len(parts) > 2 {
			return nil, errors.Errorf(
				"invalid resource %q value %q: expected {requests} or {requests}-{limits}",
				name,
				rawValue,
			)
		}

		// Get the request and limit strings
		reqStr := parts[0]
		limStr := reqStr
		if len(parts) == 2 {
			limStr = parts[1]
		}

		// Parse the quantities
		reqQty, err := resource.ParseQuantity(reqStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing resource %q requests %q", name, reqStr)
		}
		limQty, err := resource.ParseQuantity(limStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing resource %q limits %q", name, limStr)
		}

		if reqQty.Cmp(limQty) > 0 {
			return nil, errors.Errorf("resource %q requests %q must be <= limits %q", name, reqStr, limStr)
		}

		resName := corev1.ResourceName(name)
		requests[resName] = reqQty
		limits[resName] = limQty
	}

	return &corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}, nil
}
