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

// Package discovery 提供集群资源发现功能（Group/Version/ResourceKind）
package discovery

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

// GetServerVersion 获取集群版本信息
func GetServerVersion(cfg *cluster.Config) (*version.Info, error) {
	cli, err := NewRedisCacheClient(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "new redis cache client for cluster %s", cfg.ClusterID)
	}
	return cli.ServerVersion()
}

// GetGroupVersionResource 根据配置，名称等信息，获取指定资源对应的 GroupVersionResource
// 若指定 GroupVersion，则在对应的 Group 中寻找资源信息，否则获取 preferred version
// 包含刷新缓存逻辑，若首次从缓存中找不到对应资源，会刷新缓存再次查询，若还是找不到，则返回错误
func GetGroupVersionResource(
	cfg *cluster.Config, kind, groupVersion string,
) (*schema.GroupVersionResource, error) {
	cli, err := NewRedisCacheClient(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "new redis cache client for cluster %s", cfg.ClusterID)
	}

	// 定义获取资源的函数
	getResource := func() (*schema.GroupVersionResource, error) {
		if len(groupVersion) != 0 {
			return cli.getResWithGroupVersion(kind, groupVersion)
		}
		return cli.getPreferredResource(kind)
	}

	// 第一次尝试获取资源
	res, err := getResource()
	if err == nil {
		return res, nil
	}

	// 刷新缓存后重试
	cli.Invalidate()

	// 第二次尝试获取资源
	res, err = getResource()
	if err != nil {
		return nil, errors.Wrapf(err, "get group version resource")
	}

	return res, nil
}

// GetResPreferredVersion 获取某类资源在集群中的 Preferred 版本
func GetResPreferredVersion(cfg *cluster.Config, kind string) (string, error) {
	resInfo, err := GetGroupVersionResource(cfg, kind, "")
	if err != nil {
		return "", errors.Wrapf(err, "get preferred version")
	}
	if resInfo.Group != "" {
		return resInfo.Group + "/" + resInfo.Version, nil
	}
	return resInfo.Version, nil
}
