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

package accesspoint

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/installtask"
)

var (
	// ErrAccessPointNotFound 接入点不存在，或不属于指定空间
	ErrAccessPointNotFound = errors.New("polaris access point not found")
	// ErrAccessPointNameExists 空间内已存在同名接入点
	ErrAccessPointNameExists = errors.New("polaris access point name already exists")
	// ErrBindingNotFound 集群绑定记录不存在
	ErrBindingNotFound = errors.New("polaris cluster binding not found")
	// ErrBindingExists 空间内该集群已有绑定记录
	ErrBindingExists = errors.New("polaris cluster binding already exists")

	// ErrAccessPointInvalid 接入点参数不合法：地址格式错误、数量越界等
	ErrAccessPointInvalid = errors.New("invalid polaris access point")
	// ErrConnectivityFailed 保存接入点前重跑的连通测试未通过：网络不通或 Token 无效
	ErrConnectivityFailed = errors.New("polaris access point connectivity test failed")
	// ErrAccessPointUnreachable 实时读取自建北极星（如命名空间列表）失败
	ErrAccessPointUnreachable = errors.New("polaris access point is unreachable")
	// ErrAccessPointInUse 接入点仍绑定着集群，或仍有北极星配置固定使用它
	ErrAccessPointInUse = errors.New("polaris access point is still in use")
	// ErrClusterInUse 集群上有环境被北极星配置使用，不能解绑或换绑
	ErrClusterInUse = errors.New("cluster is used by polaris configs")
	// ErrClusterNotBound 集群未绑定自建接入点
	ErrClusterNotBound = errors.New("cluster is not bound to any polaris access point")
)

// apiErrorRule 一类领域错误对应的 API 错误码与详情码，详情码为空表示不附带详情。
type apiErrorRule struct {
	target error
	code   bkerrs.ErrCode
	detail bkerrs.ErrDetailCode
}

// AccessPointInUseError 接入点仍被使用，不能删除；errors.Is 可匹配 ErrAccessPointInUse
type AccessPointInUseError struct {
	// ClusterIDs 绑定到该接入点的集群
	ClusterIDs []string
	// Configs 固定使用该接入点的北极星配置
	Configs []ConfigUsage
}

func (e *AccessPointInUseError) Error() string {
	return fmt.Sprintf(
		"%s: bound to %d clusters, used by %d polaris configs",
		ErrAccessPointInUse, len(e.ClusterIDs), len(e.Configs),
	)
}

func (e *AccessPointInUseError) Unwrap() error {
	return ErrAccessPointInUse
}

// Extras 作为错误详情的 extras 返回给前端：clusterIDs 为集群 ID 的 JSON 数组，
// configs 为 [{"appID": ..., "configName": ...}] 形式的 JSON 数组
func (e *AccessPointInUseError) Extras() map[string]string {
	return map[string]string{
		"clusterIDs": jsonArray(e.ClusterIDs),
		"configs":    jsonArray(e.Configs),
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

// TODO(集群绑定子需求): POLARIS_CLUSTER_IN_USE 需要在 extras 中列出占用的应用与配置，
// 换绑时参照 AccessPointInUseError 返回实现了 extrasCarrier 的错误
var apiErrorRules = []apiErrorRule{
	{ErrAccessPointNotFound, bkerrs.ErrCodeNotFound, ""},
	{installtask.ErrNotFound, bkerrs.ErrCodeNotFound, ""},
	{installtask.ErrRunning, bkerrs.ErrCodeAborted, ""},
	{ErrAccessPointNameExists, bkerrs.ErrCodeAlreadyExists, ""},
	{ErrAccessPointInvalid, bkerrs.ErrCodeInvalidArgument, ""},
	{ErrConnectivityFailed, bkerrs.ErrCodeInvalidRequest, ""},
	{ErrAccessPointUnreachable, bkerrs.ErrCodeInvalidRequest, ""},
	{ErrClusterNotBound, bkerrs.ErrCodeInvalidRequest, ""},
	{ErrAccessPointInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodePolarisAccessPointInUse},
	{ErrClusterInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodePolarisClusterInUse},
}

// APIError 把接入点相关的领域错误包装为 API 错误：按错误类型选择错误码，
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
		opts := []bkerrs.Option{bkerrs.WithSystem(bkerrs.SystemName), bkerrs.WithModule("polaris")}
		var carrier extrasCarrier
		if errors.As(err, &carrier) {
			opts = append(opts, bkerrs.WithExtras(carrier.Extras()))
		}
		return wrapped.SetDetails(bkerrs.NewDetail(rule.detail, wrapped.Error(), opts...))
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, msg)
}
