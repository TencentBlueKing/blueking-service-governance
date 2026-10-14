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

package helmcomponent

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrHelmAppComponentNotFound 组件引用不存在
var ErrHelmAppComponentNotFound = errors.New("helm app component not found")

// Store Helm 应用组件引用存储
type Store = store.GenericStore[HelmAppComponent]

// Meta Helm 应用组件引用的存储元数据，写入完整文档前会确保组件名称已生成。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + envName + name
// - 查询提速：appID + envName
var Meta = store.Meta[HelmAppComponent]{
	Collection: helmAppComponentCollectionName,
	NotFound:   ErrHelmAppComponentNotFound,
	BeforeSave: []func(*HelmAppComponent) error{ensureName},
}

func ensureName(comp *HelmAppComponent) error {
	comp.EnsureName()
	return nil
}

// ByApp 按应用 ID 匹配组件引用
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppAndEnv 按应用 ID 和环境名称匹配组件引用
func ByAppAndEnv(appID, envName string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("envName", envName))
}

// InExecutionOrder 按执行顺序排序：priority 升序，相同时按创建时间升序
func InExecutionOrder() store.Opt {
	return store.Combine(store.Asc("priority"), store.Asc("createdAt"))
}
