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
	stderrors "errors"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// DetailSyncer 详情同步器
type DetailSyncer struct {
	db *mongo.Database
}

// NewDetailSyncer 创建 DetailSyncer 实例
func NewDetailSyncer(db *mongo.Database) *DetailSyncer {
	return &DetailSyncer{db: db}
}

// SyncDetails 执行详情同步。suppressedTags 是本次刷新命中的 suppression tag，先逐个复核 manifest 是否重新存在。
//
// 复核遇到 404 以外的错误时，状态照常回到 idle，但返回错误交给 asynq 重试，否则这批 tag 要等到下一次刷新才有机会恢复；
// 普通 tag 的详情拉取失败只记入 LastError，本身不触发重试。复核失败引发的重试会整条任务重跑，
// 上次详情拉取失败、仍未补全的普通 tag 会顺带再拉一次，已补全的不受影响
func (s *DetailSyncer) SyncDetails(ctx context.Context, info *RepoKeyInfo, suppressedTags []string) error {
	repoKey := info.RepoKey
	repoName := info.RepoName

	// 原子性尝试获取详情同步权（仅当状态为 idle 时成功）
	acquired, err := TrySetDetailSyncing(ctx, s.db, repoKey)
	if err != nil {
		return errors.Wrapf(err, "acquire detail sync for %s", repoKey)
	}
	if !acquired {
		// suppressedTags 只随本条任务消息传递，正在执行的任务不会复核它们；
		// 返回错误交给 asynq 重试，避免这批 tag 要等到下一次刷新才能恢复
		if len(suppressedTags) > 0 {
			return errors.Errorf(
				"detail sync for %s is busy, retry later to recheck suppressed tags %v",
				repoKey, suppressedTags,
			)
		}
		log.Infof(ctx, "detail sync for %s skipped: status is not idle", repoKey)
		return nil
	}

	client := registry.New(info.Username, info.Password, true)

	var lastError string
	restoreErr := s.restoreSuppressedTags(ctx, repoKey, repoName, client, suppressedTags)
	if restoreErr != nil {
		lastError = restoreErr.Error()
	}

	// 获取需要补全详情的标签
	tags, err := ListUnsyncedDetailTags(ctx, s.db, repoKey)
	if err != nil {
		return errors.Wrapf(err, "list unsynced detail tags for %s", repoKey)
	}
	if len(tags) == 0 {
		log.Infof(ctx, "no tags need detail sync for %s", repoKey)
	}

	// 按批次处理
	for i := 0; i < len(tags); i += DetailSyncBatchSize {
		end := min(i+DetailSyncBatchSize, len(tags))
		batch := tags[i:end]

		batchErr := s.syncDetailBatch(ctx, repoKey, repoName, client, batch)
		if batchErr != nil {
			lastError = batchErr.Error()
			log.Errorf(ctx, "detail sync batch failed for %s: %v", repoKey, batchErr)
			// 继续处理下一批，不中断
		}
	}

	if err = s.completeDetailSync(ctx, repoKey, lastError); err != nil {
		return errors.Wrapf(err, "complete detail sync for %s", repoKey)
	}
	return errors.Wrapf(restoreErr, "restore suppressed tags for %s", repoKey)
}

// restoreSuppressedTags 逐个 HEAD 复核 suppressed tag：manifest 重新存在时删除 suppression 并补回空快照
func (s *DetailSyncer) restoreSuppressedTags(
	ctx context.Context,
	repoKey, repoName string,
	client *registry.Client,
	tags []string,
) error {
	ctx = context.WithoutCancel(ctx)
	var restoreErrs []error
	for _, tag := range tags {
		err := client.HeadManifest(ctx, repoName, tag)
		switch {
		case err == nil:
			if delErr := DeleteTagSuppressions(ctx, s.db, repoKey, []string{tag}); delErr != nil {
				restoreErrs = append(restoreErrs, delErr)
				continue
			}
			if upsertErr := UpsertTags(ctx, s.db, repoKey, []string{tag}); upsertErr != nil {
				restoreErrs = append(restoreErrs, errors.Wrapf(upsertErr, "restore suppressed tag %s", tag))
			}
		case registry.IsTagNotFound(err):
			if _, delErr := store.Of(s.db, ImageMeta).DeleteMany(ctx, ByRepoTag(repoKey, tag)); delErr != nil {
				restoreErrs = append(
					restoreErrs, errors.Wrapf(delErr, "delete leftover snapshot of suppressed tag %s", tag),
				)
			}
		default:
			restoreErrs = append(restoreErrs, errors.Wrapf(err, "head manifest for suppressed tag %s", tag))
		}
	}
	return stderrors.Join(restoreErrs...)
}

