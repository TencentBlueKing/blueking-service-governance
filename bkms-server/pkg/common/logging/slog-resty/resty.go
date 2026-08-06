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

package slogresty

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// RestyLogger 将 resty 日志转发到统一日志封装。
type RestyLogger struct {
	ctx context.Context
}

var _ resty.Logger = (*RestyLogger)(nil)

// NewRestyLogger 创建 RestyLogger 实例。
func NewRestyLogger(ctx context.Context) *RestyLogger {
	return &RestyLogger{ctx: ctx}
}

// Errorf 实现 resty.Logger 接口，转发到统一日志封装。
func (l *RestyLogger) Errorf(format string, v ...any) {
	logging.Log(l.ctx, slog.LevelError, fmt.Sprintf(format, v...))
}

// Warnf 实现 resty.Logger 接口，转发到统一日志封装。
func (l *RestyLogger) Warnf(format string, v ...any) {
	logging.Log(l.ctx, slog.LevelWarn, fmt.Sprintf(format, v...))
}

// Debugf 实现 resty.Logger 接口，转发到统一日志封装。
func (l *RestyLogger) Debugf(format string, v ...any) {
	logging.Log(l.ctx, slog.LevelDebug, fmt.Sprintf(format, v...))
}
