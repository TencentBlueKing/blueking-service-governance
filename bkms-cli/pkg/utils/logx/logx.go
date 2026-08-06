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

// Package logx 封装 bkms-cli 使用的 slog 全局日志器及日志级别切换工具
package logx

import (
	"log/slog"
	"os"
	"strings"

	"github.com/pkg/errors"
)

var levelVar slog.LevelVar

func init() {
	levelVar.Set(slog.LevelInfo)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &levelVar,
	})))
}

// SetLevel updates the global slog level.
func SetLevel(level string) error {
	parsed, err := ParseLevel(level)
	if err != nil {
		return err
	}
	levelVar.Set(parsed)
	return nil
}

// ParseLevel parses a user provided log level string.
func ParseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, errors.Errorf("unsupported log level: %s", level)
	}
}
