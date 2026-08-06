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

package registry

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

// 存储镜像仓库的 MongoDB 集合名称
const imageRegistryCollectionName = "image_registries"

// ImageRegistryStore 是用于管理镜像仓库的存储接口
type ImageRegistryStore interface {
	// List 获取指定工作空间下的所有镜像仓库
	List(ctx context.Context, workspaceID string) ([]ImageRegistry, error)

	// GetByWorkspaceAndType 通过工作空间 & 类型获取镜像仓库
	GetByWorkspaceAndType(
		ctx context.Context, workspaceID string, registryType ImageRegistryType,
	) (*ImageRegistry, error)

	// Create 创建新的镜像仓库
	Create(ctx context.Context, registry *ImageRegistry) (bson.ObjectID, error)

	// Update 更新已经存在的镜像仓库
	Update(ctx context.Context, registry *ImageRegistry) error
}

var _ ImageRegistryStore = &ImageRegistryStoreMongo{}

// ImageRegistryStoreMongo 是 ImageRegistryStore 接口的 MongoDB 实现
type ImageRegistryStoreMongo struct {
	collection *mongo.Collection
}

// NewImageRegistryStoreMongo ...
func NewImageRegistryStoreMongo(client *mongo.Client, dbName string) (*ImageRegistryStoreMongo, error) {
	coll := client.Database(dbName).Collection(imageRegistryCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + type
	return &ImageRegistryStoreMongo{collection: coll}, nil
}

// List 获取指定工作空间下的所有镜像仓库
func (s *ImageRegistryStoreMongo) List(ctx context.Context, workspaceID string) ([]ImageRegistry, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"workspaceID": workspaceID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var imageRegistries []ImageRegistry
	if err = cursor.All(ctx, &imageRegistries); err != nil {
		return nil, errors.Wrapf(err, "decode workspace %s image registries", workspaceID)
	}

	for idx, reg := range imageRegistries {
		if err = s.handleSensitiveFields(&imageRegistries[idx], crypto.AESDecrypt); err != nil {
			return nil, errors.Wrapf(
				err, "decrypt workspace %s image registry %s sensitive fields", workspaceID, reg.Registry,
			)
		}
	}
	return imageRegistries, nil
}

// GetByWorkspaceAndType 通过工作空间 & 类型获取镜像仓库
func (s *ImageRegistryStoreMongo) GetByWorkspaceAndType(
	ctx context.Context, workspaceID string, registryType ImageRegistryType,
) (*ImageRegistry, error) {
	var registry ImageRegistry
	filter := bson.M{
		"workspaceID": workspaceID,
		"type":        registryType,
	}
	if err := s.collection.FindOne(ctx, filter).Decode(&registry); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(
				ErrImageRegistryNotFound, "workspace %s type %s not found", workspaceID, registryType,
			)
		}
		return nil, err
	}
	if err := s.handleSensitiveFields(&registry, crypto.AESDecrypt); err != nil {
		return nil, errors.Wrapf(
			err, "decrypt workspace %s type %s image registry sensitive fields", workspaceID, registryType,
		)
	}
	return &registry, nil
}

// Create 创建新的镜像仓库
func (s *ImageRegistryStoreMongo) Create(ctx context.Context, registry *ImageRegistry) (bson.ObjectID, error) {
	// 避免修改原始数据
	registryCopy := *registry
	// 对敏感字段进行加密
	if err := s.handleSensitiveFields(&registryCopy, crypto.AESEncrypt); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "encrypt sensitive fields")
	}
	ret, err := s.collection.InsertOne(ctx, &registryCopy)
	if err != nil {
		return bson.NilObjectID, err
	}
	if oid, ok := ret.InsertedID.(bson.ObjectID); ok {
		return oid, nil
	}
	return bson.NilObjectID, errors.New("failed to get inserted ID")
}

// Update 更新已经存在的镜像仓库
func (s *ImageRegistryStoreMongo) Update(ctx context.Context, registry *ImageRegistry) error {
	// 避免修改原始数据
	registryCopy := *registry
	// 对敏感字段进行加密
	if err := s.handleSensitiveFields(&registryCopy, crypto.AESEncrypt); err != nil {
		return errors.Wrap(err, "encrypt sensitive fields")
	}
	filter := bson.M{"workspaceID": registryCopy.WorkspaceID, "type": registryCopy.Type}
	update := bson.M{
		"$set": bson.M{
			"registry":         registryCopy.Registry,
			"username":         registryCopy.Username,
			"password":         registryCopy.Password,
			"bkCICredentialID": registryCopy.BkCICredentialID,
		},
	}
	_, err := s.collection.UpdateOne(ctx, filter, update)
	return err
}

// handleSensitiveFields 对 ImageRegistry 的敏感字段进行加密或解密
func (s *ImageRegistryStoreMongo) handleSensitiveFields(
	registry *ImageRegistry, handleFunc func(key, data string) (string, error),
) error {
	if registry == nil {
		return nil
	}
	// 如果密码不为空，则进行加/解密
	if registry.Password != "" {
		password, err := handleFunc(config.G.Encrypt.Secret, registry.Password)
		if err != nil {
			return err
		}
		registry.Password = password
	}
	return nil
}
