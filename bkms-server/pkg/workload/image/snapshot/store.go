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

package snapshot

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
)

// MongoDB 集合名
const (
	// collImageSnapshots 镜像快照集合
	collImageSnapshots = "image_snapshots"
	// collRepoSnapshotStatus 仓库快照状态集合
	collRepoSnapshotStatus = "repo_snapshot_statuses"
)

// SnapshotStore 镜像快照存储接口
type SnapshotStore interface {
	// UpsertSnapshots 批量创建或更新快照记录（用于标签刷新同步）
	UpsertSnapshots(ctx context.Context, repoKey string, snapshots []Image) error

	// DeleteByRepoKeyExcludeTags 删除指定仓库中不在给定标签列表中的快照（清理远程已消失的标签）
	DeleteByRepoKeyExcludeTags(ctx context.Context, repoKey string, activeTags []string) (int64, error)

	// DeleteByRepoKeyAndTag 删除指定仓库中的单个标签快照。
	// 返回实际删除的条数；若记录不存在，则返回 0 和 nil，保持幂等语义。
	DeleteByRepoKeyAndTag(ctx context.Context, repoKey, tag string) (int64, error)

	// HasTag 判断指定仓库快照中是否存在 tag
	HasTag(ctx context.Context, repoKey, tag string) (bool, error)

	// UpdateDetail 更新单条快照的详情字段（用于详情补全）
	UpdateDetail(ctx context.Context, repoKey, tag string, detail *registry.ImageDetail) error

	// MarkDetailSyncPending 将指定标签标记为需要重新拉取详情（标签不存在时忽略）
	MarkDetailSyncPending(ctx context.Context, repoKey string, tags []string) error

	// ListByRepoKey 分页查询指定仓库的快照列表（支持关键词过滤和时间排序）
	ListByRepoKey(ctx context.Context, repoKey, keyword string, page, pageSize int) ([]Image, int64, error)

	// ListByRepoKeyAndTags 在给定的 tags 列表范围内分页查询快照（用于生产环境：已晋级 tag 的交集查询）
	ListByRepoKeyAndTags(
		ctx context.Context, repoKey string, tags []string, keyword string, page, pageSize int,
	) ([]Image, int64, error)

	// ListUnsyncedDetailTags 获取指定仓库中需要补全详情的标签列表
	// （builtAt 为空、detailSyncPending，或 latest）
	ListUnsyncedDetailTags(ctx context.Context, repoKey string) ([]string, error)

	// ListAllTags 获取指定仓库中所有标签列表（用于 diff 计算）
	ListAllTags(ctx context.Context, repoKey string) ([]string, error)

	// UpsertStatus 创建或更新仓库快照状态
	UpsertStatus(ctx context.Context, status *RepoSnapshotStatus) error

	// GetStatus 获取仓库快照状态
	GetStatus(ctx context.Context, repoKey string) (*RepoSnapshotStatus, error)

	// TrySetRefreshing 原子性地将状态从非 refreshing 设为 refreshing（用于幂等性检查）
	// 返回 true 表示成功获取刷新权，false 表示已有刷新在进行中
	TrySetRefreshing(ctx context.Context, repoKey string) (bool, error)

	// TrySetDetailSyncing 原子性地将状态从 idle 设为 detail_syncing（用于详情同步并发控制）
	// 返回 true 表示成功获取同步权，false 表示当前状态不允许开始详情同步
	TrySetDetailSyncing(ctx context.Context, repoKey string) (bool, error)

	// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
	DeleteAll(ctx context.Context) error
}

// 编译期接口实现检查
var _ SnapshotStore = &SnapshotStoreMongo{}

// SnapshotStoreMongo MongoDB 实现
type SnapshotStoreMongo struct {
	snapshotColl *mongo.Collection
	statusColl   *mongo.Collection
}

// NewSnapshotStoreMongo 创建 SnapshotStoreMongo 实例。
func NewSnapshotStoreMongo(client *mongo.Client, dbName string) (*SnapshotStoreMongo, error) {
	db := client.Database(dbName)
	snapshotColl := db.Collection(collImageSnapshots)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：repoKey + tag
	// - 查询提速：repoKey
	statusColl := db.Collection(collRepoSnapshotStatus)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：repoKey
	store := &SnapshotStoreMongo{
		snapshotColl: snapshotColl,
		statusColl:   statusColl,
	}
	return store, nil
}

