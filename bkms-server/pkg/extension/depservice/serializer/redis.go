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

// Package serializer defines Gin input and output serializers for dependency service APIs.
package serializer

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/redis"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("redis_cluster_name", validateRedisClusterName)
	}
}

// redisClusterNameRegexp: 小写字母开头，仅小写字母/数字/连字符
var redisClusterNameRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func validateRedisClusterName(fl validator.FieldLevel) bool {
	return redisClusterNameRegexp.MatchString(fl.Field().String())
}

// -----------------------------------------------------------------------------
// Path / query inputs
// -----------------------------------------------------------------------------

// WorkspaceURIInput is the path input for workspace-scoped APIs.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// WorkspaceInstanceURIInput is the path input for instance-scoped APIs.
type WorkspaceInstanceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// Redis 实例 ID
	InstanceID string `uri:"instanceID" binding:"required"`
}

// ListRedisInstancesQueryInput is the query input for listing Redis instances.
type ListRedisInstancesQueryInput struct {
	// 实例状态过滤
	Status string `form:"status"`
	// 作用域类型过滤: workspace / envType / env
	ScopeType string `form:"scopeType"`
}

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

// CreateRedisInstanceInput is the JSON body for creating a Redis instance.
type CreateRedisInstanceInput struct {
	// 实例名称（同 workspace 下唯一）
	Name string `json:"name" binding:"required"`
	// 作用域类型: workspace / envType / env
	ScopeType string `json:"scopeType" binding:"required,oneof=workspace envType env"`
	// 作用域值；workspace 时为空，envType 为环境类型，env 为环境名
	ScopeValue string `json:"scopeValue"`
	// 描述
	Description string `json:"description"`

	// --- Redis / DBM 创建参数 ---
	// 业务 ID
	BkBizID int `json:"bkBizID" binding:"required"`
	// 云区域 ID
	BkCloudID int `json:"bkCloudID"`
	// 业务英文缩写
	DBAppAbbr string `json:"dbAppAbbr" binding:"required"`
	// 集群类型
	ClusterType string `json:"clusterType" binding:"required"`
	// 集群名称（小写字母开头，仅小写字母/数字/连字符）
	ClusterName string `json:"clusterName" binding:"required,redis_cluster_name"`
	// 集群别名（集群模式）
	ClusterAlias string `json:"clusterAlias"`
	// 版本号（如 Redis-6）
	DBVersion string `json:"dbVersion" binding:"required"`

	// 集群接入层端口
	ProxyPort int `json:"proxyPort"`
	// 集群分片数
	ClusterShardNum int `json:"clusterShardNum"`

	// 主从起始端口
	Port int `json:"port"`
	// DB 数量
	Databases int `json:"databases"`
	// Redis 访问密码
	RedisPwd string `json:"redisPwd"`

	// 主机来源
	IPSource string `json:"ipSource"`
	// 容灾级别
	DisasterToleranceLevel string `json:"disasterToleranceLevel"`
	// 资源池申请规格
	ResourceSpec *ResourceSpecInput `json:"resourceSpec"`
}

// ResourceSpecInput 资源池申请规格
type ResourceSpecInput struct {
	Proxy        *ResourceSpecItemInput `json:"proxy"`
	BackendGroup *ResourceSpecItemInput `json:"backendGroup"`
}

// ResourceSpecItemInput 单角色资源规格
type ResourceSpecItemInput struct {
	SpecID       int                `json:"specID"`
	Count        int                `json:"count"`
	LocationSpec *LocationSpecInput `json:"locationSpec"`
}

// LocationSpecInput 地域约束
type LocationSpecInput struct {
	City       string `json:"city"`
	SubZoneIDs []int  `json:"subZoneIDs"`
}

// ToCreateParams converts the HTTP input to redis.CreateParams.
func (in *CreateRedisInstanceInput) ToCreateParams() *redis.CreateParams {
	params := &redis.CreateParams{
		BkBizID:                in.BkBizID,
		BkCloudID:              in.BkCloudID,
		DBAppAbbr:              in.DBAppAbbr,
		ClusterType:            dbm.ClusterType(in.ClusterType),
		ClusterName:            in.ClusterName,
		ClusterAlias:           in.ClusterAlias,
		DBVersion:              in.DBVersion,
		ProxyPort:              in.ProxyPort,
		ClusterShardNum:        in.ClusterShardNum,
		Port:                   in.Port,
		Databases:              in.Databases,
		RedisPwd:               in.RedisPwd,
		IPSource:               in.IPSource,
		DisasterToleranceLevel: in.DisasterToleranceLevel,
	}
	if in.ResourceSpec != nil {
		params.ResourceSpec = in.ResourceSpec.toDBM()
	}
	return params
}

