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

// Package appmodel 为应用模型相关逻辑的实现
package appmodel

import (
	"fmt"
	"time"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	k8sworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/workload"
)

// ResourceKey 资源信息
type ResourceKey struct {
	// Kind 资源类型，如：GameDeployment, Service, ConfigMap
	Kind string `bson:"kind"`
	// Name 资源名称
	Name string `bson:"name"`
}

// String 返回格式化的字符串表示（用于日志等场景）
func (r ResourceKey) String() string {
	return fmt.Sprintf("%s/%s", r.Kind, r.Name)
}

// ResourceKeys 资源引用列表
type ResourceKeys []ResourceKey

// Diff 返回差异（rs 中存在，other 中不存在）
func (rs ResourceKeys) Diff(other ResourceKeys) ResourceKeys {
	// 计算出元素集合
	otherSet := set.NewStringSet()
	for _, r := range other {
		otherSet.Add(r.String())
	}

	result := ResourceKeys{}
	// 遍历 rs，找出 other 中不存在的元素
	for _, r := range rs {
		if otherSet.Has(r.String()) {
			continue
		}
		result = append(result, r)
	}
	return result
}

// NameByKind 返回指定 Kind 的第一个资源名称，找不到则返回空串。
func (rs ResourceKeys) NameByKind(kind string) string {
	for _, r := range rs {
		if r.Kind == kind {
			return r.Name
		}
	}
	return ""
}

// Status 部署状态
type Status string

const (
	// StatusDeploying 部署中
	StatusDeploying Status = "deploying"
	// StatusDeployed 已部署
	StatusDeployed Status = "deployed"
	// StatusUninstalling 卸载中
	StatusUninstalling Status = "uninstalling"
	// StatusUninstalled 已卸载
	StatusUninstalled Status = "uninstalled"
	// StatusFailed 部署失败
	StatusFailed Status = "failed"
	// StatusCanceled 取消部署（用户取消：部署中时重新部署，自动取消之前的部署）
	StatusCanceled Status = "canceled"
	// StatusRolledBack 分批发布已回滚：Kourse 完全回滚后所有 Pod 恢复旧版本，
	// 但工作负载 spec 与部署记录中的镜像仍为本次部署的新版本
	StatusRolledBack Status = "rolled-back"

	// StatusPollingTimeout 轮询超时
	StatusPollingTimeout Status = "polling-timeout"
	// StatusPollingBroken 轮询中断
	StatusPollingBroken Status = "polling-broken"
)

// IsStable 判断状态是否为稳定态（不再变化，直到下次人工操作）
func (s Status) IsStable() bool {
	switch s {
	case StatusDeployed,
		StatusUninstalled,
		StatusFailed,
		StatusCanceled,
		StatusRolledBack:
		return true
	case StatusPollingTimeout,
		StatusPollingBroken:
		return true
	default:
		return false
	}
}

// IsUninstall 判断状态是否为卸载中 / 已卸载
func (s Status) IsUninstall() bool {
	return s == StatusUninstalling || s == StatusUninstalled
}

const (
	// 部署记录 Extras 字段 key
	// ExtraKeyDeploySource 标记本次部署记录的来源
	ExtraKeyDeploySource = "deploySource"
	// ExtraKeyBuildBranch 记录触发本次部署的代码分支，仅按分支检出时写入
	ExtraKeyBuildBranch = "buildBranch"
	// ExtraKeyBuildRevisionType 记录代码来源类型 branch / tag / commit
	ExtraKeyBuildRevisionType = "buildRevisionType"
	// ExtraKeyBuildRevision 记录分支名、Tag 名或 CommitID
	ExtraKeyBuildRevision = "buildRevision"
	// ExtraKeyBuildCommitID 记录构建完成后的提交哈希，与检出用的修订不是同一字段
	ExtraKeyBuildCommitID = "buildCommitID"
)

const (
	// deploySource 可选值
	// DeploySourceBuildAutoDeploy 表示部署由“构建+部署”链路触发
	DeploySourceBuildAutoDeploy = "buildAutoDeploy"
	// DeploySourceDirectDeploy 表示部署由直接部署链路触发
	DeploySourceDirectDeploy = "directDeploy"
)

// ReleaseMode 发布方式
type ReleaseMode string

const (
	// ReleaseModeAll 一次更新全部实例；存量部署记录未写入发布方式，同样视为该值
	ReleaseModeAll ReleaseMode = "all"
	// ReleaseModeBatch 按发布策略分批更新（Kourse Rollout Canary）
	ReleaseModeBatch ReleaseMode = "batch"
)

// ReleaseStrategySnapshot 发起分批发布时所用发布策略的快照；
// 策略后续被修改或删除不影响已发起的发布，进度查询以快照为准
type ReleaseStrategySnapshot struct {
	// ID 策略 ID
	ID string `bson:"id"`
	// Name 策略名称
	Name string `bson:"name"`
	// Batches 下发时的批次
	Batches []releasestrategy.Batch `bson:"batches"`
}

// BatchProgress 分批发布进度，由部署轮询任务读取 Rollout CR 状态后写入
type BatchProgress struct {
	// CurrentBatchIndex 当前批次序号，从 0 开始，-1 表示回滚。
	// 发布尚未进入批次、或全部批次走完后 stepStatus 被清空，此时为 nil，必须和 0 区分
	CurrentBatchIndex *int64 `bson:"currentBatchIndex,omitempty"`
	// BatchPhase 当前批次状态
	BatchPhase batchrelease.BatchPhase `bson:"batchPhase"`
	// RolloutPhase Rollout 原始 phase
	RolloutPhase string `bson:"rolloutPhase"`
	// Message Rollout 提示信息
	Message string `bson:"message"`
	// PauseStartTime 当前批次暂停开始时间（未暂停时为空）
	PauseStartTime *time.Time `bson:"pauseStartTime,omitempty"`
	// UpdatedReplicas 已更新为新版本的实例数
	UpdatedReplicas int32 `bson:"updatedReplicas"`
	// Replicas 总实例数
	Replicas int32 `bson:"replicas"`
	// OldImageTag 旧镜像标签
	OldImageTag string `bson:"oldImageTag"`
	// NewImageTag 新镜像标签
	NewImageTag string `bson:"newImageTag"`
}

