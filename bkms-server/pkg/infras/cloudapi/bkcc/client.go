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

// Package bkcc provides api client to bkcc（蓝鲸配置平台）
package bkcc

import (
	"context"

	"github.com/pkg/errors"
)

// ErrBusinessNotFound 业务未找到
var ErrBusinessNotFound = errors.New("bkcc: business not found")

// Client bk-cmdb API 客户端接口
type Client interface {
	// ListBusinesses 查询 SRE 有权限的业务信息
	ListBusinesses(ctx context.Context) ([]Business, error)

	// GetBusinessByID 按 bk_biz_id 精确查询单个业务，查不到时返回 ErrBusinessNotFound
	GetBusinessByID(ctx context.Context, bizID int64) (*Business, error)
}
