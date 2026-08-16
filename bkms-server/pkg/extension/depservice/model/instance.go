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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
)

// ScopeType 依赖服务实例可用范围（与 envvars 包的 ScopeType 语义保持一致）
type ScopeType string

const (
	// ScopeTypeWorkspace 工作空间所有环境可用
	ScopeTypeWorkspace ScopeType = "workspace"
	// ScopeTypeEnvType 按环境类型(development/test/staging/production)生效
	ScopeTypeEnvType ScopeType = "envType"
	// ScopeTypeEnv 按具体环境名称生效
	ScopeTypeEnv ScopeType = "env"
)

type InstanceStatus string

const (
	// ProvisioningStatus 创建状态
	ProvisioningStatus InstanceStatus = "provisioning"
	// CreateFailedStatus provider 创建失败状态
	CreateFailedStatus InstanceStatus = "createFailed"
	// DeleteFailedStatus provider 删除失败状态
	DeleteFailedStatus InstanceStatus = "deleteFailed"
	// AvailableStatus 可用状态
	AvailableStatus InstanceStatus = "available"
	// DeletingStatus 删除中状态（异步删除流程进行中）
	DeletingStatus InstanceStatus = "deleting"
	// UnavailableStatus 不可用状态. 即服务实例已创建成功, 但由于某些原因处于不可用, 比如数据库在不断重启等
	UnavailableStatus InstanceStatus = "unavailable"
)

// ServiceInstance represents a service instance
type ServiceInstance struct {
	// Name 服务实例名称
	Name string        `bson:"name" validate:"required"`
	ID   bson.ObjectID `bson:"_id,omitempty"`

	ServiceName string `bson:"serviceName" validate:"required"`
	// PlanName 服务方案名
	PlanName string `bson:"planName" validate:"required"`
	// ProviderType 服务实例的来源类型. 对应 model.ServicePlan.ProviderType
	ProviderType string `bson:"providerType" validate:"required"`

	// ScopeType 表示实例的可见范围类型
	ScopeType ScopeType `bson:"scopeType" validate:"required"`
	// ScopeValue 作用域值:
	// - 当 ScopeType 为 workspace 时，固定为空字符串
	// - 当 ScopeType 为 envType 时，可选值为 development、test、staging 或 production
	// - 当 ScopeType 为 env 时，值为具体的环境名称
	ScopeValue  string `bson:"scopeValue"`
	WorkspaceID string `bson:"workspaceID" validate:"required"`

	// Config 服务实例的配置(如实例规格大小, 区域, 集群名等非敏感数据)
	Config map[string]any `bson:"config"`
	// Credentials 服务实例的敏感凭证(如数据库的账号密码等), 保存时会加密存储
	Credentials map[string]any `bson:"credentials"`

	Status InstanceStatus `bson:"status"`
	// Message 辅助记录实例的状态详情
	Message string `bson:"message"`

	Operator    string `bson:"operator" validate:"required"`
	Description string `bson:"description"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// validateScope 校验 ScopeType 与 ScopeValue 的组合是否合法
func validateScope(scopeType ScopeType, scopeValue string) error {
	switch scopeType {
	case ScopeTypeWorkspace:
		if scopeValue != "" {
			return errors.Errorf("scopeValue must be empty when scopeType is %s", scopeType)
		}
	case ScopeTypeEnvType:
		if !env.IsValidEnvType(scopeValue) {
			return errors.Errorf(
				"scopeValue %q is not a valid env type when scopeType is %s",
				scopeValue, scopeType,
			)
		}
	case ScopeTypeEnv:
		if scopeValue == "" {
			return errors.Errorf("scopeValue must not be empty when scopeType is %s", scopeType)
		}
	default:
		return errors.Errorf("unknown scopeType: %s", scopeType)
	}
	return nil
}

// MatchesEnv 判断该实例是否对给定环境(envName, envType)生效
func (i *ServiceInstance) MatchesEnv(envName, envType string) bool {
	switch i.ScopeType {
	case ScopeTypeWorkspace:
		return true
	case ScopeTypeEnvType:
		return i.ScopeValue == envType
	case ScopeTypeEnv:
		return i.ScopeValue == envName
	default:
		return false
	}
}

// String 仅用于错误信息可读性
func (i *ServiceInstance) String() string {
	return fmt.Sprintf("ServiceInstance(name=%s, ws=%s)", i.Name, i.WorkspaceID)
}
