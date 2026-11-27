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

package cluster

import (
	"context"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// unauthorizedError BCS 网关拒绝了空间的集群 Token。
// 调用方会把它包成内部错误，响应输出时还原为 CLUSTER_TOKEN_UNAUTHORIZED。
type unauthorizedError struct{}

func (unauthorizedError) Error() string {
	return "bcs gateway rejected the cluster token, check whether the token is valid " +
		"and its owner has permission on the cluster"
}

func (unauthorizedError) APIErrorDetail() (bkerrs.ErrCode, bkerrs.ErrDetailCode, string) {
	return bkerrs.ErrCodeNoPermission, bkerrs.ErrDetailCodeClusterTokenUnauthorized, "cluster"
}

// ErrUnauthorized BCS 网关拒绝了空间的集群 Token：Token 无效，或 Token 所属用户没有集群、命名空间的权限
var ErrUnauthorized error = unauthorizedError{}

var _ bkerrs.APIDetailError = unauthorizedError{}

// TokenResolver 返回空间访问集群 API 使用的 Token
type TokenResolver func(ctx context.Context, workspaceID, clusterID string) (string, error)

// tokenResolver 由 web 与 worker 启动时通过 SetTokenResolver 注入。
// 本包属于基础设施层，不直接依赖保存集群 Token 的领域包
var tokenResolver TokenResolver

// SetTokenResolver 注入集群 Token 的解析函数，须在处理请求与任务之前调用
func SetTokenResolver(r TokenResolver) {
	tokenResolver = r
}

// NewConfigForWorkspace 创建空间访问集群使用的集群配置，业务代码应使用它而不是 NewConfig。
//
// bcs.allowWorkspaceClusterTokens 关闭或使用本地集群时等价于 NewConfig；
// 打开时使用空间为集群分配的 Token，集群未接入空间时返回解析函数的错误，BCS 网关拒绝 Token 时请求返回 ErrUnauthorized
func NewConfigForWorkspace(ctx context.Context, workspaceID, clusterID string) (*Config, error) {
	if !workspaceTokensEnabled() || useLocalCluster() {
		return NewConfig(clusterID), nil
	}
	if tokenResolver == nil {
		return nil, errors.New("cluster token resolver is not set")
	}
	token, err := tokenResolver(ctx, workspaceID, clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve token of cluster %s for workspace %s", clusterID, workspaceID)
	}
	cfg := newBCSConfig(clusterID, token)
	cfg.Rest.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &unauthorizedTransport{next: rt, clusterID: clusterID}
	}
	return cfg, nil
}

// ConfigCache 在一次操作（一个请求或任务）内缓存 NewConfigForWorkspace 的结果，避免重复解析集群 Token。
// 不能跨操作复用：Token 更换后缓存的配置不会更新。零值可直接使用；nil 时不缓存
type ConfigCache struct {
	mu      sync.Mutex
	configs map[[2]string]*Config
}

// Get 返回空间访问集群使用的集群配置，解析失败时不缓存
func (c *ConfigCache) Get(ctx context.Context, workspaceID, clusterID string) (*Config, error) {
	if c == nil {
		return NewConfigForWorkspace(ctx, workspaceID, clusterID)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := [2]string{workspaceID, clusterID}
	if cfg, ok := c.configs[key]; ok {
		return cfg, nil
	}
	cfg, err := NewConfigForWorkspace(ctx, workspaceID, clusterID)
	if err != nil {
		return nil, err
	}
	if c.configs == nil {
		c.configs = map[[2]string]*Config{}
	}
	c.configs[key] = cfg
	return cfg, nil
}

func workspaceTokensEnabled() bool {
	return config.G != nil && config.G.BCS.AllowWorkspaceClusterTokens
}

// unauthorizedTransport 把 BCS 网关的 401 响应转换为 ErrUnauthorized，
// 否则调用方只能看到 client-go 的 "the server has asked for the client to provide credentials"
type unauthorizedTransport struct {
	next      http.RoundTripper
	clusterID string
}

func (t *unauthorizedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	_ = resp.Body.Close()
	return nil, errors.Wrapf(ErrUnauthorized, "cluster %s, %s %s", t.clusterID, req.Method, req.URL.Path)
}
