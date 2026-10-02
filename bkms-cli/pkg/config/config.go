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

// Defaults 为默认参数，即用户调用 bkms-cli 命令时，如果未指定相应参数，则使用这些默认值
type Defaults struct {
	WorkspaceID string `yaml:"workspaceID"`
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
	// Username 用户名
	Username string `yaml:"username"`
	// AccessToken 服务访问凭证
	AccessToken string `yaml:"accessToken"`
	// Defaults 默认参数
	Defaults Defaults `yaml:"defaults"`
	// Update overrides the official update source when both URLs are configured.
	Update UpdateSource `yaml:"update,omitempty"`
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf(
		"configFilePath: %s\n\nbkmsBaseUrl: %s\nusername: %s\naccessToken: [REDACTED]\n%s%s",
		cfgFilePath, c.BkmsBaseURL, c.Username, c.Defaults.String(), c.Update.String(),
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

	normalizeLoadedConfig(conf)

	// 初始化全局配置
	G = conf
	return conf, nil
}

// normalizeLoadedConfig 统一去除尾斜杠
func normalizeLoadedConfig(conf *Config) {
	conf.BkmsBaseURL = strings.TrimSuffix(conf.BkmsBaseURL, "/")
}

// createDefaultConfig 创建默认配置文件（含目录），并返回默认配置
func createDefaultConfig() (*Config, error) {
	// 确保配置文件所在目录存在
	cfgDir := filepath.Dir(cfgFilePath)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "failed to create config directory %s", cfgDir)
	}

	conf := &Config{}

	contents, err := yaml.Marshal(conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal default config")
	}
	if err = os.WriteFile(cfgFilePath, contents, 0o600); err != nil {
		return nil, errors.Wrapf(err, "failed to write default config file %s", cfgFilePath)
	}

	return conf, nil
}

// Dump 将当前配置保存到文件
func (c *Config) Dump() error {
	contents, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	if err = os.WriteFile(cfgFilePath, contents, 0o600); err != nil {
		return err
	}
	return nil
}

// SetBkmsBaseURL 更新 bkmsBaseUrl 并持久化。
// ifUnset 为 true 时仅在当前仍为空时写入。
func (c *Config) SetBkmsBaseURL(bkmsBaseURL string, ifUnset bool) (updated bool, err error) {
	bkmsBaseURL = strings.TrimSpace(bkmsBaseURL)
	if bkmsBaseURL == "" {
		return false, errors.New("bkmsBaseUrl is required")
	}
	return c.SetEndpoints(bkmsBaseURL, UpdateSource{}, ifUnset)
}

// SetEndpoints saves the selected endpoints together, without partially changing in-memory config on failure.
// ifUnset treats the update source as a pair: either existing field prevents replacement of both.
func (c *Config) SetEndpoints(bkmsBaseURL string, source UpdateSource, ifUnset bool) (bool, error) {
	bkmsBaseURL = strings.TrimSpace(bkmsBaseURL)
	source.LatestVersionURL = strings.TrimSpace(source.LatestVersionURL)
	source.DownloadURLTemplate = strings.TrimSpace(source.DownloadURLTemplate)
	if !source.IsZero() {
		if err := source.Validate(); err != nil {
			return false, err
		}
	}
	pending := *c
	if bkmsBaseURL != "" && (!ifUnset || strings.TrimSpace(c.BkmsBaseURL) == "") {
		pending.BkmsBaseURL = strings.TrimSuffix(bkmsBaseURL, "/")
	}
	if !source.IsZero() && (!ifUnset || c.Update.IsZero()) {
		pending.Update = source
	}
	if pending == *c {
		return false, nil
	}
	if err := pending.Dump(); err != nil {
		return false, errors.Wrap(err, "save config")
	}
	*c = pending
	return true, nil
}

// UserIsInitialized 判断配置是否已完成用户初始化，仅检查 AccessToken 等配置项是否存在
// 不验证其真实有效性。
func (c *Config) UserIsInitialized() bool {
	return c.Username != "" && c.AccessToken != ""
}

// HasBkmsBaseURL reports whether bkmsBaseUrl is set.
func (c *Config) HasBkmsBaseURL() bool {
	return c != nil && strings.TrimSpace(c.BkmsBaseURL) != ""
}

// RequireBkmsBaseURL returns an error with setup guidance when bkmsBaseUrl is empty.
func (c *Config) RequireBkmsBaseURL() error {
	if c.HasBkmsBaseURL() {
		return nil
	}
	return errors.New(
		"bkmsBaseUrl is not configured\n\n" +
			"Set it first, then continue:\n" +
			"  bkms-cli config set --bkms-base-url <url>",
	)
}
