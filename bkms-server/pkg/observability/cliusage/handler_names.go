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
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// closureSegmentPattern 匹配闭包名称段，如 "func1"、"2"
var closureSegmentPattern = regexp.MustCompile(`^(func\d+|\d+)$`)

// HandlerNames handler 全名到 API 标识的映射。
// 默认取方法名，重名时取去掉模块前缀的全名，查不到时为 "METHOD route"。
type HandlerNames struct {
	names atomic.Pointer[map[string]string]
}

// NewHandlerNames 创建 HandlerNames
func NewHandlerNames() *HandlerNames {
	return &HandlerNames{}
}

// Load 由 basePath 下的路由构建映射，须在路由注册完成后调用
func (h *HandlerNames) Load(ctx context.Context, routes gin.RoutesInfo, basePath string) {
	byMethod := map[string][]string{}
	seen := map[string]bool{}
	for _, route := range routes {
		if !strings.HasPrefix(route.Path, basePath) || seen[route.Handler] {
			continue
		}
		seen[route.Handler] = true
		if name, ok := methodName(route.Handler); ok {
			byMethod[name] = append(byMethod[name], route.Handler)
		}
	}

	names := make(map[string]string, len(seen))
	for name, fullNames := range byMethod {
		if len(fullNames) == 1 {
			names[fullNames[0]] = name
			continue
		}
		for _, fullName := range fullNames {
			names[fullName] = qualifiedName(fullName)
		}
		log.WarnAttrs(ctx, "cli usage: duplicate handler method name, use qualified names instead",
			slog.String("method_name", name), slog.Any("handlers", fullNames))
	}
	h.names.Store(&names)
}

// Lookup 返回 API 标识
func (h *HandlerNames) Lookup(handlerName, method, route string) string {
	if names := h.names.Load(); names != nil {
		if name, ok := (*names)[handlerName]; ok {
			return name
		}
	}
	return method + " " + route
}

// methodName 取方法名，闭包返回 false
func methodName(fullName string) (string, bool) {
	s := strings.TrimSuffix(fullName, "-fm")
	segments := strings.Split(s[strings.LastIndex(s, "/")+1:], ".")
	for _, segment := range segments {
		if closureSegmentPattern.MatchString(segment) {
			return "", false
		}
	}
	name := segments[len(segments)-1]
	return name, name != ""
}

// qualifiedName 返回去掉模块前缀的全名，如 "deploy/handler.(*Handler).List"
func qualifiedName(fullName string) string {
	s := strings.TrimSuffix(fullName, "-fm")
	if _, after, ok := strings.Cut(s, "/pkg/"); ok {
		return after
	}
	return s
}
