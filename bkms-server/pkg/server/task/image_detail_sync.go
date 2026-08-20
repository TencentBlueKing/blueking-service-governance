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

// FIXME: 镜像详情同步已迁至 asynq（pkg/workload/image/snapshot.DetailSyncTask）
// 本文件仅保留 RabbitMQ 存量消费路径，存量队列耗尽后移除此实现

package task

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// imageDetailSync 镜像快照详情同步任务
func imageDetailSync(ctx context.Context, args snapshot.ImageDetailSyncArgs) (*EmptyResult, error) {
	log.Infof(ctx, "start image detail sync task %s", args)

	// 按需解密凭据
	username, err := args.Username()
	if err != nil {
		return nil, errors.Wrap(err, "decrypt username")
	}
	password, err := args.Password()
	if err != nil {
		return nil, errors.Wrap(err, "decrypt password")
	}

	snapshotStore, err := snapshot.NewSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create snapshot store")
	}

	info := &snapshot.RepoKeyInfo{
		RepoKey:  args.RepoKey,
		RepoName: args.RepoName,
		Username: username,
		Password: password,
	}

	detailSyncer := snapshot.NewDetailSyncer(snapshotStore)
	if err = detailSyncer.SyncDetails(ctx, info); err != nil {
		return nil, errors.Wrapf(err, "detail sync for %s failed", args.RepoKey)
	}

	log.Infof(ctx, "image detail sync task %s completed", args)
	return &emptyResult, nil
}
