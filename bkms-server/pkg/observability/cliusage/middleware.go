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

package cliusage

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

const (
	// EventName 事件名
	EventName = "cli_api_call"
	// schemaVersion 字段版本，字段变更时递增
	schemaVersion = 1
)

// Middleware 为 bkms-cli 请求输出事件日志。
// 须挂在认证之前：c.Next() 返回后，c.Request 已被认证中间件替换为带用户信息的 ctx，
// 认证失败的请求也能记录。
func Middleware(names *HandlerNames) gin.HandlerFunc {
	return func(c *gin.Context) {
		client, ok := Identify(c.Request)
		if !ok {
			c.Next()
			return
		}

		c.Next()

		route := c.FullPath()
		if route == "" {
			return
		}
		ctx := c.Request.Context()
		username, tenantID := requestUser(ctx)
		method := c.Request.Method
		log.InfoAttrs(ctx, EventName,
			slog.String("event", EventName),
			slog.Int("schema", schemaVersion),
			slog.String("cli_version", client.Version),
			slog.String("os", client.OS),
			slog.String("arch", client.Arch),
			slog.String("device_id", client.DeviceID),
			slog.String("username", username),
			slog.String("tenant_id", tenantID),
			slog.String("handler", names.Lookup(c.HandlerName(), method, route)),
			slog.String("method", method),
			slog.String("route", route),
			slog.Int("status", c.Writer.Status()),
		)
	}
}

// requestUser 返回用户名与租户，认证失败时为空
func requestUser(ctx context.Context) (username, tenantID string) {
	if result, ok := auth.GetResult(ctx); ok {
		username = result.RequestUser.GetID()
		tenantID = result.RequestUser.GetTenantID()
	}
	if resolved, ok := tenant.GetTenantID(ctx); ok {
		tenantID = resolved
	}
	return username, tenantID
}
