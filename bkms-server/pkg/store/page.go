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

package store

import (
	"context"
	"math"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Page ListPage 返回的一页结果
type Page[T any] struct {
	// Items 当前页的文档，无结果时为空切片
	Items []T
	// Total 匹配的文档总数；传入 WithoutTotal 时不统计，恒为 0
	Total int64
	// HasNext 是否还有下一页，两种模式下均有效
	HasNext bool
}

// ListPage 分页获取匹配的文档，page 从 1 开始，size 为每页数量，两者小于 1 时返回错误。
//
// 默认先统计匹配的总数，再取当前页；总数为 0 或 page 超出范围时不再查询文档。传入 WithoutTotal 时不统计总数，
// 多取一条用来判断是否还有下一页。
//
// 统计与查询是两次独立的请求，期间有并发写入时 Total、HasNext 与 Items 可能不一致（如 Total 多于实际可取到的条数），
// Total 只应作为近似值用于展示。
//
// ListPage 不会自动追加排序字段：排序键不唯一时（如只按 createdAt 排序），翻页时顺序相同的文档可能重复或遗漏，
// 调用方应让排序最终落在唯一字段上，例如 Desc("createdAt") 之后再加 Desc("_id")。例如：
//
//	// 第 2 页，每页 20 条，按创建时间倒序
//	page, err := records.ListPage(ctx, 2, 20, ByApp(appID), store.Desc("createdAt"), store.Desc("_id"))
//
// 支持的选项：CondOpt、SortOpt、ProjectOpt、PageOpt；数量由 size 决定，不接受 LimitOpt
func (s *GenericStore[T]) ListPage(ctx context.Context, page, size int64, opts ...Opt) (*Page[T], error) {
	if page < 1 || size < 1 {
		return nil, errors.Errorf("list page of %s: page and size must be >= 1, got page %d, size %d",
			s.coll.Name(), page, size)
	}
	// 保证 page * size 与 size + 1 都不会溢出
	if size == math.MaxInt64 || page-1 > (math.MaxInt64-size)/size {
		return nil, errors.Errorf("list page of %s: page %d is too large for size %d", s.coll.Name(), page, size)
	}
	q, err := s.parseOpts("list page of", opts, pageOpts)
	if err != nil {
		return nil, err
	}

	skip := (page - 1) * size
	result := &Page[T]{Items: []T{}}
	limit := size + 1
	if !q.withoutTotal {
		if result.Total, err = s.coll.CountDocuments(ctx, q.filter()); err != nil {
			return nil, errors.Wrapf(err, "count in %s, filter %v", s.coll.Name(), q.filter())
		}
		if skip >= result.Total {
			return result, nil
		}
		result.HasNext = skip+size < result.Total
		limit = size
	}

	items, err := s.find(ctx, q, options.Find().SetSkip(skip).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	if int64(len(items)) > size {
		result.HasNext = true
		items = items[:size]
	}
	result.Items = items
	return result, nil
}
