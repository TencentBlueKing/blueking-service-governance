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

package serializer

import (
	"time"

	"github.com/samber/lo"

	bkciproject "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
)

// BindBuildNodeInput 关联或更换构建机的请求体
type BindBuildNodeInput struct {
	NodeHashID string `json:"nodeHashID" binding:"required,max=64"`
}

// BuildNodeCandidateOutput 候选构建机
type BuildNodeCandidateOutput struct {
	// 蓝盾节点 HashId，只作为选项的值，页面不展示
	NodeHashID  string `json:"nodeHashID"`
	DisplayName string `json:"displayName"`
	IP          string `json:"ip"`
	OSName      string `json:"osName"`
	Online      bool   `json:"online"`
	Selectable  bool   `json:"selectable"`
	// 不可关联的原因：offline 离线；osUnsupported 非 Linux
	UnselectableReason string `json:"unselectableReason" enums:"offline,osUnsupported"`
}

// NewBuildNodeCandidateOutput 从候选构建机构造输出
func NewBuildNodeCandidateOutput(c bkciproject.BuildNodeCandidate) *BuildNodeCandidateOutput {
	return &BuildNodeCandidateOutput{
		NodeHashID:         c.NodeHashID,
		DisplayName:        c.DisplayName,
		IP:                 c.IP,
		OSName:             c.OSName,
		Online:             c.Online,
		Selectable:         c.UnselectableReason == "",
		UnselectableReason: string(c.UnselectableReason),
	}
}

// ListBuildNodeCandidatesOutput 候选构建机列表的响应
type ListBuildNodeCandidatesOutput struct {
	Data []*BuildNodeCandidateOutput `json:"data"`
}

// BuildNodeBindingOutput 已关联的构建机
type BuildNodeBindingOutput struct {
	// 蓝盾节点 HashId，用于更换时回显当前选项，页面不展示
	NodeHashID  string `json:"nodeHashID"`
	DisplayName string `json:"displayName"`
	IP          string `json:"ip"`
	OSName      string `json:"osName"`
	// 实时状态：online 在线；offline 离线；notFound 已从蓝盾删除；unknown 查询失败
	State     string    `json:"state" enums:"online,offline,notFound,unknown"`
	Updater   string    `json:"updater"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BuildNodeBindingStateOutput 工作空间的构建机关联现状
type BuildNodeBindingStateOutput struct {
	// 工作空间的蓝盾项目 Code，为空表示尚未初始化蓝盾项目
	ProjectCode string `json:"projectCode"`
	// 当前用户能否关联、更换或解除构建机
	Editable bool `json:"editable"`
	// 当前关联的构建机，未关联时为 null，表示使用公共构建机
	Binding *BuildNodeBindingOutput `json:"binding"`
}

// NewBuildNodeBindingStateOutput 从关联现状构造输出
func NewBuildNodeBindingStateOutput(
	state *bkciproject.BuildNodeBindingState, editable bool,
) BuildNodeBindingStateOutput {
	out := BuildNodeBindingStateOutput{ProjectCode: state.ProjectCode, Editable: editable}
	if b := state.Binding; b != nil {
		out.Binding = &BuildNodeBindingOutput{
			NodeHashID:  b.NodeHashID,
			DisplayName: b.DisplayName,
			IP:          b.IP,
			OSName:      b.OSName,
			State:       string(state.NodeState),
			Updater:     b.Updater,
			UpdatedAt:   b.UpdatedAt,
		}
	}
	return out
}

// GetBuildNodeBindingOutput 构建机关联现状的响应
type GetBuildNodeBindingOutput struct {
	Data BuildNodeBindingStateOutput `json:"data"`
}

// BuildNodeAuditSnapshot 写入操作记录的构建机快照，只保留定位构建机所需的字段
type BuildNodeAuditSnapshot struct {
	NodeHashID  string `json:"nodeHashID"`
	IP          string `json:"ip"`
	DisplayName string `json:"displayName"`
}

// NewBuildNodeAuditSnapshot 从关联记录构造审计快照，关联为空时返回 nil
func NewBuildNodeAuditSnapshot(b *bkciproject.BuildNodeBinding) *BuildNodeAuditSnapshot {
	if b == nil {
		return nil
	}
	return lo.ToPtr(BuildNodeAuditSnapshot{NodeHashID: b.NodeHashID, IP: b.IP, DisplayName: b.DisplayName})
}
