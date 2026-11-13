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

// Package rainbowtconf 保存应用的七彩石入口（页面地址），受空间配置开关 rainbowTConf 控制。
package rainbowtconf

import (
	"time"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrEntryNotFound 应用未配置七彩石入口
	ErrEntryNotFound = errors.New("rainbow tconf entry not found")
	// ErrEntryAlreadyExists 应用已有七彩石入口
	ErrEntryAlreadyExists = errors.New("rainbow tconf entry already exists")
)

// Entry 应用的七彩石入口（页面地址），一个应用至多一条，没有记录表示未配置
type Entry struct {
	AppID     string    `bson:"appID"`
	TenantID  string    `json:"-" bson:"tenantID"`
	URL       string    `bson:"url"`
	Updater   string    `bson:"updater"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// EntryStore 应用七彩石入口的存储
type EntryStore = store.GenericStore[Entry]

// EntryMeta 应用七彩石入口的存储元数据，唯一索引：appID（由 golang-migrate 维护）
var EntryMeta = store.Meta[Entry]{
	Collection:   "rainbow_tconf_entries",
	TenantScoped: true,
	NotFound:     ErrEntryNotFound,
	Duplicate:    ErrEntryAlreadyExists,
}

// ByApp 按应用 ID 匹配
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}
