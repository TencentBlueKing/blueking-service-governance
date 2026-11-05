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

// Package image 应用镜像相关业务逻辑
package image

import (
	"context"
	"strings"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// PromoteResult 单个镜像 tag 的晋级结果
type PromoteResult struct {
	// Tag 镜像 tag
	Tag string
	// Err 晋级失败原因，成功时为 nil
	Err error
}

// NormalizeTags 去除空白与重复 tag，保持输入顺序
func NormalizeTags(tags []string) []string {
	items := lo.FilterMap(tags, func(tag string, _ int) (string, bool) {
		tag = strings.TrimSpace(tag)
		return tag, tag != ""
	})
	return lo.Uniq(items)
}

// PromoteImages 逐个晋级镜像 tag，单个 tag 失败不影响其余 tag 继续晋级。
// tag 是否存在由服务端校验；ctx 被取消后剩余 tag 不再请求，直接记为取消。
func PromoteImages(ctx context.Context, cli client.Client, appID string, tags []string) []PromoteResult {
	results := make([]PromoteResult, 0, len(tags))
	for _, tag := range tags {
		if ctxErr := ctx.Err(); ctxErr != nil {
			results = append(results, PromoteResult{Tag: tag, Err: ctxErr})
			continue
		}
		// client 返回的错误已包含操作名与 tag，这里不再重复包装
		results = append(results, PromoteResult{Tag: tag, Err: cli.PromoteAppImage(ctx, appID, tag)})
	}
	return results
}

// FailedPromoteResults 返回晋级失败的结果
func FailedPromoteResults(results []PromoteResult) []PromoteResult {
	return lo.Filter(results, func(r PromoteResult, _ int) bool { return r.Err != nil })
}