// UpsertSnapshots 批量 upsert 快照记录
func (s *SnapshotStoreMongo) UpsertSnapshots(ctx context.Context, repoKey string, snapshots []Image) error {
	if len(snapshots) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(snapshots))
	now := time.Now()

	for i := range snapshots {
		snap := &snapshots[i]
		snap.RepoKey = repoKey
		snap.UpdatedAt = now

		filter := bson.M{"repoKey": repoKey, "tag": snap.Tag}
		update := bson.M{
			"$set": bson.M{
				"repoKey":   repoKey,
				"tag":       snap.Tag,
				"updatedAt": now,
			},
			"$setOnInsert": bson.M{
				"createdAt": now,
			},
		}

		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, model)
	}

	_, err := s.snapshotColl.BulkWrite(ctx, models)
	if err != nil {
		return errors.Wrap(err, "bulk upsert snapshots")
	}

	return nil
}

// DeleteByRepoKeyExcludeTags 删除不在 activeTags 中的快照
func (s *SnapshotStoreMongo) DeleteByRepoKeyExcludeTags(
	ctx context.Context, repoKey string, activeTags []string,
) (int64, error) {
	filter := bson.M{
		"repoKey": repoKey,
		"tag":     bson.M{"$nin": activeTags},
	}

	result, err := s.snapshotColl.DeleteMany(ctx, filter)
	if err != nil {
		return 0, errors.Wrap(err, "delete excluded snapshots")
	}

	return result.DeletedCount, nil
}

// DeleteByRepoKeyAndTag 删除指定仓库中的单个标签快照。
// 即使未命中记录也不会报错，而是返回 0，以保持幂等删除语义。
func (s *SnapshotStoreMongo) DeleteByRepoKeyAndTag(ctx context.Context, repoKey, tag string) (int64, error) {
	result, err := s.snapshotColl.DeleteOne(ctx, bson.M{
		"repoKey": repoKey,
		"tag":     tag,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "delete snapshot for %s:%s", repoKey, tag)
	}
	return result.DeletedCount, nil
}

// HasTag 判断指定仓库快照中是否存在 tag
func (s *SnapshotStoreMongo) HasTag(ctx context.Context, repoKey, tag string) (bool, error) {
	count, err := s.snapshotColl.CountDocuments(ctx, bson.M{
		"repoKey": repoKey,
		"tag":     tag,
	}, options.Count().SetLimit(1))
	if err != nil {
		return false, errors.Wrapf(err, "check snapshot tag for %s:%s", repoKey, tag)
	}
	return count > 0, nil
}

// UpdateDetail 更新快照详情
//
// 会无条件清除 detailSyncPending；并发同 tag 构建下可能误清较新的 pending 标记，
// 详见 Image.DetailSyncPending 字段注释中的 TODO
func (s *SnapshotStoreMongo) UpdateDetail(
	ctx context.Context,
	repoKey, tag string,
	detail *registry.ImageDetail,
) error {
	now := time.Now()

	update := bson.M{
		"$set": bson.M{
			"digest":    detail.Digest,
			"size":      detail.Size,
			"builtAt":   detail.BuiltAt,
			"updatedAt": now,
		},
		// 详情已是最新，清除重新拉取标记（无条件清除，见函数注释中的并发限制）
		"$unset": bson.M{"detailSyncPending": ""},
	}

	filter := bson.M{"repoKey": repoKey, "tag": tag}
	_, err := s.snapshotColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrapf(err, "update detail for %s:%s", repoKey, tag)
	}

	return nil
}

// MarkDetailSyncPending 将指定标签标记为需要重新拉取详情
func (s *SnapshotStoreMongo) MarkDetailSyncPending(ctx context.Context, repoKey string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	filter := bson.M{"repoKey": repoKey, "tag": bson.M{"$in": tags}}
	update := bson.M{"$set": bson.M{"detailSyncPending": true}}
	if _, err := s.snapshotColl.UpdateMany(ctx, filter, update); err != nil {
		return errors.Wrapf(err, "mark detail sync pending for %s", repoKey)
	}

	return nil
}

