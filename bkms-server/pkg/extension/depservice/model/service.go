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
	"encoding/json"
	"fmt"

	"github.com/spf13/cast"
)

const (
	// ProviderTypeUserDefined represents a user-defined service instance
	// 用户自行定义和管理的服务实例（如用户自注册 mysql 凭证等）
	ProviderTypeUserDefined = "user-defined"
	// ProviderTypeSystemAllocated represents a system-allocated service instance
	// 系统通过调用外部服务自动分配的服务实例（如 Polaris）
	ProviderTypeSystemAllocated = "system-allocated"
)

// Service represents a service
type Service struct {
	Name        string `bson:"name" validate:"required"`
	DisplayName string `bson:"displayName"`
	Description string `bson:"description"`
	// Category is the category of the service
	Category string `bson:"category"`
	// Plans is all plans of the service.
	// - every plan has a unique name in the service
	Plans []ServicePlan `bson:"plans" validate:"required"`
}

// GetPlanByName gets the plan by name
func (s *Service) GetPlanByName(name string) (*ServicePlan, error) {
	for i, p := range s.Plans {
		if p.Name == name {
			return &s.Plans[i], nil
		}
	}
	return nil, NewNotFoundError(fmt.Sprintf("service plan(name:%s)", name))
}

// ServicePlan represents a service plan
type ServicePlan struct {
	Name         string         `bson:"name" validate:"required"`
	ProviderType string         `bson:"providerType" validate:"required"`
	Config       map[string]any `bson:"config"`
}

// MarshalConfig marshals the config to map {"value": "config json string"} to save in db
func (p *ServicePlan) MarshalConfig() error {
	if c, err := json.Marshal(p.Config); err != nil {
		return err
	} else {
		p.Config = map[string]any{configValueKey: string(c)}
		return nil
	}
}

// UnmarshalConfig unmarshals the config from map {"value": "config json string"}
func (p *ServicePlan) UnmarshalConfig() error {
	if err := json.Unmarshal([]byte(cast.ToString(p.Config[configValueKey])), &p.Config); err != nil {
		return err
	}

	delete(p.Config, configValueKey)
	return nil
}
