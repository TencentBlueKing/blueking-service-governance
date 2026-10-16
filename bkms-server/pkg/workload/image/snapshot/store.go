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

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrImageNotFound 镜像快照不存在
	ErrImageNotFound = errors.New("image snapshot not found")
	// ErrRepoStatusNotFound 仓库快照状态不存在
	ErrRepoStatusNotFound = errors.New("repo snapshot status not found")
)

// ImageStore 镜像快照存储，repoKey + tag 唯一确定一条快照
type ImageStore = store.GenericStore[Image]

// ImageMeta 镜像快照的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：repoKey + tag
// - 查询提速：repoKey
var ImageMeta = store.Meta[Image]{
	Collection: "image_snapshots",
	NotFound:   ErrImageNotFound,
}

// RepoStatusStore 仓库快照状态存储，每个 repoKey 一条
type RepoStatusStore = store.GenericStore[RepoSnapshotStatus]

// RepoStatusMeta 仓库快照状态的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：repoKey
var RepoStatusMeta = store.Meta[RepoSnapshotStatus]{
	Collection: "repo_snapshot_statuses",
	NotFound:   ErrRepoStatusNotFound,
}

// InRepo 按仓库匹配镜像快照或仓库快照状态
func InRepo(repoKey string) store.Opt {
	return store.Where("repoKey", repoKey)
}

// ByRepoTag 按仓库与标签匹配镜像快照
func ByRepoTag(repoKey, tag string) store.Opt {
	return store.Combine(InRepo(repoKey), store.Where("tag", tag))
}

// TagIn 匹配标签在 tags 中的镜像快照，tags 为空时不匹配任何快照
func TagIn(tags []string) store.Opt {
	// $in 不接受 null，lo.Uniq 总是返回非 nil 的切片
	return store.Where("tag", bson.M{"$in": lo.Uniq(tags)})
}

// UpsertTags 批量写入仓库的标签快照（用于标签刷新同步）；已存在的快照只刷新 updatedAt，详情字段保持不变
func UpsertTags(ctx context.Context, db *mongo.Database, repoKey string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	objs := lo.Map(lo.Uniq(tags), func(tag string, _ int) *Image {
		return &Image{RepoKey: repoKey, Tag: tag}
	})
	_, err := store.Of(db, ImageMeta).UpsertMany(ctx, objs, func(img *Image) store.Opt {
		return ByRepoTag(img.RepoKey, img.Tag)
	}, store.UpdateSelected("repoKey", "tag"))
	return errors.Wrapf(err, "upsert tags of %s", repoKey)
}

// DeleteTagsNotIn 删除仓库中不在 activeTags 中的快照（清理远程已消失的标签），返回删除的数量
func DeleteTagsNotIn(ctx context.Context, db *mongo.Database, repoKey string, activeTags []string) (int64, error) {
	n, err := store.Of(db, ImageMeta).DeleteMany(
		ctx,
		InRepo(repoKey),
		store.Where("tag", bson.M{"$nin": lo.Uniq(activeTags)}),
	)
	return n, errors.Wrapf(err, "delete tags of %s not in active tags", repoKey)
}

// UpdateDetail 更新单条快照的详情字段（用于详情补全），快照不存在时忽略
//
// 会无条件清除 detailSyncPending；并发同 tag 构建下可能误清较新的 pending 标记，
// 详见 Image.DetailSyncPending 字段注释中的 TODO
func UpdateDetail(ctx context.Context, db *mongo.Database, repoKey, tag string, detail *registry.ImageDetail) error {
	err := store.Of(db, ImageMeta).Update(ctx, map[string]any{
		"digest":  detail.Digest,
		"size":    detail.Size,
		"builtAt": lo.ToPtr(detail.BuiltAt),
		// 详情已是最新，清除重新拉取标记（无条件清除，见函数注释中的并发限制）
		"detailSyncPending": false,
	}, ByRepoTag(repoKey, tag))
	return errors.Wrapf(store.IgnoreNotFound(err), "update detail for %s:%s", repoKey, tag)
}

// MarkDetailSyncPending 将指定标签标记为需要重新拉取详情，标签不存在时忽略；不刷新 updatedAt
func MarkDetailSyncPending(ctx context.Context, db *mongo.Database, repoKey string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	_, err := store.Of(db, ImageMeta).UpdateMany(
		ctx,
		map[string]any{"detailSyncPending": true},
		InRepo(repoKey),
		TagIn(tags),
		store.KeepUpdatedAt(),
	)
	return errors.Wrapf(err, "mark detail sync pending for %s", repoKey)
}

// ListImagePage 分页查询仓库的快照，keyword 按 tag 模糊匹配（忽略大小写），conds 用于追加过滤条件（如 TagIn）。
// 按构建时间降序排列（无构建时间的排在后面），其次按 tag 逆序
func ListImagePage(
	ctx context.Context,
	db *mongo.Database,
	repoKey, keyword string,
	page, pageSize int,
	conds ...store.Opt,
) (*store.Page[Image], error) {
	opts := append([]store.Opt{
		InRepo(repoKey),
		store.ContainsIgnoreCase(keyword, "tag"),
		store.Desc("builtAt"),
		store.Desc("tag"),
	}, conds...)
	result, err := store.Of(db, ImageMeta).ListPage(ctx, int64(page), int64(pageSize), opts...)
	return result, errors.Wrapf(err, "list snapshots of %s", repoKey)
}

