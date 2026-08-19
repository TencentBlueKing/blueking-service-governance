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
	"sort"
	"strconv"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

// DefaultEnvWeight 环境未单独设置权重时使用的默认值。
const DefaultEnvWeight int32 = 100

// 注册模式，决定配置何时在北极星侧注册，以及是否参与 Workload 渲染。
const (
	// RegisterModeOnDeploy 等部署后注册：配置参与 Workload 渲染（环境变量、tRPC 框架配置），
	// CR 随应用部署下发，是历史行为，也是缺省值。
	RegisterModeOnDeploy = "on_deploy"
	// RegisterModeImmediate 绑定后立即注册：配置不参与 Workload 渲染，绑定环境时直接下发
	// PolarisConfig CR 与配套 Service 完成注册，无需业务部署。
	RegisterModeImmediate = "immediate"
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
	// ServiceLabels 服务标签
	ServiceLabels map[string]string `bson:"serviceLabels"`
	// Operator 操作人
	Operator string `bson:"operator"`
	// RegisterMode 注册模式，取值见 RegisterModeOnDeploy / RegisterModeImmediate；
	// 空值按 RegisterModeOnDeploy 解释，创建后不可修改
	RegisterMode string `bson:"registerMode"`
}

// PolarisConfig 北极星配置实体，存储在独立的数据表中
type PolarisConfig struct {
	// Name 配置名称，应用内唯一
	Name string `bson:"name" validate:"required"`
	// AppID 所属应用 ID
	AppID string `bson:"appID" validate:"required"`

	// Properties 北极星配置属性
	Properties `bson:",inline" mapstructure:",squash"`

	// ScopeEnvNames 生效的环境列表
	ScopeEnvNames []string `bson:"scopeEnvNames"`

	// DepSvcInstID 关联的依赖服务实例 ID（平台创建时有值，用于后续管理）
	DepSvcInstID bson.ObjectID `bson:"depSvcInstID,omitempty"`

	// EnvStates 各环境中已经生效的关键字段和下发错误
	EnvStates map[string]PolarisEnvState `bson:"envStates,omitempty"`

	// EnvWeights 各环境的单实例权重（key 为环境名）。
	// 基本与 EnvStates 同生命周期：未部署离域立即删除；已部署离域保留至下次离域部署/卸载；
	// 仍在 scope 内卸载时保留，供再次部署使用。缺省使用 DefaultEnvWeight。
	EnvWeights map[string]int32 `bson:"envWeights,omitempty"`

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

// IsImmediateRegister 判断配置是否为绑定后立即注册模式。
// 空值按 RegisterModeOnDeploy 解释，保证存量数据与滚动发布窗口内的行为不变。
func (c *PolarisConfig) IsImmediateRegister() bool {
	return c.RegisterMode == RegisterModeImmediate
}

// EnvNamesOutsideScope 返回仍留有环境记录、但已经不在 scope 内的环境名称。
func (c *PolarisConfig) EnvNamesOutsideScope() []string {
	envNames := make([]string, 0)
	for envName := range c.EnvStates {
		if c.IsAvailableInEnv(envName) {
			continue
		}
		envNames = append(envNames, envName)
	}
	sort.Strings(envNames)
	return envNames
}

// TrackedEnvNames 返回配置仍在跟踪的全部环境：scope ∪ 仍有 EnvState 记录的环境。
func (c *PolarisConfig) TrackedEnvNames() []string {
	envNames := lo.Uniq(append(append([]string{}, c.ScopeEnvNames...), lo.Keys(c.EnvStates)...))
	sort.Strings(envNames)
	return envNames
}

// ConfigVar 配置变量
type ConfigVar struct {
	Key   string
	Value string
}

// GetVars 返回该配置会注入到 Workload 的环境变量：
// {instanceKey}_polarisToken、{instanceKey}_serviceport。
// immediate 模式不参与 Workload 渲染，返回空列表。
func (c *PolarisConfig) GetVars() []ConfigVar {
	if c.IsImmediateRegister() {
		return []ConfigVar{}
	}
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
	ServiceLabels     map[string]string
	// ScopeEnvNames 生效环境列表；nil 表示不更新，非 nil（含空切片）表示覆盖
	ScopeEnvNames []string
	PolarisToken  *string
	Operator      *string
	// envWeights 仅由 service 在 scope 变化时生成并交给 store 持久化。
	envWeights map[string]int32
}

// affectsWorkload 判断本次更新是否影响 PolarisConfig CR / 工作负载渲染。
// operator 只同步北极星 Owners，不参与 CR，因此单独更新负责人时不触发动态下发。
func (d *ConfigUpdateData) affectsWorkload() bool {
	if d == nil {
		return false
	}
	return d.InstanceKey != nil ||
		d.ServicePort != nil ||
		d.Direct != nil ||
		d.KeepNotReadyPod != nil ||
		d.EnableHealthCheck != nil ||
		d.ServiceLabels != nil ||
		d.ScopeEnvNames != nil ||
		d.PolarisToken != nil
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
//
// immediate 模式的配置不参与 Workload 渲染，不往框架配置文件注入 registry 配置；
// 这类业务若确实需要框架侧上报，自行在配置文件中书写即可，Patcher 不会覆盖已有内容。
func CollectRegistryServiceEntries(configs []*PolarisConfig) []RegistryServiceEntry {
	var entries []RegistryServiceEntry
	for _, config := range configs {
		if config.EnableHealthCheck && !config.IsImmediateRegister() {
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

// IsDeployed 该环境是否已完成首次应用部署。
func (s PolarisEnvState) IsDeployed() bool {
	return s.AppliedFields != nil
}

// GetEnvWeight 获取指定环境的有效权重：优先使用环境级别值，否则使用默认值。
func (c *PolarisConfig) GetEnvWeight(envName string) int32 {
	if w, ok := c.EnvWeights[envName]; ok {
		return w
	}
	return DefaultEnvWeight
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
