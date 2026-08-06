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

// Package serializer defines Gin input and output serializers for port-pool APIs.
package serializer

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/portpool"
)

// portPoolNamePattern 匹配以小写字母开头、小写字母或数字结尾，中间可含小写字母、数字、连字符的字符串
var portPoolNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("portpool_name", validatePortPoolName)
		_ = v.RegisterValidation("portpool_protocol", validatePortPoolProtocol)
	}
}

func validatePortPoolProtocol(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	return s == "TCP" || s == "UDP" || s == "TCP,UDP"
}

func validatePortPoolName(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if len(s) < 1 || len(s) > 63 {
		return false
	}
	return portPoolNamePattern.MatchString(s)
}

// -----------------------------------------------------------------------------
// Path inputs
// -----------------------------------------------------------------------------

// EnvURIInput is the path input for APIs scoped by environment.
type EnvURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,min=1"`
}

// EnvNameURIInput is the path input for APIs scoped by environment and port pool name.
type EnvNameURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,min=1"`
	// 端口池名称
	Name string `uri:"name" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// Shared outputs
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// -----------------------------------------------------------------------------
// List port pools
// -----------------------------------------------------------------------------

// ListPortPoolsOutput is the JSON response for listing port pools.
type ListPortPoolsOutput struct {
	// 端口池列表
	Data []*PortPoolConfigOutputObj `json:"data"`
}

// PortPoolConfigOutputObj is the JSON representation of a port pool config.
type PortPoolConfigOutputObj struct {
	// 所属环境 ID
	EnvID string `json:"envID"`
	// 配置名称
	Name string `json:"name"`
	// 端口池 item 列表
	PoolItems []*PortPoolItemOutput `json:"poolItems"`
	// 端口池整体状态 [Ready, NotReady, Deleting]
	Status string `json:"status"`
}

// PortPoolItemOutput is the JSON representation of a port pool item.
type PortPoolItemOutput struct {
	// 端口池 item 名称
	ItemName string `json:"itemName"`
	// 负载均衡 ID 列表
	LoadBalancerIDs []string `json:"loadBalancerIDs"`
	// 端口池的协议
	Protocol string `json:"protocol"`
	// 起始端口
	StartPort int32 `json:"startPort"`
	// 结束端口
	EndPort int32 `json:"endPort"`
	// 端口段长度
	SegmentLength int32 `json:"segmentLength"`
	// 扩展字段(透传到业务)
	External string `json:"external"`
	// item 状态
	PoolItemStatus *PoolItemStatusOutput `json:"poolItemStatus"`
}

// PoolItemStatusOutput is the JSON representation of a pool item status.
type PoolItemStatusOutput struct {
	// item 状态
	Status string `json:"status"`
	// item 状态信息
	Message string `json:"message"`
}

// FromModel fills output fields from a PortPoolConfig domain model.
func (o *PortPoolConfigOutputObj) FromModel(config portpool.PortPoolConfig) *PortPoolConfigOutputObj {
	*o = PortPoolConfigOutputObj{
		EnvID:  config.EnvID,
		Name:   config.Name,
		Status: config.Status,
	}
	for _, item := range config.PoolItems {
		poolItem := &PortPoolItemOutput{
			ItemName:        item.ItemName,
			LoadBalancerIDs: item.LoadBalancerIDs,
			Protocol:        item.Protocol,
			StartPort:       item.StartPort,
			EndPort:         item.EndPort,
			SegmentLength:   item.SegmentLength,
			External:        item.External,
			PoolItemStatus: &PoolItemStatusOutput{
				Status:  item.Status.Status,
				Message: item.Status.Message,
			},
		}
		o.PoolItems = append(o.PoolItems, poolItem)
	}
	return o
}

// -----------------------------------------------------------------------------
// Create port pool
// -----------------------------------------------------------------------------

// CreatePortPoolInput is the JSON input for creating a port pool.
type CreatePortPoolInput struct {
	// 端口池名称，需符合 k8s 命名规范
	Name string `json:"name" binding:"required,portpool_name"`
	// 端口池 item 列表
	PoolItems []PortPoolItemInput `json:"poolItems" binding:"required,min=1,dive"`
}

// PortPoolItemInput is the JSON input for a port pool item.
type PortPoolItemInput struct {
	// 端口池 item 名称，新增时不填则自动生成
	ItemName string `json:"itemName"`
	// 负载均衡 ID 列表
	LoadBalancerIDs []string `json:"loadBalancerIDs" binding:"required,min=1"`
	// 端口池的协议，不填则默认同时支持 TCP 和 UDP
	Protocol string `json:"protocol" binding:"omitempty,portpool_protocol"`
	// 起始端口
	StartPort int32 `json:"startPort" binding:"required,min=1,max=65535"`
	// 结束端口，端口范围是左闭右开区间，即 [startPort, endPort)
	EndPort int32 `json:"endPort" binding:"required,min=1,max=65535"`
	// 端口段长度
	SegmentLength int32 `json:"segmentLength" binding:"omitempty,min=1"`
	// 扩展字段(透传到业务)
	External string `json:"external"`
}

// ToModel converts input to portpool.PoolItem domain model.
func (i *PortPoolItemInput) ToModel() portpool.PoolItem {
	return portpool.PoolItem{
		ItemName:        i.ItemName,
		LoadBalancerIDs: i.LoadBalancerIDs,
		Protocol:        i.Protocol,
		StartPort:       i.StartPort,
		EndPort:         i.EndPort,
		SegmentLength:   i.SegmentLength,
		External:        i.External,
	}
}

// -----------------------------------------------------------------------------
// Update port pool
// -----------------------------------------------------------------------------

// UpdatePortPoolInput is the JSON input for updating a port pool.
type UpdatePortPoolInput struct {
	// 完整的 poolItem 列表，全量替换
	PoolItems []PortPoolItemInput `json:"poolItems" binding:"required,min=1,dive"`
}