// ListByRepoKey 分页查询快照列表
func (s *SnapshotStoreMongo) ListByRepoKey(
	ctx context.Context, repoKey, keyword string, page, pageSize int,
) ([]Image, int64, error) {
	filter := bson.M{"repoKey": repoKey}

	// 关键词过滤（模糊匹配 tag）
	if keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword = regexp.QuoteMeta(keyword)
		filter["tag"] = bson.M{"$regex": keyword, "$options": "i"}
	}

	results, total, err := s.listSnapshots(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, errors.Wrap(err, "list snapshots")
	}
	return results, total, nil
}

// ListByRepoKeyAndTags 在给定的 tags 列表范围内分页查询快照
func (s *SnapshotStoreMongo) ListByRepoKeyAndTags(
	ctx context.Context, repoKey string, tags []string, keyword string, page, pageSize int,
) ([]Image, int64, error) {
	if len(tags) == 0 {
		return nil, 0, nil
	}

	filter := bson.M{
		"repoKey": repoKey,
		"tag":     bson.M{"$in": tags},
	}

	// 关键词过滤（模糊匹配 tag），使用 $and 与 $in 组合
	if keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword = regexp.QuoteMeta(keyword)
		filter = bson.M{
			"repoKey": repoKey,
			"$and": []bson.M{
				{"tag": bson.M{"$in": tags}},
				{"tag": bson.M{"$regex": keyword, "$options": "i"}},
			},
		}
	}

	results, total, err := s.listSnapshots(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, errors.Wrap(err, "list snapshots by tags")
	}
	return results, total, nil
}

// listSnapshots 通用分页查询快照方法，封装 CountDocuments → Find → cursor.All 的完整流程
func (s *SnapshotStoreMongo) listSnapshots(
	ctx context.Context,
	filter bson.M,
	page, pageSize int,
) ([]Image, int64, error) {
	// 计算总数
	total, err := s.snapshotColl.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrap(err, "count snapshots")
	}

	if total == 0 {
		return nil, 0, nil
	}

	// 分页查询，按构建时间降序排序（无构建时间的排在后面），其次按 tag 字母序逆序排列
	skip := int64((page - 1) * pageSize)
	findOpts := options.Find().
		SetSort(bson.D{
			{Key: "builtAt", Value: -1},
			{Key: "tag", Value: -1},
		}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cursor, err := s.snapshotColl.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, errors.Wrap(err, "find snapshots")
	}
	defer cursor.Close(ctx)

	var results []Image
	if err = cursor.All(ctx, &results); err != nil {
		return nil, 0, errors.Wrap(err, "decode snapshots")
	}

	return results, total, nil
}

// listTags 根据过滤条件查询标签列表的通用方法
func (s *SnapshotStoreMongo) listTags(ctx context.Context, filter bson.M) ([]string, error) {
	cursor, err := s.snapshotColl.Find(ctx, filter, options.Find().SetProjection(bson.M{"tag": 1}))
	if err != nil {
		return nil, errors.Wrap(err, "find tags")
	}
	defer cursor.Close(ctx)

	var tags []string
	for cursor.Next(ctx) {
		var doc struct {
			Tag string `bson:"tag"`
		}
		if err = cursor.Decode(&doc); err != nil {
			return nil, errors.Wrap(err, "decode tag")
		}
		tags = append(tags, doc.Tag)
	}

	return tags, nil
}

// ListUnsyncedDetailTags 获取需要补全详情的标签
func (s *SnapshotStoreMongo) ListUnsyncedDetailTags(ctx context.Context, repoKey string) ([]string, error) {
	// 查询：builtAt 为空（未补全详情）、被标记需重新拉取，或者 tag 为 latest（每次都刷新）
	filter := bson.M{
		"repoKey": repoKey,
		"$or": []bson.M{
			{"builtAt": bson.M{"$exists": false}},
			{"builtAt": nil},
			{"detailSyncPending": true},
			{"tag": TagLatest},
		},
	}

	return s.listTags(ctx, filter)
}

