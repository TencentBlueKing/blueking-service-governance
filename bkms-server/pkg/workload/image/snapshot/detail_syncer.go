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
	"sync"
	"time"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
)

// DetailSyncer 详情同步器
type DetailSyncer struct {
	store SnapshotStore
}

// NewDetailSyncer 创建 DetailSyncer 实例
func NewDetailSyncer(store SnapshotStore) *DetailSyncer {
	return &DetailSyncer{store: store}
}

// SyncDetails 执行详情同步
func (s *DetailSyncer) SyncDetails(ctx context.Context, info *RepoKeyInfo) error {
	repoKey := info.RepoKey
	repoName := info.RepoName

	// 原子性尝试获取详情同步权（仅当状态为 idle 时成功）
	acquired, err := s.store.TrySetDetailSyncing(ctx, repoKey)
	if err != nil {
		return errors.Wrap(err, "try set detail syncing")
	}
	if !acquired {
		log.Infof(ctx, "detail sync for %s skipped: status is not idle", repoKey)
		return nil
	}

	client := registry.New(info.Username, info.Password, true)

	// 获取需要补全详情的标签
	tags, err := s.store.ListUnsyncedDetailTags(ctx, repoKey)
	if err != nil {
		return errors.Wrap(err, "list unsynced detail tags")
	}

	if len(tags) == 0 {
		log.Infof(ctx, "no tags need detail sync for %s", repoKey)
		return s.completeDetailSync(ctx, repoKey, "")
	}

	// 按批次处理
	var lastError string
	for i := 0; i < len(tags); i += DetailSyncBatchSize {
		end := i + DetailSyncBatchSize
		if end > len(tags) {
			end = len(tags)
		}
		batch := tags[i:end]

		batchErr := s.syncDetailBatch(ctx, repoKey, repoName, client, batch)
		if batchErr != nil {
			lastError = batchErr.Error()
			log.Errorf(ctx, "detail sync batch failed for %s: %v", repoKey, batchErr)
			// 继续处理下一批，不中断
		}
	}

	return s.completeDetailSync(ctx, repoKey, lastError)
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
				if registry.IsTagNotFound(err) && manifestGone(gCtx, client, repoName, tag) {
					if _, delErr := s.store.DeleteByRepoKeyAndTag(gCtx, repoKey, tag); delErr != nil {
						log.Errorf(gCtx, "delete missing manifest snapshot %s:%s failed: %v", repoName, tag, delErr)
						mu.Lock()
						if firstErr == nil {
							firstErr = errors.Wrapf(delErr, "delete missing manifest snapshot %s", tag)
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
			if err = s.store.UpdateDetail(gCtx, repoKey, tag, &detail); err != nil {
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

// manifestGone 用 HEAD 确认 tag 的 manifest 是否已经不存在。
// HEAD 成功或返回非 404 时保留快照，避免把 config blob 等后续 404 误判成 tag 已删除。
func manifestGone(ctx context.Context, client *registry.Client, repoName, tag string) bool {
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

// completeDetailSync 完成详情同步，更新状态
func (s *DetailSyncer) completeDetailSync(ctx context.Context, repoKey, lastError string) error {
	detailSyncedAt := time.Now()
	status := &RepoSnapshotStatus{
		RepoKey:            repoKey,
		RefreshStatus:      RefreshStatusIdle,
		LastDetailSyncedAt: &detailSyncedAt,
		LastError:          lastError,
	}
	return s.store.UpsertStatus(ctx, status)
}
