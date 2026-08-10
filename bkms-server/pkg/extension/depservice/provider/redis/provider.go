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

package redis

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/depsvcredis"
)

// Provider 实现 ServiceProvider 接口，通过投递 asynq task 实现异步 Redis 生命周期管理。
type Provider struct{}

// NewProvider 创建一个 Redis Provider。
func NewProvider() *Provider {
	return &Provider{}
}

// CreateInstance 投递 Redis 创建任务，返回 Async=true 表示异步创建已投递。
func (p *Provider) CreateInstance(
	ctx context.Context,
	instID string,
	_ *types.ServicePlanConfig,
	params types.ProvisionParams,
) (*types.CreateInstanceResult, error) {
	redisParams, ok := params.(*CreateParams)
	if !ok {
		return nil, errors.New("invalid params type for redis service, expected *redis.CreateParams")
	}
	if err := redisParams.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate redis create params")
	}

	operator := auth.MustGetUser(ctx).ID

	task := depsvcredis.CreateTask.NewTask(depsvcredis.CreateArgs{
		InstanceID: instID,
		Username:   operator,
		DBMParams:  redisParams.ToCreateRedisParams(),
	})
	if err := taskq.Enqueue(ctx, task); err != nil {
		return nil, errors.Wrap(err, "enqueue redis create task")
	}

	return &types.CreateInstanceResult{Async: true}, nil
}

// DeleteInstance 投递 Redis 禁用任务（完成后自动串联销毁），返回 Async=true。
//
// Manager 已保证仅在稳定态（available/unavailable/deleteFailed）调用本方法，
// 因此此处总是投递销毁链路，不再处理创建中取消。
func (p *Provider) DeleteInstance(
	ctx context.Context,
	instID string,
	_ *types.ServicePlanConfig,
	_ map[string]any,
) (*types.DeleteInstanceResult, error) {
	operator := auth.MustGetUser(ctx).ID

	task := depsvcredis.DisableTask.NewTask(depsvcredis.DisableArgs{
		InstanceID: instID,
		Username:   operator,
	})
	if err := taskq.Enqueue(ctx, task); err != nil {
		return nil, errors.Wrap(err, "enqueue redis disable task")
	}

	return &types.DeleteInstanceResult{Async: true}, nil
}
