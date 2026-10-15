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

package appcfg

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// AppConfigFileVersionStore 配置文件版本记录存储
type AppConfigFileVersionStore = store.GenericStore[AppConfigFileVersion]

// AppConfigFileVersionMeta 配置文件版本记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appConfigFileID + version
// - 查询提速：appConfigFileID + isDeleted + createdAt(倒序)
// - 查询提速：appID + envName + isDeleted + createdAt(倒序)
// - 查询提速：appConfigFileID + isDeleted + creator + createdAt(倒序)
var AppConfigFileVersionMeta = store.Meta[AppConfigFileVersion]{
	Collection: "app_config_file_versions",
	NotFound:   ErrAppCfgFileVersionNotFound,
	Duplicate:  ErrAppCfgFileVersionAlreadyExists,
}

// VersionsByApp 按应用 ID 匹配版本记录
func VersionsByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// VersionsByFile 按配置文件 ID 匹配版本记录
func VersionsByFile(appConfigFileID bson.ObjectID) store.Opt {
	return store.Where("appConfigFileID", appConfigFileID)
}

// VersionNotDeleted 排除已软删除的版本记录
func VersionNotDeleted() store.Opt {
	return store.Where("isDeleted", bson.M{"$ne": true})
}

// VersionsNewestFirst 按创建时间倒序，_id 作为二级排序保证翻页稳定
func VersionsNewestFirst() store.Opt {
	return store.Combine(store.Desc("createdAt"), store.Desc("_id"))
}
