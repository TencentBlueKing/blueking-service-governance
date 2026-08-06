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

package logging

import (
	"io"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

// InitDefaultLogger 初始化并注册全局默认 logger（通过 slog.SetDefault）。
func InitDefaultLogger(cfg config.LoggingConfig) error {
	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}

func newLogger(cfg config.LoggingConfig) (*slog.Logger, error) {
	cfg = normalizeConfig(cfg)

	// 先做参数校验，配置非法时直接返回错误，避免留下已打开的 lumberjack 文件句柄。
	level, err := toSlogLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	if err = validateHandlerName(cfg.HandlerName); err != nil {
		return nil, err
	}

	// 校验通过后再构造 writer 资源。
	w, err := newMultiWriter(cfg.Writers)
	if err != nil {
		return nil, errors.Wrap(err, "create log writer")
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}
	return slog.New(newHandler(cfg.HandlerName, w, handlerOpts)), nil
}

func validateHandlerName(name string) error {
	switch name {
	case HandlerText, HandlerJSON:
		return nil
	default:
		return errors.Errorf("%s handler not supported", name)
	}
}

func newHandler(name string, w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if name == HandlerText {
		return slog.NewTextHandler(w, opts)
	}
	return slog.NewJSONHandler(w, opts)
}

func normalizeConfig(cfg config.LoggingConfig) config.LoggingConfig {
	if cfg.Level == "" {
		cfg.Level = defaultLevel
	}
	if cfg.HandlerName == "" {
		cfg.HandlerName = defaultHandlerName
	}
	if len(cfg.Writers) == 0 {
		cfg.Writers = []config.LoggingWriterConfig{
			{WriterName: defaultWriterName},
		}
	}
	return cfg
}

func toSlogLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "", "INFO":
		return slog.LevelInfo, nil
	case "DEBUG":
		return slog.LevelDebug, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errors.Errorf("%s level not supported", level)
	}
}
