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

package bkci

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrBuildNodeNotFound 构建机不存在或不在工作空间的蓝盾项目下
	ErrBuildNodeNotFound = errors.New("build node not found")
	// ErrBuildNodeOffline 构建机 Agent 离线
	ErrBuildNodeOffline = errors.New("build node agent offline")
	// ErrBuildNodeOSUnsupported 构建机不是 Linux，内置流水线脚本依赖 bash 和 docker
	ErrBuildNodeOSUnsupported = errors.New("build node os unsupported")
	// ErrBuildNodeNoUsePerm 当前用户在蓝盾没有该构建机的使用权限
	ErrBuildNodeNoUsePerm = errors.New("no permission to use build node")
)

// supportedBuildNodeOS 内置流水线支持的构建机操作系统，与蓝盾返回的 osName 忽略大小写比较
const supportedBuildNodeOS = "linux"

// UnselectableReason 构建机不能被关联的原因，空字符串表示可关联
type UnselectableReason string

const (
	// UnselectableReasonOffline Agent 离线
	UnselectableReasonOffline UnselectableReason = "offline"
	// UnselectableReasonOSUnsupported 操作系统不是 Linux
	UnselectableReasonOSUnsupported UnselectableReason = "osUnsupported"
)

// BuildNodeState 已关联构建机的实时状态
type BuildNodeState string

const (
	// BuildNodeStateOnline Agent 在线
	BuildNodeStateOnline BuildNodeState = "online"
	// BuildNodeStateOffline Agent 离线
	BuildNodeStateOffline BuildNodeState = "offline"
	// BuildNodeStateNotFound 构建机已从蓝盾删除
	BuildNodeStateNotFound BuildNodeState = "notFound"
	// BuildNodeStateUnknown 查询蓝盾失败，状态未知
	BuildNodeStateUnknown BuildNodeState = "unknown"
)

// BuildNodeCandidate 可供关联的候选构建机
type BuildNodeCandidate struct {
	NodeHashID  string
	DisplayName string
	IP          string
	OSName      string
	Online      bool
	// UnselectableReason 为空表示可关联
	UnselectableReason UnselectableReason
}

// BuildNodeBindingState 工作空间的构建机关联现状
type BuildNodeBindingState struct {
	// ProjectCode 工作空间的蓝盾项目 Code，为空表示尚未初始化蓝盾项目
	ProjectCode string
	// Binding 当前关联的构建机，未关联时为 nil
	Binding *BuildNodeBinding
	// NodeState 关联构建机的实时状态，仅 Binding 非空时有值
	NodeState BuildNodeState
}

// BindResult 关联或更换构建机的结果
type BindResult struct {
	// Before 变更前的关联，首次关联时为 nil
	Before *BuildNodeBinding
	// After 变更后的关联
	After *BuildNodeBinding
	// Changed 是否发生了变更，选择的就是当前关联的构建机时为 false
	Changed bool
}

// BuildNodeManager 工作空间的蓝盾第三方构建机关联管理。
// 读写 BuildNodeBinding，并读取 Project 获取工作空间对应的蓝盾项目
type BuildNodeManager struct {
	db          *mongo.Database
	workspaceID string
}

// NewBuildNodeManager ...
func NewBuildNodeManager(db *mongo.Database, workspaceID string) *BuildNodeManager {
	return &BuildNodeManager{db: db, workspaceID: workspaceID}
}

