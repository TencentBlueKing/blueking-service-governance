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

// Package resourcevalue parses workload resource strings of the form
// "{requests}" or "{requests}-{limits}".
package resourcevalue

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ResourceBounds is one resource value split into request and limit.
// Request and Limit keep the original text. The Quantity fields are those
// strings parsed once, so callers do not parse them again.
type ResourceBounds struct {
	Request         string
	Limit           string
	RequestQuantity resource.Quantity
	LimitQuantity   resource.Quantity
}

// ParseResourceBounds splits a workload resource value and checks both sides.
// A value without "-" uses the same text for request and limit.
// Both sides must be Kubernetes quantities, and request must not exceed limit.
func ParseResourceBounds(raw string) (ResourceBounds, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, "-")
	if len(parts) == 0 || len(parts) > 2 {
		return ResourceBounds{}, errors.Errorf(
			"invalid resource value %q: expected {requests} or {requests}-{limits}",
			raw,
		)
	}

	request := parts[0]
	limit := request
	if len(parts) == 2 {
		limit = parts[1]
	}

	reqQty, err := resource.ParseQuantity(request)
	if err != nil {
		return ResourceBounds{}, errors.Wrapf(err, "parsing requests %q", request)
	}
	limQty, err := resource.ParseQuantity(limit)
	if err != nil {
		return ResourceBounds{}, errors.Wrapf(err, "parsing limits %q", limit)
	}
	if reqQty.Cmp(limQty) > 0 {
		return ResourceBounds{}, errors.Errorf("requests %q must be <= limits %q", request, limit)
	}
	return ResourceBounds{
		Request:         request,
		Limit:           limit,
		RequestQuantity: reqQty,
		LimitQuantity:   limQty,
	}, nil
}
