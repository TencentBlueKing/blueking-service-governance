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

package devmode

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// PublishRecordStore 开发模式发布记录存储
type PublishRecordStore = store.GenericStore[PublishRecord]

// PublishRecordMeta 开发模式发布记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - appID + envName + createdAt(desc)：按环境分页列出记录
// - appID + envName + instance + createdAt(desc)：取每个实例最新的一条记录
var PublishRecordMeta = store.Meta[PublishRecord]{Collection: "devmode_publish_records"}

// ByAppEnv 按应用与环境匹配
func ByAppEnv(appID, envName string) store.Opt {
	return store.Combine(store.Where("appID", appID), store.Where("envName", envName))
}

// KeywordMatches 按二进制名称、操作人或实例名模糊匹配（不区分大小写），keyword 为空白时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "binaryName", "operator", "instance")
}

// NewestFirst 按创建时间倒序；同一次上报的记录创建时间相同，再按 _id 倒序保证翻页稳定
func NewestFirst() store.Opt {
	return store.Combine(store.Desc("createdAt"), store.Desc("_id"))
}

// ListLatestByInstance 返回指定实例中每个实例最新的一条发布记录，key 为实例名，没有记录的实例不在其中
func ListLatestByInstance(
	ctx context.Context, db *mongo.Database, appID, envName string, instanceIDs []string,
) (map[string]*PublishRecord, error) {
	if len(instanceIDs) == 0 {
		return map[string]*PublishRecord{}, nil
	}
	records, err := store.Of(db, PublishRecordMeta).ListLatestBy(ctx, []string{"instance"},
		ByAppEnv(appID, envName),
		store.Where("instance", bson.M{"$in": instanceIDs}),
		store.Asc("instance"), store.Desc("createdAt"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list latest publish records for app %s env %s", appID, envName)
	}
	return lo.SliceToMap(records, func(r PublishRecord) (string, *PublishRecord) { return r.Instance, &r }), nil
}
