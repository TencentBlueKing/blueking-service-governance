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

package bkci

import "context"

// RepositoryInitSpec carries the repository URL/alias pair required to
// initialize a BKCI repository binding.
type RepositoryInitSpec struct {
	URL   string
	Alias string
}

// EnsureWorkspaceRepositories initializes each distinct repository binding for
// a workspace at most once per request.
func EnsureWorkspaceRepositories(ctx context.Context, workspaceID string, specs []RepositoryInitSpec) error {
	initializer := NewRepositoryManager(workspaceID)
	seen := make(map[RepositoryInitSpec]struct{}, len(specs))
	for _, spec := range specs {
		if spec.URL == "" || spec.Alias == "" {
			continue
		}
		if _, ok := seen[spec]; ok {
			continue
		}
		if _, err := initializer.Initialize(ctx, spec.URL, spec.Alias); err != nil {
			return err
		}
		seen[spec] = struct{}{}
	}
	return nil
}
