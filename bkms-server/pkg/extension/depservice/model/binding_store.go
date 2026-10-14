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

package model

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrBindingNameExists 应用内同服务下绑定名已存在
	ErrBindingNameExists = errors.New("service binding name already exists")
	// ErrBindingNotFound 服务绑定不存在，AsNotFoundError 同样将其视为未找到
	ErrBindingNotFound = errors.New("service binding not found")
)

// ServiceBindingStore 服务绑定存储
type ServiceBindingStore = store.GenericStore[ServiceBinding]

// ServiceBindingMeta 服务绑定的存储元数据，写入完整文档前会校验字段并由 EnvInstanceMap 重建 InstanceIDs。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + serviceName + name
// - instanceIDs：按实例反查引用
var ServiceBindingMeta = store.Meta[ServiceBinding]{
	Collection: serviceBindingCollName,
	NotFound:   ErrBindingNotFound,
	Duplicate:  ErrBindingNameExists,
	BeforeSave: []func(*ServiceBinding) error{validateBinding},
}

func validateBinding(binding *ServiceBinding) error {
	if err := validate.Struct(binding); err != nil {
		return errors.Wrap(err, "service binding validation failed")
	}
	binding.SyncInstanceIDs()
	return nil
}

// ByBindingName 按应用 + 服务 + 绑定名匹配服务绑定（至多一条）
func ByBindingName(appID, serviceName, name string) store.Opt {
	return store.Filter(bson.M{"appID": appID, "serviceName": serviceName, "name": name})
}

// ByBindingQuery 按查询条件匹配服务绑定，零值字段不参与过滤；opts 为 nil 时匹配全部绑定
func ByBindingQuery(opts *BindingQueryOptions) store.Opt {
	if opts == nil {
		return nil
	}
	filter := bson.M{}
	if opts.AppID != "" {
		filter["appID"] = opts.AppID
	}
	if opts.WorkspaceID != "" {
		filter["workspaceID"] = opts.WorkspaceID
	}
	if opts.ServiceName != "" {
		filter["serviceName"] = opts.ServiceName
	}
	if !opts.InstanceID.IsZero() {
		filter["instanceIDs"] = opts.InstanceID
	}
	return store.Filter(filter)
}

// UpdateBinding 全量更新绑定的环境映射、环境变量与描述，并同步重建 InstanceIDs；绑定不存在时返回 ErrBindingNotFound
func UpdateBinding(
	ctx context.Context,
	bindings *ServiceBindingStore,
	appID, serviceName, name string,
	data *ServiceBindingUpdateData,
) error {
	if data == nil {
		return nil
	}
	updated := &ServiceBinding{
		EnvInstanceMap: data.EnvInstanceMap,
		EnvVars:        data.EnvVars,
		Description:    data.Description,
	}
	updated.SyncInstanceIDs()
	err := bindings.Update(
		ctx, updated,
		store.UpdateSelected("envInstanceMap", "envVars", "instanceIDs", "description"),
		ByBindingName(appID, serviceName, name),
	)
	if err != nil {
		return errors.Wrapf(err, "update service binding(app=%s, service=%s, name=%s)", appID, serviceName, name)
	}
	return nil
}
