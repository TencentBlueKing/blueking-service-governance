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

// Package role provides IAM role domain types and Mongo storage for bkms-server.
//
// This package defines the role-related domain models (Role, WorkspaceGradeManager,
// PermissionScope, BuiltinRoleCode) and a Mongo-backed RoleStore implementation.
package role

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Mongo collection name constants.
const (
	// gradeManagerCollName workspace 分级管理员表名
	gradeManagerCollName = "iam_grade_managers"
	// roleCollName 角色表名
	roleCollName = "iam_roles"
)

// RoleStore 角色 / 分级管理员存储接口
type RoleStore interface {
	// CreateWorkspaceGradeManager 创建 workspace 分级管理员
	CreateWorkspaceGradeManager(ctx context.Context, wgm *WorkspaceGradeManager) (*WorkspaceGradeManager, error)
	// GetWorkspaceGradeManager 获取指定的 workspace 分级管理员
	GetWorkspaceGradeManager(ctx context.Context, workspaceID string) (*WorkspaceGradeManager, error)
	// DeleteWorkspaceGradeManager 删除指定的 workspace 分级管理员
	DeleteWorkspaceGradeManager(ctx context.Context, workspaceID string) error

	// CreateRole 创建角色
	CreateRole(ctx context.Context, role *Role) (*Role, error)
	// GetRoleByID 根据角色 ID 获取角色
	GetRoleByID(ctx context.Context, roleID string) (*Role, error)
	// DeleteRolesByUserGroupIDs 根据用户组 ID, 批量删除某个 workspace 下的角色
	DeleteRolesByUserGroupIDs(ctx context.Context, workspaceID string, userGroupIDs []int) error
	// ListRoles 查询角色
	ListRoles(ctx context.Context, queryParams *RoleQueryParams) ([]*Role, error)
}

// 编译期检查
var _ RoleStore = (*RoleStoreMongo)(nil)

// RoleStoreMongo 是 RoleStore 的 MongoDB 实现
type RoleStoreMongo struct {
	gradeManagerColl *mongo.Collection
	roleColl         *mongo.Collection
}

// NewRoleStoreMongo 创建 RoleStoreMongo 实例。
func NewRoleStoreMongo(client *mongo.Client, dbName string) (*RoleStoreMongo, error) {
	db := client.Database(dbName)
	gradeManagerColl := db.Collection(gradeManagerCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID
	roleColl := db.Collection(roleCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	// - 查询提速：workspaceID + userGroupID

	return &RoleStoreMongo{
		gradeManagerColl: gradeManagerColl,
		roleColl:         roleColl,
	}, nil
}

// CreateWorkspaceGradeManager 创建 workspace 分级管理员
func (s *RoleStoreMongo) CreateWorkspaceGradeManager(
	ctx context.Context,
	wgm *WorkspaceGradeManager,
) (*WorkspaceGradeManager, error) {
	if _, err := s.gradeManagerColl.InsertOne(ctx, wgm); err != nil {
		return nil, errors.Wrapf(err, "create workspace(%s) grade manager", wgm.WorkspaceID)
	}
	return wgm, nil
}

// GetWorkspaceGradeManager 获取指定的 workspace 分级管理员
func (s *RoleStoreMongo) GetWorkspaceGradeManager(
	ctx context.Context,
	workspaceID string,
) (*WorkspaceGradeManager, error) {
	wgm := &WorkspaceGradeManager{}
	err := s.gradeManagerColl.FindOne(ctx, bson.M{"workspaceID": workspaceID}).Decode(wgm)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace(%s) grade manager", workspaceID)
	}
	return wgm, nil
}

// DeleteWorkspaceGradeManager 删除指定的 workspace 分级管理员
func (s *RoleStoreMongo) DeleteWorkspaceGradeManager(ctx context.Context, workspaceID string) error {
	if _, err := s.gradeManagerColl.DeleteOne(ctx, bson.M{"workspaceID": workspaceID}); err != nil {
		return errors.Wrapf(err, "delete workspace(%s) grade manager", workspaceID)
	}
	return nil
}

// CreateRole 创建角色
func (s *RoleStoreMongo) CreateRole(ctx context.Context, role *Role) (*Role, error) {
	if _, err := s.roleColl.InsertOne(ctx, role); err != nil {
		return nil, errors.Wrapf(err, "create role(%s)", role.Name)
	}
	return role, nil
}

// GetRoleByID 根据角色 ID 获取角色
func (s *RoleStoreMongo) GetRoleByID(ctx context.Context, roleID string) (*Role, error) {
	r := &Role{}
	err := s.roleColl.FindOne(ctx, bson.M{"id": roleID}).Decode(r)
	if err != nil {
		return nil, errors.Wrapf(err, "get role(%s)", roleID)
	}
	return r, nil
}

// DeleteRolesByUserGroupIDs 根据用户组 ID, 批量删除某个 workspace 下的角色
func (s *RoleStoreMongo) DeleteRolesByUserGroupIDs(
	ctx context.Context,
	workspaceID string,
	userGroupIDs []int,
) error {
	filter := bson.M{
		"workspaceID": workspaceID,
		"userGroupID": bson.M{"$in": userGroupIDs},
	}
	if _, err := s.roleColl.DeleteMany(ctx, filter); err != nil {
		return errors.Wrapf(err, "delete workspace(%s) roles by user group ids %v", workspaceID, userGroupIDs)
	}
	return nil
}

// ListRoles 查询角色
func (s *RoleStoreMongo) ListRoles(ctx context.Context, queryParams *RoleQueryParams) ([]*Role, error) {
	filter := bson.M{}
	if queryParams.WorkspaceID != nil {
		filter["workspaceID"] = *queryParams.WorkspaceID
	}
	if queryParams.IsGradeManager != nil {
		filter["isGradeManager"] = *queryParams.IsGradeManager
	}
	if queryParams.Scope != nil {
		filter["scope.resourceType"] = queryParams.Scope.ResourceType
		filter["scope.resourceID"] = queryParams.Scope.ResourceID
	}

	cursor, err := s.roleColl.Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "query roles")
	}
	defer cursor.Close(ctx)

	roles := make([]*Role, 0)
	for cursor.Next(ctx) {
		r := &Role{}
		if err = cursor.Decode(r); err != nil {
			return nil, errors.Wrap(err, "decode role")
		}
		roles = append(roles, r)
	}
	return roles, nil
}
