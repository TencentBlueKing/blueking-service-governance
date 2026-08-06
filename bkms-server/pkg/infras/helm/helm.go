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

// Package helm provides Helm SDK integration capabilities.
package helm

import (
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	clusterdiscovery "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
)

// helmReleaseStorageDriver Helm Release 存储驱动类型（使用 Kubernetes Secret 存储）
const helmReleaseStorageDriver = "secret"

// customRESTClientGetter 实现 genericclioptions.RESTClientGetter 接口
// 封装 cluster.NewConfig(clusterID) 返回的 *rest.Config，为 Helm SDK 提供集群连接能力
type customRESTClientGetter struct {
	clusterID  string
	namespace  string
	restConfig *rest.Config
}

// newCustomRESTClientGetter 创建自定义 RESTClientGetter
func newCustomRESTClientGetter(clusterID, namespace string) *customRESTClientGetter {
	cfg := cluster.NewConfig(clusterID)
	return &customRESTClientGetter{
		clusterID:  clusterID,
		namespace:  namespace,
		restConfig: cfg.Rest,
	}
}

var _ genericclioptions.RESTClientGetter = &customRESTClientGetter{}

// ToRESTConfig 返回 REST 客户端配置
func (g *customRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return g.restConfig, nil
}

// ToDiscoveryClient 返回基于 Redis 缓存的 Discovery 客户端
func (g *customRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	cfg := &cluster.Config{
		Rest:      g.restConfig,
		ClusterID: g.clusterID,
	}
	client, err := clusterdiscovery.NewRedisCacheClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "create redis cache discovery client")
	}
	return client, nil
}

// ToRESTMapper 返回 REST Mapper
func (g *customRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

// ToRawKubeConfigLoader 返回 ClientConfig（用于 Helm SDK 内部获取 namespace 等信息）
func (g *customRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return &clientConfigAdapter{
		namespace:  g.namespace,
		restConfig: g.restConfig,
	}
}

// clientConfigAdapter 将 *rest.Config 适配为 clientcmd.ClientConfig 接口
type clientConfigAdapter struct {
	namespace  string
	restConfig *rest.Config
}

var _ clientcmd.ClientConfig = &clientConfigAdapter{}

// RawConfig 返回空的 api.Config（Helm SDK 不会实际使用该方法的返回值）
func (a *clientConfigAdapter) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}

// ClientConfig 返回封装的 *rest.Config
func (a *clientConfigAdapter) ClientConfig() (*rest.Config, error) {
	return a.restConfig, nil
}

// Namespace 返回命名空间
func (a *clientConfigAdapter) Namespace() (string, bool, error) {
	return a.namespace, false, nil
}

// ConfigAccess 返回空的 ConfigAccess（Helm SDK 不会实际使用该方法）
func (a *clientConfigAdapter) ConfigAccess() clientcmd.ConfigAccess {
	return nil
}

// NewActionConfiguration 创建 Helm SDK action.Configuration
// 通过 clusterID 获取集群连接配置，namespace 指定目标命名空间
func NewActionConfiguration(clusterID, namespace string, debugLog action.DebugLog) (*action.Configuration, error) {
	getter := newCustomRESTClientGetter(clusterID, namespace)
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, helmReleaseStorageDriver, debugLog); err != nil {
		return nil, errors.Wrap(err, "init helm action configuration")
	}
	return cfg, nil
}