// BuildAutoDeployInfo 表示构建阶段透传到部署记录中的构建信息快照。
type BuildAutoDeployInfo struct {
	// Branch 兼容字段，仅按分支检出时有值
	Branch string
	// RevisionType 代码来源类型 branch / tag / commit
	RevisionType string
	// Revision 分支名、Tag 名或 CommitID
	Revision string
	// CommitID 构建完成后的提交哈希
	CommitID string
}

// NewBuildAutoDeployExtras 将构建+部署信息编码到部署记录 Extras 中。
func NewBuildAutoDeployExtras(info *BuildAutoDeployInfo) map[string]string {
	if info == nil {
		return nil
	}
	extras := map[string]string{
		ExtraKeyDeploySource: DeploySourceBuildAutoDeploy,
	}
	if info.RevisionType != "" {
		extras[ExtraKeyBuildRevisionType] = info.RevisionType
	}
	if info.Revision != "" {
		extras[ExtraKeyBuildRevision] = info.Revision
	}
	if info.Branch != "" {
		extras[ExtraKeyBuildBranch] = info.Branch
	}
	if info.CommitID != "" {
		extras[ExtraKeyBuildCommitID] = info.CommitID
	}
	return extras
}

// Record 部署记录
type Record struct {
	// ID 部署记录 ID（唯一）
	ID bson.ObjectID `bson:"_id,omitempty"`

	// WorkspaceID 工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// AppID 应用 ID
	AppID string `bson:"appID"`

	// EnvName 环境名称
	EnvName string `bson:"envName"`
	// TrafficLaneName 泳道名称
	TrafficLaneName string `bson:"trafficLaneName"`
	// ClusterID 集群 ID
	ClusterID string `bson:"clusterID"`
	// Namespace 命名空间名称
	Namespace string `bson:"namespace"`
	// ImageTag 当前 pod 使用的镜像标签
	ImageTag string `bson:"imageTag"`
	// UpdateStrategy 更新策略
	UpdateStrategy string `bson:"updateStrategy"`
	// Replicas app model app 实例数量（pod 数量）
	Replicas int32 `bson:"replicas"`
	// DeletionCost 即 Pod DeletionCost，用于控制 pod 删除/更新顺序，对当前
	// 部署的每次灰度都会导致其值 -1，确保每次都能让当前灰度指定的 Pod 优先被处理
	DeletionCost int64 `bson:"deletionCost"`
	// LabelSelector 标签选择器
	LabelSelector map[string]string `bson:"labelSelector"`
	// WorkloadKind 本次部署的主工作负载类型（GameDeployment 或 Deployment）。
	WorkloadKind string `bson:"workloadKind"`
	// ResourceKeys 本次部署关联的资源
	ResourceKeys ResourceKeys `bson:"resourceKeys"`

	// ReleaseMode 发布方式，为空时视为一次更新全部（存量记录无需迁移），读取时使用 GetReleaseMode
	ReleaseMode ReleaseMode `bson:"releaseMode,omitempty"`
	// ReleaseStrategy 分批发布所用策略的快照，仅分批发布时有值
	ReleaseStrategy *ReleaseStrategySnapshot `bson:"releaseStrategy,omitempty"`
	// BatchProgress 分批发布进度，仅分批发布时有值
	BatchProgress *BatchProgress `bson:"batchProgress,omitempty"`

	// Message 部署相关信息
	Message string `bson:"message"`
	// Status 部署状态
	Status Status `bson:"status"`
	// Extras 额外信息
	Extras map[string]string `bson:"extras"`

	// StartedAt 开始时间
	StartedAt time.Time `bson:"startedAt"`
	// EndedAt 结束时间
	EndedAt time.Time `bson:"endedAt"`
	// Creator 创建人
	Creator string `bson:"creator"`
	// Updater 更新人
	Updater string `bson:"updater"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// MainWorkload 返回本次部署的主工作负载 Kind 与名称。
// 优先使用 WorkloadKind；为空时按 k8sworkload.MainDrivers 的优先级从 ResourceKeys 推断。
func (r *Record) MainWorkload() (kind, name string) {
	if r == nil {
		return "", ""
	}
	kind = r.WorkloadKind
	if kind == "" {
		for _, driver := range k8sworkload.MainDrivers() {
			if name = r.ResourceKeys.NameByKind(driver.Kind()); name != "" {
				return driver.Kind(), name
			}
		}
		return "", ""
	}
	return kind, r.ResourceKeys.NameByKind(kind)
}

// GetReleaseMode 返回发布方式，未写入时（存量记录）视为一次更新全部
func (r *Record) GetReleaseMode() ReleaseMode {
	if r.ReleaseMode == "" {
		return ReleaseModeAll
	}
	return r.ReleaseMode
}

// IsBatchReleasing 判断是否处于分批发布中：发布方式为分批且状态为部署中。
// 对环境 / 泳道的最新部署记录调用，用于拦截与分批发布冲突的操作；没有部署记录（nil）时返回 false。
func (r *Record) IsBatchReleasing() bool {
	if r == nil {
		return false
	}
	return r.GetReleaseMode() == ReleaseModeBatch && r.Status == StatusDeploying
}
