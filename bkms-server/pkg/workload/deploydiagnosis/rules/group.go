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

package rules

// 规则标识，前端用作标签 key，也是埋点统计维度，取值发布后不要再改
const (
	// IDFrameworkConfigNotFound 框架配置文件读取失败
	IDFrameworkConfigNotFound = "framework-config-not-found"
	// IDUnknown 实例异常但未命中更具体的规则
	IDUnknown = "unknown"
	// IDDeployFailed 部署已失败但没有匹配到更具体原因
	IDDeployFailed = "deploy-failed"
	// IDBuildFailed 一键构建部署在构建阶段失败
	IDBuildFailed = "build-failed"
)

// 出错原因的分类标识，前端可按它做图标或筛选，展示文案取 Group.Category
const (
	// CategoryConfigDelivery 配置下发
	CategoryConfigDelivery = "config_delivery"
	// CategoryStartupFailure 启动失败
	CategoryStartupFailure = "startup_failure"
	// CategoryDeployFailure 部署失败
	CategoryDeployFailure = "deploy_failure"
	// CategoryBuildFailure 构建失败
	CategoryBuildFailure = "build_failure"
)

// 置信度，面板不展示，供排序与统计
const (
	// ConfidenceHigh 有明确证据（如日志行）支撑
	ConfidenceHigh = "high"
	// ConfidenceLow 兜底结论，只能给出排查方向
	ConfidenceLow = "low"
)

// 解决方案按钮的跳转目标，前端按取值写死映射
const (
	// ActionFrameworkConfig 前往框架配置文件
	ActionFrameworkConfig = "frameworkConfig"
	// ActionDeployConfig 前往部署配置
	ActionDeployConfig = "deployConfig"
	// ActionBuildHistory 前往构建记录
	ActionBuildHistory = "buildHistory"
	// ActionRedeploy 重新部署
	ActionRedeploy = "redeploy"
)

// Group 一条命中规则及其覆盖的实例
type Group struct {
	// RuleID 规则标识，取值见 ID* 常量
	RuleID string
	// RuleName 标签文案，展示为「规则名 · 实例数」
	RuleName string
	// CategoryID 分类标识，取值见 Category* 常量
	CategoryID string
	// Category 分类的展示文案，如「配置下发」
	Category string
	// Confidence 置信度，取值见 Confidence* 常量
	Confidence string
	// Cause 出错原因主句
	Cause string
	// CauseDetail 主句下的说明
	CauseDetail string
	// Solutions 处理步骤，按顺序编号展示
	Solutions []string
	// Actions 解决方案按钮
	Actions []Action
	// Instances 命中该规则的实例；阶段规则的结论为空
	Instances []*GroupInstance
	// RelatedEvents 不归属某个实例的工作负载事件，如副本创建失败、配额不足
	RelatedEvents []Event
}

// Action 解决方案区的一个按钮
type Action struct {
	// ID 跳转目标，取值见 Action* 常量
	ID string
	// Label 按钮文案
	Label string
}

// ConfigRow 相关配置表格的一行
type ConfigRow struct {
	// Label 左列名称
	Label string
	// Value 右列取值
	Value string
	// Danger 为 true 时 Value 标红，表示与日志中的路径不一致
	Danger bool
	// Hint Value 旁的提示，空则不展示
	Hint string
}

// GroupInstance 分组内的一个异常实例及其证据
type GroupInstance struct {
	// ID Pod 名，下载日志时传给现有日志接口
	ID string
	// Status 实例状态，如 CrashLoopBackOff
	Status string
	// RestartCount 重启次数，分组内按它降序
	RestartCount int64
	// Evidence 命中规则的日志行，最多 MaxEvidenceLines 行
	Evidence []string
	// RelatedConfigs 与该实例相关的配置，路径比对结果已算好
	RelatedConfigs []ConfigRow
	// LogSource 日志来源，取值见 LogSource* 常量
	LogSource string
	// LogHint 日志区说明，空则不展示
	LogHint string
	// LogLines 日志正文
	LogLines []string
	// RelatedEvents 与该实例相关的事件，最多 maxRelatedEvents 条
	RelatedEvents []Event
}
