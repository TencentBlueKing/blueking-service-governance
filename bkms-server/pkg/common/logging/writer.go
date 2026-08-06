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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

const (
	defaultFileMaxSize    = 100
	defaultFileMaxBackups = 5
	// defaultFileMaxAge 默认最长保留天数（超过将由 lumberjack 删除）
	defaultFileMaxAge = 30
	// defaultLogDirPerm 是自动创建日志目录时使用的权限位
	defaultLogDirPerm = 0o755
)

func newMultiWriter(writerCfgs []config.LoggingWriterConfig) (io.Writer, error) {
	if len(writerCfgs) == 0 {
		return newWriter(defaultWriterName, config.LoggingWriterFileConfig{})
	}

	if err := validateFileWriterUniqueness(writerCfgs); err != nil {
		return nil, err
	}

	writers := make([]io.Writer, 0, len(writerCfgs))
	for _, writerCfg := range writerCfgs {
		writer, err := newWriter(writerCfg.WriterName, writerCfg.WriterConfig)
		if err != nil {
			return nil, err
		}
		writers = append(writers, writer)
	}
	if len(writers) == 1 {
		return writers[0], nil
	}
	return io.MultiWriter(writers...), nil
}

// validateFileWriterUniqueness 检查所有 file writer 的 filename 归一化后是否唯一，
// 避免多个 lumberjack 实例争抢同一文件导致日志切分/丢失。
func validateFileWriterUniqueness(writerCfgs []config.LoggingWriterConfig) error {
	seen := make(map[string]struct{}, len(writerCfgs))
	for _, cfg := range writerCfgs {
		if cfg.WriterName != WriterFile {
			continue
		}
		cleaned := filepath.Clean(cfg.WriterConfig.Filename)
		if _, dup := seen[cleaned]; dup {
			return errors.Errorf("duplicate file writer filename %s", cleaned)
		}
		seen[cleaned] = struct{}{}
	}
	return nil
}

func newWriter(name string, cfg config.LoggingWriterFileConfig) (io.Writer, error) {
	switch name {
	case "", WriterStdout:
		return os.Stdout, nil
	case WriterStderr:
		return os.Stderr, nil
	case WriterFile:
		return newRotateFileWriter(cfg)
	default:
		return nil, errors.Errorf("%s writer not supported", name)
	}
}

func newRotateFileWriter(cfg config.LoggingWriterFileConfig) (io.Writer, error) {
	if cfg.Filename == "" {
		return nil, errors.New("writer config must provide non-empty filename")
	}

	dir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(dir, defaultLogDirPerm); err != nil {
		return nil, errors.Wrapf(err, "create log dir %s", dir)
	}

	maxSize := cfg.MaxSize
	if maxSize <= 0 {
		maxSize = defaultFileMaxSize
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = defaultFileMaxBackups
	}
	if cfg.MaxAge < 0 {
		return nil, errors.Errorf("writer config maxAge %d is invalid", cfg.MaxAge)
	}
	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = defaultFileMaxAge
	}

	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		LocalTime:  true,
		Compress:   cfg.Compress,
	}, nil
}
