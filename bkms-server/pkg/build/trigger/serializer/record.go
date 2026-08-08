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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
)

// ListTriggerRecordsQueryInput is the query input for listing trigger records.
type ListTriggerRecordsQueryInput struct {
	// 结果筛选：built 已构建，skipped 已跳过，failed 触发失败；留空表示不筛选
	Result string `form:"result" binding:"omitempty,oneof=built skipped failed"`
	// 分页参数：页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页参数：每页数量，支持 5/10/20/50/100
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// TriggerRecordOutputObj is the JSON representation of one trigger record.
type TriggerRecordOutputObj struct {
	// 归属策略 ID
	PolicyID string `json:"policyID"`
	// 归属应用 ID
	AppID string `json:"appID"`
	// 触发时间
	TriggeredAt time.Time `json:"triggeredAt"`
	// 事件类型
	Event string `json:"event"`
	// 分支名
	Branch string `json:"branch"`
	// commit 哈希
	CommitID string `json:"commitID"`
	// commit 作者
	CommitAuthor string `json:"commitAuthor"`
	// 处理结果：built 已构建，skipped 已跳过，failed 触发失败
	Result string `json:"result"`
	// 结果为 built 时关联的构建号，其余为空
	BuildID string `json:"buildID"`
	// 跳过或失败原因，结果为 built 时为空
	Reason string `json:"reason"`
}

// FromModel fills output fields from a trigger record model.
func (o *TriggerRecordOutputObj) FromModel(r trigger.Record) *TriggerRecordOutputObj {
	*o = TriggerRecordOutputObj{
		PolicyID:     r.PolicyID,
		AppID:        r.AppID,
		TriggeredAt:  r.TriggeredAt,
		Event:        string(r.Event),
		Branch:       r.Branch,
		CommitID:     r.CommitID,
		CommitAuthor: r.CommitAuthor,
		Result:       string(r.Result),
		BuildID:      r.BuildID,
		Reason:       r.Reason,
	}
	return o
}

// PaginatedTriggerRecordOutputObjs is the paginated trigger record payload.
type PaginatedTriggerRecordOutputObjs struct {
	// 记录总数，按当前筛选条件统计
	Count int64 `json:"count,string"`
	// 当前页结果，按触发时间倒序
	Results []*TriggerRecordOutputObj `json:"results"`
}

// ListTriggerRecordsOutput is the JSON response for listing trigger records.
type ListTriggerRecordsOutput struct {
	// 触发记录分页结果
	Data *PaginatedTriggerRecordOutputObjs `json:"data"`
}
