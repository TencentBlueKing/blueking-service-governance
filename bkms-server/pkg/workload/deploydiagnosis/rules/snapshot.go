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

// Package rules 存放部署诊断的具体规则。规则集按顺序调用 match，命中后只渲染这一条。
package rules

import (
	"time"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

const (
	// IDConfigFileNotFound 框架配置文件读取失败的规则标识
	IDConfigFileNotFound = "config-file-not-found"
	// IDUnknown 未命中具体规则时的兜底标识
	IDUnknown = "unknown"

	// MaxEvidenceLines 单条规则最多保留的证据行数
	MaxEvidenceLines = 3
	// maxRelatedEvents 单个实例最多挂上的相关事件条数
	maxRelatedEvents = 6
	// PathMismatchHint 日志路径与配置路径不一致时，标红行旁的提示
	PathMismatchHint = "与上方配置不一致，需改成同一路径"
)

// Snapshot 一次诊断用的原始数据。规则只读这份快照
type Snapshot struct {
	// AppType 应用类型。只有 tRPC / TAF 才匹配框架配置规则
	AppType string
	// Deploy 最新一次部署的摘要，空态要靠它判断部署是否已失败
	Deploy serializer.DeployOutputObj
	// Instances 当前环境的全部实例，含健康实例和没拉到日志的实例
	Instances []*Instance
	// Events 本次部署开始后的环境事件，匹配时按资源名挂到实例上
	Events []Event
	// Framework 应用模型里的框架配置，用来和日志中的读取路径比对
	Framework Framework
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
	// LogSource 日志来源：previous / current / none
	LogSource string
	// LogHint 交给前端的日志区说明，没拉到日志时为空
	LogHint string
	// LogLines 已截取的日志正文，没拉日志时为空
	LogLines []string
}

// Event 一条集群事件，用来填进实例的相关事件
type Event struct {
	// InstanceID 事件关联的资源名，通常是 Pod 名；空表示归不到某个实例
	InstanceID string
	// CreatedAt 事件发生时间
	CreatedAt time.Time
	// Level 事件级别，如 Normal、Warning
	Level string
	// Content 事件内容
	Content string
}

// Framework 应用模型中的框架配置文件和启动命令，只给配置文件规则做路径比对
type Framework struct {
	// FileName 框架配置文件名，如 trpc_go.yaml
	FileName string
	// FilePath 配置文件在容器内的目录
	FilePath string
	// Command 启动命令和参数，拼成一行展示
	Command string
}

// AbnormalInstances 筛出不健康实例。健康实例不跑规则
func (s *Snapshot) AbnormalInstances() []*Instance {
	out := make([]*Instance, 0)
	for _, inst := range s.Instances {
		if !inst.IsHealthy {
			out = append(out, inst)
		}
	}
	return out
}

// IsDeployFailed 认失败、轮询超时、轮询中断。Helm 用同一套字面量，快照里只存字符串
func IsDeployFailed(status string) bool {
	switch appmodeldeploy.Status(status) {
	case appmodeldeploy.StatusFailed, appmodeldeploy.StatusPollingTimeout, appmodeldeploy.StatusPollingBroken:
		return true
	default:
		return false
	}
}
