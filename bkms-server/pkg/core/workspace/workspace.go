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

// Package workspace 提供工作空间相关模型 & 功能
package workspace

import (
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

type State string

const (
	// StateReady 就绪, 由系统自动设置, 只有就绪状态下, 前端允许用户对空间下的资源进行操作, 如创建环境、创建应用等
	StateReady State = "Ready"
	// StateProcessing 处理中, 仅创建工作空间时由系统自动设置.
	StateProcessing State = "Processing"
	// StateDisabled 停用, 由用户指定
	StateDisabled State = "Disabled"
)

// Workspace 工作空间是 bkms 最顶层概念，包含环境，应用等资源
// 一般工作空间会与蓝盾、BCS、制品库、监控、日志项目相互绑定
type Workspace struct {
	// ID 工作空间唯一标识
	// 1-27 字符的空间 ID，由小写字母、数字、中划线组成，以小写字母开头
	ID string `bson:"id"`
	// DisplayName 展示用名称，一般为中文名（1-64 字符）
	DisplayName string `bson:"displayName"`
	// Description 描述信息（0-512 字符）
	Description string `bson:"description"`
	// ImageRegistryType 使用的镜像仓库类型
	ImageRegistryType registry.ImageRegistryType `bson:"imageRegistryType"`

	// BkSystems 关联的蓝鲸体系系统（如：监控、BCS）的信息
	BkSystems BkSystems `bson:"bkSystems"`

	// State 工作空间状态, 可选值["Ready", "Processing", "Disabled"]
	State State `bson:"state"`
	// Creator 创建人
	Creator string `bson:"creator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// Updater 更新人
	Updater string `bson:"updater"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// BkSystems 工作空间关联的蓝鲸体系服务（如：蓝盾、容器服务、监控等）的项目 ID
type BkSystems struct {
	// BkCIProjectID 蓝盾项目 ID
	BkCIProjectID string `bson:"bkCIProjectID"`
	// BkCIProjectUID 蓝盾项目 UID, 32 位字符串
	BkCIProjectUID string `bson:"bkCIProjectUID"`
	// BkRepoProjectID 蓝盾制品库项目 ID
	BkRepoProjectID string `bson:"bkRepoProjectID"`

	// BkBCSProjectID 蓝鲸容器服务（BCS）项目 ID
	BkBCSProjectID string `bson:"bkBcsProjectID"`
	// BkBCSProjectCode 蓝鲸容器服务（BCS）项目 Code, 可读唯一字符串，如：bkce
	BkBCSProjectCode string `bson:"bkBcsProjectCode"`

	// BkLogProjectID 蓝鲸日志平台项目 ID
	BkLogProjectID string `bson:"bkLogProjectID"`
	// BkMonitorProjectID 蓝鲸监控平台项目 ID
	BkMonitorProjectID string `bson:"bkMonitorProjectID"`

	// BkCCBizID bkcc 业务 ID
	BkCCBizID string `bson:"bkCCBizID"`
	// Level2BizID 二级业务 ID
	Level2BizID string `bson:"level2BizID"`
	// ObsProductID 运营产品 ID
	ObsProductID string `bson:"obsProductID"`
	// ObsProductName 运营产品名称
	ObsProductName string `bson:"obsProductName"`

	// IsBoundExistedBKCIProject 是否关联已有蓝盾项目, true 代表关联已有项目, false 代表新建项目
	IsBoundExistedBKCIProject bool `bson:"isBoundExistedBKCIProject"`
}

// ResolveBkMonitorProjectID 解析并校验 workspace 上的 bkMonitorProjectID。
func (ws *Workspace) ResolveBkMonitorProjectID() (int64, error) {
	if ws == nil {
		return 0, errors.New("workspace is nil")
	}
	rawID := ws.BkSystems.BkMonitorProjectID
	if rawID == "" {
		return 0, errors.New("bkMonitorProjectID is empty")
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "bkMonitorProjectID must be a valid int64")
	}
	return id, nil
}
