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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/instancestats"
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
	// 是否启用权重因子（开启后才能为单个环境开启动态权重）
	EnableWeightFactor bool `json:"enableWeightFactor"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels"`
	// 注册模式：immediate（绑定后立即注册）| on_deploy（等部署后注册）
	RegisterMode string `json:"registerMode"`

	// 生效的环境列表
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
	// 各环境的单实例权重，key 为环境名称
	EnvWeights map[string]int32 `json:"envWeights"`
	// 各环境是否开启动态权重，key 为环境名称；未出现的环境表示未开启
	EnvDynamicWeights map[string]bool `json:"envDynamicWeights"`
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

// FromModel fills output fields from a PolarisConfig domain model and optional warnings.
func (o *PolarisConfigOutputObj) FromModel(config polaris.PolarisConfig, warnings []string) *PolarisConfigOutputObj {
	envWeights := config.EnvWeights
	if envWeights == nil {
		envWeights = map[string]int32{}
	}
	envDynamicWeights := config.EnvDynamicWeights
	if envDynamicWeights == nil {
		envDynamicWeights = map[string]bool{}
	}
	*o = PolarisConfigOutputObj{
		AppID:              config.AppID,
		Name:               config.Name,
		DepSvcInstID:       depSvcInstIDToString(config.DepSvcInstID),
		InstanceKey:        config.InstanceKey,
		PolarisName:        config.PolarisName,
		PolarisNamespace:   config.PolarisNamespace,
		PolarisToken:       config.PolarisToken,
		ServicePort:        config.ServicePort,
		Direct:             config.Direct,
		KeepNotReadyPod:    config.KeepNotReadyPod,
		EnableHealthCheck:  config.EnableHealthCheck,
		EnableWeightFactor: config.EnableWeightFactor,
		ServiceLabels:      config.ServiceLabels,
		RegisterMode:       registerModeOutput(config.RegisterMode),
		ScopeEnvNames:      config.ScopeEnvNames,
		Operator:           config.Operator,
		CreatedAt:          config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          config.UpdatedAt.Format(time.RFC3339),
		Warnings:           warnings,
		EnvStates:          toEnvStateOutputs(&config),
		EnvWeights:         envWeights,
		EnvDynamicWeights:  envDynamicWeights,
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
		result[envName] = PolarisEnvStateOutput{Status: polaris.PolarisEnvStatusPendingCreate}
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
		PolarisTokenChanged: polaris.PolarisTokenChanged(config, envName, state),
		Status:              polaris.PolarisEnvStatus(config, envName, state),
	}
}

