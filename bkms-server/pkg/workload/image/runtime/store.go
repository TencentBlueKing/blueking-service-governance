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

package runtime

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
	// collRuntimeImages 平台运行时镜像集合
	collRuntimeImages = "runtime_images"
)

// Store 平台运行时镜像存储接口
type Store interface {
	// Create 创建运行时镜像记录
	Create(ctx context.Context, image *Image) error

	// GetByID 按记录 ID 获取运行时镜像记录
	GetByID(ctx context.Context, id string) (*Image, error)

	// GetByTypeAndName 按镜像类型和仓库名称获取运行时镜像记录
	GetByTypeAndName(ctx context.Context, imageType ImageType, name string) (*Image, error)

	// UpdateDescription 按镜像类型和仓库名称更新运行时镜像描述
	// 仅更新 description 与 updatedAt 两个字段，其余字段保持不变
	// 记录不存在时返回 ErrRuntimeImageNotFound
	UpdateDescription(ctx context.Context, imageType ImageType, name, description string) error

	// List 查询运行时镜像记录列表
	List(ctx context.Context, opts ListOptions) ([]Image, error)

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
	coll := client.Database(dbName).Collection(collRuntimeImages)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：type + name
	store := &StoreMongo{coll: coll}
	return store, nil
}

// Create 创建运行时镜像记录
func (s *StoreMongo) Create(ctx context.Context, image *Image) error {
	if err := image.Validate(); err != nil {
		return err
	}

	now := time.Now()
	image.ID = bson.NewObjectID().Hex()
	image.Name = strings.TrimSpace(image.Name)
	image.CreatedAt = now
	image.UpdatedAt = now

	_, err := s.coll.InsertOne(ctx, image)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrRuntimeImageAlreadyExists
		}
		return errors.Wrapf(err, "create runtime image %s/%s", image.Type, image.Name)
	}
	return nil
}

// GetByID 按记录 ID 获取运行时镜像记录
func (s *StoreMongo) GetByID(ctx context.Context, id string) (*Image, error) {
	id = strings.TrimSpace(id)
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return nil, ErrRuntimeImageNotFound
	}

	var image Image
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&image)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRuntimeImageNotFound
		}
		return nil, errors.Wrapf(err, "get runtime image %s", id)
	}
	return &image, nil
}

// GetByTypeAndName 按镜像类型和仓库名称获取运行时镜像记录
func (s *StoreMongo) GetByTypeAndName(ctx context.Context, imageType ImageType, name string) (*Image, error) {
	image := &Image{Type: imageType, Name: name}
	if err := image.validateType(); err != nil {
		return nil, err
	}
	if err := image.validateName(); err != nil {
		return nil, err
	}

	var result Image
	err := s.coll.FindOne(ctx, bson.M{"type": imageType, "name": strings.TrimSpace(name)}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRuntimeImageNotFound
		}
		return nil, errors.Wrapf(err, "get runtime image %s/%s", imageType, name)
	}
	return &result, nil
}

// UpdateDescription 按镜像类型和仓库名称更新运行时镜像描述
// 仅更新 description 与 updatedAt 两个字段；记录不存在时返回 ErrRuntimeImageNotFound
func (s *StoreMongo) UpdateDescription(
	ctx context.Context,
	imageType ImageType,
	name string,
	description string,
) error {
	// 复用 Image 的字段校验逻辑，保证 type/name/description 的合法性
	image := &Image{Type: imageType, Name: name, Description: description}
	if err := image.validateType(); err != nil {
		return err
	}
	if err := image.validateName(); err != nil {
		return err
	}
	if err := image.validateDescription(); err != nil {
		return err
	}

	filter := bson.M{"type": imageType, "name": strings.TrimSpace(name)}
	update := bson.M{"$set": bson.M{
		"description": description,
		"updatedAt":   time.Now(),
	}}
	result, err := s.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrapf(err, "update runtime image description %s/%s", imageType, name)
	}
	if result.MatchedCount == 0 {
		return ErrRuntimeImageNotFound
	}
	return nil
}

// List 查询运行时镜像记录列表
func (s *StoreMongo) List(ctx context.Context, opts ListOptions) ([]Image, error) {
	filter := bson.M{}
	if opts.Type != "" {
		validator := &Image{Type: opts.Type}
		if err := validator.validateType(); err != nil {
			return nil, err
		}
		filter["type"] = opts.Type
	}
	// 关键字搜索名称或描述
	if keyword := strings.TrimSpace(opts.Keyword); keyword != "" {
		pattern := regexp.QuoteMeta(keyword)
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": pattern, "$options": "i"}},
			{"description": bson.M{"$regex": pattern, "$options": "i"}},
		}
	}

	cursor, err := s.coll.Find(ctx, filter, options.Find().SetSort(bson.D{
		{Key: "type", Value: 1},
		{Key: "name", Value: 1},
	}))
	if err != nil {
		return nil, errors.Wrap(err, "find runtime images")
	}
	defer cursor.Close(ctx)

	var images []Image
	if err = cursor.All(ctx, &images); err != nil {
		return nil, errors.Wrap(err, "decode runtime images")
	}
	return images, nil
}

// DeleteAll 删除所有记录并保留集合及索引（仅用于测试）
func (s *StoreMongo) DeleteAll(ctx context.Context) error {
	if _, err := s.coll.DeleteMany(ctx, bson.M{}); err != nil {
		return errors.Wrap(err, "delete runtime image documents")
	}
	return nil
}
