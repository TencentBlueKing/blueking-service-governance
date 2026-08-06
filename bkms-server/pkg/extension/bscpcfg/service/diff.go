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

package service

import (
	"github.com/samber/lo"

	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

// DiffScopes 计算当前 Scope 与目标 Scope 的增量 diff。
// 返回 nil 表示无需更新（当前与目标完全一致）。
// 导出此函数以便于单元测试。
func DiffScopes(
	current []bscpapi.CredentialScope, target []bscpapi.CredentialScopeItem,
) *bscpapi.UpdateCredentialScopeReq {
	// 以 App 为 key 构建当前 Scope 的 map
	currentMap := lo.Associate(current, func(s bscpapi.CredentialScope) (string, bscpapi.CredentialScope) {
		return s.App, s
	})

	// 以 App 为 key 构建目标 Scope 的 set
	targetMap := lo.Associate(target, func(t bscpapi.CredentialScopeItem) (string, bscpapi.CredentialScopeItem) {
		return t.App, t
	})

	var delIDs []int64
	var alterScope []bscpapi.AlterScopeItem
	var addScope []bscpapi.CredentialScopeItem

	// 遍历目标：判断新增或更新
	for _, t := range target {
		if existing, ok := currentMap[t.App]; ok {
			// app 存在，检查 scope 是否一致
			if existing.Scope != t.Scope {
				alterScope = append(alterScope, bscpapi.AlterScopeItem{
					ID:    existing.ID,
					App:   t.App,
					Scope: t.Scope,
				})
			}
		} else {
			// app 不存在，需要新增
			addScope = append(addScope, t)
		}
	}

	// 遍历当前：判断删除（当前存在但目标中不存在）
	for _, c := range current {
		if _, ok := targetMap[c.App]; !ok {
			delIDs = append(delIDs, c.ID)
		}
	}

	// 若三个列表均为空，返回 nil 表示无需更新
	if len(addScope) == 0 && len(alterScope) == 0 && len(delIDs) == 0 {
		return nil
	}

	return &bscpapi.UpdateCredentialScopeReq{
		AddScope:   addScope,
		AlterScope: alterScope,
		DelID:      delIDs,
	}
}
