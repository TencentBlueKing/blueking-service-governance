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

package serializer

import (
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// AppURIInput is the path input for app-scoped binding APIs.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 依赖服务名，目前仅支持 redis
	ServiceName string `uri:"serviceName" binding:"required,uri_slug"`
}

// AppBindingNameURIInput is the path input for a named binding.
type AppBindingNameURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 依赖服务名，目前仅支持 redis
	ServiceName string `uri:"serviceName" binding:"required,uri_slug"`
	// 绑定名称
	Name string `uri:"name" binding:"required,uri_slug"`
}

// CreateBindingInput is the JSON body for creating a service binding.
type CreateBindingInput struct {
	// 绑定名称（应用内同服务下唯一）
	Name string `json:"name" binding:"required,uri_slug"`
	// 描述
	Description string `json:"description"`
	// 环境名 → 实例 ID；允许为空
	EnvInstanceMap map[string]string `json:"envInstanceMap"`
	// 环境变量模板；允许为空
	EnvVars map[string]string `json:"envVars"`
}

// UpdateBindingInput is the JSON body for replacing a service binding.
type UpdateBindingInput struct {
	// 描述
	Description string `json:"description"`
	// 环境名 → 实例 ID；省略或空表示清空映射
	EnvInstanceMap map[string]string `json:"envInstanceMap"`
	// 环境变量模板；省略或空表示清空
	EnvVars map[string]string `json:"envVars"`
}

// BindingOutput is a single binding response wrapper.
type BindingOutput struct {
	Data *BindingOutputObj `json:"data"`
}

// ListBindingsOutput is the JSON response for listing bindings.
type ListBindingsOutput struct {
	Data []*BindingOutputObj `json:"data"`
}

// BindingOutputObj is the JSON representation of a service binding.
type BindingOutputObj struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	AppID          string            `json:"appID"`
	WorkspaceID    string            `json:"workspaceID"`
	ServiceName    string            `json:"serviceName"`
	EnvInstanceMap map[string]string `json:"envInstanceMap"`
	EnvVars        map[string]string `json:"envVars"`
	Description    string            `json:"description"`
	CreatedAt      string            `json:"createdAt"`
	UpdatedAt      string            `json:"updatedAt"`
}

// FromModel converts a ServiceBinding to BindingOutputObj.
func (o *BindingOutputObj) FromModel(binding *model.ServiceBinding) *BindingOutputObj {
	if o == nil {
		o = &BindingOutputObj{}
	}
	envMap := make(map[string]string, len(binding.EnvInstanceMap))
	for envName, instID := range binding.EnvInstanceMap {
		envMap[envName] = instID.Hex()
	}
	envVars := binding.EnvVars
	if envVars == nil {
		envVars = map[string]string{}
	}

	o.ID = binding.ID.Hex()
	o.Name = binding.Name
	o.AppID = binding.AppID
	o.WorkspaceID = binding.WorkspaceID
	o.ServiceName = binding.ServiceName
	o.EnvInstanceMap = envMap
	o.EnvVars = envVars
	o.Description = binding.Description
	o.CreatedAt = binding.CreatedAt.UTC().Format(time.RFC3339)
	o.UpdatedAt = binding.UpdatedAt.UTC().Format(time.RFC3339)
	return o
}

// ParseEnvInstanceMap converts envName → hex instance ID into ObjectIDs.
func ParseEnvInstanceMap(raw map[string]string) (map[string]bson.ObjectID, error) {
	if len(raw) == 0 {
		return map[string]bson.ObjectID{}, nil
	}
	result := make(map[string]bson.ObjectID, len(raw))
	for envName, instID := range raw {
		objID, err := bson.ObjectIDFromHex(instID)
		if err != nil {
			return nil, errors.Errorf("invalid instance id %q for env %q", instID, envName)
		}
		result[envName] = objID
	}
	return result, nil
}

// ValidateEnvVars 校验绑定 EnvVars 的 key 命名。value 是渲染模板，写入时无法预知最终结果，此处不校验。
func ValidateEnvVars(envVars map[string]string) error {
	for key := range envVars {
		if err := envvartypes.ValidateEnvVarKey(key); err != nil {
			return err
		}
	}
	return nil
}
