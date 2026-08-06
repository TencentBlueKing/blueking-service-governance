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

package audit

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// 存储操作记录数据的 MongoDB 集合名称
const collectionName = "operation_records"

// OperationRecordStore 是用于管理操作记录的存储接口
type OperationRecordStore interface {
	// List 获取操作记录列表，支持分页 & 关键字查询
	// opts: 查询选项
	// 返回值：
	// - records: 操作记录列表
	// - total: 总记录数
	// - err: 错误信息
	List(ctx context.Context, opts ListOptions) ([]OperationRecord, int64, error)

	// Get 通过 ID 获取操作记录
	Get(ctx context.Context, id string) (*OperationRecord, error)

	// Create 创建新的操作记录
	Create(ctx context.Context, record *OperationRecord) (string, error)

	// Update 更新已经存在的操作记录
	Update(ctx context.Context, record *OperationRecord) error

	// GetLatestOperationTimesByWorkspacesForUser 批量查询指定用户在各 workspace 的最近操作时间
	GetLatestOperationTimesByWorkspacesForUser(
		ctx context.Context, workspaceIDs []string, username string,
	) (map[string]time.Time, error)

	// GetLatestOperationTimesByAppsForUser 批量查询指定用户在指定 workspace 下各 app 的最近操作时间
	GetLatestOperationTimesByAppsForUser(
		ctx context.Context, workspaceID string, appIDs []string, username string,
	) (map[string]time.Time, error)

	// DeleteAll 删除所有操作记录并保留集合及索引（仅用于测试）
	DeleteAll(ctx context.Context) error
}

var _ OperationRecordStore = &OperationRecordStoreMongo{}

// OperationRecordStoreMongo 是 OperationRecordStore 接口的 MongoDB 实现
type OperationRecordStoreMongo struct {
	collection *mongo.Collection
}

// NewOperationRecordStoreMongo 创建新的 OperationRecordStoreMongo 实例
func NewOperationRecordStoreMongo(client *mongo.Client, dbName string) (*OperationRecordStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 查询提速：username + group.workspaceID + createdAt(倒序)
	// - 查询提速：username + group.workspaceID + group.appID + createdAt(倒序)
	return &OperationRecordStoreMongo{collection: coll}, nil
}

// DeleteAll 删除所有操作记录并保留集合及索引（仅用于测试）
func (s *OperationRecordStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}

// ListOptions 列表查询选项
type ListOptions struct {
	// 分组参数
	// WorkspaceID 工作空间 ID（分组过滤）
	WorkspaceID string
	// AppID 应用 ID（分组过滤）
	AppID string
	// EnvName 环境名称（分组过滤，需要搭配 workspaceID 使用）
	EnvName string

	// 时间范围
	StartedAt time.Time
	EndedAt   time.Time

	// Username 操作人
	Username string
	// OpType 操作类型
	OpType OperationType
	// ResourceType 资源类型
	ResType ResourceType
	// Result 操作结果
	Result Result

	// 分页参数
	Page, PageSize int64
}

// AsFilter 将选项转换为过滤器
func (o *ListOptions) AsFilter() bson.M {
	filter := bson.M{}
	// 分组过滤
	if o.WorkspaceID != "" {
		filter["group.workspaceID"] = o.WorkspaceID
	}
	if o.AppID != "" {
		filter["group.appID"] = o.AppID
	}
	if o.EnvName != "" {
		filter["group.envName"] = o.EnvName
	}
	// 时间范围过滤
	if o.StartedAt.Before(o.EndedAt) {
		filter["createdAt"] = bson.M{
			"$gte": o.StartedAt,
			"$lte": o.EndedAt,
		}
	}
	// 数据字段过滤
	if o.Username != "" {
		filter["username"] = buildCaseInsensitiveMatch(o.Username)
	}
	if o.OpType != "" {
		filter["operationType"] = o.OpType
	}
	if o.ResType != "" {
		filter["resourceType"] = o.ResType
	}
	if o.Result != "" {
		filter["result"] = o.Result
	}
	return filter
}

