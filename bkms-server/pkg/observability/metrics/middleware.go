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

package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
)

const unknownRoute = "unknown"

// GinMiddleware 记录 Gin 入站请求的 Prometheus 指标
//
// 指标在 c.Next 之后上报，确保能拿到最终响应状态码；未匹配到 Gin 路由模板时统一归类为 unknown，
// 避免 404 等场景直接使用原始 URL 造成高基数标签
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		handler := c.FullPath()
		if handler == "" {
			handler = unknownRoute
		}
		ReportServerRequestMetric(handler, c.Request.Method, c.Writer.Status(), started)
	}
}
