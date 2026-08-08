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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

// CreateArgs 创建 Redis 任务参数
type CreateArgs struct {
	InstanceID string `json:"instanceID"`
	Username   string `json:"username"`
	Handle     string `json:"handle"` // 空=Submit, 非空=Poll（值为 ticketID 字符串）

	// DBM CreateRedis 所需参数（Submit 时使用）
	BkBizID                int               `json:"bkBizID"`
	TicketType             dbm.TicketType    `json:"ticketType"`
	BkCloudID              int               `json:"bkCloudID"`
	DBAppAbbr              string            `json:"dbAppAbbr"`
	ClusterType            dbm.ClusterType   `json:"clusterType"`
	ClusterName            string            `json:"clusterName"` // 集群模式写 details.cluster_name；主从模式写 Infos[].cluster_name，并作回查名
	ClusterAlias           string            `json:"clusterAlias"`
	DBVersion              string            `json:"dbVersion"`
	ProxyPort              int               `json:"proxyPort"`
	ClusterShardNum        int               `json:"clusterShardNum"`
	IPSource               string            `json:"ipSource"`
	ResourceSpec           *dbm.ResourceSpec `json:"resourceSpec"`
	DisasterToleranceLevel string            `json:"disasterToleranceLevel"`
	Port                   int               `json:"port"`
	Databases              int               `json:"databases"`
	RedisPwd               string            `json:"redisPwd"`
	// Infos 主从部署（REDIS_INS_APPLY）专用；由 provider 侧按 ClusterName/Databases 预组装
	Infos []dbm.RedisInsInfo `json:"infos,omitempty"`
}

// toCreateRedisParams 按工单类型组装 DBM 创建参数。
// REDIS_INS_APPLY 不消费顶层 ClusterName，使用 Infos（缺省时回退 ClusterName+Databases）。
func (a CreateArgs) toCreateRedisParams() *dbm.CreateRedisParams {
	params := &dbm.CreateRedisParams{
		BkBizID:                a.BkBizID,
		TicketType:             a.TicketType,
		BkCloudID:              a.BkCloudID,
		DBAppAbbr:              a.DBAppAbbr,
		ClusterType:            a.ClusterType,
		DBVersion:              a.DBVersion,
		IPSource:               a.IPSource,
		ResourceSpec:           a.ResourceSpec,
		DisasterToleranceLevel: a.DisasterToleranceLevel,
	}
	if a.TicketType == dbm.TicketTypeRedisInsApply {
		params.Port = a.Port
		params.RedisPwd = a.RedisPwd
		// AppendApply 保持零值 false：只支持新建
		params.Infos = a.Infos
		if len(params.Infos) == 0 && a.ClusterName != "" {
			params.Infos = []dbm.RedisInsInfo{{
				ClusterName: a.ClusterName,
				Databases:   a.Databases,
			}}
		}
		return params
	}
	params.ClusterName = a.ClusterName
	params.ClusterAlias = a.ClusterAlias
	params.ProxyPort = a.ProxyPort
	params.ClusterShardNum = a.ClusterShardNum
	return params
}

func createHandler(ctx context.Context, args CreateArgs) error {
	objID, err := parseObjectID(args.InstanceID)
	if err != nil {
		return err
	}

	inst, err := instStore.Get(ctx, objID)
	if err != nil {
		if model.AsNotFoundError(err) {
			return fmt.Errorf("instance %s already gone: %w", args.InstanceID, taskq.ErrStopRetry)
		}
		return fmt.Errorf("get instance %s: %w: %w", args.InstanceID, err, taskq.ErrFixedRetry)
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

	ticketID, err := dbmClient.CreateRedis(ctx, args.toCreateRedisParams(), args.Username)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	// 先持久化 ticketID，再入队 Poll。Enqueue 失败时 asynq 重试会走上面的恢复分支，不会重复 CreateRedis。
	if err = instStore.PatchConfig(ctx, objID, map[string]any{configKeyCreateTicketID: ticketID}); err != nil {
		// 工单已创建但未能落库：停止重试以免重复开单，错误信息中保留 ticketID 便于人工恢复
		persistErr := fmt.Errorf("persist createTicketID=%d after DBM submit: %w", ticketID, err)
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, persistErr)
	}

	args.Handle = strconv.Itoa(ticketID)
	if err = taskq.Enqueue(ctx, CreateTask.NewTask(args)); err != nil {
		// ticket 已落库，返回可重试错误即可安全恢复
		return fmt.Errorf("enqueue create poll task for ticket %d: %w: %w", ticketID, err, taskq.ErrFixedRetry)
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
		return fmt.Errorf("poll create ticket %d: %w: %w", ticketID, err, taskq.ErrFixedRetry)
	}
	if !done {
		return fmt.Errorf("create ticket %s in progress: %w", args.Handle, taskq.ErrFixedRetry)
	}

	// ClusterName 在主从模式下对应 Infos[].cluster_name，与提交工单时一致，可用于回查
	clusterInfo, err := dbmClient.FindClusterByName(
		ctx, args.BkBizID, args.ClusterName, args.ClusterType, args.Username,
	)
	if err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	if err = instStore.PatchConfig(ctx, objID, map[string]any{
		configKeyClusterID:   clusterInfo.ID,
		configKeyClusterName: args.ClusterName,
		configKeyClusterType: string(args.ClusterType),
		configKeyDomain:      clusterInfo.Domain,
		configKeyPort:        clusterInfo.Port,
		configKeyBkBizID:     args.BkBizID,
	}); err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	if args.RedisPwd != "" {
		if err = instStore.PatchCredentials(ctx, objID, map[string]any{
			"password": args.RedisPwd,
		}); err != nil {
			return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
		}
	}

	if err = instStore.UpdateStatus(ctx, objID, model.AvailableStatus, ""); err != nil {
		return failWithStopErr(ctx, args.InstanceID, model.CreateFailedStatus, err)
	}

	return nil
}