// ListAllTags 获取指定仓库中所有标签列表（用于 diff 计算）
func (s *SnapshotStoreMongo) ListAllTags(ctx context.Context, repoKey string) ([]string, error) {
	filter := bson.M{"repoKey": repoKey}
	return s.listTags(ctx, filter)
}

// UpsertStatus 创建或更新仓库快照状态
func (s *SnapshotStoreMongo) UpsertStatus(ctx context.Context, status *RepoSnapshotStatus) error {
	now := time.Now()
	status.UpdatedAt = now

	filter := bson.M{"repoKey": status.RepoKey}
	setFields := bson.M{
		"refreshStatus": string(status.RefreshStatus),
		"lastError":     status.LastError,
		"updatedAt":     now,
	}

	if status.RepoName != "" {
		setFields["repoName"] = status.RepoName
	}

	update := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"repoKey":   status.RepoKey,
			"createdAt": now,
		},
	}

	// 可选时间字段
	if status.LastRefreshedAt != nil {
		update["$set"].(bson.M)["lastRefreshedAt"] = status.LastRefreshedAt
	}
	if status.LastDetailSyncedAt != nil {
		update["$set"].(bson.M)["lastDetailSyncedAt"] = status.LastDetailSyncedAt
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.statusColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return errors.Wrapf(err, "upsert status for %s", status.RepoKey)
	}

	return nil
}

// GetStatus 获取仓库快照状态
func (s *SnapshotStoreMongo) GetStatus(ctx context.Context, repoKey string) (*RepoSnapshotStatus, error) {
	var status RepoSnapshotStatus
	err := s.statusColl.FindOne(ctx, bson.M{"repoKey": repoKey}).Decode(&status)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get status for %s", repoKey)
	}
	return &status, nil
}

// TrySetRefreshing 原子性设置为 refreshing 状态
func (s *SnapshotStoreMongo) TrySetRefreshing(ctx context.Context, repoKey string) (bool, error) {
	now := time.Now()

	filter := bson.M{
		"repoKey":       repoKey,
		"refreshStatus": bson.M{"$ne": string(RefreshStatusRefreshing)},
	}
	update := bson.M{
		"$set": bson.M{
			"refreshStatus": string(RefreshStatusRefreshing),
			"lastError":     "",
			"updatedAt":     now,
		},
		"$setOnInsert": bson.M{
			"repoKey":   repoKey,
			"createdAt": now,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	result, err := s.statusColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		// 如果是 duplicate key error（说明已存在且状态为 refreshing），返回 false
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "try set refreshing for %s", repoKey)
	}

	// 如果 ModifiedCount > 0 或 UpsertedCount > 0，说明成功获取刷新权
	return result.ModifiedCount > 0 || result.UpsertedCount > 0, nil
}

// TrySetDetailSyncing 原子性设置为 detail_syncing 状态（仅当当前状态为 idle 时）
func (s *SnapshotStoreMongo) TrySetDetailSyncing(ctx context.Context, repoKey string) (bool, error) {
	now := time.Now()

	filter := bson.M{
		"repoKey":       repoKey,
		"refreshStatus": string(RefreshStatusIdle),
	}
	update := bson.M{
		"$set": bson.M{
			"refreshStatus": string(RefreshStatusDetailSyncing),
			"lastError":     "",
			"updatedAt":     now,
		},
		"$setOnInsert": bson.M{
			"repoKey":   repoKey,
			"createdAt": now,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	result, err := s.statusColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		// 如果是 duplicate key error（说明已存在且状态非 idle），返回 false
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "try set detail syncing for %s", repoKey)
	}

	// 如果 ModifiedCount > 0 或 UpsertedCount > 0，说明成功获取同步权
	return result.ModifiedCount > 0 || result.UpsertedCount > 0, nil
}

// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
func (s *SnapshotStoreMongo) DeleteAll(ctx context.Context) error {
	if _, err := s.snapshotColl.DeleteMany(ctx, bson.M{}); err != nil {
		return errors.Wrap(err, "delete snapshot documents")
	}
	if _, err := s.statusColl.DeleteMany(ctx, bson.M{}); err != nil {
		return errors.Wrap(err, "delete status documents")
	}
	return nil
}
