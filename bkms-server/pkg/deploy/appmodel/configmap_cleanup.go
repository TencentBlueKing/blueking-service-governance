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

package appmodel

import (
	"slices"

	"github.com/samber/lo"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// recentRecordsLimit 计算保留与清理的 ConfigMap 时回看的部署记录条数。
// 每次部署都会清理窗口内过期的 ConfigMap，窗口只需覆盖上一次清理之后新增的记录
const recentRecordsLimit = 10

// getExpiredConfigMaps 返回 records 引用过、current 下发之后不再可能被挂载的 ConfigMap。
//
// 从最新的记录往前，直到最近一条部署成功的记录（含）为止，这些记录对应版本的 Pod 可能仍在运行（分批发布中的
// 旧版本、未完成的滚动更新），部署成功的记录还是分批发布回滚的目标版本，它们引用的 ConfigMap 都需要保留，
// 否则这些 Pod 重建时无法挂载。配置内容未变化时不同记录引用同名 ConfigMap，被保留或被 current 引用的都不会返回
func getExpiredConfigMaps(records []Record, current ResourceKeys) ResourceKeys {
	records = sortRecord(records)
	inUse := slices.Clone(current)
	for _, record := range records {
		// 被卸载后，更早的 ConfigMap（包括被卸载的记录本身）都可以清理
		if record.Status == StatusUninstalled {
			break
		}
		inUse = append(inUse, listCmFromRecord(record)...)
		// 部署成功的记录本身需要保留，更早的 ConfigMap 都可以清理
		if record.Status == StatusDeployed {
			break
		}
	}
	return listRefConfigMaps(records).Diff(inUse)
}

// listRefConfigMaps 返回 records 引用过的全部 ConfigMap，按记录由新到旧去重, 确保返回顺序稳定
func listRefConfigMaps(records []Record) ResourceKeys {
	return lo.Uniq(lo.FlatMap(sortRecord(records), func(record Record, _ int) []ResourceKey {
		return listCmFromRecord(record)
	}))
}

// listCmFromRecord 返回 record 中包含的 ConfigMap
func listCmFromRecord(record Record) ResourceKeys {
	return lo.Filter(record.ResourceKeys, func(key ResourceKey, _ int) bool {
		return key.Kind == k8skind.CM
	})
}