// registerModeOutput 把存量数据里缺失的注册模式补成缺省值，避免响应中出现空字符串。
func registerModeOutput(mode string) string {
	if mode == "" {
		return polaris.RegisterModeOnDeploy
	}
	return mode
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

// -----------------------------------------------------------------------------
// Create app polaris config
// -----------------------------------------------------------------------------

// CreateAppPolarisConfigInput is the JSON input for creating a polaris config.
type CreateAppPolarisConfigInput struct {
	// 是否由平台创建新的北极星服务
	CreateNewService bool `json:"createNewService"`
	// 生效的环境列表
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
	// 是否启用权重因子，默认 false。仅 createNewService 为 true 时写入北极星；
	// 开启后北极星按实例机型标记权重因子，
	// 各环境还需单独开启动态权重才会真正按机型分流
	EnableWeightFactor *bool `json:"enableWeightFactor"`
	// 服务标签
	ServiceLabels map[string]string `json:"serviceLabels"`
	// 操作人(即北极星负责人, 仅 createNewService 为 true 时有效)
	Operator *string `json:"operator"`
	// 注册模式，默认 on_deploy（等部署后注册）。
	// immediate 表示绑定环境后立即下发 PolarisConfig CR 与配套 Service 完成注册，
	// 该配置不再注入环境变量和 tRPC 框架配置。创建后不可修改
	RegisterMode *string `json:"registerMode" binding:"omitempty,oneof=immediate on_deploy"`
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
	// 是否启用权重因子（可选更新）；关闭只屏蔽各环境的动态权重，不清除各环境的开关取值
	EnableWeightFactor *bool `json:"enableWeightFactor"`
	// 服务标签（可选更新，传入时全量替换）
	ServiceLabels map[string]string `json:"serviceLabels"`
	// 组件实例标识（可选更新）
	InstanceKey *string `json:"instanceKey"`
	// 生效的环境列表（可选更新；传入时全量替换，空数组表示清空，nil 表示不更新）
	ScopeEnvNames []string `json:"scopeEnvNames"`
	// 北极星 Token（可选更新）
	PolarisToken *string `json:"polarisToken"`
	// 操作人/负责人（可选更新；未出现表示不改；空字符串非法）
	Operator *string `json:"operator"`
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

// -----------------------------------------------------------------------------
// Put env weight
// -----------------------------------------------------------------------------

// AppConfigEnvNameURIInput is the path input for APIs scoped by application, config name, and env name.
type AppConfigEnvNameURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 配置名称
	ConfigName string `uri:"configName" binding:"required,min=1"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// PutEnvWeightInput is the JSON input for updating an environment's weight.
type PutEnvWeightInput struct {
	// 单实例权重，取值范围 0-10000
	Weight *int32 `json:"weight" binding:"required,min=0,max=10000"`
	// 该环境是否开启动态权重，不传表示保持原值。
	// 开启后上面的权重作为动态调权的基准权重
	DynamicWeight *bool `json:"dynamicWeight"`
}

// PutEnvWeightOutput is the JSON response for updating an environment's weight.
type PutEnvWeightOutput struct {
	// 更新后的北极星配置
	Data *PolarisConfigOutputObj `json:"data"`
}

// FromModel fills output fields from the updated config.
func (o *PutEnvWeightOutput) FromModel(config *polaris.PolarisConfig) *PutEnvWeightOutput {
	o.Data = new(PolarisConfigOutputObj).FromModel(*config, nil)
	return o
}

// -----------------------------------------------------------------------------
// Env instance stats
// -----------------------------------------------------------------------------

// GetEnvInstanceStatsOutput is the JSON response for per-env polaris instance stats.
type GetEnvInstanceStatsOutput struct {
	Data *GetEnvInstanceStatsOutputObj `json:"data"`
}

// GetEnvInstanceStatsOutputObj is the payload of GetEnvInstanceStats.
type GetEnvInstanceStatsOutputObj struct {
	// 各环境匹配到的北极星实例统计，key 为环境名
	EnvStats map[string]EnvInstanceStatsOutput `json:"envStats"`
	// 北极星服务下全部健康实例数（含非平台注册，例如迁移业务）
	TotalHealthyInstanceCount int32 `json:"totalHealthyInstanceCount"`
	// 北极星服务下全部健康实例的权重总和
	TotalHealthyInstanceWeight int32 `json:"totalHealthyInstanceWeight"`
}

// EnvInstanceStatsOutput is the JSON representation of matched polaris instance counts.
type EnvInstanceStatsOutput struct {
	// 匹配实例中健康的数量（isHealthy && !isIsolated && weight > 0）
	HealthyInstanceCount int32 `json:"healthyInstanceCount"`
	// 匹配健康实例的权重总和
	HealthyInstanceWeight int32 `json:"healthyInstanceWeight"`
	// 匹配实例中隔离的数量（isIsolated == true）
	IsolatedInstanceCount int32 `json:"isolatedInstanceCount"`
	// 匹配到本环境 Pod 的实例总数
	TotalInstanceCount int32 `json:"totalInstanceCount"`
	// 本环境被单独设置过权重的实例数，其实际权重可能与配置的单实例权重不一致
	WeightOverriddenInstanceCount int32 `json:"weightOverriddenInstanceCount"`
}

// FromModel fills output fields from collected env instance stats.
func (o *GetEnvInstanceStatsOutput) FromModel(result *instancestats.Result) *GetEnvInstanceStatsOutput {
	obj := &GetEnvInstanceStatsOutputObj{
		EnvStats: map[string]EnvInstanceStatsOutput{},
	}
	if result != nil {
		obj.EnvStats = make(map[string]EnvInstanceStatsOutput, len(result.EnvStats))
		for envName, s := range result.EnvStats {
			obj.EnvStats[envName] = EnvInstanceStatsOutput{
				HealthyInstanceCount:  int32(s.HealthyInstanceCount),  //nolint:gosec // G115: counts fit in int32
				HealthyInstanceWeight: int32(s.HealthyInstanceWeight), //nolint:gosec // G115: weights fit in int32
				IsolatedInstanceCount: int32(s.IsolatedInstanceCount), //nolint:gosec // G115: counts fit in int32
				TotalInstanceCount:    int32(s.TotalInstanceCount),    //nolint:gosec // G115: counts fit in int32
				//nolint:gosec // G115: counts fit in int32
				WeightOverriddenInstanceCount: int32(s.WeightOverriddenInstanceCount),
			}
		}
		//nolint:gosec // G115: counts fit in int32
		obj.TotalHealthyInstanceCount = int32(result.TotalHealthyInstanceCount)
		//nolint:gosec // G115: weights fit in int32
		obj.TotalHealthyInstanceWeight = int32(result.TotalHealthyInstanceWeight)
	}
	o.Data = obj
	return o
}
