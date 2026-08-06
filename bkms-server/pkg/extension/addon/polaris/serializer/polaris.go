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

// Package serializer defines Gin input and output serializers for polaris-config APIs.
package serializer

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("instance_key", validateInstanceKey)
	}
}

// instanceKeyRegexp 匹配以字母开头，且只包含字母、数字、下划线的字符串
var instanceKeyRegexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// validateInstanceKey 校验 instanceKey 只能包含字母、数字、下划线，且以字母开头
func validateInstanceKey(fl validator.FieldLevel) bool {
	return instanceKeyRegexp.MatchString(fl.Field().String())
}

// depSvcInstIDToString converts bson.ObjectID to string, returning empty string for zero values.
// This matches the v1 behavior where mapstructurex.BsonIDToStringHook converts zero ObjectIDs to "".
func depSvcInstIDToString(id bson.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

// -----------------------------------------------------------------------------
// Path inputs
// -----------------------------------------------------------------------------

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppConfigNameURIInput is the path input for APIs scoped by application and config name.
type AppConfigNameURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 配置名称
	ConfigName string `uri:"configName" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// Shared outputs
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// -----------------------------------------------------------------------------
// List app polaris configs
// -----------------------------------------------------------------------------

// ListAppPolarisConfigsOutput is the JSON response for listing polaris configs.
type ListAppPolarisConfigsOutput struct {
	// 北极星配置列表
	Data []*PolarisConfigOutputObj `json:"data"`
}

// PolarisConfigOutputObj is the JSON representation of a polaris config.
type PolarisConfigOutputObj struct {
	// 所属应用 ID
	AppID string `json:"appID"`
	// 组件名称
	Name string `json:"name"`
	// 关联的依赖服务实例 ID（平台创建时有值，用于后续管理）
	DepSvcInstID string `json:"depSvcInstID"`
	// 组件实例标识，用于环境变量拼接
	InstanceKey string `json:"instanceKey"`

	// 北极星实例名称
	PolarisName string `json:"polarisName"`
	// 北极星环境（命名空间）
	PolarisNamespace string `json:"polarisNamespace"`
	// 北极星 Token（敏感信息，返回时脱敏）
	PolarisToken string `json:"polarisToken"`

	// 服务端口
	ServicePort int32 `json:"servicePort"`
	// 是否为直连模式（注册 PodIP 到北极星）
	Direct bool `json:"direct"`
	// 是否保留未就绪的 Pod 在北极星
	KeepNotReadyPod bool `json:"keepNotReadyPod"`
	// 是否启用健康检查
	EnableHealthCheck bool `json:"enableHealthCheck"`
	// 服务权重
	Weight int32 `json:"weight"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels"`

	// 组件生效范围类型
	ScopeType string `json:"scopeType"`
	// 组件生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames"`
	// 负责人
	Operator string `json:"operator"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
	// 校验警告信息
	Warnings []string `json:"warnings"`
	// 各环境中已经生效的关键字段、下发错误和部署状态
	EnvStates map[string]PolarisEnvStateOutput `json:"envStates"`
}

// PolarisEnvStateOutput is the JSON representation of a single environment's applied state.
type PolarisEnvStateOutput struct {
	// 集群中已经生效的关键字段；nil 表示尚未完成首次应用部署
	AppliedFields *RedeployRequiredFieldsOutput `json:"appliedFields"`
	// 最近一次记录的动态下发错误；应用部署完成或正常下发成功后清空
	LastError string `json:"lastError"`
	// 环境信息最后更新时间；尚无环境记录时为空
	UpdatedAt string `json:"updatedAt"`
	// 生成响应时，当前配置期望的 Polaris Token 是否不同于该环境最近一次应用部署记录的 Token；
	// 环境不在当前生效范围或尚无部署快照时为 false，Token 值本身始终脱敏
	PolarisTokenChanged bool `json:"polarisTokenChanged"`
	// 部署状态: deployed / pendingCreate / pendingModify / pendingDelete
	Status string `json:"status"`
}

// RedeployRequiredFieldsOutput is the JSON representation of fields that require app redeployment.
type RedeployRequiredFieldsOutput struct {
	// InstanceKey 北极星实例标识，用于生成环境变量
	InstanceKey string `json:"instanceKey"`
	// PolarisToken 北极星访问令牌
	PolarisToken string `json:"polarisToken"`
	// ServicePort 北极星服务端口
	ServicePort int32 `json:"servicePort"`
}

const (
	// PolarisEnvStatusDeployed 表示环境在 scope 内，且部署关联字段均与部署快照一致。
	PolarisEnvStatusDeployed = "deployed"
	// PolarisEnvStatusPendingCreate 表示环境在 scope 内，但尚无部署快照。
	PolarisEnvStatusPendingCreate = "pendingCreate"
	// PolarisEnvStatusPendingModify 表示环境在 scope 内，且至少一个部署关联字段与部署快照不同。
	PolarisEnvStatusPendingModify = "pendingModify"
	// PolarisEnvStatusPendingDelete 表示环境已移出 scope，但仍保留部署快照等待下次部署删除。
	PolarisEnvStatusPendingDelete = "pendingDelete"
)

// FromModel fills output fields from a PolarisConfig domain model and optional warnings.
func (o *PolarisConfigOutputObj) FromModel(config polaris.PolarisConfig, warnings []string) *PolarisConfigOutputObj {
	*o = PolarisConfigOutputObj{
		AppID:             config.AppID,
		Name:              config.Name,
		DepSvcInstID:      depSvcInstIDToString(config.DepSvcInstID),
		InstanceKey:       config.InstanceKey,
		PolarisName:       config.PolarisName,
		PolarisNamespace:  config.PolarisNamespace,
		PolarisToken:      config.PolarisToken,
		ServicePort:       config.ServicePort,
		Direct:            config.Direct,
		KeepNotReadyPod:   config.KeepNotReadyPod,
		EnableHealthCheck: config.EnableHealthCheck,
		Weight:            config.Weight,
		ServiceLabels:     config.ServiceLabels,
		ScopeType:         string(config.ScopeType),
		ScopeEnvNames:     config.ScopeEnvNames,
		Operator:          config.Operator,
		CreatedAt:         config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         config.UpdatedAt.Format(time.RFC3339),
		Warnings:          warnings,
		EnvStates:         toEnvStateOutputs(&config),
	}
	return o
}

// toEnvStateOutputs preserves recorded states and supplements scoped environments that have not been deployed.
func toEnvStateOutputs(config *polaris.PolarisConfig) map[string]PolarisEnvStateOutput {
	result := make(map[string]PolarisEnvStateOutput, len(config.ScopeEnvNames)+len(config.EnvStates))
	for envName, state := range config.EnvStates {
		result[envName] = newPolarisEnvStateOutput(config, envName, state)
	}
	for _, envName := range config.ScopeEnvNames {
		if _, ok := result[envName]; ok {
			continue
		}
		result[envName] = PolarisEnvStateOutput{Status: PolarisEnvStatusPendingCreate}
	}
	return result
}

func newPolarisEnvStateOutput(
	config *polaris.PolarisConfig,
	envName string,
	state polaris.PolarisEnvState,
) PolarisEnvStateOutput {
	return PolarisEnvStateOutput{
		AppliedFields:       toRedeployRequiredFieldsOutput(state.AppliedFields),
		LastError:           state.LastError,
		UpdatedAt:           state.UpdatedAt.Format(time.RFC3339),
		PolarisTokenChanged: polarisTokenChanged(config, envName, state),
		Status:              polarisEnvStatus(config, envName, state),
	}
}

func toRedeployRequiredFieldsOutput(fields *polaris.RedeployRequiredFields) *RedeployRequiredFieldsOutput {
	if fields == nil {
		return nil
	}
	return &RedeployRequiredFieldsOutput{
		InstanceKey:  fields.InstanceKey,
		PolarisToken: "******",
		ServicePort:  fields.ServicePort,
	}
}

func polarisEnvStatus(
	config *polaris.PolarisConfig,
	envName string,
	state polaris.PolarisEnvState,
) string {
	if !config.IsAvailableInEnv(envName) {
		return PolarisEnvStatusPendingDelete
	}
	if state.AppliedFields == nil {
		return PolarisEnvStatusPendingCreate
	}
	if state.AppliedFields.InstanceKey != config.InstanceKey ||
		state.AppliedFields.PolarisToken != config.PolarisToken ||
		state.AppliedFields.ServicePort != config.ServicePort {
		return PolarisEnvStatusPendingModify
	}
	return PolarisEnvStatusDeployed
}

// polarisTokenChanged 比较响应生成时的配置期望 Token 与该环境最近一次部署快照中的 Token。
func polarisTokenChanged(config *polaris.PolarisConfig, envName string, state polaris.PolarisEnvState) bool {
	return config.IsAvailableInEnv(envName) &&
		state.AppliedFields != nil && state.AppliedFields.PolarisToken != config.PolarisToken
}

// -----------------------------------------------------------------------------
// Create app polaris config
// -----------------------------------------------------------------------------

// CreateAppPolarisConfigInput is the JSON input for creating a polaris config.
type CreateAppPolarisConfigInput struct {
	// 是否由平台创建新的北极星服务
	CreateNewService bool `json:"createNewService"`
	// 组件生效范围类型，可选值只能为 environment
	ScopeType string `json:"scopeType" binding:"required,oneof=environment"`
	// 组件生效的环境列表，当 scopeType 为 environment 时有效
	ScopeEnvNames []string `json:"scopeEnvNames"`
	// 组件实例标识，用于环境变量拼接，只能包含字母、数字、下划线
	InstanceKey string `json:"instanceKey" binding:"required,instance_key"`
	// 北极星实例名称
	PolarisName string `json:"polarisName" binding:"required,min=1"`
	// 北极星环境（命名空间）
	PolarisNamespace string `json:"polarisNamespace" binding:"required,oneof=Test Production Development Pre-release"`
	// 北极星 Token（当 createNewService 为 false 时必填，为 true 时由平台创建后回填）
	PolarisToken *string `json:"polarisToken"`
	// 服务端口
	ServicePort int32 `json:"servicePort" binding:"required,min=1,max=65535"`
	// 是否为直连模式，默认 false
	Direct *bool `json:"direct"`
	// 是否保留未就绪的 Pod 在北极星，默认 true
	KeepNotReadyPod *bool `json:"keepNotReadyPod"`
	// 是否启用健康检查，默认 false
	EnableHealthCheck *bool `json:"enableHealthCheck"`
	// 服务权重，默认 10
	Weight *int32 `json:"weight" binding:"omitempty,min=0"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels"`
	// 操作人(即北极星负责人, 仅 createNewService 为 true 时有效)
	Operator *string `json:"operator"`
}

