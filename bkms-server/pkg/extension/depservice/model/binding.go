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
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const serviceBindingCollName = "depservice_bindings"

// ServiceBinding 是应用侧对空间级依赖服务实例的一份使用关系。
// 同一应用可有多份绑定（靠 Name 区分）；一份绑定内每个环境最多映射一个实例。
type ServiceBinding struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	// Name 绑定名称，应用内同 serviceName 下唯一
	Name string `bson:"name" validate:"required"`
	// AppID 所属应用
	AppID string `bson:"appID" validate:"required"`
	// WorkspaceID 所属工作空间
	WorkspaceID string `bson:"workspaceID" validate:"required"`
	// ServiceName 依赖服务名，如 redis
	ServiceName string `bson:"serviceName" validate:"required"`

	// EnvInstanceMap 环境名 → 实例 ID。允许为空（先建绑定再选实例）。
	EnvInstanceMap map[string]bson.ObjectID `bson:"envInstanceMap"`
	// EnvVars 渲染环境变量模板,
	// 模板引用「当前环境命中的那台实例」的 Credentials。
	EnvVars map[string]string `bson:"envVars"`

	// InstanceIDs 由 EnvInstanceMap 派生，仅用于按实例反查，不对外暴露。
	InstanceIDs []bson.ObjectID `bson:"instanceIDs"`

	Description string `bson:"description"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// ServiceBindingUpdateData 全量更新绑定的可写字段。
type ServiceBindingUpdateData struct {
	EnvInstanceMap map[string]bson.ObjectID
	EnvVars        map[string]string
	Description    string
}

// BindingQueryOptions 绑定列表查询条件
type BindingQueryOptions struct {
	AppID       string
	WorkspaceID string
	ServiceName string
	InstanceID  bson.ObjectID
}

// SyncInstanceIDs 根据 EnvInstanceMap 重建 InstanceIDs。
func (b *ServiceBinding) SyncInstanceIDs() {
	if b.EnvInstanceMap == nil {
		b.EnvInstanceMap = map[string]bson.ObjectID{}
	}
	if b.EnvVars == nil {
		b.EnvVars = map[string]string{}
	}
	b.InstanceIDs = lo.Uniq(lo.Values(b.EnvInstanceMap))
}