func (in *ResourceSpecInput) toDBM() *dbm.ResourceSpec {
	if in == nil {
		return nil
	}
	spec := &dbm.ResourceSpec{}
	if in.Proxy != nil {
		spec.Proxy = in.Proxy.toDBM()
	}
	if in.BackendGroup != nil {
		spec.BackendGroup = in.BackendGroup.toDBM()
	}
	return spec
}

func (in *ResourceSpecItemInput) toDBM() *dbm.ResourceSpecItem {
	if in == nil {
		return nil
	}
	item := &dbm.ResourceSpecItem{
		SpecID: in.SpecID,
		Count:  in.Count,
	}
	if in.LocationSpec != nil {
		item.LocationSpec = &dbm.LocationSpec{
			City:       in.LocationSpec.City,
			SubZoneIDs: in.LocationSpec.SubZoneIDs,
		}
	}
	return item
}

// -----------------------------------------------------------------------------
// Outputs
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// CreateRedisInstanceOutput is the JSON response for creating a Redis instance.
type CreateRedisInstanceOutput struct {
	Data CreateRedisInstanceOutputObj `json:"data"`
}

// CreateRedisInstanceOutputObj contains the created instance id and status.
type CreateRedisInstanceOutputObj struct {
	// 实例 ID
	ID string `json:"id"`
	// 实例状态
	Status string `json:"status"`
}

// ListRedisInstancesOutput is the JSON response for listing Redis instances.
type ListRedisInstancesOutput struct {
	// 实例列表
	Data []*RedisInstanceOutputObj `json:"data"`
}

// GetRedisInstanceOutput is the JSON response for getting a Redis instance.
type GetRedisInstanceOutput struct {
	Data *RedisInstanceOutputObj `json:"data"`
}

// RedisInstanceOutputObj is the JSON representation of a Redis service instance.
// Credentials are never included.
type RedisInstanceOutputObj struct {
	// 实例 ID
	ID string `json:"id"`
	// 实例名称
	Name string `json:"name"`
	// 服务名
	ServiceName string `json:"serviceName"`
	// Provider 类型
	ProviderType string `json:"providerType"`
	// 作用域类型
	ScopeType string `json:"scopeType"`
	// 作用域值
	ScopeValue string `json:"scopeValue"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 引用该实例的应用 ID 列表（由绑定反查）
	UsedAppIDs []string `json:"usedAppIDs"`
	// 非敏感配置
	Config RedisInstanceConfigOutput `json:"config"`
	// 实例状态
	Status string `json:"status"`
	// 状态详情
	Message string `json:"message"`
	// 操作人
	Operator string `json:"operator"`
	// 描述
	Description string `json:"description"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

// RedisInstanceConfigOutput exposes non-sensitive Redis config fields.
type RedisInstanceConfigOutput struct {
	ClusterID   int    `json:"clusterID,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	ClusterType string `json:"clusterType,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Port        int    `json:"port,omitempty"`
	BkBizID     int    `json:"bkBizID,omitempty"`
}

// FromModel converts a ServiceInstance to RedisInstanceOutputObj without credentials.
func (o *RedisInstanceOutputObj) FromModel(inst *model.ServiceInstance) *RedisInstanceOutputObj {
	if o == nil {
		o = &RedisInstanceOutputObj{}
	}

	o.ID = inst.ID.Hex()
	o.Name = inst.Name
	o.ServiceName = inst.ServiceName
	o.ProviderType = inst.ProviderType
	o.ScopeType = string(inst.ScopeType)
	o.ScopeValue = inst.ScopeValue
	o.WorkspaceID = inst.WorkspaceID
	o.UsedAppIDs = []string{}
	o.Config = redisConfigFromMap(inst.Config)
	o.Status = string(inst.Status)
	o.Message = inst.Message
	o.Operator = inst.Operator
	o.Description = inst.Description
	o.CreatedAt = inst.CreatedAt.UTC().Format(time.RFC3339)
	o.UpdatedAt = inst.UpdatedAt.UTC().Format(time.RFC3339)
	return o
}

// WithUsedAppIDs sets the reverse-lookup app IDs that reference this instance.
func (o *RedisInstanceOutputObj) WithUsedAppIDs(appIDs []string) *RedisInstanceOutputObj {
	if o == nil {
		return o
	}
	if appIDs == nil {
		appIDs = []string{}
	}
	o.UsedAppIDs = appIDs
	return o
}

func redisConfigFromMap(cfg map[string]any) RedisInstanceConfigOutput {
	if cfg == nil {
		return RedisInstanceConfigOutput{}
	}
	return RedisInstanceConfigOutput{
		ClusterID:   cast.ToInt(cfg["clusterID"]),
		ClusterName: cast.ToString(cfg["clusterName"]),
		ClusterType: cast.ToString(cfg["clusterType"]),
		Domain:      cast.ToString(cfg["domain"]),
		Port:        cast.ToInt(cfg["port"]),
		BkBizID:     cast.ToInt(cfg["bkBizID"]),
	}
}
