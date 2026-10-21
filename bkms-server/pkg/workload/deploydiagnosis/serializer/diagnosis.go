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

// Package serializer defines Gin input and output serializers for deploy diagnosis APIs.
package serializer

import (
	"time"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// 空态文案由 emptyReason 决定，有结论时必须是空字符串
const (
	// EmptyReasonAllHealthy 实例均已启动并通过健康检查，无需处理
	EmptyReasonAllHealthy = "all_healthy"
	// EmptyReasonNoInstance 当前环境没有可诊断的实例
	EmptyReasonNoInstance = "no_instance"
)

// 日志来源
const (
	// LogSourcePrevious 最近一次重启前的容器日志
	LogSourcePrevious = "previous"
	// LogSourceCurrent 当前容器日志
	LogSourceCurrent = "current"
	// LogSourceNone 未取到该实例日志
	LogSourceNone = "none"
)

// 动作 id 仅四值，前端写死映射
const (
	// ActionFrameworkConfig 前往框架配置文件
	ActionFrameworkConfig = "frameworkConfig"
	// ActionEnvVarConfig 前往环境变量
	ActionEnvVarConfig = "envVarConfig"
	// ActionDeployConfig 前往部署配置
	ActionDeployConfig = "deployConfig"
	// ActionRedeploy 重新部署
	ActionRedeploy = "redeploy"
)

// AppEnvURIInput 按应用和环境限定的路径参数
type AppEnvURIInput struct {
	AppID   string `uri:"appID" binding:"required,uri_slug"`
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// GetDeployDiagnosisQueryInput 诊断接口查询参数
type GetDeployDiagnosisQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// GetDeployDiagnosisOutput 诊断接口成功响应，data 即契约业务 JSON
type GetDeployDiagnosisOutput struct {
	Data *DeployDiagnosisOutputObj `json:"data"`
}

// DeployDiagnosisOutputObj 一次环境诊断结论
type DeployDiagnosisOutputObj struct {
	// GeneratedAt 诊断计算完成时间
	GeneratedAt time.Time `json:"generatedAt"`
	// CacheHit 是否直接返回缓存结果，面板不强制展示
	CacheHit bool `json:"cacheHit"`
	// Partial 日志或事件未取全时为 true，面板提示「部分证据未取到」
	Partial bool `json:"partial"`
	// PartialReasons 未取全的数据源，取值为 logs / events；没有则为空数组
	PartialReasons []string `json:"partialReasons"`
	// EmptyReason groups 为空时的空态：all_healthy / no_instance；有结论时为空字符串
	EmptyReason string `json:"emptyReason"`
	// Deploy 最新部署记录，DeployID 只供查看完整资源 YAML
	Deploy DeployOutputObj `json:"deploy"`
	// Summary 实例计数
	Summary DiagnosisSummaryObj `json:"summary"`
	// Groups 按规则分组的结论，多条时前端做标签，默认第一组
	Groups []*DiagnosisGroupObj `json:"groups"`
}

// DeployOutputObj 最新部署记录摘要
type DeployOutputObj struct {
	// DeployID 最新部署记录 ID
	DeployID string `json:"deployID"`
	// Status 部署状态，如 deployed / failed
	Status string `json:"status"`
	// Message 部署记录上的补充说明
	Message string `json:"message"`
}

// DiagnosisSummaryObj 实例计数，与入口异常数对齐
type DiagnosisSummaryObj struct {
	// TotalCount 当前环境实例总数，用于判断「有实例但都健康」
	TotalCount int `json:"totalCount"`
	// AbnormalCount 异常实例数，与部署入口的异常数对齐
	AbnormalCount int `json:"abnormalCount"`
}

// DiagnosisGroupObj 一条命中规则及其覆盖的实例
type DiagnosisGroupObj struct {
	// RuleID 规则标识，作标签 key，不直接展示
	RuleID string `json:"ruleId"`
	// RuleName 标签文案，展示为「规则名 · 实例数」
	RuleName string `json:"ruleName"`
	// Category 出错原因旁的分类，如「配置下发」
	Category string `json:"category"`
	// Confidence 置信度，面板不展示
	Confidence string `json:"confidence"`
	// Cause 出错原因主句，原样展示
	Cause string `json:"cause"`
	// CauseDetail 主句下的说明，原样展示
	CauseDetail string `json:"causeDetail"`
	// Solutions 处理步骤，按数组顺序编号展示
	Solutions []string `json:"solutions"`
	// Actions 解决方案按钮，ID 决定跳转，Label 是按钮文案
	Actions []*DiagnosisActionObj `json:"actions"`
	// RelatedConfigs 无实例时的相关配置；有实例时改读实例上的同名字段
	RelatedConfigs []*RelatedConfigRowObj `json:"relatedConfigs"`
	// Instances 命中该规则的实例，已按重启次数降序；无实例的结论为空数组
	Instances []*DiagnosisInstanceObj `json:"instances"`
}

// DiagnosisActionObj 解决方案区按钮
type DiagnosisActionObj struct {
	// ID 按钮行为，仅 frameworkConfig / envVarConfig / deployConfig / redeploy
	ID string `json:"id"`
	// Label 按钮文案
	Label string `json:"label"`
}

// RelatedConfigRowObj 相关配置表格的一行
type RelatedConfigRowObj struct {
	// Label 左列名称
	Label string `json:"label"`
	// Value 右列取值，环境变量只放变量名
	Value string `json:"value"`
	// Danger 为 true 时 Value 标红，表示与日志中的路径不一致
	Danger bool `json:"danger"`
	// Hint Value 旁的提示，空则不展示
	Hint string `json:"hint"`
}

// DiagnosisInstanceObj 同规则下的一个异常实例
type DiagnosisInstanceObj struct {
	// ID Pod 名，下载日志时传给现有日志接口
	ID string `json:"id"`
	// Status 实例状态，如 CrashLoopBackOff
	Status string `json:"status"`
	// RestartCount 重启次数，分组内已按它降序
	RestartCount int64 `json:"restartCount"`
	// Evidence 命中规则的日志行，空则不展示
	Evidence []string `json:"evidence"`
	// RelatedConfigs 该实例的相关配置，路径比对结果已算好
	RelatedConfigs []*RelatedConfigRowObj `json:"relatedConfigs"`
	// LogSource 日志来源：previous / current / none
	LogSource string `json:"logSource"`
	// LogHint 日志区说明，空则不展示
	LogHint string `json:"logHint"`
	// LogLines 日志正文，空则前端展示未取到该实例日志
	LogLines []string `json:"logLines"`
	// RelatedEvents 本次部署开始后与该实例相关的事件，空则前端展示没有相关事件
	RelatedEvents []*DiagnosisEventObj `json:"relatedEvents"`
}

// DiagnosisEventObj 与实例相关的集群事件
type DiagnosisEventObj struct {
	// CreatedAt 事件发生时间
	CreatedAt time.Time `json:"createdAt"`
	// Level 事件级别，如 Normal / Warning
	Level string `json:"level"`
	// Content 事件内容
	Content string `json:"content"`
}
