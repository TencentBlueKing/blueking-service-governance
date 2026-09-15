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

	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
)

// PublishRecordURIInput DevMode Publish Record URI 路径参数
type PublishRecordURIInput struct {
	// AppID 应用 ID
	AppID string `uri:"appID" binding:"required"`
	// EnvName 环境名称
	EnvName string `uri:"envName" binding:"required"`
}

// CreatePublishRecordsInput DevMode 发布记录上报请求体
type CreatePublishRecordsInput struct {
	// BinaryName 发布的二进制名称
	BinaryName string `json:"binaryName" binding:"required"`
	// FileSize 文件大小（字节）
	FileSize int64 `json:"fileSize" binding:"gte=0"`
	// MD5 文件 MD5 值
	MD5 string `json:"md5" binding:"required"`
	// Results 逐实例发布结果
	Results []PublishInstanceResult `json:"results" binding:"required,min=1,dive"`
}

// PublishInstanceResult 单个实例的发布结果
type PublishInstanceResult struct {
	// Instance 目标实例（pod）名称
	Instance string `json:"instance" binding:"required"`
	// Status 发布状态：success / failed
	Status string `json:"status" binding:"required,oneof=success failed"`
	// Message 失败原因等附加信息
	Message string `json:"message"`
}

// CreatePublishRecordsOutput DevMode 发布记录上报成功响应
type CreatePublishRecordsOutput struct {
	Data *CreatePublishRecordsData `json:"data"`
}

// CreatePublishRecordsData DevMode 发布记录上报成功响应数据
type CreatePublishRecordsData struct {
	// RecordIDs 新建的发布记录 ID 列表
	RecordIDs []string `json:"recordIDs"`
}

// ListPublishRecordsInput DevMode 发布记录列表查询参数
type ListPublishRecordsInput struct {
	// Keyword 搜索关键字
	Keyword string `form:"keyword"`
	// Page 分页页码
	Page int64 `form:"page" binding:"required,gte=1"`
	// PageSize 分页大小
	PageSize int64 `form:"pageSize" binding:"required,oneof=1 5 10 20 50 100"`
}

// ListPublishRecordsOutput DevMode 发布记录列表成功响应
type ListPublishRecordsOutput struct {
	Data PaginatedPublishRecords `json:"data"`
}

// PaginatedPublishRecords 分页发布记录
type PaginatedPublishRecords struct {
	Count   int64                     `json:"count,string"`
	Results []*PublishRecordOutputObj `json:"results"`
}

// PublishRecordOutputObj 单条发布记录
type PublishRecordOutputObj struct {
	ID         string    `json:"id"`
	Instance   string    `json:"instance"`
	BinaryName string    `json:"binaryName"`
	FileSize   int64     `json:"fileSize"`
	MD5        string    `json:"md5"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Operator   string    `json:"operator"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// FromModel 将领域模型转换为输出对象
func (o *PublishRecordOutputObj) FromModel(record devmode.PublishRecord) *PublishRecordOutputObj {
	o.ID = record.ID.Hex()
	o.Instance = record.Instance
	o.BinaryName = record.BinaryName
	o.FileSize = record.FileSize
	o.MD5 = record.MD5
	o.Status = string(record.Status)
	o.Message = record.Message
	o.Operator = record.Operator
	o.CreatedAt = record.CreatedAt
	o.UpdatedAt = record.UpdatedAt
	return o
}
