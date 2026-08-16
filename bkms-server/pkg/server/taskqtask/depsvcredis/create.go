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

// CreateArgs 创建 Redis 任务参数。
// DBM 创建参数由 provider 侧 ToCreateRedisParams 一次性组装，worker 不再二次映射。
type CreateArgs struct {
	InstanceID string `json:"instanceID"`
	Username   string `json:"username"`
	// 空=Submit, 非空=Poll（值为 ticketID 字符串）
	Handle string `json:"handle"`

	// DBMParams 组装好的 DBM CreateRedis 参数
	DBMParams *dbm.CreateRedisParams `json:"dbmParams"`
}

// clusterName 返回创建后回查用的集群名。
// 集群模式取 DBMParams.ClusterName；主从模式取 Infos[0].ClusterName。
func (a CreateArgs) clusterName() string {
	if a.DBMParams == nil {
		return ""
	}
	if a.DBMParams.ClusterName != "" {
		return a.DBMParams.ClusterName
	}
	if len(a.DBMParams.Infos) > 0 {
		return a.DBMParams.Infos[0].ClusterName
	}
	return ""
}

func createHandler(ctx context.Context, args CreateArgs) error {
	objID, err := parseObjectID(args.InstanceID)
	if err != nil {
		return err
	}

	inst, err := instStore.Get(ctx, objID)
	if err != nil {
		if model.AsNotFoundError(err) {
			return errors.Wrapf(taskq.ErrStopRetry, "instance %s already gone", args.InstanceID)
		}
		return errors.Wrapf(taskq.ErrFixedRetry, "get instance %s: %v", args.InstanceID, err)
	}

	if args.Handle == "" {
		return createSubmit(ctx, objID, inst, args)
	}
	return createPoll(ctx, objID, args)
}

func createSubmit(ctx context.Context, objID bson.ObjectID, inst *model.ServiceInstance, args CreateArgs) error {
	// 若上次 Submit 已成功但 Enqueue Poll 失败，从 DB 恢复 ticket，避免重复开单
	if ticketID := cast.ToInt(inst.Config[configKeyCreateTicketID]); ticketID > 0 {
		args.Handle = strconv.Itoa(ticketID)
		return taskq.Enqueue(ctx, CreateTask.NewTask(args))
	}

	if args.DBMParams == nil {
		return failWithStopErr(
			ctx,
			args.InstanceID,
			model.CreateFailedStatus,
			errors.New("missing dbmParams in create task"),
		)
	}

	ticketID, err := dbmClient.CreateRedis(ctx, args.DBMParams, args.Username)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	// 先持久化 ticketID，再入队 Poll。Enqueue 失败时 asynq 重试会走上面的恢复分支，不会重复 CreateRedis。
	if err = instStore.PatchConfig(ctx, objID, map[string]any{configKeyCreateTicketID: ticketID}); err != nil {
		// 工单已创建但未能落库：停止重试以免重复开单，错误信息中保留 ticketID 便于人工恢复
		persistErr := errors.Wrapf(err, "persist createTicketID=%d after DBM submit", ticketID)
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, persistErr)
	}

	args.Handle = strconv.Itoa(ticketID)
	if err = taskq.Enqueue(ctx, CreateTask.NewTask(args)); err != nil {
		// ticket 已落库，返回可重试错误即可安全恢复
		return errors.Wrapf(taskq.ErrFixedRetry, "enqueue create poll task for ticket %d: %v", ticketID, err)
	}
	return nil
}

func createPoll(ctx context.Context, objID bson.ObjectID, args CreateArgs) error {
	ticketID, err := parseTicketID(args.Handle)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	done, err := pollTicket(ctx, ticketID, args.Username)
	if err != nil {
		if errors.Is(err, taskq.ErrStopRetry) {
			return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
		}
		return errors.Wrapf(taskq.ErrFixedRetry, "poll create ticket %d: %v", ticketID, err)
	}
	if !done {
		return errors.Wrapf(taskq.ErrFixedRetry, "create ticket %s in progress", args.Handle)
	}

	// 工单已 SUCCEEDED：后续收尾失败多为瞬时错误，必须可重试。
	// 仅在重试耗尽时由 OnExhausted 落 createFailed，并保留 createTicketID 便于人工对账。
	if args.DBMParams == nil {
		return failWithStopErr(
			ctx,
			args.InstanceID,
			model.CreateFailedStatus,
			errors.New("missing dbmParams in create task"),
		)
	}

	clusterName := args.clusterName()
	if clusterName == "" {
		// 参数缺陷无法靠重试恢复
		return failWithStopErr(
			ctx,
			args.InstanceID,
			model.CreateFailedStatus,
			errors.New("empty cluster name for lookup"),
		)
	}

	clusterInfo, err := dbmClient.FindClusterByName(
		ctx, args.DBMParams.BkBizID, clusterName, args.DBMParams.ClusterType, args.Username,
	)
	if err != nil {
		return errors.Wrapf(taskq.ErrFixedRetry, "find cluster by name %s: %v", clusterName, err)
	}

	if err = instStore.PatchConfig(ctx, objID, map[string]any{
		configKeyClusterID:   clusterInfo.ID,
		configKeyClusterName: clusterName,
		configKeyClusterType: string(args.DBMParams.ClusterType),
		configKeyDomain:      clusterInfo.Domain,
		configKeyPort:        clusterInfo.Port,
		configKeyBkBizID:     args.DBMParams.BkBizID,
	}); err != nil {
		return errors.Wrapf(taskq.ErrFixedRetry, "persist cluster config after create ticket: %v", err)
	}

	// 绑定 EnvVars 用 ${{env.KEY}} 引用这些 Credentials 键
	credentials := map[string]any{
		CredHost: clusterInfo.Domain,
		CredPort: strconv.Itoa(clusterInfo.Port),
	}
	// 密码为空时由 DBM 随机生成（集群模式还走 proxy_pwd），此处无从得知实际值，
	// 写入空串反而会向应用注入错误的 REDIS_PWD
	if args.DBMParams.RedisPwd != "" {
		credentials[CredPwd] = args.DBMParams.RedisPwd
	}
	if err = instStore.PatchCredentials(ctx, objID, credentials); err != nil {
		return errors.Wrapf(taskq.ErrFixedRetry, "persist credentials after create ticket: %v", err)
	}

	if err = instStore.UpdateStatus(ctx, objID, model.AvailableStatus, ""); err != nil {
		return errors.Wrapf(taskq.ErrFixedRetry, "mark instance available after create ticket: %v", err)
	}

	return nil
}
