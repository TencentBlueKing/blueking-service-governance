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
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// Config k8s 集群配置
type Config struct {
	Rest      *rest.Config
	ClusterID string
}

// NewConfig 创建集群配置，clusterID 为 BCS 集群 ID。
//
// 函数也可以被配置为返回本地集群（开发时适用）：
// - 修改配置文件中的 UseKubeConfigCluster / StubKubeConfigPath
func NewConfig(clusterID string) *Config {
	// 全局配置未初始化 / 特别指定时，使用本地 kubeConfig 指向的集群
	if config.G == nil || config.G.Development.UseKubeConfigCluster {
		var kubeCfgPath string
		if config.G != nil && config.G.Development.StubKubeConfigPath != "" {
			kubeCfgPath = config.G.Development.StubKubeConfigPath
		}

		cfg, err := BuildLocalKubeConfig(clusterID, kubeCfgPath)
		// 本地 kubeconfig 只会是用在开发场景下，如果失败直接 panic 即可
		if err != nil {
			panic(fmt.Sprintf("failed to build config from local kubeconfig: %v", err))
		}
		return cfg
	}

	// 默认使用 BCS 集群
	return &Config{
		Rest: &rest.Config{
			Host:            fmt.Sprintf("%s/clusters/%s", config.G.BCS.BaseUrl, clusterID),
			BearerToken:     config.G.BCS.Token,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		},
		ClusterID: clusterID,
	}
}

// BuildLocalKubeConfig builds cluster config from local kubeconfig file.
//
// Args:
// - clusterID: the cluster ID to set in the returned Config
// - configPath: path to the kubeconfig file; if empty, use the default path (~/.kube/config)
func BuildLocalKubeConfig(clusterID, configPath string) (*Config, error) {
	if configPath == "" {
		configPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	if _, err := os.Stat(configPath); err != nil {
		return nil, errors.Wrap(err, "stat kubeconfig")
	}
	restConf, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "build config from %s", configPath)
	}
	return &Config{Rest: restConf, ClusterID: clusterID}, nil
}