// ListUnsyncedDetailTags 获取仓库中需要补全详情的标签：builtAt 为空、被标记需重新拉取，或者 latest（每次都刷新）
func ListUnsyncedDetailTags(ctx context.Context, db *mongo.Database, repoKey string) ([]string, error) {
	tags, err := listTags(ctx, db, InRepo(repoKey), store.Filter(bson.M{
		"$or": []bson.M{
			{"builtAt": bson.M{"$exists": false}},
			{"builtAt": nil},
			{"detailSyncPending": true},
			{"tag": TagLatest},
		},
	}))
	return tags, errors.Wrapf(err, "list unsynced detail tags of %s", repoKey)
}

// ListAllTags 获取仓库中的全部标签（用于 diff 计算）
func ListAllTags(ctx context.Context, db *mongo.Database, repoKey string) ([]string, error) {
	tags, err := listTags(ctx, db, InRepo(repoKey))
	return tags, errors.Wrapf(err, "list all tags of %s", repoKey)
}

// listTags 获取匹配快照的标签，只读出 tag 字段
func listTags(ctx context.Context, db *mongo.Database, conds ...store.Opt) ([]string, error) {
	opts := append([]store.Opt{store.SelectFields("tag")}, conds...)
	images, err := store.Of(db, ImageMeta).List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return lo.Map(images, func(img Image, _ int) string { return img.Tag }), nil
}

// GetRepoStatus 获取仓库快照状态，不存在时返回 nil
func GetRepoStatus(ctx context.Context, db *mongo.Database, repoKey string) (*RepoSnapshotStatus, error) {
	status, err := store.Of(db, RepoStatusMeta).Get(ctx, InRepo(repoKey))
	if errors.Is(err, ErrRepoStatusNotFound) {
		return nil, nil
	}
	return status, errors.Wrapf(err, "get status for %s", repoKey)
}

// UpsertRepoStatus 创建或更新仓库快照状态。refreshStatus 与 lastError 总会写入；
// repoName 为空、两个时间字段为 nil 时保留库中已有的值
func UpsertRepoStatus(ctx context.Context, db *mongo.Database, status *RepoSnapshotStatus) error {
	fields := []string{"refreshStatus", "lastError"}
	if status.RepoName != "" {
		fields = append(fields, "repoName")
	}
	if status.LastRefreshedAt != nil {
		fields = append(fields, "lastRefreshedAt")
	}
	if status.LastDetailSyncedAt != nil {
		fields = append(fields, "lastDetailSyncedAt")
	}
	_, err := store.Of(db, RepoStatusMeta).Upsert(ctx, status, InRepo(status.RepoKey), store.UpdateSelected(fields...))
	return errors.Wrapf(err, "upsert status for %s", status.RepoKey)
}

// TrySetRefreshing 原子性地将状态从非 refreshing 设为 refreshing（用于幂等性检查）。
// 返回 true 表示成功获取刷新权，false 表示已有刷新在进行中
func TrySetRefreshing(ctx context.Context, db *mongo.Database, repoKey string) (bool, error) {
	acquired, err := trySetRepoStatus(ctx, db, repoKey, RefreshStatusRefreshing,
		bson.M{"$ne": RefreshStatusRefreshing})
	return acquired, errors.Wrapf(err, "try set refreshing for %s", repoKey)
}

// TrySetDetailSyncing 原子性地将状态从 idle 设为 detail_syncing（用于详情同步并发控制）。
// 返回 true 表示成功获取同步权，false 表示当前状态不允许开始详情同步
func TrySetDetailSyncing(ctx context.Context, db *mongo.Database, repoKey string) (bool, error) {
	acquired, err := trySetRepoStatus(ctx, db, repoKey, RefreshStatusDetailSyncing, RefreshStatusIdle)
	return acquired, errors.Wrapf(err, "try set detail syncing for %s", repoKey)
}

// trySetRepoStatus 在当前状态满足 from 时将状态设为 to，状态记录不存在时直接创建。
//
// 状态记录存在但不满足 from 时，upsert 会尝试插入一条同 repoKey 的新记录并因唯一索引冲突失败，
// 以此判定未获取到状态
func trySetRepoStatus(
	ctx context.Context,
	db *mongo.Database,
	repoKey string,
	to RefreshStatus,
	from any,
) (bool, error) {
	_, err := store.Of(db, RepoStatusMeta).Upsert(
		ctx,
		map[string]any{"refreshStatus": to, "lastError": ""},
		InRepo(repoKey),
		store.Where("refreshStatus", from),
	)
	if errors.Is(err, store.ErrDuplicate) {
		return false, nil
	}
	return err == nil, err
}
