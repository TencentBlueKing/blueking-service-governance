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

	"github.com/samber/lo"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
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

// GetDeployDiagnosisOutput 诊断接口成功响应
type GetDeployDiagnosisOutput struct {
	Data *DeployDiagnosisOutputObj `json:"data"`
}

// DeployDiagnosisOutputObj 一次环境诊断结论
type DeployDiagnosisOutputObj struct {
	// GeneratedAt 诊断完成时间
	GeneratedAt time.Time `json:"generatedAt"`
	// Partial 日志或事件未取全时为 true，提示「部分证据未取到」
	Partial bool `json:"partial"`
	// PartialReasons 未取全的数据源：logs / events；没有则为空数组
	PartialReasons []string `json:"partialReasons"`
	// EmptyReason groups 为空时的空态：no_deploy / deploying / all_healthy / no_instance / instances_unavailable；
	// 有结论时为空字符串
	EmptyReason string `json:"emptyReason"`
	// Deploy 最新一次部署；从未部署过时各字段为空
	Deploy DeployOutputObj `json:"deploy"`
	// Summary 实例计数
	Summary DiagnosisSummaryObj `json:"summary"`
	// Groups 按规则分组的结论，多条时默认展示第一组
	Groups []*DiagnosisGroupObj `json:"groups"`
}

// DeployOutputObj 最新一次部署摘要
type DeployOutputObj struct {
	// DeployID 运行实例所属的部署记录 ID，用于查看完整资源 YAML；一键构建从未进入部署阶段时为空
	DeployID string `json:"deployID"`
	// BuildID 一键构建部署的构建 ID，用于跳转构建记录；直接部署时为空
	BuildID string `json:"buildID"`
	// Stage 一键构建部署所处阶段：build / deploy；直接部署时为空
	Stage string `json:"stage"`
	// Status 最新记录上的原始状态，只用于展示
	Status string `json:"status"`
	// Phase 归一后的阶段：inProgress / succeeded / failed / other
	Phase string `json:"phase"`
	// Message 最新记录上的说明，失败时通常是失败原因
	Message string `json:"message"`
}

// DiagnosisSummaryObj 实例计数
type DiagnosisSummaryObj struct {
	// TotalCount 当前环境实例总数
	TotalCount int `json:"totalCount"`
	// AbnormalCount 异常实例数，与部署入口的异常数口径一致
	AbnormalCount int `json:"abnormalCount"`
}

// DiagnosisGroupObj 一条命中规则及其覆盖的实例
type DiagnosisGroupObj struct {
	// RuleID 规则标识，作标签 key：framework-config-not-found / unknown / deploy-failed / build-failed
	RuleID string `json:"ruleId"`
	// RuleName 标签文案，展示为「规则名 · 实例数」
	RuleName string `json:"ruleName"`
	// CategoryID 分类标识：config_delivery / startup_failure / deploy_failure / build_failure
	CategoryID string `json:"categoryId"`
	// Category 分类的展示文案，如「配置下发」
	Category string `json:"category"`
	// Confidence 置信度：high / low，面板不展示
	Confidence string `json:"confidence"`
	// Cause 出错原因主句
	Cause string `json:"cause"`
	// CauseDetail 主句下的说明
	CauseDetail string `json:"causeDetail"`
	// Solutions 处理步骤，按数组顺序编号展示
	Solutions []string `json:"solutions"`
	// Actions 解决方案按钮
	Actions []*DiagnosisActionObj `json:"actions"`
	// Instances 命中该规则的实例，已按重启次数降序；阶段性结论（部署 / 构建失败）为空数组
	Instances []*DiagnosisInstanceObj `json:"instances"`
	// RelatedEvents 不归属某个实例的工作负载事件，如副本创建失败、配额不足
	RelatedEvents []*DiagnosisEventObj `json:"relatedEvents"`
}

// DiagnosisActionObj 解决方案区按钮
type DiagnosisActionObj struct {
	// ID 跳转目标：frameworkConfig / deployConfig / buildHistory / redeploy
	ID string `json:"id"`
	// Label 按钮文案
	Label string `json:"label"`
}

