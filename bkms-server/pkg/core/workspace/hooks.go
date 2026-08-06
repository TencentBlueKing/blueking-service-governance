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

package workspace

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// NewComponentRefCountHooks 构建 WorkspaceCompsStore 的组件引用计数 Hook。
// 在组件增删后自动维护 ComponentDef.WorkspaceCompInstanceCount 字段。
func NewComponentRefCountHooks(compDefStore component.ComponentDefStore) *ComponentHooks {
	return &ComponentHooks{
		AfterAdd: func(ctx context.Context, comps []*Component) error {
			typeCounts := map[string]int{}
			for _, comp := range comps {
				if comp.Type != "" {
					typeCounts[comp.Type]++
				}
			}
			for compType, count := range typeCounts {
				if err := compDefStore.UpdateInstanceCount(
					ctx,
					compType,
					component.FieldWorkspaceCompInstance,
					count,
				); err != nil {
					return errors.Wrapf(err, "increment workspaceCompInstanceCount for %s", compType)
				}
			}
			return nil
		},
		AfterRemove: func(ctx context.Context, comps []*Component) error {
			typeCounts := map[string]int{}
			for _, comp := range comps {
				if comp.Type != "" {
					typeCounts[comp.Type]++
				}
			}
			for compType, count := range typeCounts {
				if err := compDefStore.UpdateInstanceCount(
					ctx, compType, component.FieldWorkspaceCompInstance, -count,
				); err != nil {
					return errors.Wrapf(err, "decrement workspaceCompInstanceCount for %s", compType)
				}
			}
			return nil
		},
	}
}
