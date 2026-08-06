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

// Package httpcli 提供带 OTel 追踪的 HTTP 客户端工厂
package httpcli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	slogresty "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging/slog-resty"
)

// sensitiveQueryKeys 需要脱敏的关键词
var sensitiveQueryKeys = map[string]bool{
	"token":      true,
	"password":   true,
	"passwd":     true,
	"secret":     true,
	"ticket":     true,
	"key":        true,
	"sign":       true,
	"auth":       true,
	"credential": true,
}

type propagatingTransport struct {
	base http.RoundTripper
}

// NewRestyClient 创建带追踪和日志的 resty 客户端。
func NewRestyClient(ctx context.Context) *resty.Client {
	return resty.New().
		SetTransport(NewTransport()).
		SetLogger(slogresty.NewRestyLogger(ctx))
}

// NewTransport 返回带追踪的 RoundTripper。
func NewTransport() http.RoundTripper {
	return NewTransportWith(nil)
}

// NewTransportWith 同 NewTransport，可指定 base
func NewTransportWith(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(
		&propagatingTransport{base: base},
		otelhttp.WithFilter(requestFilter),
		otelhttp.WithSpanNameFormatter(spanNameFormatter),
	)
}

func (t *propagatingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(r)

	if span := trace.SpanFromContext(r.Context()); span.IsRecording() {
		span.SetAttributes(
			attribute.String("url.full", redactedURL(r.URL)),
		)
	}

	return resp, err
}

// redactedURL 仅对敏感 query 参数的非空值替换为 REDACTED
func redactedURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	// 复制一份 URL，避免修改原始对象
	copied := *u
	if copied.RawQuery == "" {
		return fmt.Sprintf("%s://%s%s", copied.Scheme, copied.Host, copied.Path)
	}

	redactedQuery := lo.MapValues(copied.Query(), func(values []string, key string) []string {
		if !isSensitiveKey(key) {
			return values
		}
		return lo.Map(values, func(v string, _ int) string {
			if v != "" {
				return "REDACTED"
			}
			return v
		})
	})
	copied.RawQuery = url.Values(redactedQuery).Encode()

	return fmt.Sprintf("%s://%s%s?%s", copied.Scheme, copied.Host, copied.Path, copied.RawQuery)
}

// isSensitiveKey 判断 query 参数名是否包含敏感关键词（忽略大小写）
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for k := range sensitiveQueryKeys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func spanNameFormatter(_ string, r *http.Request) string {
	return r.Method + " " + r.URL.Host + r.URL.Path
}

func requestFilter(r *http.Request) bool {
	switch r.URL.Path {
	case "/healthz", "/readyz", "/health":
		return false
	default:
		return true
	}
}
