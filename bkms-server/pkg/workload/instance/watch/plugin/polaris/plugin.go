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

// Package polaris 以 Watch 插件的形式为流内实例补充北极星注册状态
package polaris

import (
	"context"

	"github.com/pkg/errors"

	polarisaddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	watchplugin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/watch/plugin"
)

// pluginName 写入事件的 plugin 字段，前端据此把载荷落到实例行的北极星列
const pluginName = "polaris"

// Lister 拉取当前应用环境的北极星实例；返回 error 时 Runner 跳过本轮，不拆流
type Lister func(ctx context.Context) ([]*polarisaddon.PolarisServiceInstances, error)

// Plugin 北极星附属数据插件
//
// 北极星没有原生 Watch，只能周期拉取，且 SDK 自身有约 15s 缓存，因此拉得比
// Runner 的周期更密只会拿回同一份数据
type Plugin struct {
	lister Lister
}

var _ watchplugin.Plugin = &Plugin{}

// New 绑定该应用环境的北极星拉取
func New(lister Lister) *Plugin {
	return &Plugin{lister: lister}
}

// Name 插件名
func (p *Plugin) Name() string {
	return pluginName
}

// Fetch 按快照返回各实例当前的北极星注册状态
// 拉取失败直接返回 error：Runner 会跳过本轮，页面保留上次已知状态，不会被清成空数组
func (p *Plugin) Fetch(
	ctx context.Context,
	snapshot []watchplugin.InstanceSnapshot,
) (map[string]any, error) {
	// 按应用环境拉北极星实例；失败抛给 Runner 跳过本轮
	svcInstances, err := p.lister(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list polaris service instances")
	}

	// 匹配走领域层，写出前再投影成 API 对象，保证与 List 口径一致
	matcher := polarisaddon.NewInstanceMatcher(svcInstances)

	payloads := make(map[string]any, len(snapshot))
	for _, instance := range snapshot {
		payloads[instance.ID] = serializer.PolarisInfosFromModels(matcher.ForIP(instance.IP))
	}

	return payloads, nil
}
