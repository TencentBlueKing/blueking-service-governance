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
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrAppConfigFileDefNotFound 配置文件 def 不存在
	ErrAppConfigFileDefNotFound = errors.New("app config file def not found")
	// ErrAppConfigFileDefAlreadyExists 应用下已存在同名（或同挂载目录）的配置文件 def
	ErrAppConfigFileDefAlreadyExists = errors.New("app config file def already exists")
)

// AppConfigFileDefStore 配置文件 def 存储
type AppConfigFileDefStore = store.GenericStore[AppConfigFileDef]

// AppConfigFileDefMeta 配置文件 def 的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + name
// - 唯一（partial，mountDir 非空）：appID + mountDir + name，保证最终挂载路径 Join(mountDir, name) 不重复
// - 查询：appID + configKind
var AppConfigFileDefMeta = store.Meta[AppConfigFileDef]{
	Collection: "app_config_file_defs",
	NotFound:   ErrAppConfigFileDefNotFound,
	Duplicate:  ErrAppConfigFileDefAlreadyExists,
}

// DefsByApp 按应用 ID 匹配 def 记录
func DefsByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// DefFilterConfigKind 按 ConfigKind 过滤 def 记录
func DefFilterConfigKind(kind ConfigKind) store.Opt {
	return store.Where("configKind", kind)
}

// ListDefsByApp 返回指定应用下的全部 def 记录（按名称排序），可通过 DefFilterConfigKind 等条件进一步过滤
func ListDefsByApp(
	ctx context.Context,
	defs *AppConfigFileDefStore,
	appID string,
	opts ...store.Opt,
) ([]AppConfigFileDef, error) {
	list, err := defs.List(ctx, append([]store.Opt{DefsByApp(appID), store.Asc("name")}, opts...)...)
	if err != nil {
		return nil, errors.Wrapf(err, "list app config file defs of app %s", appID)
	}
	return list, nil
}

// UpdateDef 按 ID 更新 def 记录的可变字段，def 不存在时返回 ErrAppConfigFileDefNotFound。
// EnvConfigMode 以 inline 方式展开为 isUnifiedConfig / mountedEnvNames / envInstanceStrategy。
func UpdateDef(ctx context.Context, defs *AppConfigFileDefStore, def *AppConfigFileDef) error {
	if def.ID.IsZero() {
		return errors.New("def ID is required for update")
	}
	mutable := store.UpdateSelected(
		"name", "mountDir", "isUnifiedConfig", "mountedEnvNames", "enableEnvVarRender", "envInstanceStrategy",
	)
	if err := defs.Update(ctx, def, mutable, store.ByID(def.ID)); err != nil {
		return errors.Wrapf(err, "update app config file def %s", def.ID.Hex())
	}
	return nil
}
