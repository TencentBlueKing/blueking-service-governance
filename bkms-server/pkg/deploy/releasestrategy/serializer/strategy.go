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

// Package serializer 空间发布策略相关 API 的请求与响应结构
package serializer

import (
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	if err := v.RegisterValidation("batch_target", validateBatchTarget); err != nil {
		panic("failed to register batch_target validator: " + err.Error())
	}
}

// validateBatchTarget 校验批次目标值：正整数（≥ 1）或百分比（1% ~ 100%）
func validateBatchTarget(fl validator.FieldLevel) bool {
	_, err := releasestrategy.ParseBatchTarget(fl.Field().String())
	return err == nil
}

// WorkspaceURIInput binds a workspace-scoped path.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// StrategyURIInput binds a release strategy path.
type StrategyURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 发布策略 ID
	StrategyID string `uri:"strategyID" binding:"required,mongodb"`
}

// BatchInput 发布策略中的一个批次
type BatchInput struct {
	// 本批"累计更新到"的实例数：正整数（如 "1"）或百分比（如 "25%"），百分比范围 1% ~ 100%
	Target string `json:"target" binding:"required,batch_target"`
	// 本批完成后的暂停方式：none 不暂停 / duration 定时暂停 / permanent 永久暂停（需人工继续）
	PauseMode string `json:"pauseMode" binding:"required,oneof=none duration permanent"`
	// 定时暂停时长（秒），仅 pauseMode 为 duration 时必填且只能在此时填写，范围 60 ~ 86400
	PauseSeconds int `json:"pauseSeconds" binding:"required_if=PauseMode duration,excluded_unless=PauseMode duration,omitempty,min=60,max=86400"`
}

// ReleaseStrategyInput is the JSON body shared by creating and updating a release strategy.
type ReleaseStrategyInput struct {
	// 策略名称，1 ~ 32 个字符，同一空间内唯一
	Name string `json:"name" binding:"required,max=32"`
	// 策略描述，不超过 200 个字符
	Description string `json:"description" binding:"max=200"`
	// 批次列表，1 ~ 10 个；不校验跨批次递增
	Batches []BatchInput `json:"batches" binding:"required,min=1,max=10,dive"`
}

// ToModel converts the input into a release strategy model of the given workspace.
// 审计字段（创建人 / 更新人 / 时间）由调用方与 store 填写
func (i ReleaseStrategyInput) ToModel(workspaceID string) *releasestrategy.ReleaseStrategy {
	return &releasestrategy.ReleaseStrategy{
		WorkspaceID: workspaceID,
		Name:        i.Name,
		Description: i.Description,
		Batches: lo.Map(i.Batches, func(b BatchInput, _ int) releasestrategy.Batch {
			return releasestrategy.Batch{
				Target:       b.Target,
				PauseMode:    releasestrategy.PauseMode(b.PauseMode),
				PauseSeconds: b.PauseSeconds,
			}
		}),
	}
}

// BatchOutputObj is the JSON representation of one batch.
type BatchOutputObj struct {
	// 本批"累计更新到"的实例数：正整数或百分比
	Target string `json:"target"`
	// 本批完成后的暂停方式：none / duration / permanent
	PauseMode string `json:"pauseMode"`
	// 定时暂停时长（秒），pauseMode 不为 duration 时为 0
	PauseSeconds int `json:"pauseSeconds"`
}

// ReleaseStrategyOutputObj is the JSON representation of one release strategy.
type ReleaseStrategyOutputObj struct {
	// 策略 ID
	ID string `json:"id"`
	// 所属工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 策略名称
	Name string `json:"name"`
	// 策略描述
	Description string `json:"description"`
	// 批次列表
	Batches []BatchOutputObj `json:"batches"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 最近更新人
	Updater string `json:"updater"`
	// 最近更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a release strategy model.
func (o *ReleaseStrategyOutputObj) FromModel(s releasestrategy.ReleaseStrategy) *ReleaseStrategyOutputObj {
	*o = ReleaseStrategyOutputObj{
		ID:          s.ID.Hex(),
		WorkspaceID: s.WorkspaceID,
		Name:        s.Name,
		Description: s.Description,
		Batches: lo.Map(s.Batches, func(b releasestrategy.Batch, _ int) BatchOutputObj {
			return BatchOutputObj{
				Target:       b.Target,
				PauseMode:    string(b.PauseMode),
				PauseSeconds: b.PauseSeconds,
			}
		}),
		Creator:   s.Creator,
		CreatedAt: s.CreatedAt,
		Updater:   s.Updater,
		UpdatedAt: s.UpdatedAt,
	}
	return o
}

// ListReleaseStrategiesOutput is the JSON response for listing release strategies.
type ListReleaseStrategiesOutput struct {
	// 空间内全部发布策略，按创建时间升序，不分页
	Data []*ReleaseStrategyOutputObj `json:"data"`
}

// ReleaseStrategyOutput is the JSON response for APIs returning a single release strategy.
type ReleaseStrategyOutput struct {
	// 发布策略
	Data *ReleaseStrategyOutputObj `json:"data"`
}

// EmptyOutput is an empty successful response.
type EmptyOutput struct{}
