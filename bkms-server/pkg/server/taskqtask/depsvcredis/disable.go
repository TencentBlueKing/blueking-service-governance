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

package depsvcredis

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// DisableArgs 禁用 Redis 任务参数
type DisableArgs struct {
	InstanceID string `json:"instanceID"`
	Username   string `json:"username"`
	// 空=Submit, 非空=Poll（值为 ticketID 字符串）
	Handle string `json:"handle"`
}

func disableHandler(ctx context.Context, args DisableArgs) error {
	objID, err := parseObjectID(args.InstanceID)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	if args.Handle == "" {
		return disableSubmit(ctx, objID, args)
	}
	return disablePoll(ctx, args)
}

func disableSubmit(ctx context.Context, objID bson.ObjectID, args DisableArgs) error {
	inst, err := instStore.Get(ctx, objID)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	// 恢复已提交但未成功入队 Poll 的工单，避免重复 Disable
	if ticketID := cast.ToInt(inst.Config[configKeyDisableTicketID]); ticketID > 0 {
		args.Handle = strconv.Itoa(ticketID)
		return taskq.Enqueue(ctx, DisableTask.NewTask(args))
	}

	ref, err := clusterRefFromConfig(inst.Config)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	ticketID, err := dbmClient.DisableRedis(ctx, &dbm.DisableRedisParams{
		BkBizID:    ref.BkBizID,
		TicketType: dbm.DisableTicketType(ref.ClusterType),
		ClusterID:  ref.ClusterID,
	}, args.Username)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	if err = instStore.PatchConfig(ctx, objID, map[string]any{configKeyDisableTicketID: ticketID}); err != nil {
		persistErr := errors.Wrapf(err, "persist disableTicketID=%d after DBM submit", ticketID)
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, persistErr)
	}

	args.Handle = strconv.Itoa(ticketID)
	if err = taskq.Enqueue(ctx, DisableTask.NewTask(args)); err != nil {
		return errors.Wrapf(taskq.ErrFixedRetry, "enqueue disable poll task for ticket %d: %v", ticketID, err)
	}
	return nil
}

func disablePoll(ctx context.Context, args DisableArgs) error {
	ticketID, err := parseTicketID(args.Handle)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
	}

	done, err := pollTicket(ctx, ticketID, args.Username)
	if err != nil {
		if errors.Is(err, taskq.ErrStopRetry) {
			return failWithStopErr(ctx, args.InstanceID, model.DeleteFailedStatus, err)
		}
		return errors.Wrapf(taskq.ErrFixedRetry, "poll disable ticket %d: %v", ticketID, err)
	}
	if !done {
		return errors.Wrapf(taskq.ErrFixedRetry, "disable ticket %s in progress", args.Handle)
	}

	// ─── 完成 → 串联 DestroyTask ───
	return taskq.Enqueue(ctx, DestroyTask.NewTask(DestroyArgs{
		InstanceID: args.InstanceID,
		Username:   args.Username,
	}))
}
