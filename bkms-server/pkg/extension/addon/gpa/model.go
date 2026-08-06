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

// Package gpa 定义了 GPA（GeneralPodAutoscaler）自动扩缩容配置相关的实体和方法。
package gpa

import (
	"time"
)

// ResourceName 扩缩容指标所基于的资源类型
type ResourceName string

const (
	// ResourceCPU 基于 CPU 使用率
	ResourceCPU ResourceName = "cpu"
	// ResourceMemory 基于内存使用率
	ResourceMemory ResourceName = "memory"
)

// GPAMetric 单条扩缩容指标，对应 GPA CRD 中 spec.metric.metrics[] 的一项。
type GPAMetric struct {
	// Resource 指标资源类型：cpu / memory
	Resource ResourceName `bson:"resource" validate:"required,oneof=cpu memory"`
	// AverageUtilization 平均使用率阈值（百分比），取值 1-100
	AverageUtilization int32 `bson:"averageUtilization" validate:"required,gte=1,lte=100"`
}

// GPATimeRange 单条定时扩缩容规则，对应 GPA CRD 中 spec.time.ranges[] 的一项。
// 当当前时间命中 Schedule 指定的时间段时，将副本数调整为 DesiredReplicas。
type GPATimeRange struct {
	// DesiredReplicas 命中时间段时的期望副本数（>= 1）
	DesiredReplicas int32 `bson:"desiredReplicas" validate:"required,gte=1"`
	// Schedule 标准 5 段 Crontab 表达式（分 时 日 月 周）。
	// 应为时间段（如 "* 2-3 * * *"），此处仅校验语法合法性。
	Schedule string `bson:"schedule" validate:"required,crontab"`
	// Enabled 是否启用该定时规则。仅启用的规则会被写入底层 K8s CR 的 spec.time.ranges，
	// 未启用的规则仅保留在 DB 配置中，不对工作负载生效。
	Enabled bool `bson:"enabled"`
	// Remark 备注说明，仅用于展示，不参与扩缩容逻辑，也不写入 CR。
	Remark string `bson:"remark" validate:"omitempty,max=256"`
}

// GPAConfig GPA 自动扩缩容配置实体，存储在独立的数据表中。
// 每个应用 + 环境唯一一份配置。
type GPAConfig struct {
	// Name 配置名称，由 GenerateName 按 "gpa-{appID}" 规则生成，同时作为 GPA CR 的 metadata.name
	Name string `bson:"name" validate:"required"`
	// AppID 所属应用 ID
	AppID string `bson:"appID" validate:"required"`
	// EnvName 生效环境名称，配置在应用 + 环境维度唯一
	EnvName string `bson:"envName" validate:"required"`

	// MinReplicas 最小副本数（本期强制 >= 1）
	MinReplicas int32 `bson:"minReplicas" validate:"required,gte=1"`
	// MaxReplicas 最大副本数（须 >= MinReplicas）
	MaxReplicas int32 `bson:"maxReplicas" validate:"required,gtefield=MinReplicas"`
	// Metrics 指标模式的扩缩容指标列表，最多 2 条（cpu、memory）。
	// 与 TimeRanges 二者可选，但至少配置其一（见 validateAtLeastOneMode）。
	Metrics []GPAMetric `bson:"metrics" validate:"omitempty,max=2,dive"`
	// TimeRanges 定时模式的扩缩容规则列表。
	// 与 Metrics 二者可选，但至少配置其一（见 validateAtLeastOneMode）。
	TimeRanges []GPATimeRange `bson:"timeRanges" validate:"omitempty,dive"`

	// ComputeByLimits GPA 级别的利用率计算基准开关。
	// true 时该 GPA 下所有 type=Utilization 的指标（cpu、memory）以 limits 为基准计算利用率
	//（利用率 = 实际使用量 / limits），对应下发 CR 时写入 annotation compute-by-limits="true"；
	// false（默认）时沿用 requests 作为基准
	ComputeByLimits bool `bson:"computeByLimits"`

	// Enabled 是否启用。false 时不下发/清理集群中的 GPA CR，DB 配置保留
	Enabled bool `bson:"enabled"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// GenerateName 生成配置名称（同时作为 GPA CR 的 metadata.name）。
// 固定为 "gpa-{appID}"：每应用 + 环境一份，CR 部署在各环境独立 namespace，跨环境同名不冲突。
// appID 由 app_id validator 约束（小写字母开头、仅小写字母/数字/中划线、长度 1~63），
// 故 "gpa-{appID}" 必然满足 K8s RFC 1123 命名要求。
func (c *GPAConfig) GenerateName() string {
	return "gpa-" + c.AppID
}
