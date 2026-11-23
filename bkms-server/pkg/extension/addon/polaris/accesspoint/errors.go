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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
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
	// ErrConnectivityFailed 保存接入点前重跑的连通测试未通过：网络不通、开启了客户端鉴权或 Token 无效，
	ErrConnectivityFailed = errors.New("polaris access point connectivity test failed")
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

// TODO(北极星接入点管理、集群绑定子需求): IN_USE 两类详情需要在 extras 中列出占用的应用与配置，
// 实现删除与换绑时参照 bkerrs.WrapEnvClusterNamespaceOccupied 单独包装，不再走这里的通用规则
var apiErrorRules = []apiErrorRule{
	{ErrAccessPointNotFound, bkerrs.ErrCodeNotFound, ""},
	{ErrAccessPointNameExists, bkerrs.ErrCodeAlreadyExists, ""},
	{ErrAccessPointInvalid, bkerrs.ErrCodeInvalidArgument, ""},
	{ErrConnectivityFailed, bkerrs.ErrCodeInvalidRequest, ""},
	{ErrClusterNotBound, bkerrs.ErrCodeInvalidRequest, ""},
	{ErrAccessPointInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodePolarisAccessPointInUse},
	{ErrClusterInUse, bkerrs.ErrCodeInvalidRequest, bkerrs.ErrDetailCodePolarisClusterInUse},
}

// APIError 把接入点相关的领域错误包装为 API 错误：按错误类型选择错误码，
// 前端需要识别的错误附带对应的详情码；其余错误按服务端内部错误处理
func APIError(err error, msg string) error {
	for _, rule := range apiErrorRules {
		if !errors.Is(err, rule.target) {
			continue
		}
		wrapped := bkerrs.Wrap(err, rule.code, msg)
		if rule.detail == "" {
			return wrapped
		}
		return wrapped.SetDetails(bkerrs.NewDetail(
			rule.detail,
			wrapped.Error(),
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule("polaris"),
		))
	}
	return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, msg)
}