// CreateAppPolarisConfigOutput is the JSON response for creating a polaris config.
type CreateAppPolarisConfigOutput struct {
	// 配置名称
	Data PolarisNameOutputObj `json:"data"`
}

// PolarisNameOutputObj is the JSON representation of a polaris config name.
type PolarisNameOutputObj struct {
	// 配置名称
	Name string `json:"name"`
}

// -----------------------------------------------------------------------------
// Patch app polaris config
// -----------------------------------------------------------------------------

// PatchAppPolarisConfigInput is the JSON input for updating a polaris config.
type PatchAppPolarisConfigInput struct {
	// 服务端口（可选更新）
	ServicePort *int32 `json:"servicePort" binding:"omitempty,min=1,max=65535"`
	// 是否为直连模式（可选更新）
	Direct *bool `json:"direct"`
	// 是否保留未就绪的 Pod 在北极星（可选更新）
	KeepNotReadyPod *bool `json:"keepNotReadyPod"`
	// 是否启用健康检查（可选更新）
	EnableHealthCheck *bool `json:"enableHealthCheck"`
	// 服务权重（可选更新）
	Weight *int32 `json:"weight" binding:"omitempty,min=0"`
	// 服务标签（可选更新，传入时全量替换）
	ServiceLabels map[string]string `json:"serviceLabels"`
	// 组件实例标识（可选更新）
	InstanceKey *string `json:"instanceKey"`
	// 组件生效范围（可选更新）
	Scope *PatchPolarisScopeInput `json:"scope"`
	// 北极星 Token（可选更新）
	PolarisToken *string `json:"polarisToken"`
}