// syncDetailBatch 处理一批标签的详情同步，使用信号量控制并发
func (s *DetailSyncer) syncDetailBatch(
	ctx context.Context,
	repoKey, repoName string,
	client *registry.Client,
	batch []string,
) error {
	sem := make(chan struct{}, DetailSyncMaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, tag := range batch {
		// 获取信号量
		sem <- struct{}{}

		wg.Go(func() {
			// 释放信号量
			defer func() { <-sem }()
			gCtx := context.WithoutCancel(ctx)
			detail, err := client.GetTagDetail(gCtx, repoName, tag)
			if err != nil {
				// GetTagDetail 后半段（子 manifest、config blob）的 404 不能当成 tag 已删除。
				// 只有 manifest 本身也 HEAD 不到时才删快照。
				if registry.IsTagNotFound(err) && s.manifestGone(gCtx, client, repoName, tag) {
					if dropErr := s.dropMissingManifestTag(gCtx, repoKey, repoName, tag); dropErr != nil {
						log.Errorf(gCtx, "drop missing manifest tag %s:%s failed: %v", repoName, tag, dropErr)
						mu.Lock()
						if firstErr == nil {
							firstErr = dropErr
						}
						mu.Unlock()
						return
					}
					log.Infof(gCtx, "dropped snapshot %s:%s because the remote manifest is gone", repoName, tag)
					return
				}
				log.Errorf(gCtx, "get tag detail for %s:%s failed: %v", repoName, tag, err)
				mu.Lock()
				if firstErr == nil {
					firstErr = errors.Wrapf(err, "get detail for %s", tag)
				}
				mu.Unlock()
				return
			}

			// 更新快照详情
			if err = UpdateDetail(gCtx, s.db, repoKey, tag, &detail); err != nil {
				log.Errorf(gCtx, "update detail for %s:%s failed: %v", repoKey, tag, err)
				mu.Lock()
				if firstErr == nil {
					firstErr = errors.Wrapf(err, "update detail for %s", tag)
				}
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	return firstErr
}

// dropMissingManifestTag 为 manifest 已不存在的 tag 写入 suppression 并删除快照。
// tags/list 仍残留该名字时，suppression 阻止刷新把它当成新增写回，所以必须先于快照删除写入
func (s *DetailSyncer) dropMissingManifestTag(ctx context.Context, repoKey, repoName, tag string) error {
	if err := UpsertTagSuppression(ctx, s.db, repoKey, repoName, tag); err != nil {
		return errors.Wrapf(err, "record suppression for missing manifest %s", tag)
	}
	if _, err := store.Of(s.db, ImageMeta).DeleteMany(ctx, ByRepoTag(repoKey, tag)); err != nil {
		return errors.Wrapf(err, "delete missing manifest snapshot %s", tag)
	}
	return nil
}

// completeDetailSync 完成详情同步，更新状态
func (s *DetailSyncer) completeDetailSync(ctx context.Context, repoKey, lastError string) error {
	detailSyncedAt := time.Now()
	status := &RepoSnapshotStatus{
		RepoKey:            repoKey,
		RefreshStatus:      RefreshStatusIdle,
		LastDetailSyncedAt: &detailSyncedAt,
		LastError:          lastError,
	}
	return UpsertRepoStatus(ctx, s.db, status)
}

// manifestGone 用 HEAD 确认 tag 的 manifest 是否已经不存在。
// HEAD 成功或返回非 404 时保留快照，避免把 config blob 等后续 404 误判成 tag 已删除。
func (*DetailSyncer) manifestGone(ctx context.Context, client *registry.Client, repoName, tag string) bool {
	err := client.HeadManifest(ctx, repoName, tag)
	if err == nil {
		return false
	}
	if registry.IsTagNotFound(err) {
		return true
	}
	log.Errorf(ctx, "head manifest %s:%s failed, keep snapshot: %v", repoName, tag, err)
	return false
}
