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

// Package audit provides user operation audit records functions.
package audit

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// AddOperationRecord 添加操作记录，返回新记录的 ID（hex）和错误信息
// NOTE:
// 1. 这是同步版本，适用于需要获取记录 ID 或处理错误的场景。
// 2. 如果不需要返回值，推荐使用 AddOperationRecordAsync 异步版本。
// 3. 返回 Record ID 的目的是某些特定场景可能需要二次更新操作记录（如：后台任务执行）。
//
// 使用示例：
//
//	// 基本用法：记录创建应用操作
//	recordID, err := AddOperationRecord(ctx, "admin", OperationTypeCreate, ResourceTypeApp, appID)
//	if err != nil {
//	    return err
//	}
//
//	// 带选项：记录更新操作并附加操作数据
//	recordID, err := AddOperationRecord(
//	    ctx,
//	    "blueking",
//	    OperationTypeUpdate,
//	    ResourceTypeApp,
//	    appID,
//	    WithAttribute("name"),
//	    WithBeforeData(map[string]any{"appName": "old-name"}),
//	    WithAfterData(map[string]any{"appName": "new-name"}),
//	)
//	if err != nil {
//	    return err
//	}
func AddOperationRecord(
	ctx context.Context,
	username string,
	operationType OperationType,
	resourceType ResourceType,
	resourceID string,
	opts ...Option,
) (string, error) {
	// 获取 store
	store, err := NewOperationRecordStoreMongo(database.Client(), database.Name())
	if err != nil {
		return "", errors.Wrap(err, "new operation record store")
	}

	// 将数据写入数据库
	id, err := store.Create(ctx, newOperationRecord(
		username, operationType, resourceType, resourceID, opts...,
	))
	if err != nil {
		return "", errors.Wrapf(
			err, "create operation record (username: %s, opType: %s, resType: %s, resID: %s)",
			username, operationType, resourceType, resourceID,
		)
	}
	return id, nil
}

// AddOperationRecordAsync 异步添加操作记录，不返回任何值
// NOTE:
// 1. 推荐使用 `go audit.AddOperationRecordAsync()` 的方式执行。
// 2. 内部会自动记录错误日志，调用者无需处理错误。
// 3. 如果需要获取记录 ID 或处理错误，请使用 AddOperationRecord 同步版本。
//
// 使用示例：
//
//	// 基本用法：异步记录创建应用操作
//	go AddOperationRecordAsync(ctx, OperationTypeCreate, ResourceTypeApp, appID)
//
//	// 异步记录删除操作
//	go AddOperationRecordAsync(ctx, OperationTypeDelete, ResourceTypeWorkspace, workspaceID)
//
// // 带选项：异步记录更新操作并附加操作数据
// go AddOperationRecordAsync(
//
//	    ctx,
//		OperationTypeUpdate,
//		ResourceTypeApp,
//		appID,
//		WithAttribute("name"),
//		WithBeforeData(map[string]any{"appName": "old-name"}),
//		WithAfterData(map[string]any{"appName": "new-name"}),
//
// )
//
// // 带分组信息：异步记录删除操作并附加操作数据
// go AddOperationRecordAsync(
//
//	   ctx,
//	   OperationTypeDelete,
//	   ResourceTypeApp,
//	   appID,
//	   ...
//	   WithGroup(OperationGroup{
//	      WorkspaceID: "workspace-abcd",
//	      AppID:       "app-abcd",
//	      EnvID:       "6913216fdbbfcca1c0f403cf"
//	}),
//
// )
func AddOperationRecordAsync(
	ctx context.Context,
	operationType OperationType,
	resourceType ResourceType,
	resourceID string,
	opts ...Option,
) {
	// 复制上下文，避免协程执行中途被取消
	ctx = context.WithoutCancel(ctx)
	// 从 ctx 中获取用户名
	username := auth.MustGetUser(ctx).ID

	if _, err := AddOperationRecord(ctx, username, operationType, resourceType, resourceID, opts...); err != nil {
		log.Errorf(ctx, "add operation record failed: %v", err)
		return
	}
	log.Debugf(
		ctx, "operation record created (username: %s, opType: %s, resType: %s, resID: %s)",
		username, operationType, resourceType, resourceID,
	)
}
