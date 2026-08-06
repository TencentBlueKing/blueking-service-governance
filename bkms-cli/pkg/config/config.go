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

// Package config 提供 bkms-cli 配置定义
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/envx"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/pathx"
)

// 以下变量值可通过环境变量指定

// cfgFilePath 为配置文件所在路径
var cfgFilePath = envx.Get("BKMS_CLI_CONFIG", filepath.Join(pathx.HomeDir(), ".bkms", "config.yaml"))

// bkmsBaseURL 默认的 bkms 服务基础地址，通过 go build -ldflags 注入
var bkmsBaseURL = ""

// Defaults 为默认参数，即用户调用 bkms-cli 命令时，如果未指定相应参数，则使用这些默认值
type Defaults struct {
	WorkspaceID string `yaml:"workspaceID"`
}

// BCSConfig 蓝鲸 BCS 容器平台配置
type BCSConfig struct {
	// Token BCS 个人 token，用于通过 BCS API 网关连接 K8s 集群，该 token 可在 BCS 平台个人设置中获取；
	// 请求经过 BCS 网关时，该 token 会被用来做用户身份鉴权；
	// 仅用于 publish 命令的文件发布场景，首次通过 --bcs-token 参数传入后会持久化到配置文件，后续自动读取；
	// 注意：该 token 具有个人权限，请勿泄露；如 token 过期或权限变更，需重新通过 --bcs-token 传入覆盖
	Token string `yaml:"token"`
}

// String 返回 Defaults 的字符串表示
func (d *Defaults) String() string {
	// 如果所有字段都没有设置默认值，则不返回
	if d.WorkspaceID == "" {
		return ""
	}
	return fmt.Sprintf("defaults:\n  workspaceID: %s\n", d.WorkspaceID)
}

// Config 是 bkms-cli 的配置结构体
type Config struct {
	// BkmsBaseURL 服务治理平台服务访问基础地址
	BkmsBaseURL string `yaml:"bkmsBaseUrl"`
	// Deprecated: 兼容历史配置项，读取后会迁移到 BkmsBaseURL
	BkmsApigwURL string `yaml:"bkmsApigwUrl,omitempty"`
	// Username 用户名
	Username string `yaml:"username"`
	// AccessToken 服务访问凭证
	AccessToken string `yaml:"accessToken"`
	// Defaults 默认参数
	Defaults Defaults `yaml:"defaults"`
	// BCS 蓝鲸容器平台配置，仅用于 publish 命令通过 BCS API 网关连接 K8s 集群，替代本地 kubeconfig 文件；
	// 目前仅存储个人 Token，首次使用时通过 --bcs-token 参数传入并持久化，后续自动读取；
	// 注意：该配置仅适用于 CLI publish 的文件发布场景，不适用于服务端 BCS 集群管理
	BCS BCSConfig `yaml:"bcs"`
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf(
		"configFilePath: %s\n\nbkmsBaseUrl: %s\nusername: %s\naccessToken: [REDACTED]\n%sbcs:\n  token: [REDACTED]",
		cfgFilePath, G.BkmsBaseURL, G.Username, c.Defaults.String(),
	)
}

// G 是全局配置实例，可在代码逻辑中使用
var G *Config

// Load 从文件加载配置；若配置文件不存在，则自动创建目录和默认配置文件
func (c *Config) Load() (*Config, error) {
	yamlFile, err := os.ReadFile(cfgFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// 配置文件不存在，自动创建
		conf, createErr := createDefaultConfig()
		if createErr != nil {
			return nil, createErr
		}
		G = conf
		return conf, nil
	}

	conf := &Config{}
	if err = yaml.Unmarshal(yamlFile, conf); err != nil {
		return nil, err
	}

	// 兼容历史配置项，并统一去除尾斜杠
	if conf.BkmsBaseURL == "" {
		conf.BkmsBaseURL = conf.BkmsApigwURL
	}
	conf.BkmsBaseURL = strings.TrimSuffix(conf.BkmsBaseURL, "/")
	conf.BkmsApigwURL = ""

	// 初始化全局配置
	G = conf
	return conf, nil
}

// createDefaultConfig 创建默认配置文件（含目录），并返回默认配置
func createDefaultConfig() (*Config, error) {
	// 确保配置文件所在目录存在
	cfgDir := filepath.Dir(cfgFilePath)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "failed to create config directory %s", cfgDir)
	}

	conf := &Config{
		BkmsBaseURL: bkmsBaseURL,
	}

	contents, err := yaml.Marshal(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal default config")
	}
	if err = os.WriteFile(cfgFilePath, contents, 0o600); err != nil {
		return nil, errors.Wrapf(err, "failed to write default config file %s", cfgFilePath)
	}

	return conf, nil
}

// Dump 将全局配置保存到文件
func (c *Config) Dump() error {
	contents, err := yaml.Marshal(G)
	if err != nil {
		return err
	}
	if err = os.WriteFile(cfgFilePath, contents, 0o600); err != nil {
		return err
	}
	return nil
}

// UserIsInitialized 判断配置是否已完成用户初始化，仅检查 AccessToken 等配置项是否存在
// 不验证其真实有效性。
func (c *Config) UserIsInitialized() bool {
	return c.Username != "" && c.AccessToken != ""
}
