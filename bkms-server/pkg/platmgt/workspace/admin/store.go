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

package admin

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrRecordNotFound 表示目标临时管理员记录不存在。
	ErrRecordNotFound = errors.New("temporary admin record not found")
	// ErrRecordAlreadyExists 表示已存在未回收的临时管理员记录。
	ErrRecordAlreadyExists = errors.New("temporary admin record already exists")
)

// Store 持久化临时管理员记录。
type Store = store.GenericStore[WorkspaceTempAdmin]

// Meta 临时管理员授权记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 条件唯一：workspaceID + username（仅 isRecycled = false）
// - 查询提速：isRecycled + expiresAt
var Meta = store.Meta[WorkspaceTempAdmin]{
	Collection: collectionName,
	NotFound:   ErrRecordNotFound,
	Duplicate:  ErrRecordAlreadyExists,
}

// ByActiveGrant 匹配某个空间/用户当前仍生效（未回收）的临时管理员授权记录。
// 未回收记录受 partial unique index 约束，因此至多匹配一条。
func ByActiveGrant(workspaceID, username string) store.Opt {
	return store.Filter(bson.M{"workspaceID": workspaceID, "username": username, "isRecycled": false})
}

// ExpiredGrants 匹配截至 now 已过期且未回收的记录，未设置过期时间（零值）的记录不在其列。
func ExpiredGrants(now time.Time) store.Opt {
	return store.Filter(bson.M{
		"isRecycled": false,
		"expiresAt":  bson.M{"$lte": now, "$ne": time.Time{}},
	})
}

// updateRecord 按 ID 更新一条临时管理员记录的全部可变字段，workspaceID、username、creator 不可变。
func updateRecord(ctx context.Context, records *Store, record *WorkspaceTempAdmin) error {
	mutable := store.UpdateSelected("expiresAt", "isRecycled", "updater")
	if err := records.Update(ctx, record, mutable, store.ByID(record.ID)); err != nil {
		return errors.Wrapf(err, "update temporary admin record %s", record.ID.Hex())
	}
	return nil
}
