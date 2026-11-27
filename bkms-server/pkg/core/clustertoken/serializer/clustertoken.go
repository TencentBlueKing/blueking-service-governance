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

// Package serializer 集群 Token 与集群接入相关 API 的请求与响应结构。
//
// 这里只用 binding 校验必填；Token 格式、重复、集群归属等校验由领域逻辑完成
package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/clustertoken"
)

// WorkspaceURIInput 空间路径参数
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// TokenURIInput Token 路径参数
type TokenURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 集群 Token ID
	TokenID string `uri:"tokenID" binding:"required"`
}

// ListClustersQueryInput 已接入集群列表的查询参数
type ListClustersQueryInput struct {
	// 只返回使用该 Token 的集群；为空时返回空间内全部已接入集群
	TokenID string `form:"tokenID"`
}

// ClusterURIInput 集群路径参数
type ClusterURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// BCS 集群 ID
	ClusterID string `uri:"clusterID" binding:"required"`
}

// TokenInput 新建或更换 Token 的请求
type TokenInput struct {
	// Token 明文，只能包含可见 ASCII 字符，最长 1024
	Token string `json:"token" binding:"required"`
}

// TokenRefInput 引用 Token：tokenID 为已有 Token，token 为新 Token，必须且只能填一个；
// 新 Token 与空间内已有 Token 相同时复用已有 Token
type TokenRefInput struct {
	// 已有 Token 的 ID
	TokenID string `json:"tokenID"`
	// 新 Token 的明文
	Token string `json:"token"`
}

// ToRef 转换为领域参数
func (in TokenRefInput) ToRef() clustertoken.TokenRef {
	return clustertoken.TokenRef{TokenID: in.TokenID, Token: in.Token}
}

// WorkspaceClusterCreateInput 接入集群的请求
type WorkspaceClusterCreateInput struct {
	TokenRefInput
	// 待接入的集群 ID，须属于空间关联的 BCS 项目
	ClusterIDs []string `json:"clusterIDs" binding:"required,min=1"`
}

// ClusterTokenOutput 集群 Token，不返回明文
type ClusterTokenOutput struct {
	// Token ID
	ID string `json:"id"`
	// 脱敏后的 Token
	Masked string `json:"masked"`
	// 是否为空间的默认 Token
	IsDefault bool `json:"isDefault"`
	// 使用该 Token 的集群数量
	ClusterCount int `json:"clusterCount"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 最近配置人（新建或更换 Token 值）
	Updater string `json:"updater"`
	// 最近配置时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewClusterTokenOutput 由集群 Token 构造输出
func NewClusterTokenOutput(t *clustertoken.ClusterToken, clusterCount int) *ClusterTokenOutput {
	return &ClusterTokenOutput{
		ID:           t.ID.Hex(),
		Masked:       t.Masked,
		IsDefault:    t.IsDefault,
		ClusterCount: clusterCount,
		Creator:      t.Creator,
		CreatedAt:    t.CreatedAt,
		Updater:      t.Updater,
		UpdatedAt:    t.UpdatedAt,
	}
}

// ClusterCandidateOutput 可接入的集群
type ClusterCandidateOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 集群名称
	ClusterName string `json:"clusterName"`
	// 集群类型，如 virtual（虚拟）、single（独立）
	Type string `json:"type"`
	// 集群环境，如 prod
	Environment string `json:"environment"`
	// 集群状态，如 RUNNING
	Status string `json:"status"`
	// 是否为共享集群
	IsShared bool `json:"isShared"`
	// 是否已接入空间
	Accessed bool `json:"accessed"`
}

// NewClusterCandidateOutput 由候选集群构造输出
func NewClusterCandidateOutput(c *clustertoken.Candidate) *ClusterCandidateOutput {
	return &ClusterCandidateOutput{
		ClusterID:   c.ID,
		ClusterName: c.Name,
		Type:        c.Type,
		Environment: c.Environment,
		Status:      c.Status,
		IsShared:    c.IsShared,
		Accessed:    c.Accessed,
	}
}

// WorkspaceClusterTokenOutput 接入集群使用的 Token
type WorkspaceClusterTokenOutput struct {
	// Token ID
	ID string `json:"id"`
	// 脱敏后的 Token
	Masked string `json:"masked"`
	// 是否为空间的默认 Token
	IsDefault bool `json:"isDefault"`
}

// ClusterCheckOutput 集群权限检测结果
type ClusterCheckOutput struct {
	// 检测状态；权限检测方式确定前固定为 unchecked（未检测）
	Status clustertoken.CheckStatus `json:"status" enums:"unchecked"`
}

// WorkspaceClusterOutput 空间已接入的集群
type WorkspaceClusterOutput struct {
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 接入时的集群名称
	ClusterName string `json:"clusterName"`
	// 是否为共享集群
	IsShared bool `json:"isShared"`
	// 使用的 Token；Token 记录缺失时为 null
	Token *WorkspaceClusterTokenOutput `json:"token"`
	// 使用该集群的环境数量（含特性环境）
	EnvCount int `json:"envCount"`
	// 权限检测结果
	Check ClusterCheckOutput `json:"check"`
	// 接入人
	Creator string `json:"creator"`
	// 接入时间
	CreatedAt time.Time `json:"createdAt"`
	// 最近修改人
	Updater string `json:"updater"`
	// 最近修改时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewWorkspaceClusterOutput 由接入记录构造输出，token 为 nil 表示 Token 记录缺失
func NewWorkspaceClusterOutput(
	a *clustertoken.WorkspaceCluster, token *clustertoken.ClusterToken, envCount int,
) *WorkspaceClusterOutput {
	out := &WorkspaceClusterOutput{
		ClusterID:   a.ClusterID,
		ClusterName: a.ClusterName,
		IsShared:    a.IsShared,
		EnvCount:    envCount,
		Check:       ClusterCheckOutput{Status: clustertoken.CheckStatusUnchecked},
		Creator:     a.Creator,
		CreatedAt:   a.CreatedAt,
		Updater:     a.Updater,
		UpdatedAt:   a.UpdatedAt,
	}
	if token != nil {
		out.Token = &WorkspaceClusterTokenOutput{ID: token.ID.Hex(), Masked: token.Masked, IsDefault: token.IsDefault}
	}
	return out
}

// ClusterTokenListResponse Token 列表的响应
type ClusterTokenListResponse struct {
	Data []*ClusterTokenOutput `json:"data"`
}

// ClusterTokenResponse 单个 Token 的响应
type ClusterTokenResponse struct {
	Data *ClusterTokenOutput `json:"data"`
}

// ClusterCandidateListResponse 候选集群的响应
type ClusterCandidateListResponse struct {
	Data []*ClusterCandidateOutput `json:"data"`
}

// WorkspaceClusterListResponse 已接入集群列表的响应
type WorkspaceClusterListResponse struct {
	Data []*WorkspaceClusterOutput `json:"data"`
}

// WorkspaceClusterResponse 单个已接入集群的响应
type WorkspaceClusterResponse struct {
	Data *WorkspaceClusterOutput `json:"data"`
}

// ClusterTokenEmptyResponse 无数据的响应
type ClusterTokenEmptyResponse struct{}
