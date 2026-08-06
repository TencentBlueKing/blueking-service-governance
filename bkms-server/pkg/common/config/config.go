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

// Package config 管理配置项，支持从配置文件中读取配置
package config

import (
	"cmp"
	"context"
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var validate *validator.Validate

const (
	// HTTPServerReleaseMode 等同于 gin.ReleaseMode，为了避免依赖 gin 包而定义的常量
	HTTPServerReleaseMode = "release"
	// DefaultHTTPServerReadHeaderTimeout 是 HTTP Server 读取请求头超时时间，单位：秒
	DefaultHTTPServerReadHeaderTimeout = 10
	// DefaultHTTPServerReadTimeout 是 HTTP Server 读取完整请求超时时间，单位：秒
	DefaultHTTPServerReadTimeout = 65
	// DefaultHTTPServerWriteTimeout 是 HTTP Server 写响应超时时间，单位：秒
	DefaultHTTPServerWriteTimeout = 65
	// DefaultHTTPServerIdleTimeout 是 HTTP Server keep-alive 空闲连接超时时间，单位：秒
	DefaultHTTPServerIdleTimeout = 120
	// DefaultHTTPServerShutdownTimeout 是进程退出时等待 HTTP Server 优雅关闭的最长时间，单位：秒
	DefaultHTTPServerShutdownTimeout = 30
)

// use a single instance of Validate, it caches struct info
func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

var G *Config

// Load 加载项目配置，返回配置对象和错误信息（如果有的话），同时也会设置模块级全部变量
// `G` 以方便其他模块读取。
func Load(ctx context.Context, cfgFile string) (*Config, error) {
	var cfg Config

	// 检查配置文件是否存在
	if _, err := os.Stat(cfgFile); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("config file %s not found", cfgFile)
		}
		return nil, errors.Wrapf(err, "config file %s unavailable", cfgFile)
	}

	// 使用 viper 从 cfgFile 加载配置
	vp := viper.New()
	vp.SetConfigFile(cfgFile)
	setDefaultConfigValues(vp)

	// logger 尚未初始化，使用 slog 默认 handler（Go 1.21+ 默认输出到 stderr），
	// 确保配置错误场景下仍能看到启动日志。
	logger := slog.Default()
	logger.Info("loading config from file", "filename", cfgFile)

	if err := vp.ReadInConfig(); err != nil {
		return nil, errors.Wrapf(err, "read config file %s", cfgFile)
	}

	if err := vp.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrapf(err, "unmarshal config file %s", cfgFile)
	}

	if cfg.Encrypt.Secret != "" {
		// base64 解码
		bytes, err := base64.StdEncoding.DecodeString(cfg.Encrypt.Secret)
		if err != nil {
			return nil, errors.Wrapf(err, "decode encrypt secret")
		}
		cfg.Encrypt.Secret = string(bytes)
	}

	if err := validate.Struct(cfg.Metrics); err != nil {
		return nil, errors.Wrap(err, "validate MetricsConfig")
	}

	if err := validate.Struct(cfg.HTTPServer); err != nil {
		return nil, errors.Wrap(err, "validate HTTPServerConfig")
	}

	// Set default values
	cfg.Account.AuthEnvName = cmp.Or(cfg.Account.AuthEnvName, "prod")
	cfg.Account.BackendType = cmp.Or(cfg.Account.BackendType, "bk_token")

	if err := validate.Struct(cfg.Account); err != nil {
		return nil, errors.Wrap(err, "validate AccountConfig")
	}

	// 设置全局环境变量
	G = &cfg
	return &cfg, nil
}

// setDefaultConfigValues 设置可选配置的默认值，必填配置仍需由配置文件显式提供
func setDefaultConfigValues(vp *viper.Viper) {
	vp.SetDefault("httpServer.mode", HTTPServerReleaseMode)
	vp.SetDefault("httpServer.readHeaderTimeout", DefaultHTTPServerReadHeaderTimeout)
	vp.SetDefault("httpServer.readTimeout", DefaultHTTPServerReadTimeout)
	vp.SetDefault("httpServer.writeTimeout", DefaultHTTPServerWriteTimeout)
	vp.SetDefault("httpServer.idleTimeout", DefaultHTTPServerIdleTimeout)
	vp.SetDefault("httpServer.shutdownTimeout", DefaultHTTPServerShutdownTimeout)
}