// ListCandidates 列出工作空间蓝盾项目下的第三方构建机，并标注每台是否可关联。
// 工作空间尚未初始化蓝盾项目时返回 ErrProjectNotFound
func (m *BuildNodeManager) ListCandidates(ctx context.Context) ([]BuildNodeCandidate, error) {
	project, err := m.getProject(ctx)
	if err != nil {
		return nil, err
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	nodes, err := client.ListThirdPartyAgentNodes(ctx, project.Code)
	if err != nil {
		return nil, errors.Wrapf(err, "list build nodes of workspace %s", m.workspaceID)
	}

	return lo.Map(nodes, func(n bkci.AgentNode, _ int) BuildNodeCandidate {
		return BuildNodeCandidate{
			NodeHashID:         n.NodeHashID,
			DisplayName:        m.resolveDisplayName(n),
			IP:                 n.IP,
			OSName:             n.OSName,
			Online:             n.AgentOnline,
			UnselectableReason: m.checkSelectable(n),
		}
	}), nil
}

// Get 查询工作空间的构建机关联现状，已关联时实时查询构建机状态。
// 查询状态失败只会把状态降级为未知，不会让整个查询失败
func (m *BuildNodeManager) Get(ctx context.Context) (*BuildNodeBindingState, error) {
	state := &BuildNodeBindingState{}

	project, err := m.getProject(ctx)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return state, nil
		}
		return nil, err
	}
	state.ProjectCode = project.Code

	binding, err := m.getCurrentBinding(ctx)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return state, nil
	}
	state.Binding = binding
	state.NodeState = m.queryNodeState(ctx, binding)
	return state, nil
}

// Bind 关联或更换工作空间的构建机。
// 构建机必须在线、是 Linux，且当前用户在蓝盾有使用权限；校验失败时原关联保持不变
func (m *BuildNodeManager) Bind(ctx context.Context, nodeHashID string) (*BindResult, error) {
	before, err := m.getCurrentBinding(ctx)
	if err != nil {
		return nil, err
	}

	// 重复提交同一台构建机时不重新校验、不刷新更新时间，避免产生无意义的操作记录
	if before != nil && before.NodeHashID == nodeHashID {
		return &BindResult{Before: before, After: before}, nil
	}

	project, err := m.getProject(ctx)
	if err != nil {
		return nil, err
	}

	user := auth.MustGetUser(ctx)
	client, err := bkci.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	status, err := client.GetThirdPartyAgentNodeStatus(ctx, project.Code, nodeHashID)
	if err != nil {
		if errors.Is(err, bkci.ObjectNotFound) {
			return nil, errors.Wrapf(ErrBuildNodeNotFound, "node %s in project %s", nodeHashID, project.Code)
		}
		return nil, errors.Wrapf(err, "get status of build node %s", nodeHashID)
	}
	if err = m.validateBindable(status); err != nil {
		return nil, errors.Wrapf(err, "node %s", nodeHashID)
	}

	after := &BuildNodeBinding{
		WorkspaceID: m.workspaceID,
		ProjectCode: project.Code,
		NodeHashID:  status.NodeHashID,
		AgentHashID: status.AgentHashID,
		DisplayName: m.resolveDisplayName(status.AgentNode),
		IP:          status.IP,
		OSName:      status.OSName,
		Updater:     user.ID,
	}
	bindingStore := store.Of(m.db, BuildNodeBindingMeta)
	if _, err = bindingStore.Upsert(ctx, after, ByWorkspaceAndNode(m.workspaceID, after.NodeHashID)); err != nil {
		return nil, errors.Wrapf(err, "save build node binding of workspace %s", m.workspaceID)
	}

	// 目前一个工作空间只保留一台构建机。数据库不做唯一约束（后续可能支持多台），
	// 由这里在写入后删掉其余构建机；并发关联时多写出的记录也会在下一次关联时被清掉
	if _, err = bindingStore.DeleteMany(
		ctx, ByWorkspace(m.workspaceID), store.Where("nodeHashID", bson.M{"$ne": after.NodeHashID}),
	); err != nil {
		return nil, errors.Wrapf(err, "remove other build node bindings of workspace %s", m.workspaceID)
	}
	return &BindResult{Before: before, After: after, Changed: true}, nil
}

