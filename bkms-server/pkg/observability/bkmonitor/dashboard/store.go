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

package dashboard

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrNotFound 应用仪表盘绑定未找到
	ErrNotFound = errors.New("AppDashboard not found")

	// ErrDuplicate 应用已绑定相同 uid 的仪表盘
	ErrDuplicate = errors.New("AppDashboard duplicate uid error")

	// ErrDashboardNotExist 提交的仪表盘 uid 在 bkmonitor 侧不存在
	ErrDashboardNotExist = errors.New("dashboard uid does not exist in bkmonitor")
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// Store 应用仪表盘绑定存储
type Store = store.GenericStore[AppDashboard]

// Meta 应用仪表盘绑定的存储元数据，写入完整文档前会校验必填字段。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + uid
var Meta = store.Meta[AppDashboard]{
	Collection: "bkmonitor_app_dashboard",
	NotFound:   ErrNotFound,
	Duplicate:  ErrDuplicate,
	BeforeSave: []func(*AppDashboard) error{validateDashboard},
}

func validateDashboard(d *AppDashboard) error {
	return errors.Wrap(validate.Struct(d), "AppDashboard validation failed")
}

// ByApp 按应用 ID 匹配仪表盘绑定
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppAndUID 按应用 ID 与仪表盘 uid 匹配仪表盘绑定（至多一条）
func ByAppAndUID(appID, uid string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("uid", uid))
}
