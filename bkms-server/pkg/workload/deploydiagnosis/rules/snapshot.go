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

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
)

// 实例日志来源
const (
	// LogSourcePrevious 最近一次重启前的容器日志
	LogSourcePrevious = "previous"
	// LogSourceCurrent 当前容器日志
	LogSourceCurrent = "current"
	// LogSourceNone 未取到该实例日志
	LogSourceNone = "none"
)

// Snapshot 一次诊断用的原始数据，规则只读这份快照
type Snapshot struct {
	// AppType 应用类型，只有 tRPC / TAF 才匹配框架配置规则
	AppType string
	// Deploy 最新一次部署；从未部署过时为 nil
	Deploy *Deploy
	// Instances 当前环境的全部实例，含健康实例和没拉到日志的实例
	Instances []*Instance
	// InstancesListed 为 true 表示已经按部署记录列过 Pod。有部署但仍为 false 时，不能当成没有实例
	InstancesListed bool
	// Events 本次部署开始后的事件。InstanceID 为空的是工作负载事件，挂到分组上
	Events []Event
	// PartialReasons 采集未取全的数据源，规则不读，原样带到诊断结论
	PartialReasons []string
	// FrameworkConfig 应用模型里的框架配置，用来和日志中的读取路径比对
	FrameworkConfig FrameworkConfig
}

// Deploy 最新一次部署的摘要
type Deploy struct {
	// DeployID 运行实例所属的部署记录 ID，可用来查看资源清单快照；一键构建从未进入部署阶段时为空
	DeployID string
	// BuildID 一键构建部署的构建 ID；直接部署时为空
	BuildID string
	// Stage 一键构建部署所处阶段 build / deploy；直接部署时为空
	Stage autodeploy.Stage
	// Status 最新记录上的原始状态，只用于展示
	Status string
	// Phase 归一后的阶段，判断成功、失败、进行中一律用它
	Phase deploystatus.DeployPhase
	// Message 最新记录上的说明，失败时通常是失败原因
	Message string
}

// Instance 单个 Pod 的状态，以及已经为它拉到的日志
type Instance struct {
	// ID Pod 名
	ID string
	// Status 实例状态，如 Running、CrashLoopBackOff
	Status string
	// RestartCount 重启次数，决定先给谁拉日志，也决定分组内的排序
	RestartCount int64
	// IsHealthy 与实例列表的健康口径一致；为 false 才进入规则匹配
	IsHealthy bool
	// LogSource 日志来源，取值见 LogSource* 常量
	LogSource string
	// LogHint 交给前端的日志区说明，没拉到日志时为空
	LogHint string
	// LogLines 已截取的日志正文，没拉日志时为空
	LogLines []string
}

// Event 一条集群事件。Pod 事件挂到对应实例，工作负载事件挂到分组
type Event struct {
	// InstanceID 关联的 Pod 名。空表示工作负载级事件，不归属某个实例
	InstanceID string
	// ResourceKind 事件关联的资源类型，如 Pod、Deployment
	ResourceKind string
	// ResourceName 事件关联的资源名
	ResourceName string
	// CreatedAt 事件发生时间
	CreatedAt time.Time
	// Level 事件级别，如 Normal、Warning
	Level string
	// Content 事件内容
	Content string
}

// FrameworkConfig 框架配置文件的名称、挂载目录和启动命令，只给框架配置规则做路径比对
type FrameworkConfig struct {
	// FileName 框架配置文件名，如 trpc_go.yaml
	FileName string
	// FilePath 配置文件在容器内的目录
	FilePath string
	// Command 启动命令和参数，拼成一行展示
	Command string
}

// AbnormalInstances 筛出不健康实例，健康实例不跑规则
func (s *Snapshot) AbnormalInstances() []*Instance {
	out := make([]*Instance, 0)
	for _, inst := range s.Instances {
		if !inst.IsHealthy {
			out = append(out, inst)
		}
	}
	return out
}
