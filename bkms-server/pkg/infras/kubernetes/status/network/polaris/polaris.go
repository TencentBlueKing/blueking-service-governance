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

// Package polaris parses the synchronization status of Polaris configuration resources.
package polaris

import (
	"github.com/TencentBlueKing/gopkg/mapx"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// Parse 获取状态
func Parse(manifest map[string]any) *k8sstatus.Result {
	// 目前北极星配置仅关注同步阶段的状态（下发一般不会报错）
	// -> 返回 status.syncStatus.state 字段，如果不存在则返回 unknown
	status := mapx.GetStr(manifest, "status.syncStatus.state")
	if status == "" {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}
	return &k8sstatus.Result{Code: status}
}
