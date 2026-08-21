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

package customruntime

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// collCustomRuntimeImages 工作空间自定义运行时镜像集合
	collCustomRuntimeImages = "custom_runtime_images"
)

// Store 工作空间自定义运行时镜像存储接口。
//
// 记录归属工作空间而非应用，因此接口不提供任何以 appID 为条件的查询或删除方法。
type Store interface {
	// Upsert 幂等写入自定义运行时镜像记录。
	//
	// 以 workspaceID + type + name 为唯一键：命中已有记录时只刷新 updatedAt，
	// 其余字段保持不变。调用方无需在写入前先查重。
	Upsert(ctx context.Context, image *Image) error

	// GetByWorkspaceTypeAndName 按工作空间、镜像类型与仓库名称获取记录，
	// 未命中返回 ErrCustomRuntimeImageNotFound
	GetByWorkspaceTypeAndName(
		ctx context.Context,
		workspaceID string,
		imageType ImageType,
		name string,
	) (*Image, error)

	// List 查询指定工作空间下的自定义运行时镜像记录列表。
	//
	// 候选口径仅以落库记录为准：不过滤快照同步状态，也不校验镜像在 registry 中是否
	// 仍然存在。结果不分页，预期单工作空间的记录数量在百条以内。
	List(ctx context.Context, workspaceID string, opts ListOptions) ([]Image, error)

	// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
	DeleteAll(ctx context.Context) error
}

// 编译期接口实现检查
var _ Store = &StoreMongo{}

// StoreMongo Store 的 MongoDB 实现
type StoreMongo struct {
	coll *mongo.Collection
}

// NewStoreMongo 创建 StoreMongo。
func NewStoreMongo(client *mongo.Client, dbName string) (*StoreMongo, error) {
	coll := client.Database(dbName).Collection(collCustomRuntimeImages)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + type + name
	store := &StoreMongo{coll: coll}
	return store, nil
}

// Upsert 幂等写入自定义运行时镜像记录。
//
// 命中已有记录时只刷新 updatedAt，其余字段保持不变。
func (s *StoreMongo) Upsert(ctx context.Context, image *Image) error {
	if err := image.Validate(); err != nil {
		return err
	}

	// 唯一键按 trim 后的值落库，后续 Get/List 也用同一口径，避免空白差异造成查不到
	image.WorkspaceID = strings.TrimSpace(image.WorkspaceID)
	image.Name = strings.TrimSpace(image.Name)

	now := time.Now()
	filter := bson.M{
		"workspaceID": image.WorkspaceID,
		"type":        image.Type,
		"name":        image.Name,
	}

	id := image.ID
	if id == "" {
		id = bson.NewObjectID().Hex()
	}

	// $set 只刷新 updatedAt；身份字段放在 $setOnInsert，命中已有记录时不得被覆盖
	update := bson.M{
		"$set": bson.M{
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{
			"_id":         id,
			"workspaceID": image.WorkspaceID,
			"type":        image.Type,
			"name":        image.Name,
			"createdAt":   now,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var result Image
	if err := s.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result); err != nil {
		return errors.Wrapf(
			err, "upsert custom runtime image %s/%s/%s", image.WorkspaceID, image.Type, image.Name,
		)
	}

	// 把落库后的文档回写到入参，调用方能拿到生成的 ID 与时间戳
	*image = result
	return nil
}

// GetByWorkspaceTypeAndName 按工作空间、镜像类型与仓库名称获取记录
func (s *StoreMongo) GetByWorkspaceTypeAndName(
	ctx context.Context,
	workspaceID string,
	imageType ImageType,
	name string,
) (*Image, error) {
	// 查询键与 Upsert 落库口径对齐：先 trim 再 Validate，空白或非法名称直接拒绝
	image := &Image{
		WorkspaceID: strings.TrimSpace(workspaceID),
		Type:        imageType,
		Name:        strings.TrimSpace(name),
	}
	if err := image.Validate(); err != nil {
		return nil, err
	}

	var result Image
	err := s.coll.FindOne(ctx, bson.M{
		"workspaceID": image.WorkspaceID,
		"type":        image.Type,
		"name":        image.Name,
	}).Decode(&result)
	if err != nil {
		// 未命中统一成哨兵错误，调用方无需感知 mongo.ErrNoDocuments
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCustomRuntimeImageNotFound
		}
		return nil, errors.Wrapf(
			err, "get custom runtime image %s/%s/%s", image.WorkspaceID, image.Type, image.Name,
		)
	}
	return &result, nil
}

// List 查询指定工作空间下的自定义运行时镜像记录列表
func (s *StoreMongo) List(ctx context.Context, workspaceID string, opts ListOptions) ([]Image, error) {
	// 工作空间是必填过滤条件，空值不能退化成全库扫描
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, errors.New("workspaceID is required")
	}

	filter := bson.M{"workspaceID": workspaceID}
	// Type 为空表示查全部类型；非空时先校验枚举，避免把非法 type 当查询条件
	if opts.Type != "" {
		validator := &Image{Type: opts.Type}
		if err := validator.validateType(); err != nil {
			return nil, err
		}
		filter["type"] = opts.Type
	}
	// keyword 只按名称模糊匹配，QuoteMeta 防止正则注入；不分页，按 type + name 稳定排序
	if keyword := strings.TrimSpace(opts.Keyword); keyword != "" {
		filter["name"] = bson.M{"$regex": regexp.QuoteMeta(keyword), "$options": "i"}
	}

	cursor, err := s.coll.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "type", Value: 1},
		{Key: "name", Value: 1},
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "find custom runtime images in workspace %s", workspaceID)
	}
	defer cursor.Close(ctx)

	var images []Image
	if err = cursor.All(ctx, &images); err != nil {
		return nil, errors.Wrap(err, "decode custom runtime images")
	}
	return images, nil
}

// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
func (s *StoreMongo) DeleteAll(ctx context.Context) error {
	// 只清文档，不 drop 集合，避免测试后把 golang-migrate 维护的唯一索引一并删掉
	if _, err := s.coll.DeleteMany(ctx, bson.M{}); err != nil {
		return errors.Wrap(err, "delete custom runtime image documents")
	}
	return nil
}
