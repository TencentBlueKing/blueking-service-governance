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

package ginutils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSON writes JSON response data.
func JSON(c *gin.Context, status int, obj any) {
	if obj == nil {
		c.Status(status)
		return
	}
	c.JSON(status, obj)
}

// OK writes a 200 JSON response.
func OK(c *gin.Context, obj any) {
	JSON(c, http.StatusOK, obj)
}

// Created writes a 201 JSON response.
func Created(c *gin.Context, obj any) {
	JSON(c, http.StatusCreated, obj)
}

// NoContent writes a 204 No Content response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
