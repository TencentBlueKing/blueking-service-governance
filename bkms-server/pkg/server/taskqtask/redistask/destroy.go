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

package redistask

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// DestroyArgs 销毁 Redis 任务参数
type DestroyArgs struct {
	InstanceID string `json:"instanceID"`
	Username   string `json:"username"`
	Handle     string `json:"handle"` // 空=Submit, 非空=Poll
}

func destroyHandler(ctx context.Context, args DestroyArgs) error {
	objID, err := parseObjectID(args.InstanceID)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	if args.Handle == "" {
		return destroySubmit(ctx, objID, args)
	}
	return destroyPoll(ctx, objID, args)
}

func destroySubmit(ctx context.Context, objID bson.ObjectID, args DestroyArgs) error {
	inst, err := instStore.Get(ctx, objID)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	// 恢复已提交但未成功入队 Poll 的工单，避免重复 Destroy
	if ticketID := cast.ToInt(inst.Config[configKeyDestroyTicketID]); ticketID > 0 {
		args.Handle = strconv.Itoa(ticketID)
		return taskq.Enqueue(ctx, DestroyTask.NewTask(args))
	}

	ref, err := clusterRefFromConfig(inst.Config)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	ticketID, err := dbmClient.DeleteRedis(ctx, &dbm.DeleteRedisParams{
		BkBizID:    ref.BkBizID,
		TicketType: dbm.DeleteTicketType(ref.ClusterType),
		ClusterID:  ref.ClusterID,
	}, args.Username)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	if err = instStore.PatchConfig(ctx, objID, map[string]any{configKeyDestroyTicketID: ticketID}); err != nil {
		persistErr := fmt.Errorf("persist destroyTicketID=%d after DBM submit: %w", ticketID, err)
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, persistErr)
	}

	args.Handle = strconv.Itoa(ticketID)
	if err = taskq.Enqueue(ctx, DestroyTask.NewTask(args)); err != nil {
		return fmt.Errorf("enqueue destroy poll task for ticket %d: %w: %w", ticketID, err, taskq.ErrFixedRetry)
	}
	return nil
}

func destroyPoll(ctx context.Context, objID bson.ObjectID, args DestroyArgs) error {
	ticketID, err := parseTicketID(args.Handle)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	done, err := pollTicket(ctx, ticketID, args.Username)
	if err != nil {
		if errors.Is(err, taskq.ErrStopRetry) {
			return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
		}
		return fmt.Errorf("poll destroy ticket %d: %w: %w", ticketID, err, taskq.ErrFixedRetry)
	}
	if !done {
		return fmt.Errorf("destroy ticket %s in progress: %w", args.Handle, taskq.ErrFixedRetry)
	}

	// ─── 完成: 删除实例记录；失败需重试，避免永久卡在 deleting ───
	if err = instStore.Delete(ctx, objID); err != nil {
		if model.AsNotFoundError(err) {
			log.Infof(ctx, "redistask: instance %s already deleted after destroy", args.InstanceID)
			return nil
		}
		return fmt.Errorf("delete instance %s after destroy: %w: %w", args.InstanceID, err, taskq.ErrFixedRetry)
	}
	return nil
}
