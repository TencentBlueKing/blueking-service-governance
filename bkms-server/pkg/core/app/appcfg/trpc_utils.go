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

package appcfg

import (
	"context"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// trpcServiceConfig 是用于提取 tRPC 配置中 server.service 信息的轻量结构体
// 当前业务只需要解析 server.service[].name，因此这里只覆盖少量字段；完整字段可参考：
// https://github.com/trpc-group/trpc-go/blob/9b5c63e5/config.go#L514
type trpcServiceConfig struct {
	Server struct {
		Service []*struct {
			Name string `yaml:"name"`
		} `yaml:"service"`
	} `yaml:"server"`
}

// GetTrpcServiceNames 获取指定应用和环境下 tRPC 配置文件中的所有服务名
// 该方法组合了 GetEnvContent 和 parseTrpcServiceNames，供需要从环境配置中读取服务名的业务流程使用
func GetTrpcServiceNames(
	ctx context.Context,
	store AppConfigFileStore,
	appID, envName string,
) ([]string, error) {
	_, content, err := GetEnvContent(ctx, store, appID, envName)
	if err != nil {
		return nil, err
	}
	return parseTrpcServiceNames(content)
}

// parseTrpcServiceNames 从 tRPC 配置 YAML 内容中提取所有 server.service[].name
func parseTrpcServiceNames(content string) ([]string, error) {
	var cfg trpcServiceConfig
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, errors.Wrap(err, "unmarshal tRPC config YAML")
	}

	var names []string
	for _, svc := range cfg.Server.Service {
		if svc != nil && svc.Name != "" {
			names = append(names, svc.Name)
		}
	}
	return names, nil
}