// PatchPolarisScopeInput is the JSON input for updating polaris config scope.
type PatchPolarisScopeInput struct {
	// 组件生效范围类型，可选值只能为 environment
	ScopeType string `json:"scopeType" binding:"required,oneof=environment"`
	// 组件生效的环境列表，当 scopeType 为 environment 时有效
	ScopeEnvNames []string `json:"scopeEnvNames"`
}

// PatchAppPolarisConfigOutput is the JSON response for updating a polaris config.
type PatchAppPolarisConfigOutput struct {
	// 更新后的北极星配置
	Data *PolarisConfigOutputObj `json:"data"`
}

// FromModel fills output fields from the updated config.
func (o *PatchAppPolarisConfigOutput) FromModel(config *polaris.PolarisConfig) *PatchAppPolarisConfigOutput {
	o.Data = new(PolarisConfigOutputObj).FromModel(*config, nil)
	return o
}

// -----------------------------------------------------------------------------
// List app polaris config vars
// -----------------------------------------------------------------------------

// ListAppPolarisConfigVarsOutput is the JSON response for listing polaris config vars.
type ListAppPolarisConfigVarsOutput struct {
	// 变量列表
	Data []PolarisConfigVarOutput `json:"data"`
}

// PolarisConfigVarOutput is the JSON representation of a polaris config var.
type PolarisConfigVarOutput struct {
	// 变量名
	Key string `json:"key"`
	// 变量值
	Value string `json:"value"`
}

// FromModel fills output fields from a polaris ConfigVar.
func (o *PolarisConfigVarOutput) FromModel(v polaris.ConfigVar) *PolarisConfigVarOutput {
	o.Key = v.Key
	o.Value = v.Value
	return o
}

// -----------------------------------------------------------------------------
// Validate app polaris config
// -----------------------------------------------------------------------------

// ValidateAppPolarisConfigOutput is the JSON response for validating a polaris config.
type ValidateAppPolarisConfigOutput struct {
	// 校验警告信息
	Warnings []string `json:"warnings"`
}
