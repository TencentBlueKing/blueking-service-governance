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

// Package framework 提供 e2e 基础框架功能
package framework

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
)

// NOTE: fmt 仍用于 ginkgo.Fail 中的 Sprintf，日志输出已统一使用 Logf

const (
	// e2eConfigFileName 是 E2E 测试专属配置文件名，避免覆盖用户的正式配置 (~/.bkms/config.yaml)
	e2eConfigFileName = "e2e-config.yaml"
)

// configFilePath 返回 E2E 测试专属 CLI 配置文件路径
// 优先使用 BKMS_CLI_CONFIG 环境变量，否则使用 ~/.bkms/e2e-config.yaml
func configFilePath() string {
	if p := os.Getenv("BKMS_CLI_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	return filepath.Join(home, ".bkms", e2eConfigFileName)
}

// buildCLIConfig 根据环境配置和认证状态构建 CLI Config 结构体。
// authenticated 为 true 时填充完整认证信息，为 false 时仅保留 API 地址。
func buildCLIConfig(cfg *EnvConfig, authenticated bool) *config.Config {
	c := &config.Config{
		BkmsBaseURL: cfg.ApiUrl,
	}
	if authenticated {
		c.Username = cfg.Username
		c.AccessToken = cfg.Token
		c.Defaults = config.Defaults{WorkspaceID: cfg.WorkspaceID}
		c.BCS = config.BCSConfig{Token: cfg.BCSToken}
	}
	return c
}

// GenerateConfigFile 根据环境配置生成 CLI 配置文件。
// authenticated 为 true 时填充完整认证信息，为 false 时生成空白配置（用于未认证测试）。
func GenerateConfigFile(cfg *EnvConfig, authenticated bool) string {
	cfgPath := configFilePath()
	// 确保配置目录存在
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to create config directory: %v", err))
	}
	// 构建 CLI Config 结构体并序列化为 YAML
	data, err := yaml.Marshal(buildCLIConfig(cfg, authenticated))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to marshal config to YAML: %v", err))
	}
	if err = os.WriteFile(cfgPath, data, 0o644); err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to write config file: %v", err))
	}

	// 设置环境变量，确保 CLI 能找到配置文件
	if err = os.Setenv("BKMS_CLI_CONFIG", cfgPath); err != nil {
		ginkgo.Fail(fmt.Sprintf("failed to set env variable: %v", err))
	}

	authLabel := "authenticated"
	if !authenticated {
		authLabel = "unauthenticated"
	}
	Logf("CONFIG", "Generated %s CLI config file: %s", authLabel, cfgPath)

	return cfgPath
}

// CleanupConfigFile 清理 CLI 配置文件
func CleanupConfigFile() {
	cfgPath := configFilePath()
	if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
		Logf("CONFIG", "Failed to clean up config file: %v", err)
		return
	}
	Logf("CONFIG", "Cleaned up config file: %s", cfgPath)
}
