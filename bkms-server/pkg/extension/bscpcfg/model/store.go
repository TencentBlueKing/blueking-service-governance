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

// Package model 定义了应用配置管理相关的方法。
package model

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var _ Store = (*StoreMongo)(nil)

// Store 应用配置管理统一存储接口。
type Store interface {
	// === Metadata 操作 ===

	// CreateMetadata 创建 Metadata（一个 app 一条记录）
	CreateMetadata(ctx context.Context, meta *Metadata) error
	// GetMetadata 获取指定 app 的 Metadata
	GetMetadata(ctx context.Context, appID string) (*Metadata, error)
	// UpdateMetadata 更新 Metadata（可更新 mountPath、token、credentialID、credential）
	UpdateMetadata(ctx context.Context, appID string, updateData *MetadataUpdate) error
	// DeleteMetadata 删除指定 app 的 Metadata
	DeleteMetadata(ctx context.Context, appID string) error

	// === EnvBinding 操作 ===

	// CreateEnvBinding 创建 EnvBinding（一个 app+env 一条记录）
	CreateEnvBinding(ctx context.Context, binding *EnvBinding) error
	// GetEnvBinding 获取指定 app+env 的绑定
	GetEnvBinding(ctx context.Context, appID, envName string) (*EnvBinding, error)
	// UpdateEnvBinding 更新绑定（可更新 services 数组）
	UpdateEnvBinding(ctx context.Context, appID, envName string, updateData *EnvBindingUpdate) error
	// DeleteEnvBinding 删除指定 app+env 的绑定
	DeleteEnvBinding(ctx context.Context, appID, envName string) error

	// === 聚合操作 ===

	// DeleteEnvBindingsByApp 删除应用下所有 EnvBinding（应用删除时级联）
	DeleteEnvBindingsByApp(ctx context.Context, appID string) error
	// ListEnvBindingsByApp 获取应用下所有环境的绑定列表
	ListEnvBindingsByApp(ctx context.Context, appID string) ([]*EnvBinding, error)
	// GetSnapshot 获取指定 app+env 的聚合快照，不存在返回 nil, nil。
	GetSnapshot(ctx context.Context, appID, envName string) (*Snapshot, error)
}

// StoreMongo Store 的 MongoDB 实现，组合 Metadata 和 EnvBinding 存储
type StoreMongo struct {
	metaStore    MetadataStore
	bindingStore EnvBindingStore
}

// NewStoreMongo 创建统一的配置管理存储
func NewStoreMongo(client *mongo.Client, dbName string) (Store, error) {
	metaStore, err := NewMetadataStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	bindingStore, err := NewEnvBindingStoreMongo(client, dbName)
	if err != nil {
		return nil, err
	}
	return &StoreMongo{
		metaStore:    metaStore,
		bindingStore: bindingStore,
	}, nil
}

// === Metadata 操作 ===

// CreateMetadata 创建 Metadata（一个 app 一条记录）
func (s *StoreMongo) CreateMetadata(ctx context.Context, meta *Metadata) error {
	return s.metaStore.Create(ctx, meta)
}

// GetMetadata 获取指定 app 的 Metadata
func (s *StoreMongo) GetMetadata(ctx context.Context, appID string) (*Metadata, error) {
	return s.metaStore.Get(ctx, appID)
}

// UpdateMetadata 更新 Metadata（可更新 mountPath、token、credentialID、credential）
func (s *StoreMongo) UpdateMetadata(
	ctx context.Context,
	appID string,
	updateData *MetadataUpdate,
) error {
	return s.metaStore.Update(ctx, appID, updateData)
}

// DeleteMetadata 删除指定 app 的 Metadata
func (s *StoreMongo) DeleteMetadata(ctx context.Context, appID string) error {
	return s.metaStore.Delete(ctx, appID)
}

// === EnvBinding 操作 ===

// CreateEnvBinding 创建 EnvBinding（一个 app+env 一条记录）
func (s *StoreMongo) CreateEnvBinding(ctx context.Context, binding *EnvBinding) error {
	return s.bindingStore.Create(ctx, binding)
}

// GetEnvBinding 获取指定 app+env 的绑定
func (s *StoreMongo) GetEnvBinding(ctx context.Context, appID, envName string) (*EnvBinding, error) {
	return s.bindingStore.Get(ctx, appID, envName)
}

// UpdateEnvBinding 更新绑定（可更新 services 数组）
func (s *StoreMongo) UpdateEnvBinding(
	ctx context.Context,
	appID, envName string,
	updateData *EnvBindingUpdate,
) error {
	return s.bindingStore.Update(ctx, appID, envName, updateData)
}

// DeleteEnvBinding 删除指定 app+env 的绑定
func (s *StoreMongo) DeleteEnvBinding(ctx context.Context, appID, envName string) error {
	return s.bindingStore.Delete(ctx, appID, envName)
}

// === 聚合操作 ===

// DeleteEnvBindingsByApp 删除应用下所有 EnvBinding（应用删除时级联）
func (s *StoreMongo) DeleteEnvBindingsByApp(ctx context.Context, appID string) error {
	return s.bindingStore.DeleteByApp(ctx, appID)
}

// ListEnvBindingsByApp 获取应用下所有环境的绑定列表
func (s *StoreMongo) ListEnvBindingsByApp(ctx context.Context, appID string) ([]*EnvBinding, error) {
	return s.bindingStore.ListByApp(ctx, appID)
}

// GetSnapshot 获取指定 app+env 的聚合快照。
// 如果 Metadata 不存在或 EnvBinding 不存在，返回 nil, nil。
func (s *StoreMongo) GetSnapshot(ctx context.Context, appID, envName string) (*Snapshot, error) {
	meta, err := s.metaStore.Get(ctx, appID)
	if err != nil {
		if errors.Is(err, ErrMetadataNotFound) {
			return nil, nil
		}
		return nil, err
	}

	binding, err := s.bindingStore.Get(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, ErrEnvBindingNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &Snapshot{
		Metadata:   meta,
		EnvBinding: binding,
	}, nil
}
