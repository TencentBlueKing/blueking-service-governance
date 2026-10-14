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
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrOperationRecordNotFound is returned when an operation record is not found.
var ErrOperationRecordNotFound = errors.New("operation record not found")

// OperationRecordStore 操作记录存储
type OperationRecordStore = store.GenericStore[OperationRecord]

// OperationRecordMeta 操作记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 查询提速：username + group.workspaceID + createdAt(倒序)
// - 查询提速：username + group.workspaceID + group.appID + createdAt(倒序)
var OperationRecordMeta = store.Meta[OperationRecord]{
	Collection: "operation_records",
	NotFound:   ErrOperationRecordNotFound,
}

// NewestFirst 按创建时间倒序，以 _id 兜底保证翻页稳定
func NewestFirst() store.Opt {
	return store.Combine(store.Desc("createdAt"), store.Desc("_id"))
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

// Cond 将选项中的过滤条件（不含分页）转换为查询条件，空值字段不参与过滤
func (o *ListOptions) Cond() store.Opt {
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
	if o.OpType != "" {
		filter["operationType"] = o.OpType
	}
	if o.ResType != "" {
		filter["resourceType"] = o.ResType
	}
	if o.Result != "" {
		filter["result"] = o.Result
	}
	return store.Combine(store.Filter(filter), store.ContainsIgnoreCase(o.Username, "username"))
}

// ListRecords 按选项分页获取操作记录，最新的在前
func ListRecords(ctx context.Context, db *mongo.Database, opts ListOptions) (*store.Page[OperationRecord], error) {
	page, err := store.Of(db, OperationRecordMeta).ListPage(ctx, opts.Page, opts.PageSize, opts.Cond(), NewestFirst())
	return page, errors.Wrap(err, "list operation records")
}

// GetLatestOperationTimesByWorkspacesForUser 批量查询指定用户在各 workspace 的最近操作时间，key 为 workspaceID
func GetLatestOperationTimesByWorkspacesForUser(
	ctx context.Context, db *mongo.Database, workspaceIDs []string, username string,
) (map[string]time.Time, error) {
	m, err := latestOperationTimes(ctx, db, "group.workspaceID", bson.M{
		"group.workspaceID": bson.M{"$in": workspaceIDs},
		"username":          username,
	})
	return m, errors.Wrapf(err, "get latest operation times of user %s by workspaces", username)
}

// GetLatestOperationTimesByAppsForUser 批量查询指定用户在指定 workspace 下各 app 的最近操作时间，key 为 appID
func GetLatestOperationTimesByAppsForUser(
	ctx context.Context, db *mongo.Database, workspaceID string, appIDs []string, username string,
) (map[string]time.Time, error) {
	m, err := latestOperationTimes(ctx, db, "group.appID", bson.M{
		"group.workspaceID": workspaceID,
		"group.appID":       bson.M{"$in": appIDs},
		"username":          username,
	})
	return m, errors.Wrapf(err, "get latest operation times of user %s by apps in workspace %s", username, workspaceID)
}

// latestTimeResult 用于解码 aggregation 结果
type latestTimeResult struct {
	ID       string    `bson:"_id"`
	LatestAt time.Time `bson:"latestAt"`
}

// latestOperationTimes 按 groupField 分组，统计匹配 match 的记录中各组的最近操作时间
func latestOperationTimes(
	ctx context.Context, db *mongo.Database, groupField string, match bson.M,
) (map[string]time.Time, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":      "$" + groupField,
			"latestAt": bson.M{"$max": "$createdAt"},
		}}},
	}
	results, err := store.Aggregate[latestTimeResult](ctx, store.Of(db, OperationRecordMeta), pipeline)
	if err != nil {
		return nil, errors.Wrapf(err, "aggregate latest operation times by %s", groupField)
	}
	return lo.SliceToMap(results, func(r latestTimeResult) (string, time.Time) { return r.ID, r.LatestAt }), nil
}
