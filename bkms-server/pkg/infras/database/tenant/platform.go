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

package tenant

// platformTables 平台表：不注入 tenant_id。集合名与对应 store 中的常量保持一致。
var platformTables = map[string]struct{}{
	"cluster_addon_defs":  {}, // pkg/core/env/clusteraddon
	"depservice_services": {}, // pkg/extension/depservice/model
}

// isPlatform 判断集合是否为平台表。
func isPlatform(name string) bool {
	_, ok := platformTables[name]
	return ok
}
