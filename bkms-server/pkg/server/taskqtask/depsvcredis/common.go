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

// Package depsvcredis 实现依赖服务（depservice）Redis 实例生命周期的 asynq 异步任务。
//
// 包含三个独立的 TaskType：CreateTask（创建）、DisableTask（禁用）、DestroyTask（销毁），
// 通过 handler 末尾硬编码 Enqueue 实现串联（禁用完成后自动触发销毁）。
//
// TaskType 在包加载时完成注册，投递侧（webserver）无需任何初始化；消费侧（worker）
// 在启动阶段调用一次 Init，准备执行任务所需的 DBM client 与 ServiceInstanceStore。
package depsvcredis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/dbm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

var (
	dbmClient dbm.Client
	instStore model.ServiceInstanceStore
)

// Init 初始化 depsvcredis 包依赖。在 worker 启动阶段（store 初始化之后）调用一次。
//
// DBM client 在此构造而非首次执行任务时惰性构造，让配置缺失等问题在启动阶段就暴露。
func Init(store model.ServiceInstanceStore) error {
	client, err := dbm.New()
	if err != nil {
		return err
	}
	dbmClient, instStore = client, store
	return nil
}

// pollTicket 查询 DBM 工单状态。
// 返回 (true, nil) = 成功完成；
// 返回 (false, nil) = 仍在进行；
// 返回 (false, ErrStopRetry-wrapped) = 终态失败，不应重试；
// 返回 (false, other) = 瞬时错误，可重试。
func pollTicket(ctx context.Context, ticketID int, username string) (bool, error) {
	ticket, err := dbmClient.GetTicketStatus(ctx, ticketID, username)
	if err != nil {
		return false, err
	}
	switch ticket.Status {
	case dbm.TicketStatusSucceeded:
		return true, nil
	case dbm.TicketStatusFailed, dbm.TicketStatusTerminated:
		return false, errors.Wrapf(taskq.ErrStopRetry, "DBM ticket %d terminal status: %s", ticketID, ticket.Status)
	default:
		return false, nil // PENDING / RUNNING
	}
}

// failWithStopErr 标记实例状态为失败并返回 ErrStopRetry 终止 task 重试
func failWithStopErr(ctx context.Context, instID string, status model.InstanceStatus, err error) error {
	metrics.DepserviceRedisFailed(operationFromStatus(status))
	objID, parseErr := bson.ObjectIDFromHex(instID)
	if parseErr != nil {
		log.Errorf(ctx, "depsvcredis: invalid instID %q: %v", instID, parseErr)
		return errors.Wrap(taskq.ErrStopRetry, "invalid instID")
	}
	if updateErr := instStore.UpdateStatus(ctx, objID, status, err.Error()); updateErr != nil {
		log.Errorf(ctx, "depsvcredis: update status for %s: %v", instID, updateErr)
	}
	return errors.Wrap(taskq.ErrStopRetry, err.Error())
}

// failOnExhausted 在重试耗尽回调中标记实例为失败状态
func failOnExhausted(ctx context.Context, instID string, status model.InstanceStatus, lastErr error) {
	metrics.DepserviceRedisFailed(operationFromStatus(status))
	objID, parseErr := bson.ObjectIDFromHex(instID)
	if parseErr != nil {
		log.Errorf(ctx, "depsvcredis exhausted: invalid instID %q: %v", instID, parseErr)
		return
	}
	msg := fmt.Sprintf("task exhausted: %v", lastErr)
	if updateErr := instStore.UpdateStatus(ctx, objID, status, msg); updateErr != nil {
		log.Errorf(ctx, "depsvcredis exhausted: update status for %s: %v", instID, updateErr)
	}
}

// clusterRef 是 disable/destroy 提交 DBM 工单所需的集群定位信息。
type clusterRef struct {
	ClusterID   int
	BkBizID     int
	ClusterType dbm.ClusterType
}

// clusterRefFromConfig 从实例 Config 解析集群定位信息。
func clusterRefFromConfig(cfg map[string]any) (clusterRef, error) {
	ref := clusterRef{
		ClusterID:   cast.ToInt(cfg[configKeyClusterID]),
		BkBizID:     cast.ToInt(cfg[configKeyBkBizID]),
		ClusterType: dbm.ClusterType(cast.ToString(cfg[configKeyClusterType])),
	}
	if ref.ClusterID <= 0 || ref.BkBizID <= 0 || ref.ClusterType == "" {
		return clusterRef{}, errors.Errorf(
			"incomplete cluster config: clusterID=%d bkBizID=%d clusterType=%q",
			ref.ClusterID, ref.BkBizID, ref.ClusterType,
		)
	}
	return ref, nil
}

// parseObjectID 解析实例 ID，失败时终止重试。
func parseObjectID(instID string) (bson.ObjectID, error) {
	objID, err := bson.ObjectIDFromHex(instID)
	if err != nil {
		return bson.NilObjectID, errors.Wrapf(taskq.ErrStopRetry, "invalid instID %q", instID)
	}
	return objID, nil
}

// parseTicketID 解析 ticket handle，失败时终止重试。
func parseTicketID(handle string) (int, error) {
	ticketID, err := strconv.Atoi(handle)
	if err != nil || ticketID <= 0 {
		return 0, errors.Wrapf(taskq.ErrStopRetry, "invalid ticket handle %q", handle)
	}
	return ticketID, nil
}

// operationFromStatus 根据实例终态状态推断操作类型，用于指标上报。
func operationFromStatus(status model.InstanceStatus) string {
	switch status {
	case model.CreateFailedStatus:
		return "create"
	default:
		return "delete"
	}
}
