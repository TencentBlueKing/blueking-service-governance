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

// Package polaris 提供北极星 SDK 的统一抽象接口与开源实现。
//
// 本包导出 Provider 接口和 Instance 数据类型，并提供基于开源北极星 SDK
// (github.com/polarismesh/polaris-go) 的默认实现。
//
// 内部私有版本通过 go.work replace 替换本 module 为私有 SDK 实现，
// 对外接口保持一致，调用方无需改动。
package polaris

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	polarissdk "github.com/polarismesh/polaris-go"
	polarisapi "github.com/polarismesh/polaris-go/api"
)

// Provider 抽象北极星消费端能力，屏蔽私有 SDK 与开源 SDK 之间的实现差异。
//
// 两版 SDK 接口大体一致但不可互通（例如私有版支持 JoinPoint 接入点，开源版仅支持
// 直连地址），因此由各自的 Provider 实现内部消化差异，对外提供一致的行为。
type Provider interface {
	// Init 初始化底层 SDK。
	//
	// joinPoint 为北极星接入点，仅私有 SDK 支持；开源实现应忽略非空值并给出告警。
	// sdkAddress 为直连地址列表（逗号分隔），当 joinPoint 为空时使用。
	Init(ctx context.Context, joinPoint, sdkAddress string) error

	// GetInstances 查询指定服务下的所有实例，返回已归一化的 Instance 列表。
	GetInstances(ctx context.Context, namespace, serviceName string) ([]*Instance, error)
}

// Instance 北极星实例信息
type Instance struct {
	// IP 实例 IP（对应 Pod IP）
	IP string
	// Port 实例端口（对应应用监听的服务端口）
	Port uint32
	// Weight 实例权重
	Weight int
	// IsHealthy 健康状态
	IsHealthy bool
	// IsIsolated 隔离状态
	IsIsolated bool
	// EnableHealthCheck 是否启用健康检查
	EnableHealthCheck bool
	// Metadata 元数据
	Metadata map[string]string
}

// WarnFunc 用于输出告警日志的回调函数类型。
// 由调用方注入，避免 adapter 包依赖具体的日志框架。
type WarnFunc func(ctx context.Context, format string, args ...any)

// InfoFunc 用于输出信息日志的回调函数类型。
type InfoFunc func(ctx context.Context, format string, args ...any)

// defaultProvider 基于开源北极星 SDK（github.com/polarismesh/polaris-go）的 Provider 实现。
//
// ConsumerAPI 是协程安全的，进程内只需创建一个实例，重复使用即可。
type defaultProvider struct {
	consumer polarissdk.ConsumerAPI
	warnf    WarnFunc
	infof    InfoFunc
}

// ProviderOption 配置 Provider 的选项函数。
type ProviderOption func(*defaultProvider)

// WithWarnFunc 设置告警日志输出函数。
func WithWarnFunc(f WarnFunc) ProviderOption {
	return func(p *defaultProvider) {
		p.warnf = f
	}
}

// WithInfoFunc 设置信息日志输出函数。
func WithInfoFunc(f InfoFunc) ProviderOption {
	return func(p *defaultProvider) {
		p.infof = f
	}
}

// NewDefaultProvider 创建基于开源 SDK 的 Provider 实例。
func NewDefaultProvider(opts ...ProviderOption) Provider {
	p := &defaultProvider{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Init 初始化开源北极星 ConsumerAPI。
//
// 开源 SDK 不支持 JoinPoint 接入点，若配置了 joinPoint 将忽略并告警，仅使用直连地址。
func (p *defaultProvider) Init(ctx context.Context, joinPoint, sdkAddress string) error {
	if joinPoint != "" && p.warnf != nil {
		p.warnf(ctx, "polaris joinPoint %q is not supported by the open-source SDK, ignored", joinPoint)
	}

	apiCfg := polarisapi.NewConfiguration()
	if sdkAddress != "" {
		apiCfg.GetGlobal().GetServerConnector().SetAddresses(strings.Split(sdkAddress, ","))
	}

	consumer, err := polarissdk.NewConsumerAPIByConfig(apiCfg)
	if err != nil {
		return errors.Wrap(err, "failed to create polaris consumer api")
	}

	p.consumer = consumer
	if p.infof != nil {
		p.infof(ctx, "polaris consumer api initialized (open-source sdk)")
	}
	return nil
}

// GetInstances 查询指定服务下的所有实例并归一化为 Instance。
func (p *defaultProvider) GetInstances(_ context.Context, namespace, serviceName string) ([]*Instance, error) {
	if p.consumer == nil {
		return nil, errors.New("polaris consumer api not initialized, call Init first")
	}

	req := &polarissdk.GetAllInstancesRequest{}
	req.Namespace = namespace
	req.Service = serviceName

	resp, err := p.consumer.GetAllInstances(req)
	if err != nil {
		return nil, errors.Wrapf(err, "get instances for polaris service %s/%s", namespace, serviceName)
	}

	instances := make([]*Instance, 0, len(resp.Instances))
	for _, item := range resp.Instances {
		instances = append(instances, &Instance{
			IP:                item.GetHost(),
			Port:              item.GetPort(),
			Weight:            item.GetWeight(),
			IsHealthy:         item.IsHealthy(),
			IsIsolated:        item.IsIsolated(),
			EnableHealthCheck: item.IsEnableHealthCheck(),
			Metadata:          item.GetMetadata(),
		})
	}

	return instances, nil
}