// List 获取操作记录列表
func (s *OperationRecordStoreMongo) List(ctx context.Context, opts ListOptions) ([]OperationRecord, int64, error) {
	// 统计总数
	total, err := s.collection.CountDocuments(ctx, opts.AsFilter())
	if err != nil {
		return nil, 0, errors.Wrap(err, "count operation records")
	}

	// 分页查询
	cursor, err := s.collection.Find(
		ctx,
		opts.AsFilter(),
		options.Find().
			SetLimit(opts.PageSize).
			SetSkip((opts.Page-1)*opts.PageSize).
			SetSort(bson.D{{"createdAt", -1}}),
	)
	if err != nil {
		return nil, 0, errors.Wrap(err, "find operation records")
	}
	defer cursor.Close(ctx)

	var records []OperationRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, errors.Wrap(err, "decode operation records")
	}
	return records, total, nil
}

// Get 通过 ID 获取操作记录
func (s *OperationRecordStoreMongo) Get(ctx context.Context, id string) (*OperationRecord, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid format id %s", id)
	}

	var record OperationRecord
	if err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(err, "operation record for id %s not found", id)
		}
		return nil, err
	}
	return &record, nil
}

// Create 创建新的操作记录
func (s *OperationRecordStoreMongo) Create(ctx context.Context, record *OperationRecord) (string, error) {
	ret, err := s.collection.InsertOne(ctx, record)
	if err != nil {
		return "", err
	}
	if oid, ok := ret.InsertedID.(bson.ObjectID); ok {
		return oid.Hex(), nil
	}
	return "", errors.New("failed to get inserted ID")
}

// Update 更新已经存在的操作记录
// 注意：只更新可变字段，不更新创建时的固定字段（如 username、type、resourceType 等）
func (s *OperationRecordStoreMongo) Update(ctx context.Context, record *OperationRecord) error {
	updateDoc := bson.M{"updatedAt": time.Now()}
	// 操作结果（从 running → success / failed / canceled）
	if record.Result != "" {
		updateDoc["result"] = record.Result
	}
	// 操作前数据（可能需要补充）
	if record.Data.Before != nil {
		updateDoc["data.before"] = record.Data.Before
	}
	// 操作后数据（可能需要补充）
	if record.Data.After != nil {
		updateDoc["data.after"] = record.Data.After
	}
	// 额外信息
	if record.Data.Extras != nil {
		updateDoc["data.extras"] = record.Data.Extras
	}

	// 更新记录
	if _, err := s.collection.UpdateOne(ctx, bson.M{"_id": record.ID}, bson.M{"$set": updateDoc}); err != nil {
		return err
	}
	return nil
}

// latestTimeResult 用于解码 aggregation 结果
type latestTimeResult struct {
	ID       string    `bson:"_id"`
	LatestAt time.Time `bson:"latestAt"`
}

// GetLatestOperationTimesByWorkspacesForUser 批量查询指定用户在各 workspace 的最近操作时间
func (s *OperationRecordStoreMongo) GetLatestOperationTimesByWorkspacesForUser(
	ctx context.Context, workspaceIDs []string, username string,
) (map[string]time.Time, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"group.workspaceID": bson.M{"$in": workspaceIDs},
			"username":          username,
		}}},
		{{"$group", bson.M{
			"_id":      "$group.workspaceID",
			"latestAt": bson.M{"$max": "$createdAt"},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate latest operation times by workspaces for user")
	}
	defer cursor.Close(ctx)

	var results []latestTimeResult
	if err = cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode aggregation results")
	}

	m := make(map[string]time.Time, len(results))
	for _, r := range results {
		m[r.ID] = r.LatestAt
	}
	return m, nil
}

// GetLatestOperationTimesByAppsForUser 批量查询指定用户在指定 workspace 下各 app 的最近操作时间
func (s *OperationRecordStoreMongo) GetLatestOperationTimesByAppsForUser(
	ctx context.Context, workspaceID string, appIDs []string, username string,
) (map[string]time.Time, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"group.workspaceID": workspaceID,
			"group.appID":       bson.M{"$in": appIDs},
			"username":          username,
		}}},
		{{"$group", bson.M{
			"_id":      "$group.appID",
			"latestAt": bson.M{"$max": "$createdAt"},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate latest operation times by apps for user")
	}
	defer cursor.Close(ctx)

	var results []latestTimeResult
	if err = cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode aggregation results")
	}

	m := make(map[string]time.Time, len(results))
	for _, r := range results {
		m[r.ID] = r.LatestAt
	}
	return m, nil
}

// buildCaseInsensitiveMatch 构造过滤器，转义特殊字符，忽略大小写进行模糊匹配
func buildCaseInsensitiveMatch(s string) bson.M {
	pattern := regexp.QuoteMeta(s)
	return bson.M{
		"$regex":   pattern,
		"$options": "i",
	}
}
