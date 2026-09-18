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

package trigger

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// PolicyCollName 触发策略表名
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + name
// - 普通：appID
const PolicyCollName = "build_trigger_policies"

// RecordCollName 触发记录表名
//
// 索引（由 golang-migrate 维护）：
// - 普通：policyID + triggeredAt
// - 普通：policyID + commitID
//
// 注意 policyID + commitID 不能设唯一约束：去重规则是「同一策略下同一 commit 已成功构建过则跳过」，
// 同一 commit 仍可能产生多条 skipped / failed 记录，唯一约束会让这些记录写不进去
const RecordCollName = "build_trigger_records"

var (
	// ErrPolicyNotFound 触发策略不存在
	ErrPolicyNotFound = errors.New("build trigger policy not found")
	// ErrPolicyNameDuplicated 同应用下已存在同名触发策略
	ErrPolicyNameDuplicated = errors.New("build trigger policy name already exists in the app")
)

// PolicyStore 触发策略存储接口
type PolicyStore interface {
	// Create 创建触发策略，名称在应用内重复时返回 ErrPolicyNameDuplicated
	Create(ctx context.Context, policy *Policy) error

	// Update 更新触发策略，策略不存在时返回 ErrPolicyNotFound。
	// 仅更新策略表单字段与蓝盾关联标识，不改动 appID / creator / createdAt
	Update(ctx context.Context, policy *Policy) error

	// UpdateStatus 更新策略启停状态，策略不存在时返回 ErrPolicyNotFound
	UpdateStatus(ctx context.Context, appID, policyID string, status Status) error

	// Get 获取单条触发策略，不存在时返回 ErrPolicyNotFound
	Get(ctx context.Context, appID, policyID string) (*Policy, error)

	// List 获取应用下的全部触发策略。
	// 单应用策略上限为 MaxPoliciesPerApp，无需分页；生效中与已停用一并返回
	List(ctx context.Context, appID string) ([]Policy, error)

	// Delete 删除触发策略，策略不存在时返回 ErrPolicyNotFound。
	// 不级联删除触发记录与已产出的构建记录
	Delete(ctx context.Context, appID, policyID string) error
}

// RecordStore 触发记录存储接口。
// Mongo 实现由「自动触发记录存储与查询」子需求落地
type RecordStore interface {
	// Create 创建触发记录
	Create(ctx context.Context, record *Record) error

	// List 获取策略的触发记录列表（支持分页），按触发时间倒序。
	// result 为空表示不按结果筛选
	List(ctx context.Context, policyID string, result Result, page, pageSize int64) ([]Record, int64, error)

	// ExistsBuiltByCommit 判断该策略下指定 commit 是否已成功发起过构建，用于回调去重。
	// 只统计 result 为 ResultBuilt 的记录
	ExistsBuiltByCommit(ctx context.Context, policyID, commitID string) (bool, error)
}

var _ PolicyStore = &PolicyStoreMongo{}

// PolicyStoreMongo 基于 MongoDB 的触发策略存储
type PolicyStoreMongo struct {
	collection *mongo.Collection
}

// NewPolicyStoreMongo 创建 PolicyStore Mongo 实现，索引由迁移维护，此处不建
func NewPolicyStoreMongo(client *mongo.Client, dbName string) (*PolicyStoreMongo, error) {
	coll := client.Database(dbName).Collection(PolicyCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + name
	// - 普通：appID
	return &PolicyStoreMongo{collection: coll}, nil
}

// Create 创建触发策略，名称在应用内重复时返回 ErrPolicyNameDuplicated
func (s *PolicyStoreMongo) Create(ctx context.Context, policy *Policy) error {
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now
	if _, err := s.collection.InsertOne(ctx, policy); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrPolicyNameDuplicated
		}
		return errors.Wrap(err, "create build trigger policy")
	}
	return nil
}

// Update 更新触发策略表单字段与蓝盾关联标识，不改动 appID / creator / createdAt
func (s *PolicyStoreMongo) Update(ctx context.Context, policy *Policy) error {
	filter := bson.M{"appID": policy.AppID, "id": policy.ID}
	updateDoc := bson.M{"$set": bson.M{
		"name":             policy.Name,
		"event":            policy.Event,
		"branchMatchMode":  policy.BranchMatchMode,
		"branchMatchValue": policy.BranchMatchValue,
		"pathFilter":       policy.PathFilter,
		"pipelineID":       policy.PipelineID,
		"triggerID":        policy.TriggerID,
		"updatedAt":        time.Now(),
	}}
	ret, err := s.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrPolicyNameDuplicated
		}
		return errors.Wrapf(err, "update build trigger policy %s", policy.ID)
	}
	if ret.MatchedCount == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

// UpdateStatus 更新策略启停状态，策略不存在时返回 ErrPolicyNotFound
func (s *PolicyStoreMongo) UpdateStatus(ctx context.Context, appID, policyID string, status Status) error {
	filter := bson.M{"appID": appID, "id": policyID}
	ret, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{
		"status":    status,
		"updatedAt": time.Now(),
	}})
	if err != nil {
		return errors.Wrapf(err, "update build trigger policy %s status", policyID)
	}
	if ret.MatchedCount == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

// Get 获取单条触发策略，不存在时返回 ErrPolicyNotFound
func (s *PolicyStoreMongo) Get(ctx context.Context, appID, policyID string) (*Policy, error) {
	var policy Policy
	err := s.collection.FindOne(ctx, bson.M{"appID": appID, "id": policyID}).Decode(&policy)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPolicyNotFound
		}
		return nil, errors.Wrapf(err, "get build trigger policy %s", policyID)
	}
	return &policy, nil
}

// List 获取应用下全部触发策略，按创建时间升序
func (s *PolicyStoreMongo) List(ctx context.Context, appID string) ([]Policy, error) {
	cursor, err := s.collection.Find(
		ctx,
		bson.M{"appID": appID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list build trigger policies of app %s", appID)
	}
	defer cursor.Close(ctx)

	// 无策略时返回空切片，避免调用方把 nil 与「未查到」混淆
	policies := make([]Policy, 0)
	if err = cursor.All(ctx, &policies); err != nil {
		return nil, errors.Wrapf(err, "decode build trigger policies of app %s", appID)
	}
	return policies, nil
}

// Delete 删除触发策略，不级联删除触发记录与构建记录
func (s *PolicyStoreMongo) Delete(ctx context.Context, appID, policyID string) error {
	ret, err := s.collection.DeleteOne(ctx, bson.M{"appID": appID, "id": policyID})
	if err != nil {
		return errors.Wrapf(err, "delete build trigger policy %s", policyID)
	}
	if ret.DeletedCount == 0 {
		return ErrPolicyNotFound
	}
	return nil
}
