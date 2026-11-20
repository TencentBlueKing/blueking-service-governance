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

package apm

import (
	"context"
	"log/slog"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/urlmask"
)

// maskHandler 脱敏 sloggin 访问日志 request 分组中的 query 与 referer
type maskHandler struct {
	slog.Handler
}

// Handle 脱敏后交给下游 handler
func (h maskHandler) Handle(ctx context.Context, r slog.Record) error {
	masked := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		masked.AddAttrs(maskRequestAttr(a))
		return true
	})
	return h.Handler.Handle(ctx, masked)
}

// WithAttrs 实现 slog.Handler
func (h maskHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return maskHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup 实现 slog.Handler
func (h maskHandler) WithGroup(name string) slog.Handler {
	return maskHandler{h.Handler.WithGroup(name)}
}

func maskRequestAttr(a slog.Attr) slog.Attr {
	if a.Key != "request" || a.Value.Kind() != slog.KindGroup {
		return a
	}
	group := a.Value.Group()
	attrs := make([]slog.Attr, len(group))
	for i, attr := range group {
		switch attr.Key {
		case "query":
			attr.Value = slog.StringValue(urlmask.Query(attr.Value.String()))
		case "referer":
			attr.Value = slog.StringValue(urlmask.URL(attr.Value.String()))
		}
		attrs[i] = attr
	}
	return slog.Attr{Key: a.Key, Value: slog.GroupValue(attrs...)}
}
