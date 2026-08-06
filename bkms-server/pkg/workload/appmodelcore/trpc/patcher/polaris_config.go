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

package patcher

import (
	"context"

	gabs "github.com/Jeffail/gabs/v2"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
)

// PolarisRegistryPatcher 在 tRPC 配置中注入北极星注册中心的 service 配置。
// 当应用存在启用了健康检查的北极星配置时，将 plugins.registry.polaris.service 配置块
// 注入到 YAML 配置文件中。如果原始配置已有该路径，则不覆盖。
type PolarisRegistryPatcher struct {
	polarisConfigStore polaris.PolarisConfigStore
}

// NewPolarisRegistryPatcher 创建北极星注册配置补丁器
func NewPolarisRegistryPatcher(store polaris.PolarisConfigStore) *PolarisRegistryPatcher {
	return &PolarisRegistryPatcher{polarisConfigStore: store}
}

// Patch 实现 ConfigPatcher 接口
func (p *PolarisRegistryPatcher) Patch(ctx context.Context, appID, envName, content string) (string, error) {
	if p.polarisConfigStore == nil {
		return content, nil
	}

	// 获取当前环境下可用的北极星配置列表
	configs, listErr := p.polarisConfigStore.ListByEnv(ctx, appID, envName)
	if listErr != nil {
		return "", errors.Wrap(listErr, "listing polaris configs")
	}

	// 收集启用了健康检查的配置条目
	entries := polaris.CollectRegistryServiceEntries(configs)
	if len(entries) == 0 {
		return content, nil
	}

	// 解析原始 YAML 配置
	var configMap map[string]any
	if err := yaml.Unmarshal([]byte(content), &configMap); err != nil {
		return "", errors.Wrap(err, "parsing tRPC config YAML")
	}
	if configMap == nil {
		configMap = make(map[string]any)
	}

	container := gabs.Wrap(configMap)

	// 检查是否已有 plugins.registry.polaris.service 配置
	if container.ExistsP("plugins.registry.polaris.service") {
		return content, nil
	}

	// 注入配置
	if injectErr := injectRegistryServiceEntries(container, entries); injectErr != nil {
		return "", errors.Wrap(injectErr, "injecting polaris service entries")
	}

	// 序列化回 YAML
	result, marshalErr := yaml.Marshal(configMap)
	if marshalErr != nil {
		return "", errors.Wrap(marshalErr, "marshaling merged tRPC config")
	}
	return string(result), nil
}

// injectRegistryServiceEntries 将 service 条目注入到 configMap 的
// plugins.registry.polaris.service 路径下
func injectRegistryServiceEntries(container *gabs.Container, entries []polaris.RegistryServiceEntry) error {
	serviceList := make([]any, 0, len(entries))
	for _, entry := range entries {
		serviceList = append(serviceList, map[string]any{
			"name":      entry.Name,
			"namespace": entry.Namespace,
			"token":     entry.Token,
		})
	}
	// SetP 会自动创建中间层级的 map
	if _, err := container.SetP(serviceList, "plugins.registry.polaris.service"); err != nil {
		return err
	}
	return nil
}
