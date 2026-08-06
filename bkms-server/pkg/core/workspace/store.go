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

package workspace

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// 存储工作空间数据的 MongoDB 集合名称
const workspaceCollectionName = "workspaces"

// ErrWorkspaceNotFound 工作空间未找到时, 返回固定错误
var ErrWorkspaceNotFound = errors.New("workspace not found")

// ListOptions 工作空间查询参数
type ListOptions struct {
	// Keywork 模糊搜索, 匹配 id、displayName, 忽视大小写
	Keyword string `json:"keyword"`
	State   *State `json:"state"`
	// Sort 自定义排序规则，如果为空则使用函数内默认排序
	Sort bson.D `json:"-"`
}

// ListPageOptions 工作空间分页查询参数。
type ListPageOptions struct {
	ListOptions
	// Page 页码，从 1 开始。
	Page int64
	// PageSize 每页数量。
	PageSize int64
}

// WorkspaceStore 是用于管理工作空间的存储接口
type WorkspaceStore interface {
	// List 获取工作空间列表，由于每个用户的工作空间数量有限，不设分页
	List(ctx context.Context, opts *ListOptions) ([]Workspace, error)

	// ListWithPagination 获取工作空间分页列表，同时返回满足条件的总数，获取当前所有的workspace，需分页拉取
	ListWithPagination(ctx context.Context, opts *ListPageOptions) ([]Workspace, int64, error)

	// Get 通过 ID 获取工作空间
	Get(ctx context.Context, id string) (*Workspace, error)

	// Create 创建新的工作空间
	Create(ctx context.Context, workspace *Workspace) error

	// Update 更新已经存在的工作空间
	Update(ctx context.Context, workspace *Workspace) error

	// Delete 删除已经存在的工作空间
	Delete(ctx context.Context, id string) error

	// CountByState returns workspace counts grouped by lifecycle state for the matched filter.
	CountByState(ctx context.Context, opts *ListOptions) (map[State]int64, error)
}

var _ WorkspaceStore = &WorkspaceStoreMongo{}

// WorkspaceStoreMongo 是 WorkspaceStore 接口的 MongoDB 实现
type WorkspaceStoreMongo struct {
	collection *mongo.Collection
}

func (s *WorkspaceStoreMongo) buildListFilter(opts *ListOptions) bson.M {
	filter := bson.M{}
	if opts == nil {
		return filter
	}
	if opts.Keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword := regexp.QuoteMeta(opts.Keyword)
		filter["$or"] = []bson.M{
			{"id": bson.M{"$regex": keyword, "$options": "i"}}, // i 表示不区分大小写
			{"displayName": bson.M{"$regex": keyword, "$options": "i"}},
		}
	}
	if opts.State != nil {
		filter["state"] = *opts.State
	}
	return filter
}

// NewWorkspaceStoreMongo ...
func NewWorkspaceStoreMongo(client *mongo.Client, dbName string) (*WorkspaceStoreMongo, error) {
	coll := client.Database(dbName).Collection(workspaceCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	return &WorkspaceStoreMongo{collection: coll}, nil
}

// List 获取工作空间列表
func (s *WorkspaceStoreMongo) List(ctx context.Context, opts *ListOptions) ([]Workspace, error) {
	filter := s.buildListFilter(opts)
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "find workspaces")
	}
	defer cursor.Close(ctx)

	var workspaces []Workspace
	if err = cursor.All(ctx, &workspaces); err != nil {
		return nil, errors.Wrap(err, "decode workspaces")
	}
	return workspaces, nil
}

// ListWithPagination 获取工作空间分页列表，同时返回满足条件的总数
func (s *WorkspaceStoreMongo) ListWithPagination(
	ctx context.Context,
	opts *ListPageOptions,
) ([]Workspace, int64, error) {
	if opts == nil {
		opts = &ListPageOptions{}
	}
	filter := s.buildListFilter(&opts.ListOptions)

	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrap(err, "count workspaces")
	}

	// 使用自定义排序，如果未指定则使用默认排序
	sort := opts.Sort
	// 默认工作空间排序：按更新时间降序，相同则按 ID 升序（保证分页稳定）
	if sort == nil {
		sort = bson.D{
			{Key: "updatedAt", Value: -1},
			{Key: "id", Value: 1},
		}
	}

	findOpts := options.Find().
		SetSort(sort).
		SetSkip((opts.Page - 1) * opts.PageSize).
		SetLimit(opts.PageSize)
	cursor, err := s.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, errors.Wrap(err, "find workspaces with pagination")
	}
	defer cursor.Close(ctx)

	var workspaces []Workspace
	if err = cursor.All(ctx, &workspaces); err != nil {
		return nil, 0, errors.Wrap(err, "decode paginated workspaces")
	}
	return workspaces, total, nil
}

// Get 通过 ID 获取工作空间
func (s *WorkspaceStoreMongo) Get(ctx context.Context, id string) (*Workspace, error) {
	var workspace Workspace
	if err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(&workspace); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	return &workspace, nil
}

// Create 创建新的工作空间
func (s *WorkspaceStoreMongo) Create(ctx context.Context, workspace *Workspace) error {
	// 设置创建 / 更新时间
	timeNow := time.Now()
	// FIXME: 完成数据迁移后，可以移除该兼容逻辑，即创建时候直接设置 CreatedAt = timeNow
	if workspace.CreatedAt.IsZero() {
		workspace.CreatedAt = timeNow
	}
	workspace.UpdatedAt = timeNow

	if _, err := s.collection.InsertOne(ctx, workspace); err != nil {
		return err
	}
	return nil
}

// Update 更新已经存在的工作空间
func (s *WorkspaceStoreMongo) Update(ctx context.Context, workspace *Workspace) error {
	if _, err := s.collection.UpdateOne(
		ctx,
		bson.M{"id": workspace.ID},
		bson.M{"$set": bson.M{
			"displayName":       workspace.DisplayName,
			"description":       workspace.Description,
			"bkSystems":         workspace.BkSystems,
			"imageRegistryType": workspace.ImageRegistryType,
			"state":             workspace.State,
			"updater":           workspace.Updater,
			"updatedAt":         time.Now(),
		}},
	); err != nil {
		return err
	}
	return nil
}

// Delete 删除一个工作空间
func (s *WorkspaceStoreMongo) Delete(ctx context.Context, id string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

// workspaceCountByState is the decoded row from MongoDB $group aggregation by state.
type workspaceCountByState struct {
	// State decodes MongoDB $group output field "_id", which holds the grouping key "$state".
	State State `bson:"_id"`
	// Count is the number of workspaces in this state.
	Count int64 `bson:"count"`
}

// CountByState returns workspace counts grouped by lifecycle state for the matched filter.
func (s *WorkspaceStoreMongo) CountByState(ctx context.Context, opts *ListOptions) (map[State]int64, error) {
	filter := s.buildListFilter(opts)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$state",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate workspace counts by state")
	}
	defer cursor.Close(ctx)

	var results []workspaceCountByState
	if err := cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode workspace counts by state")
	}

	counts := lo.SliceToMap(results, func(item workspaceCountByState) (State, int64) {
		return item.State, item.Count
	})
	return counts, nil
}
