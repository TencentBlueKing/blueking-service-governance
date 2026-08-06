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

package redis

import (
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// compile-time check
var _ types.ProvisionParams = (*CreateParams)(nil)

// CreateParams 创建 Redis 服务实例所需的业务参数
type CreateParams struct {
	// BkBizID 业务 ID
	BkBizID int `mapstructure:"bkBizID" validate:"required"`
	// BkCloudID 云区域 ID
	BkCloudID int `mapstructure:"bkCloudID"`
	// DBAppAbbr 业务英文缩写
	DBAppAbbr string `mapstructure:"dbAppAbbr" validate:"required"`
	// ClusterType 集群类型
	ClusterType dbm.ClusterType `mapstructure:"clusterType" validate:"required"`
	// ClusterName 集群名称
	ClusterName string `mapstructure:"clusterName" validate:"required"`
	// ClusterAlias 集群别名
	ClusterAlias string `mapstructure:"clusterAlias"`
	// DBVersion 版本号（如 Redis-6）
	DBVersion string `mapstructure:"dbVersion" validate:"required"`

	// --- 集群部署专用 ---
	// ProxyPort 集群接入层端口
	ProxyPort int `mapstructure:"proxyPort"`
	// ClusterShardNum 集群分片数
	ClusterShardNum int `mapstructure:"clusterShardNum"`

	// --- 主从部署专用 ---
	// Port 集群起始端口
	Port int `mapstructure:"port"`
	// RedisPwd Redis 访问密码
	RedisPwd string `mapstructure:"redisPwd"`

	// --- 通用 ---
	// IPSource 主机来源，默认 resource_pool
	IPSource string `mapstructure:"ipSource"`
	// DisasterToleranceLevel 容灾级别
	DisasterToleranceLevel string `mapstructure:"disasterToleranceLevel"`
	// ResourceSpec 资源池申请规格
	ResourceSpec *dbm.ResourceSpec `mapstructure:"resourceSpec"`
}

// Validate 校验创建参数
func (p *CreateParams) Validate() error {
	return validate.Struct(p)
}

// ToTicketType 根据集群类型推断创建工单类型
func (p *CreateParams) ToTicketType() dbm.TicketType {
	if p.ClusterType == dbm.ClusterTypeRedisInstance {
		return dbm.TicketTypeRedisInsApply
	}
	return dbm.TicketTypeRedisClusterApply
}
