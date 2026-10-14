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
	"strconv"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ListLatestBy 按 groupBy 中的字段分组，返回每组中按 SortOpt 排在第一位的文档，例如每个应用最近的一条部署记录：
//
//	records.ListLatestBy(ctx, []string{"appID"}, InApps(appIDs), store.Asc("appID"), store.Desc("createdAt"))
//
// 必须传入 SortOpt，返回的文档同样按它排序，无结果时返回空切片；分组字段在文档中不存在时这些文档归为一组。
// 排序发生在分组之前，让 SortOpt 以分组字段开头并为（分组字段, 排序字段）建立索引，可以避免内存排序。
// 聚合没有显式设置 allowDiskUse：用不上索引时，$sort 的数据超过内存限制（默认 100MB）后是落盘还是报错
// 取决于服务端配置（MongoDB 6.0 起默认落盘，更早的版本报错），匹配的文档较多时应建立上述索引或用 CondOpt 缩小范围。
// 支持的选项：CondOpt、SortOpt、ProjectOpt
func (s *GenericStore[T]) ListLatestBy(ctx context.Context, groupBy []string, opts ...Opt) ([]T, error) {
	q, err := s.parseOpts("list latest in", opts, latestOpts)
	if err != nil {
		return nil, err
	}
	if len(groupBy) == 0 {
		return nil, errors.Errorf("list latest in %s: fields to group by are required", s.coll.Name())
	}
	if len(q.sort) == 0 {
		return nil, errors.Errorf(
			"list latest in %s: a sort option is required to decide the latest one",
			s.coll.Name(),
		)
	}
	// 分组字段可能是嵌套路径，不能直接作为 _id 文档的 key（不允许包含 "."），按序号命名
	groupID := make(bson.D, 0, len(groupBy))
	for i, field := range groupBy {
		var key string
		if key, err = s.schema.resolveKey(field); err != nil {
			return nil, errors.Wrapf(err, "list latest in %s", s.coll.Name())
		}
		groupID = append(groupID, bson.E{Key: "k" + strconv.Itoa(i), Value: "$" + key})
	}

	pipeline := mongo.Pipeline{}
	if filter := q.filter(); len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: q.sort}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: groupID},
			{Key: "doc", Value: bson.D{{Key: "$first", Value: "$$ROOT"}}},
		}}},
		bson.D{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$doc"}}}},
		bson.D{{Key: "$sort", Value: q.sort}},
	)
	if proj := q.projection(); proj != nil {
		pipeline = append(pipeline, bson.D{{Key: "$project", Value: proj}})
	}
	cur, err := s.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrapf(err, "list latest in %s, filter %v", s.coll.Name(), q.filter())
	}
	return s.decodeAll(ctx, cur)
}

// Aggregate 在 s 的集合上执行聚合管道，将每个结果文档解码为 R，用于分组统计、取去重组合等读操作；
// 管道（即统计口径）由领域层定义。结果不经过 Decode 钩子，需要读出模型时请使用 List / ListLatestBy。
// 无结果时返回空切片。例如：
//
//	type appCount struct {
//		WorkspaceID string `bson:"_id"`
//		Count       int64  `bson:"count"`
//	}
//	rows, err := store.Aggregate[appCount](ctx, apps, mongo.Pipeline{
//		{{Key: "$match", Value: bson.M{"workspaceID": bson.M{"$in": ids}}}},
//		{{Key: "$group", Value: bson.M{"_id": "$workspaceID", "count": bson.M{"$sum": 1}}}},
//	})
//
// opts 中的条件会作为管道的第一个 $match 阶段（字段名按模型解析），便于复用领域层的具名 Opt，例如：
//
//	rows, err := store.Aggregate[stateCount](ctx, workspaces, mongo.Pipeline{
//		{{Key: "$group", Value: bson.M{"_id": "$state", "count": bson.M{"$sum": 1}}}},
//	}, workspace.KeywordMatches(keyword))
//
// 支持的选项：CondOpt
func Aggregate[R, T any](ctx context.Context, s *GenericStore[T], pipeline mongo.Pipeline, opts ...Opt) ([]R, error) {
	q, err := s.parseOpts("aggregate", opts, condOpts)
	if err != nil {
		return nil, err
	}
	if filter := q.filter(); len(filter) > 0 {
		pipeline = append(mongo.Pipeline{{{Key: "$match", Value: filter}}}, pipeline...)
	}
	cur, err := s.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrapf(err, "aggregate %s", s.coll.Name())
	}
	rows := []R{}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, errors.Wrapf(err, "decode aggregation results from %s", s.coll.Name())
	}
	return rows, nil
}