// Unbind 解除工作空间的构建机关联，返回被解除的关联；未关联时返回 nil 且不报错
func (m *BuildNodeManager) Unbind(ctx context.Context) (*BuildNodeBinding, error) {
	removed, err := m.getCurrentBinding(ctx)
	if err != nil || removed == nil {
		return nil, err
	}

	// 按工作空间整体删除，连同并发关联可能多写出的记录一起清掉
	if _, err = store.Of(m.db, BuildNodeBindingMeta).DeleteMany(ctx, ByWorkspace(m.workspaceID)); err != nil {
		return nil, errors.Wrapf(err, "delete build node bindings of workspace %s", m.workspaceID)
	}
	return removed, nil
}

// getCurrentBinding 获取工作空间当前关联的构建机，未关联时返回 nil。
// 数据库允许同一工作空间存在多条记录，取最近更新的一条，保证结果确定
func (m *BuildNodeManager) getCurrentBinding(ctx context.Context) (*BuildNodeBinding, error) {
	binding, err := store.Of(m.db, BuildNodeBindingMeta).Get(
		ctx, ByWorkspace(m.workspaceID), store.Desc("updatedAt"),
	)
	if err != nil {
		if errors.Is(err, ErrBuildNodeBindingNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get build node binding of workspace %s", m.workspaceID)
	}
	return binding, nil
}

// getProject 获取工作空间对应的蓝盾项目，不存在时返回可用 errors.Is 判定的 ErrProjectNotFound
func (m *BuildNodeManager) getProject(ctx context.Context) (*Project, error) {
	project, err := store.Of(m.db, ProjectMeta).Get(ctx, ByWorkspace(m.workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci project of workspace %s", m.workspaceID)
	}
	return project, nil
}

// queryNodeState 实时查询已关联构建机的状态，失败时降级为未知
func (m *BuildNodeManager) queryNodeState(ctx context.Context, binding *BuildNodeBinding) BuildNodeState {
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		log.Warnf(ctx, "create bkci client to query build node %s: %v", binding.NodeHashID, err)
		return BuildNodeStateUnknown
	}

	status, err := client.GetThirdPartyAgentNodeStatus(ctx, binding.ProjectCode, binding.NodeHashID)
	if err != nil {
		if errors.Is(err, bkci.ObjectNotFound) {
			return BuildNodeStateNotFound
		}
		// 错误在此被吞掉并降级为未知状态，因此只在这里打一次日志
		log.Warnf(ctx, "query status of build node %s: %v", binding.NodeHashID, err)
		return BuildNodeStateUnknown
	}
	return lo.Ternary(status.AgentOnline, BuildNodeStateOnline, BuildNodeStateOffline)
}

// checkSelectable 返回构建机不能被关联的原因，可关联时返回空字符串
func (m *BuildNodeManager) checkSelectable(n bkci.AgentNode) UnselectableReason {
	if !n.AgentOnline {
		return UnselectableReasonOffline
	}
	if !strings.EqualFold(n.OSName, supportedBuildNodeOS) {
		return UnselectableReasonOSUnsupported
	}
	return ""
}

// validateBindable 在候选条件之外，还要求有蓝盾使用权限并能拿到 agentHashId，否则后续下发流水线会失败
func (m *BuildNodeManager) validateBindable(status *bkci.AgentNodeStatus) error {
	switch reason := m.checkSelectable(status.AgentNode); reason {
	case "":
	case UnselectableReasonOffline:
		return ErrBuildNodeOffline
	case UnselectableReasonOSUnsupported:
		return ErrBuildNodeOSUnsupported
	default:
		return errors.Errorf("build node unselectable: %s", reason)
	}
	if !status.CanUse {
		return ErrBuildNodeNoUsePerm
	}
	if status.AgentHashID == "" {
		return errors.New("bkci returned empty agentHashId")
	}
	return nil
}

// resolveDisplayName 节点别名为空时用主机名兜底，保证页面和操作记录里总有可读的名称
func (m *BuildNodeManager) resolveDisplayName(n bkci.AgentNode) string {
	return lo.CoalesceOrEmpty(n.DisplayName, n.Hostname)
}
