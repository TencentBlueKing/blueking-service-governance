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

package dbfactory

import (
	"cmp"
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/onsi/gomega"
	"github.com/samber/lo"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
)

// PolarisConfigOpts 定义创建北极星配置时的参数
type PolarisConfigOpts struct {
	// AppID 配置所属应用 ID（必填）
	AppID string `validate:"required"`
	// Name 配置名称；为空时使用随机值
	Name string
	// AccessPointID 配置固定的接入点 ID；为空表示平台托管
	AccessPointID string
	// ScopeEnvs 配置的生效环境
	ScopeEnvs []*envmodel.Environment
}

// PolarisConfig 创建一个已持久化的测试用北极星配置
func PolarisConfig(ctx context.Context, store *polaris.ConfigStore, opts *PolarisConfigOpts) *polaris.PolarisConfig {
	validateOpts(opts)

	cfg := &polaris.PolarisConfig{
		AppID:         opts.AppID,
		Name:          cmp.Or(opts.Name, "test-polaris-"+stringx.Random(6)),
		AccessPointID: opts.AccessPointID,
		ScopeEnvNames: lo.Map(opts.ScopeEnvs, func(e *envmodel.Environment, _ int) string { return e.Name }),
	}
	gomega.Expect(store.Create(ctx, cfg)).To(gomega.Succeed())
	return cfg
}

// PolarisAccessPoint 在 workspaceID 下创建一个已持久化的测试用北极星自建接入点，名称使用随机后缀避免空间内重名
func PolarisAccessPoint(
	ctx context.Context, store *accesspoint.AccessPointStore, workspaceID string,
) *accesspoint.PolarisAccessPoint {
	p := &accesspoint.PolarisAccessPoint{
		WorkspaceID: workspaceID,
		Name:        "test-ap-" + stringx.Random(6),
		ConsoleURL:  "http://console.example.com",
		ClientHosts: []string{"10.0.0.1"},
	}
	gomega.Expect(store.Create(ctx, p)).To(gomega.Succeed())
	return p
}

// PolarisClusterBinding 创建一个已持久化的测试用集群绑定，把 workspaceID 下的集群绑定到接入点
func PolarisClusterBinding(
	ctx context.Context, store *accesspoint.BindingStore, workspaceID, clusterID, accessPointID string,
) *accesspoint.PolarisClusterBinding {
	b := &accesspoint.PolarisClusterBinding{
		WorkspaceID:   workspaceID,
		ClusterID:     clusterID,
		AccessPointID: accessPointID,
	}
	gomega.Expect(store.Create(ctx, b)).To(gomega.Succeed())
	return b
}
