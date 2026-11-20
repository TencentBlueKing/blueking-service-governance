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

package apitracking

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/urlmask"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

const (
	// EventName 事件名
	EventName = "api_call"
	// schemaVersion 字段版本，字段变更时递增
	schemaVersion = 1
)

type userInfo struct {
	username string
	tenantID string
	cred     auth.UserCredential
}

// Attrs 返回追踪字段，未匹配路由时为 nil；须在 c.Next() 后调用
func Attrs(c *gin.Context, names *HandlerNames) []slog.Attr {
	route := c.FullPath()
	if route == "" {
		return nil
	}
	req := c.Request
	cli, isCLI := IdentifyCLI(req)
	user := requestUser(req.Context())
	client := ClientCLI
	if !isCLI {
		client = clientType(user.cred)
	}
	return []slog.Attr{
		slog.String("event", EventName),
		slog.Int("schema", schemaVersion),
		slog.String("client", client),
		slog.String("cli_version", cli.Version),
		slog.String("os", cli.OS),
		slog.String("arch", cli.Arch),
		slog.String("device_id", cli.DeviceID),
		slog.String("user_agent", truncate(req.UserAgent(), maxUserAgentLen)),
		slog.String("username", user.username),
		slog.String("tenant_id", user.tenantID),
		slog.String("client_ip", c.ClientIP()),
		slog.String("handler", names.Lookup(c.HandlerName(), req.Method, route)),
		slog.String("method", req.Method),
		slog.String("route", route),
		slog.String("path", req.URL.Path),
		slog.String("query", urlmask.Query(req.URL.RawQuery)),
		slog.Int("status", c.Writer.Status()),
	}
}

// requestUser 返回用户名、租户与凭据，认证失败时用户名与租户为空
func requestUser(ctx context.Context) userInfo {
	var info userInfo
	if result, ok := auth.GetResult(ctx); ok {
		info.username = result.RequestUser.GetID()
		info.tenantID = result.RequestUser.GetTenantID()
		info.cred = result.RequestUser.Credential()
	}
	if resolved, ok := tenant.GetTenantID(ctx); ok {
		info.tenantID = resolved
	}
	return info
}
