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

// Package apm 提供 APM 观测辅助能力
package apm

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartClientSpan 创建 client span（span_name: "{system}/{operation}"），需配合 EndClientSpan 结束。
func StartClientSpan(ctx context.Context, system, operation string) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, fmt.Sprintf("%s/%s", system, operation),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("rpc.system", system),
			attribute.String("rpc.service", operation),
		),
	)
}

// EndClientSpan 结束 span 并记录响应状态码与错误。
func EndClientSpan(span trace.Span, resp *http.Response, err *error) {
	if resp != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
		span.SetAttributes(
			attribute.String("url.full",
				fmt.Sprintf("%s://%s%s", resp.Request.URL.Scheme, resp.Request.URL.Host, resp.Request.URL.Path)),
		)
	}
	if err != nil && *err != nil {
		span.RecordError(*err)
		span.SetStatus(codes.Error, (*err).Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}
