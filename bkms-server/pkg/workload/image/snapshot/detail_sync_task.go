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

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

const (
	// detailSyncTaskName asynq 任务类型名
	detailSyncTaskName = "taskq.imageDetailSync"
	// detailSyncMaxRetry 单次任务意外失败的 asynq 重试上限，覆盖进程重启接续
	detailSyncMaxRetry = 10
)

// DetailSyncTask 镜像详情同步任务；init 赋值避免与投递侧引用形成包初始化环
var DetailSyncTask *taskq.TaskType[ImageDetailSyncArgs]

func init() {
	DetailSyncTask = taskq.NewTaskType[ImageDetailSyncArgs](
		detailSyncTaskName, handleImageDetailSync, asynq.MaxRetry(detailSyncMaxRetry),
	)
}

// handleImageDetailSync asynq 入口：按现网口径建 store，解密失败停重试，其余错误交给 asynq 重试
func handleImageDetailSync(ctx context.Context, args ImageDetailSyncArgs) error {
	store, err := NewSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create snapshot store")
	}
	return NewDetailSyncManager(store).Handle(ctx, args)
}

// DetailSyncManager 执行一次镜像详情同步
type DetailSyncManager struct {
	store SnapshotStore
}

// NewDetailSyncManager 注入快照 store，供 asynq handler 与单测共用
func NewDetailSyncManager(store SnapshotStore) *DetailSyncManager {
	return &DetailSyncManager{store: store}
}

// Handle 解密仓库凭据后调用 DetailSyncer 补全 tag 详情。
// 解密失败属不可恢复，wrap taskq.ErrStopRetry；SyncDetails 的 store 类错误原样返回，
// 由 asynq MaxRetry 重试。部分 tag 拉取失败由 SyncDetails 写 LastError 后成功返回，不在此改判。
func (m *DetailSyncManager) Handle(ctx context.Context, args ImageDetailSyncArgs) error {
	if m.store == nil {
		return errors.Wrap(taskq.ErrStopRetry, "snapshot store not initialized")
	}

	log.Infof(ctx, "start image detail sync task %s", args)

	username, err := args.Username()
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "decrypt username: %v", err)
	}
	password, err := args.Password()
	if err != nil {
		return errors.Wrapf(taskq.ErrStopRetry, "decrypt password: %v", err)
	}

	info := &RepoKeyInfo{
		RepoKey:  args.RepoKey,
		RepoName: args.RepoName,
		Username: username,
		Password: password,
	}
	if err = NewDetailSyncer(m.store).SyncDetails(ctx, info); err != nil {
		return errors.Wrapf(err, "detail sync for %s failed", args.RepoKey)
	}

	log.Infof(ctx, "image detail sync task %s completed", args)
	return nil
}
