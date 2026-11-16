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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

var errRestartScopeConflict = bkerrs.Define(
	bkerrs.ErrCodeInvalidArgument,
	"EXTENSION_DEVMODE_RESTART_SCOPE_CONFLICT",
	"exactly one of envs, envTypes or instances is required",
)

// RestartURIInput DevMode 进程重启 URI 路径参数
type RestartURIInput struct {
	// AppID 应用 ID
	AppID string `uri:"appID" binding:"required"`
}

// RestartInput DevMode 进程重启请求体，envs / envTypes / instances 必须且只能指定其中一个
type RestartInput struct {
	// Envs 按环境名称重启（各环境下所有 Running 实例）
	Envs *RestartEnvsTarget `json:"envs"`
	// EnvTypes 按环境类型重启（匹配环境下所有 Running 实例）
	EnvTypes *RestartEnvTypesTarget `json:"envTypes"`
	// Instances 重启指定环境下的指定实例
	Instances *RestartInstancesTarget `json:"instances"`
}

// RestartEnvsTarget 按环境名称重启的目标
type RestartEnvsTarget struct {
	// Names 环境名称列表
	Names []string `json:"names" binding:"required,min=1,unique,dive,uri_slug"`
}

// RestartEnvTypesTarget 按环境类型重启的目标
type RestartEnvTypesTarget struct {
	// Types 环境类型列表，不允许 production
	Types []string `json:"types" binding:"required,min=1,unique,dive,oneof=development test staging"`
}

// RestartInstancesTarget 指定实例的重启目标
type RestartInstancesTarget struct {
	// EnvName 环境名称
	EnvName string `json:"envName" binding:"required,uri_slug"`
	// InstanceIDs 实例 ID 列表，须属于 EnvName 环境
	InstanceIDs []string `json:"instanceIDs" binding:"required,min=1,unique,dive,required"`
}

// Validate 校验 envs / envTypes / instances 必须且只能指定其中一个，必须在 Bind 之后单独调用
func (i *RestartInput) Validate() error {
	specified := lo.CountBy([]bool{i.Envs != nil, i.EnvTypes != nil, i.Instances != nil}, func(ok bool) bool {
		return ok
	})
	if specified != 1 {
		return errRestartScopeConflict
	}
	return nil
}

// RestartOutput DevMode 进程重启提交响应
type RestartOutput struct {
	Data *RestartData `json:"data"`
}

// RestartData DevMode 进程重启提交结果。请求已受理，实例执行结果需通过发布记录查询。
type RestartData struct {
	// RequestedAt 服务端受理时间，用于按条件轮询本次提交产生的记录
	RequestedAt time.Time `json:"requestedAt"`
	// InstanceCount 本次受理的实例数
	InstanceCount int `json:"instanceCount"`
	// RecordIDs 本次创建的发布记录 ID，用于精确轮询执行结果
	RecordIDs []string `json:"recordIDs"`
	// Skipped 被跳过的环境（仅按环境类型指定时可能出现）
	Skipped []*RestartSkippedEnv `json:"skipped"`
}

// RestartSkippedEnv 被跳过的环境
type RestartSkippedEnv struct {
	// EnvName 环境名称
	EnvName string `json:"envName"`
	// EnvType 环境类型
	EnvType string `json:"envType"`
	// Reason 跳过原因
	Reason string `json:"reason"`
}
