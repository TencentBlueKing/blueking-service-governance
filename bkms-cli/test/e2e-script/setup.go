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

package e2escript_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	cenv "github.com/caarlos0/env/v11"
	"github.com/rogpeppe/go-internal/testscript"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
)

// envConfig 持有从环境变量中读取的 E2E 配置，缺失任一项则 Parse 报错。
type envConfig struct {
	APIUrl      string `env:"BKMS_API_URL,required"`
	Username    string `env:"BKMS_USERNAME,required"`
	Token       string `env:"BKMS_TOKEN,required"`
	WorkspaceID string `env:"BKMS_WORKSPACE_ID,required"`
	AppID       string `env:"BKMS_APP_ID,required"`
	EnvName     string `env:"BKMS_ENV_NAME,required"`
}

// setup 是已认证场景的 Setup 钩子，每个脚本执行前调用一次。
func setup(env *testscript.Env) error {
	cfg, err := loadEnvConfig()
	if err != nil {
		return err
	}

	binPath, err := findBinary()
	if err != nil {
		return err
	}

	cfgPath, err := writeConfigFile(env.WorkDir, cfg, true)
	if err != nil {
		return err
	}

	injectEnv(env, cfg, binPath, cfgPath)

	return loginAndSetWorkspace(binPath, cfgPath, cfg)
}

// setupUnauth 是未认证场景的 Setup 钩子：配置仅含 API 地址，不执行 login。
func setupUnauth(env *testscript.Env) error {
	cfg, err := loadEnvConfig()
	if err != nil {
		return err
	}

	binPath, err := findBinary()
	if err != nil {
		return err
	}

	cfgPath, err := writeConfigFile(env.WorkDir, cfg, false)
	if err != nil {
		return err
	}

	injectEnv(env, cfg, binPath, cfgPath)
	return nil
}

// loadEnvConfig 解析所有必填 BKMS_* 环境变量。
func loadEnvConfig() (*envConfig, error) {
	cfg := new(envConfig)
	if err := cenv.Parse(cfg); err != nil {
		return nil, fmt.Errorf("missing required env vars: %w", err)
	}
	return cfg, nil
}

// findBinary 查找 bkms-cli 可执行文件，优先取 BKMS_CLI_BIN，
// 否则在 build 目录（可由 BKMS_CLI_BUILD_DIR 覆盖）中按候选名依次探测。
func findBinary() (string, error) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	if p := os.Getenv("BKMS_CLI_BIN"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("BKMS_CLI_BIN=%q: file not found", p)
		}
		return p, nil
	}

	buildDir := os.Getenv("BKMS_CLI_BUILD_DIR")
	if buildDir == "" {
		buildDir = "build"
	}

	candidates := []string{
		filepath.Join(buildDir, "bkms-cli-e2e"+ext),
		filepath.Join(buildDir, fmt.Sprintf("bkms-cli-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)),
		filepath.Join(buildDir, "bkms-cli"+ext),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, err := filepath.Abs(p)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
	}
	return "", fmt.Errorf("bkms-cli binary not found; set BKMS_CLI_BIN or run 'make e2e-build' first")
}

// writeConfigFile 在 workDir 中生成 CLI 配置文件并返回其路径。
// authenticated=false 时只写 API 地址，不写认证信息。
func writeConfigFile(workDir string, cfg *envConfig, authenticated bool) (string, error) {
	c := &config.Config{BkmsBaseURL: cfg.APIUrl}
	if authenticated {
		c.Username = cfg.Username
		c.AccessToken = cfg.Token
		c.Defaults = config.Defaults{WorkspaceID: cfg.WorkspaceID}
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	cfgPath := filepath.Join(workDir, "bkms-e2e-config.yaml")
	if err = os.WriteFile(cfgPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}
	return cfgPath, nil
}

// injectEnv 注入脚本所需的全部环境变量（testscript 默认只传递 PATH）。
// UNIQUE 为脚本独享时间戳，用于生成无冲突的资源名。
func injectEnv(env *testscript.Env, cfg *envConfig, binPath, cfgPath string) {
	unique := fmt.Sprintf("e2e-%d", time.Now().UnixMilli())

	env.Setenv("BKMS_CLI_BIN", binPath)
	env.Setenv("BKMS_CLI_CONFIG", cfgPath)
	env.Setenv("BKMS_API_URL", cfg.APIUrl)
	env.Setenv("BKMS_USERNAME", cfg.Username)
	env.Setenv("BKMS_TOKEN", cfg.Token)
	env.Setenv("BKMS_WORKSPACE_ID", cfg.WorkspaceID)
	env.Setenv("BKMS_APP_ID", cfg.AppID)
	env.Setenv("BKMS_ENV_NAME", cfg.EnvName)
	env.Setenv("UNIQUE", unique)

	// txtar 内嵌文件段中 $VAR 不展开，含唯一名的 fixture 只能在此生成
	env.Setenv("APP_SPEC", writeAppSpec(env.WorkDir, unique))
}

// writeAppSpec 生成带唯一 name 的 trpc app spec 文件并返回其路径。
// 写入失败时返回空串，脚本会因找不到文件而自然失败。
func writeAppSpec(workDir, unique string) string {
	spec := fmt.Sprintf(`name: %s
framework: trpc
language: go
buildConfig:
  sourceType: imageRegistry
  imageBuildConfig:
    name: mirrors.example.com/test/e2e-image
appModelSpec:
  command: ["/usr/local/trpc/bin/server"]
frameworkConfigFile:
  name: trpc_go.yaml
  mountDir: /usr/local/trpc/conf
`, unique)

	path := filepath.Join(workDir, "app-spec.yaml")
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		return ""
	}
	return path
}

// loginAndSetWorkspace 执行 login 与 workspace set，使后续脚本处于已认证状态。
func loginAndSetWorkspace(binPath, cfgPath string, cfg *envConfig) error {
	run := func(args ...string) error {
		cmd := exec.Command(binPath, args...) //nolint:gosec
		cmd.Env = append(os.Environ(), "BKMS_CLI_CONFIG="+cfgPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("setup cmd %s: %w\noutput: %s",
				strings.Join(args, " "), err, out)
		}
		return nil
	}

	if err := run("login", "--access-token", cfg.Token); err != nil {
		return err
	}
	return run("workspace", "set", cfg.WorkspaceID)
}
