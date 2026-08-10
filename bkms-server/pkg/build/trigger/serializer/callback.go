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

import "time"

// CallbackCredentialHeader 回调凭证请求头。
//
// 每个应用独享一个凭证，由蓝盾凭证管理下发并在触发专用流水线初始化时注入脚本环境。
// 该凭证不会出现在任何接口的响应体中
const CallbackCredentialHeader = "X-Bkms-Build-Trigger-Token" // #nosec G101

// CallbackEventInput is the JSON body posted by the trigger-only BKCI pipeline.
//
// 各字段与蓝盾流水线变量的映射关系见 design_notes/build_trigger_contract.md，
// 字段名变更必须同步流水线模板中的回调脚本
type CallbackEventInput struct {
	// 触发策略 ID，用于定位策略
	PolicyID string `json:"policyID" binding:"required,min=1,max=63"`
	// 事件类型，本期仅 push
	Event string `json:"event" binding:"required,oneof=push"`
	// 推送的分支名，构建时使用该分支而非构建配置中的默认分支
	Branch string `json:"branch" binding:"required,max=255"`
	// 本次推送的 HEAD commit 哈希
	CommitID string `json:"commitID" binding:"required,max=64"`
	// commit 作者，用于审计
	CommitAuthor string `json:"commitAuthor" binding:"max=255"`
	// 事件发生时间
	EventTime time.Time `json:"eventTime"`
}

// CallbackResultOutputObj is the JSON representation of a callback processing result.
//
// 三态均以 HTTP 200 返回，由 Result 区分，使流水线侧脚本可从响应体判断处理结果并留痕
type CallbackResultOutputObj struct {
	// 处理结果：built 已发起构建，skipped 已跳过，failed 触发失败
	Result string `json:"result"`
	// 结果为 built 时的构建号，其余为空
	BuildID string `json:"buildID"`
	// 跳过或失败原因，结果为 built 时为空
	Reason string `json:"reason"`
}

// CallbackOutput is the JSON response for the build trigger callback.
type CallbackOutput struct {
	// 回调处理结果
	Data *CallbackResultOutputObj `json:"data"`
}
