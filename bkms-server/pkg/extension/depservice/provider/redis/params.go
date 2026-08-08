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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// compile-time check
var _ types.ProvisionParams = (*CreateParams)(nil)

// CreateParams 创建 Redis 服务实例所需的业务参数。
//
// 集群部署（Twemproxy*/Predixy*）与主从部署（RedisInstance）共用本结构，
// 但 DBM 工单 details 字段不同：主从模式会把 ClusterName/Databases 组装为 Infos。
type CreateParams struct {
	// BkBizID 业务 ID
	BkBizID int `mapstructure:"bkBizID" validate:"required"`
	// BkCloudID 云区域 ID
	BkCloudID int `mapstructure:"bkCloudID"`
	// DBAppAbbr 业务英文缩写
	DBAppAbbr string `mapstructure:"dbAppAbbr" validate:"required"`
	// ClusterType 集群类型
	ClusterType dbm.ClusterType `mapstructure:"clusterType" validate:"required"`
	// ClusterName 集群名称。
	// 集群模式：写入 REDIS_CLUSTER_APPLY.details.cluster_name；
	// 主从模式：写入 REDIS_INS_APPLY.details.infos[].cluster_name，并作为创建后回查依据。
	ClusterName string `mapstructure:"clusterName" validate:"required"`
	// ClusterAlias 集群别名（集群模式）
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
	// Databases DB 数量（REDIS_INS_APPLY.infos[].databases）
	Databases int `mapstructure:"databases"`
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

// Validate 校验创建参数（含按部署模式的必填字段）
func (p *CreateParams) Validate() error {
	if err := validate.Struct(p); err != nil {
		return err
	}
	switch {
	case p.ClusterType == dbm.ClusterTypeRedisInstance:
		if p.Port <= 0 {
			return errors.New("port is required for RedisInstance (master-slave) deploy")
		}
		if p.Databases <= 0 {
			return errors.New("databases is required for RedisInstance (master-slave) deploy")
		}
	case dbm.IsProxyClusterType(p.ClusterType):
		if p.ProxyPort <= 0 {
			return errors.New("proxyPort is required for cluster deploy")
		}
		if p.ClusterShardNum <= 0 {
			return errors.New("clusterShardNum is required for cluster deploy")
		}
	default:
		return errors.Errorf("unsupported clusterType: %s", p.ClusterType)
	}
	return nil
}

// ToTicketType 根据集群类型推断创建工单类型。
// 调用方应先 Validate；未知类型默认按集群工单处理。
func (p *CreateParams) ToTicketType() dbm.TicketType {
	switch p.ClusterType {
	case dbm.ClusterTypeRedisInstance:
		return dbm.TicketTypeRedisInsApply
	default:
		return dbm.TicketTypeRedisClusterApply
	}
}

// ToCreateRedisParams 组装 DBM CreateRedis 参数。
// 主从模式下将 ClusterName/Databases 映射为 Infos，供 REDIS_INS_APPLY 消费。
// 调用方应先 Validate，本方法按已通过校验的部署模式填充字段。
func (p *CreateParams) ToCreateRedisParams() *dbm.CreateRedisParams {
	params := &dbm.CreateRedisParams{
		BkBizID:                p.BkBizID,
		TicketType:             p.ToTicketType(),
		BkCloudID:              p.BkCloudID,
		DBAppAbbr:              p.DBAppAbbr,
		ClusterType:            p.ClusterType,
		DBVersion:              p.DBVersion,
		IPSource:               p.IPSource,
		ResourceSpec:           p.ResourceSpec,
		DisasterToleranceLevel: p.DisasterToleranceLevel,
	}
	switch {
	case p.ClusterType == dbm.ClusterTypeRedisInstance:
		params.Port = p.Port
		params.RedisPwd = p.RedisPwd
		params.Infos = []dbm.RedisInsInfo{{
			ClusterName: p.ClusterName,
			Databases:   p.Databases,
		}}
	case dbm.IsProxyClusterType(p.ClusterType):
		params.ClusterName = p.ClusterName
		params.ClusterAlias = p.ClusterAlias
		params.ProxyPort = p.ProxyPort
		params.ClusterShardNum = p.ClusterShardNum
	}
	return params
}
