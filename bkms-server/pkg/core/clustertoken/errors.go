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

package clustertoken

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

var (
	// ErrTokenNotFound Token 不存在，或不属于指定空间
	ErrTokenNotFound = errors.New("cluster token not found")
	// ErrTokenExists 空间内已存在相同的 Token
	ErrTokenExists = errors.New("cluster token already exists")
	// ErrTokenInvalid Token 格式不合法
	ErrTokenInvalid = errors.New("invalid cluster token")
	// ErrTokenRefInvalid 引用 Token 的参数不合法：tokenID 与 token 必须且只能填一个
	ErrTokenRefInvalid = errors.New("exactly one of tokenID and token is required")
	// ErrTokenInUse Token 仍被集群使用，不能删除
	ErrTokenInUse = errors.New("cluster token is still used by clusters")
	// ErrDefaultTokenConflict 并发设置默认 Token 冲突，重试即可
	ErrDefaultTokenConflict = errors.New("default cluster token was changed concurrently, please retry")

	// ErrWorkspaceClusterNotFound 集群未接入空间
	ErrWorkspaceClusterNotFound = errors.New("workspace cluster not found")
	// ErrWorkspaceClusterExists 集群已接入空间
	ErrWorkspaceClusterExists = errors.New("cluster is already accessed by the workspace")
	// ErrClusterNotInProject 集群不属于空间关联的 BCS 项目
	ErrClusterNotInProject = errors.New("cluster does not belong to the workspace's bcs project")
	// ErrClusterInUse 集群仍被环境或北极星集群绑定使用，不能移除
	ErrClusterInUse = errors.New("cluster is still in use")
)

// clusterNotAccessedError 开关打开时使用了未接入空间的集群。
// 访问集群 API 的调用方会把它包成内部错误，响应输出时按 APIDetailError 还原为 CLUSTER_NOT_ACCESSED。
type clusterNotAccessedError struct{}

func (clusterNotAccessedError) Error() string {
	return "cluster is not accessed by the workspace"
}

func (clusterNotAccessedError) APIErrorDetail() (bkerrs.ErrCode, bkerrs.ErrDetailCode, string) {
	return bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodeClusterNotAccessed, "cluster"
}

// ErrClusterNotAccessed 开关打开时使用了未接入空间的集群
var ErrClusterNotAccessed error = clusterNotAccessedError{}

var _ bkerrs.APIDetailError = clusterNotAccessedError{}

// ClusterRef 错误详情中引用的集群
type ClusterRef struct {
	ClusterID   string `json:"clusterID"`
	ClusterName string `json:"clusterName"`
}

// TokenInUseError Token 仍被集群使用；errors.Is 可匹配 ErrTokenInUse
type TokenInUseError struct {
	// Clusters 使用该 Token 的集群
	Clusters []ClusterRef
}

func (e *TokenInUseError) Error() string {
	return fmt.Sprintf("%s: used by %d clusters", ErrTokenInUse, len(e.Clusters))
}

func (e *TokenInUseError) Unwrap() error { return ErrTokenInUse }

// Extras 作为错误详情的 extras 返回给前端：clusters 为 [{"clusterID": ..., "clusterName": ...}] 形式的 JSON 数组
func (e *TokenInUseError) Extras() map[string]string {
	return map[string]string{"clusters": jsonArray(e.Clusters)}
}

// EnvRef 错误详情中引用的环境
type EnvRef struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// ClusterInUseError 集群仍被使用；errors.Is 可匹配 ErrClusterInUse
type ClusterInUseError struct {
	// Envs 使用该集群的环境（含特性环境）
	Envs []EnvRef
	// PolarisBound 集群是否绑定了北极星接入点
	PolarisBound bool
}

func (e *ClusterInUseError) Error() string {
	return fmt.Sprintf("%s: used by %d envs, polaris bound: %t", ErrClusterInUse, len(e.Envs), e.PolarisBound)
}

func (e *ClusterInUseError) Unwrap() error { return ErrClusterInUse }

// Extras 作为错误详情的 extras 返回给前端：envs 为 [{"name": ..., "displayName": ...}] 形式的 JSON 数组，
// polarisBound 为 "true" / "false"
func (e *ClusterInUseError) Extras() map[string]string {
	return map[string]string{
		"envs":         jsonArray(e.Envs),
		"polarisBound": strconv.FormatBool(e.PolarisBound),
	}
}

func jsonArray[T any](items []T) string {
	if items == nil {
		items = []T{}
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// extrasCarrier 需要在错误详情中附带 extras 的领域错误
type extrasCarrier interface {
	Extras() map[string]string
}

// apiErrorRule 一类领域错误对应的 API 错误码与详情码，详情码为空表示不附带详情。
type apiErrorRule struct {
	target error
	code   bkerrs.ErrCode
	detail bkerrs.ErrDetailCode
}

var apiErrorRules = []apiErrorRule{
	{ErrTokenNotFound, bkerrs.ErrCodeNotFound, ""},
	{ErrWorkspaceClusterNotFound, bkerrs.ErrCodeNotFound, ""},
	{ErrTokenExists, bkerrs.ErrCodeAlreadyExists, ""},
	{ErrWorkspaceClusterExists, bkerrs.ErrCodeAlreadyExists, ""},
	{ErrTokenInvalid, bkerrs.ErrCodeInvalidArgument, ""},
	{ErrTokenRefInvalid, bkerrs.ErrCodeInvalidArgument, ""},
	{ErrClusterNotInProject, bkerrs.ErrCodeInvalidArgument, ""},
	{ErrDefaultTokenConflict, bkerrs.ErrCodeAborted, ""},
	{ErrTokenInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodeClusterTokenInUse},
	{ErrClusterInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodeWorkspaceClusterInUse},
}

// APIError 把集群 Token 相关的领域错误包装为 API 错误：按错误类型选择错误码，
// 前端需要识别的错误附带对应的详情码（错误实现了 extrasCarrier 时一并附带 extras）；其余错误按服务端内部错误处理
func APIError(err error, msg string) error {
	for _, rule := range apiErrorRules {
		if !errors.Is(err, rule.target) {
			continue
		}
		wrapped := bkerrs.Wrap(err, rule.code, msg)
		if rule.detail == "" {
			return wrapped
		}
		opts := []bkerrs.Option{bkerrs.WithSystem(bkerrs.SystemName), bkerrs.WithModule("cluster")}
		var carrier extrasCarrier
		if errors.As(err, &carrier) {
			opts = append(opts, bkerrs.WithExtras(carrier.Extras()))
		}
		return wrapped.SetDetails(bkerrs.NewDetail(rule.detail, wrapped.Error(), opts...))
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, msg)
}