// RelatedConfigRowObj 相关配置表格的一行
type RelatedConfigRowObj struct {
	// Label 左列名称
	Label string `json:"label"`
	// Value 右列取值
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
	// RestartCount 重启次数
	RestartCount int64 `json:"restartCount"`
	// Evidence 命中规则的日志行，空则不展示
	Evidence []string `json:"evidence"`
	// RelatedConfigs 该实例的相关配置，路径比对结果已算好
	RelatedConfigs []*RelatedConfigRowObj `json:"relatedConfigs"`
	// LogSource 日志来源：previous / current / none
	LogSource string `json:"logSource"`
	// LogHint 日志区说明，空则不展示
	LogHint string `json:"logHint"`
	// LogLines 日志正文，空表示未取到该实例日志
	LogLines []string `json:"logLines"`
	// RelatedEvents 本次部署开始后与该实例相关的事件，空表示没有相关事件
	RelatedEvents []*DiagnosisEventObj `json:"relatedEvents"`
}

// DiagnosisEventObj 一条集群事件
type DiagnosisEventObj struct {
	// CreatedAt 事件发生时间
	CreatedAt time.Time `json:"createdAt"`
	// Level 事件级别，如 Normal / Warning
	Level string `json:"level"`
	// ResourceKind 事件关联的资源类型，如 Pod、Deployment
	ResourceKind string `json:"resourceKind"`
	// ResourceName 事件关联的资源名
	ResourceName string `json:"resourceName"`
	// Content 事件内容
	Content string `json:"content"`
}

// FromDiagnosis 把诊断结论转成响应结构，所有切片字段都保证输出 [] 而不是 null
func (o *DeployDiagnosisOutputObj) FromDiagnosis(d *deploydiagnosis.Diagnosis) *DeployDiagnosisOutputObj {
	o.GeneratedAt = d.GeneratedAt
	o.Partial = len(d.PartialReasons) > 0
	o.PartialReasons = append([]string{}, d.PartialReasons...)
	o.EmptyReason = d.EmptyReason
	if d.Deploy != nil {
		o.Deploy = DeployOutputObj{
			DeployID: d.Deploy.DeployID,
			BuildID:  d.Deploy.BuildID,
			Stage:    string(d.Deploy.Stage),
			Status:   d.Deploy.Status,
			Phase:    string(d.Deploy.Phase),
			Message:  d.Deploy.Message,
		}
	}
	o.Summary = DiagnosisSummaryObj{TotalCount: d.TotalCount, AbnormalCount: d.AbnormalCount}
	o.Groups = lo.Map(d.Groups, func(g *rules.Group, _ int) *DiagnosisGroupObj { return newGroupObj(g) })
	return o
}

func newGroupObj(g *rules.Group) *DiagnosisGroupObj {
	return &DiagnosisGroupObj{
		RuleID:      g.RuleID,
		RuleName:    g.RuleName,
		CategoryID:  g.CategoryID,
		Category:    g.Category,
		Confidence:  g.Confidence,
		Cause:       g.Cause,
		CauseDetail: g.CauseDetail,
		Solutions:   append([]string{}, g.Solutions...),
		Actions: lo.Map(g.Actions, func(a rules.Action, _ int) *DiagnosisActionObj {
			return &DiagnosisActionObj{ID: a.ID, Label: a.Label}
		}),
		Instances: lo.Map(
			g.Instances,
			func(inst *rules.GroupInstance, _ int) *DiagnosisInstanceObj { return newInstanceObj(inst) },
		),
		RelatedEvents: newEventObjs(g.RelatedEvents),
	}
}

func newInstanceObj(inst *rules.GroupInstance) *DiagnosisInstanceObj {
	return &DiagnosisInstanceObj{
		ID:           inst.ID,
		Status:       inst.Status,
		RestartCount: inst.RestartCount,
		Evidence:     append([]string{}, inst.Evidence...),
		RelatedConfigs: lo.Map(inst.RelatedConfigs, func(row rules.ConfigRow, _ int) *RelatedConfigRowObj {
			return &RelatedConfigRowObj{Label: row.Label, Value: row.Value, Danger: row.Danger, Hint: row.Hint}
		}),
		LogSource:     inst.LogSource,
		LogHint:       inst.LogHint,
		LogLines:      append([]string{}, inst.LogLines...),
		RelatedEvents: newEventObjs(inst.RelatedEvents),
	}
}

func newEventObjs(events []rules.Event) []*DiagnosisEventObj {
	return lo.Map(events, func(ev rules.Event, _ int) *DiagnosisEventObj {
		return &DiagnosisEventObj{
			CreatedAt:    ev.CreatedAt,
			Level:        ev.Level,
			ResourceKind: ev.ResourceKind,
			ResourceName: ev.ResourceName,
			Content:      ev.Content,
		}
	})
}
