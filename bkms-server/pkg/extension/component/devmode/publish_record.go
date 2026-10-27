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

package devmode

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// RecordStatus 开发模式发布记录状态（publish / restart 共用）
type RecordStatus string

const (
	// RecordStatusSuccess 执行成功
	RecordStatusSuccess RecordStatus = "success"
	// RecordStatusFailed 执行失败
	RecordStatusFailed RecordStatus = "failed"
	// RecordStatusPending 等待执行
	RecordStatusPending RecordStatus = "pending"
	// RecordStatusRunning 执行中
	RecordStatusRunning RecordStatus = "running"
)

// RecordAction 开发模式发布记录的操作类型
type RecordAction string

const (
	// RecordActionPublish 发布二进制并热重启
	RecordActionPublish RecordAction = "publish"
	// RecordActionRestart 仅重启业务进程（不替换二进制）
	RecordActionRestart RecordAction = "restart"
)

// PublishRecord 开发模式发布记录（每个实例/pod 一条）
type PublishRecord struct {
	// ID 发布记录 ID（唯一）
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`

	// AppID 应用 ID
	AppID string `bson:"appID"`
	// EnvName 环境名称
	EnvName string `bson:"envName"`

	// Action 操作类型：publish / restart，历史数据为空，按 publish 处理
	Action RecordAction `bson:"action,omitempty"`
	// Instance 目标实例（pod）名称
	Instance string `bson:"instance"`
	// BinaryName 发布的二进制名称
	BinaryName string `bson:"binaryName"`
	// FileSize 文件大小（字节）
	FileSize int64 `bson:"fileSize"`
	// MD5 文件 MD5 值
	MD5 string `bson:"md5"`
	// Status 发布状态：pending / running / success / failed
	Status RecordStatus `bson:"status"`
	// Message 失败原因等附加信息
	Message string `bson:"message"`
	// Operator 操作人
	Operator string `bson:"operator"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// GetAction 获取操作类型，历史数据未设置时按 publish 处理
func (r *PublishRecord) GetAction() RecordAction {
	if r.Action == "" {
		return RecordActionPublish
	}
	return r.Action
}
