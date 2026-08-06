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

package promotion

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDB 集合名
const (
	// collImagePromotions 镜像晋级记录集合
	collImagePromotions = "image_promotions"
)

// PromotionStore 定义镜像晋级记录的存储接口
type PromotionStore interface {
	// Upsert 创建或更新晋级记录（以 appID + repoKey + tag 为条件执行幂等 upsert）
	Upsert(ctx context.Context, appID, repoKey, tag, operator string) error

	// ListByAppAndRepoKey 按 appID + repoKey 批量查询该应用下所有已晋级的 Tag 记录
	ListByAppAndRepoKey(ctx context.Context, appID, repoKey string) ([]Image, error)

	// ListTagsByAppAndRepoKey 按 appID + repoKey 查询该应用下所有已晋级的 tag 字符串列表（投影查询，仅取 tag 字段）
	ListTagsByAppAndRepoKey(ctx context.Context, appID, repoKey string) ([]string, error)

	// IsTagPromoted 精确查询指定 appID + repoKey + tag 的晋级记录是否存在
	IsTagPromoted(ctx context.Context, appID, repoKey, tag string) (bool, error)

	// DeleteByApp 删除应用下所有晋级记录
	DeleteByApp(ctx context.Context, appID string) error

	// DeleteTag 删除单个标签的晋级记录。
	// 若记录不存在，视为幂等删除成功并返回 nil。
	DeleteTag(ctx context.Context, appID, repoKey, tag string) error

	// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
	DeleteAll(ctx context.Context) error
}

// 编译期接口实现检查
var _ PromotionStore = &PromotionStoreMongo{}

// PromotionStoreMongo MongoDB 实现
type PromotionStoreMongo struct {
	coll *mongo.Collection
}

// NewPromotionStoreMongo 创建 PromotionStoreMongo 实例。
func NewPromotionStoreMongo(client *mongo.Client, dbName string) (*PromotionStoreMongo, error) {
	db := client.Database(dbName)
	coll := db.Collection(collImagePromotions)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + repoKey + tag
	store := &PromotionStoreMongo{coll: coll}
	return store, nil
}

// Upsert 创建或更新晋级记录（幂等操作）
// 重复晋级时会更新 promotedBy / promotedAt 为最后一次操作的信息
func (s *PromotionStoreMongo) Upsert(ctx context.Context, appID, repoKey, tag, operator string) error {
	now := time.Now()

	filter := bson.M{"appID": appID, "repoKey": repoKey, "tag": tag}
	update := bson.M{
		"$set": bson.M{
			"promotedAt": now,
			"promotedBy": operator,
			"updatedAt":  now,
		},
		"$setOnInsert": bson.M{
			"appID":     appID,
			"repoKey":   repoKey,
			"tag":       tag,
			"createdAt": now,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return errors.Wrapf(err, "upsert promotion for app %s, repo %s, tag %s", appID, repoKey, tag)
	}
	return nil
}

// ListByAppAndRepoKey 按 appID + repoKey 批量查询晋级记录
func (s *PromotionStoreMongo) ListByAppAndRepoKey(ctx context.Context, appID, repoKey string) ([]Image, error) {
	filter := bson.M{"appID": appID, "repoKey": repoKey}

	cursor, err := s.coll.Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrapf(err, "find promotions for app %s, repo %s", appID, repoKey)
	}
	defer cursor.Close(ctx)

	var results []Image
	if err = cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode promotions")
	}
	return results, nil
}

// ListTagsByAppAndRepoKey 按 appID + repoKey 查询已晋级的 tag 字符串列表
func (s *PromotionStoreMongo) ListTagsByAppAndRepoKey(
	ctx context.Context,
	appID, repoKey string,
) ([]string, error) {
	filter := bson.M{"appID": appID, "repoKey": repoKey}

	// 使用 Projection 只返回 tag 字段，避免传输不必要的文档内容
	cursor, err := s.coll.Find(ctx, filter, options.Find().SetProjection(bson.M{"tag": 1}))
	if err != nil {
		return nil, errors.Wrapf(err, "find promoted tags for app %s, repo %s", appID, repoKey)
	}
	defer cursor.Close(ctx)

	var tags []string
	for cursor.Next(ctx) {
		var doc struct {
			Tag string `bson:"tag"`
		}
		if err = cursor.Decode(&doc); err != nil {
			return nil, errors.Wrap(err, "decode promoted tag")
		}
		tags = append(tags, doc.Tag)
	}

	return tags, nil
}

// IsTagPromoted 精确查询指定 appID + repoKey + tag 的晋级记录是否存在
func (s *PromotionStoreMongo) IsTagPromoted(ctx context.Context, appID, repoKey, tag string) (bool, error) {
	filter := bson.M{"appID": appID, "repoKey": repoKey, "tag": tag}
	count, err := s.coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, errors.Wrapf(err, "check promotion exists for app %s, repo %s, tag %s", appID, repoKey, tag)
	}
	return count > 0, nil
}

// DeleteByApp 删除应用下所有晋级记录
func (s *PromotionStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.coll.DeleteMany(ctx, bson.M{"appID": appID})
	return err
}

// DeleteTag 删除 tag 晋级记录
// 采用幂等删除，未查到时不报错
func (s *PromotionStoreMongo) DeleteTag(
	ctx context.Context,
	appID, repoKey, tag string,
) error {
	_, err := s.coll.DeleteOne(ctx, bson.M{
		"appID":   appID,
		"repoKey": repoKey,
		"tag":     tag,
	})
	if err != nil {
		return errors.Wrapf(err, "delete promotion for app %s, repo %s, tag %s", appID, repoKey, tag)
	}
	return nil
}

// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
func (s *PromotionStoreMongo) DeleteAll(ctx context.Context) error {
	if _, err := s.coll.DeleteMany(ctx, bson.M{}); err != nil {
		return errors.Wrap(err, "delete promotion documents")
	}
	return nil
}
