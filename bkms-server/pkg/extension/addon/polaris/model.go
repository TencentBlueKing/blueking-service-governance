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

// Package polaris 定义了北极星配置相关的实体和方法，app 相关的北极星配置会在部署时生效，
// 生成对应的 PolarisConfig CR 并影响 Workload 渲染等
//
//	 Q: 为什么北极星配置要独立于应用模型，而不是使用现有的应用组件/空间组件等概念？
//	 A: 虽然北极星是应用级别的配置（每个应用关联到独立的北极星服务），但北极星需要支持不同的环境生效范围.
//		例如，一个应用的测试环境和生产环境实例应该被注册到不同的北极星服务。而应用组件无法区分环境生效。
//		空间组件目前的设计虽然支持环境生效范围，但空间组件产品设计上应该是允许空间下多个不同应用复用的能力，
//		与北极星的应用场景不太一致。
//
// NOTE：对于这种一个应用在不同环境下需要有不同配置的情况，后续可能需要更加一致的设计。
package polaris

import (
	"strconv"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

// PolarisServiceInstances 存储单个北极星服务的配置和实例信息
type PolarisServiceInstances struct {
	// ServiceNamespace 北极星命名空间
	ServiceNamespace string
	// ServiceName 北极星服务名
	ServiceName string
	// ServicePort 应用侧服务端口（用于匹配 Pod IP+Port）
	ServicePort int32
	// Instances 北极星实例列表
	Instances []*polarisInfra.Instance
}

// Properties 北极星配置属性
type Properties struct {
	// InstanceKey 实例标识
	InstanceKey string `bson:"instanceKey"`
	// PolarisName 北极星服务名称
	PolarisName string `bson:"polarisName"`
	// PolarisNamespace 北极星命名空间
	PolarisNamespace string `bson:"polarisNamespace"`
	// PolarisToken 北极星访问令牌
	PolarisToken string `bson:"polarisToken"`
	// ServicePort 服务端口
	ServicePort int32 `bson:"servicePort"`
	// Direct 是否直连
	Direct bool `bson:"direct"`
	// KeepNotReadyPod 是否保留未就绪的 Pod
	KeepNotReadyPod bool `bson:"keepNotReadyPod"`
	// EnableHealthCheck 是否启用健康检查
	EnableHealthCheck bool `bson:"enableHealthCheck"`
	// Weight 权重
	Weight int32 `bson:"weight"`
	// ServiceLabels 服务标签
	ServiceLabels map[string]string `bson:"serviceLabels"`
	// Operator 操作人
	Operator string `bson:"operator"`
}

// PolarisConfig 北极星配置实体，存储在独立的数据表中
type PolarisConfig struct {
	// Name 配置名称，应用内唯一
	Name string `bson:"name" validate:"required"`
	// AppID 所属应用 ID
	AppID string `bson:"appID" validate:"required"`

	// Properties 北极星配置属性
	Properties `bson:",inline" mapstructure:",squash"`

	// ScopeType 生效范围类型: environment
	ScopeType component.ScopeType `bson:"scopeType"`
	// ScopeEnvNames 生效的环境列表，当 ScopeType 为 environment 时有效
	ScopeEnvNames []string `bson:"scopeEnvNames"`

	// DepSvcInstID 关联的依赖服务实例 ID（平台创建时有值，用于后续管理）
	DepSvcInstID bson.ObjectID `bson:"depSvcInstID,omitempty"`

	// EnvStates 各环境中已经生效的关键字段和下发错误
	EnvStates map[string]PolarisEnvState `bson:"envStates,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// GenerateName 生成配置名称
func (c *PolarisConfig) GenerateName() string {
	return "polaris-" + stringx.Random(5)
}

// IsAvailableInEnv 检查配置是否在指定环境中可用
func (c *PolarisConfig) IsAvailableInEnv(envName string) bool {
	return lo.Contains(c.ScopeEnvNames, envName)
}

// ConfigVar 配置变量
type ConfigVar struct {
	Key   string
	Value string
}

// GetVars 获取北极星配置的变量列表
// 返回变量: {instanceKey}_polarisToken, {instanceKey}_servicePort
func (c *PolarisConfig) GetVars() []ConfigVar {
	return []ConfigVar{
		{
			Key:   c.InstanceKey + "_polarisToken",
			Value: c.PolarisToken,
		},
		{
			Key:   c.InstanceKey + "_serviceport",
			Value: strconv.Itoa(int(c.ServicePort)),
		},
	}
}

// ConfigUpdateData 定义了更新 PolarisConfig 时允许修改的数据
type ConfigUpdateData struct {
	InstanceKey       *string
	ServicePort       *int32
	Direct            *bool
	KeepNotReadyPod   *bool
	EnableHealthCheck *bool
	Weight            *int32
	ServiceLabels     map[string]string
	Scope             *PatchPolarisScope
	PolarisToken      *string
}

// PatchPolarisScope 更新 PolarisConfig 的生效范围
type PatchPolarisScope struct {
	ScopeType     component.ScopeType
	ScopeEnvNames []string
}

// RegistryServiceEntry 表示 tRPC 配置中 plugins.registry.polaris.service 的单条服务配置
type RegistryServiceEntry struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Token     string `yaml:"token"`
}

// GenerateRegistryConfig 生成北极星注册中心的 service 配置条目。
// 仅当 EnableHealthCheck 为 true 时才有意义。
func (c *PolarisConfig) GenerateRegistryConfig() RegistryServiceEntry {
	return RegistryServiceEntry{
		Name:      c.PolarisName,
		Namespace: c.PolarisNamespace,
		Token:     c.PolarisToken,
	}
}

// CollectRegistryServiceEntries 从给定的 PolarisConfig 列表中收集所有启用健康检查的配置的
// registry service 条目。
func CollectRegistryServiceEntries(configs []*PolarisConfig) []RegistryServiceEntry {
	var entries []RegistryServiceEntry
	for _, config := range configs {
		if config.EnableHealthCheck {
			entries = append(entries, config.GenerateRegistryConfig())
		}
	}
	return entries
}

// RedeployRequiredFields 保存要求应用重新部署才能生效的字段。
type RedeployRequiredFields struct {
	// InstanceKey 北极星实例标识，用于生成环境变量
	InstanceKey string `bson:"instanceKey"`
	// PolarisToken 北极星访问令牌
	PolarisToken string `bson:"polarisToken"`
	// ServicePort 北极星服务端口
	ServicePort int32 `bson:"servicePort"`
}

// PolarisEnvState 记录单个环境的部署快照和最近一次动态下发错误。
type PolarisEnvState struct {
	// AppliedFields 集群中已经生效的关键字段；nil 表示尚未完成首次应用部署
	AppliedFields *RedeployRequiredFields `bson:"appliedFields,omitempty"`
	// LastError 最近一次记录的动态下发错误；应用部署完成或正常下发成功后清空
	LastError string `bson:"lastError,omitempty"`
	// UpdatedAt 环境信息最后更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// GetEnvState 返回指定环境的状态；不存在时返回零值。
func (c *PolarisConfig) GetEnvState(envName string) PolarisEnvState {
	return c.EnvStates[envName]
}

// NewRedeployRequiredFields 根据北极星配置创建要求应用重新部署才能生效的字段。
func NewRedeployRequiredFields(config *PolarisConfig) *RedeployRequiredFields {
	return &RedeployRequiredFields{
		InstanceKey:  config.InstanceKey,
		PolarisToken: config.PolarisToken,
		ServicePort:  config.ServicePort,
	}
}
