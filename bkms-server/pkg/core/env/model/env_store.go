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

package model

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// The name of the MongoDB collection for storing environment data.
const environmentCollectionName = "environments"

var (
	// ErrEnvNotFound represents the error of environment not found.
	ErrEnvNotFound = errors.New("environment not found")
	// ErrEnvAlreadyExists 环境与已有环境冲突：同一工作空间内重名，或绑定了已被占用的集群命名空间
	ErrEnvAlreadyExists = errors.New("environment already exists")
)

// EnvironmentUpdateData represents the fields that can be updated for an environment.
type EnvironmentUpdateData struct {
	Type        *string
	DisplayName *string
	Description *string
	// ClusterID, ClusterType, Namespace 是集群相关信息
	ClusterID   *string
	ClusterType *string
	Namespace   *string
	// IsFederation 是否为联邦集群
	IsFederation *bool
}

// ToBSON converts EnvironmentUpdateData to bson.M for update
func (d *EnvironmentUpdateData) ToBSON() (bson.M, bool) {
	data := bson.M{}

	isEmpty := true

	setIfNotNil := func(field *string, bsonField string) {
		if field != nil {
			data[bsonField] = *field
			isEmpty = false
		}
	}

	setIfNotNil(d.Type, "type")
	setIfNotNil(d.DisplayName, "displayName")
	setIfNotNil(d.Description, "description")
	setIfNotNil(d.ClusterID, "cluster.clusterID")
	setIfNotNil(d.ClusterType, "cluster.clusterType")
	setIfNotNil(d.Namespace, "cluster.namespace")
	if d.IsFederation != nil {
		data["cluster.isFederation"] = *d.IsFederation
		isEmpty = false
	}

	return data, isEmpty
}

// EnvironmentStore 环境的 GenericStore
type EnvironmentStore = store.GenericStore[Environment]

// EnvMeta 环境的存储元信息：创建时补齐 kind 与 appIDs 的默认值，读出时根据集群信息计算 Status。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + name
// - 唯一（partial）：cluster.clusterID + cluster.namespace（仅非空时生效）
// - 查询提速：ownerAppID
var EnvMeta = store.Meta[Environment]{
	Collection: environmentCollectionName,
	NotFound:   ErrEnvNotFound,
	Duplicate:  ErrEnvAlreadyExists,
	BeforeSave: []func(*Environment) error{setEnvDefaults},
	Decode:     []func(*Environment) error{fillEnvStatus},
}

// -----------------------------------------------------------------------------
// 查询条件
// -----------------------------------------------------------------------------

// InWorkspace 匹配工作空间内的环境
func InWorkspace(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }

// ByName 按工作空间与名称匹配，两类环境都可能命中，用于工作空间级的管理。
// 调用方须自行校验工作空间权限；应用视角的访问须使用 AppEnvByName
func ByName(workspaceID, name string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("name", name))
}

// AppEnvByName 应用视角按名称匹配：同名的标准环境，或该应用拥有的同名特性环境
func AppEnvByName(workspaceID, appID, name string) store.Opt {
	return store.Combine(ByName(workspaceID, name), VisibleToApp(appID))
}

// StdEnvs 只匹配标准环境。
//
// NOTE: 只匹配 kind == "standard"。上线特性环境前，须把存量文档回填 kind=standard，
// 否则缺失 kind 的历史记录将不可见。
func StdEnvs() store.Opt { return store.Where("kind", EnvironmentKindStandard) }

// StdEnvsIn 匹配工作空间内的标准环境
func StdEnvsIn(workspaceID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), StdEnvs())
}

// FeatEnvsOf 匹配应用拥有的特性环境
func FeatEnvsOf(appID string) store.Opt {
	return store.Combine(store.Where("kind", EnvironmentKindFeature), store.Where("ownerAppID", appID))
}

// VisibleToApp 匹配应用可用的环境：全部标准环境，以及该应用拥有的特性环境
func VisibleToApp(appID string) store.Opt { return visibleToApps(appID) }

// BoundToApp 匹配与应用关联的环境：appIDs 中包含该应用的标准环境，以及该应用拥有的特性环境
func BoundToApp(appID string) store.Opt {
	return store.Where("$or", bson.A{
		bson.M{"kind": EnvironmentKindStandard, "appIDs": appID},
		bson.M{"kind": EnvironmentKindFeature, "ownerAppID": appID},
	})
}

// NewestFirst 按创建时间倒序
func NewestFirst() store.Opt { return store.Desc("createdAt") }

func visibleToApps(appIDs ...string) store.Opt {
	return store.Where("$or", bson.A{
		bson.M{"kind": EnvironmentKindStandard},
		bson.M{"kind": EnvironmentKindFeature, "ownerAppID": bson.M{"$in": lo.Uniq(appIDs)}},
	})
}

// -----------------------------------------------------------------------------
// 领域函数
// -----------------------------------------------------------------------------

// ListBatchAppEnvs lists all environments available to a batch of apps.
//
// The result consists of standard environments and feature environments that belong to the
// given applications, newest first. Returns an empty list when appIDs is empty.
func ListBatchAppEnvs(
	ctx context.Context,
	db *mongo.Database,
	workspaceID string,
	appIDs []string,
) ([]Environment, error) {
	if len(appIDs) == 0 {
		return []Environment{}, nil
	}
	envs, err := store.Of(db, EnvMeta).List(ctx, InWorkspace(workspaceID), visibleToApps(appIDs...), NewestFirst())
	return envs, errors.Wrapf(err, "list environments of apps %v in workspace %s", appIDs, workspaceID)
}

// UpdateEnv updates the given fields of an environment, it is a no-op when no field is given.
func UpdateEnv(ctx context.Context, db *mongo.Database, envID bson.ObjectID, updateData *EnvironmentUpdateData) error {
	fields, isEmpty := updateData.ToBSON()
	if isEmpty {
		return nil
	}
	return errors.Wrapf(
		store.Of(db, EnvMeta).Update(ctx, fields, store.ByID(envID)),
		"update environment %s",
		envID.Hex(),
	)
}

// AddApp adds an app to an environment.
// 当应用部署到当前环境时, 会调用此方法；应用已在环境中时不重复添加，也不刷新 updatedAt
func AddApp(ctx context.Context, db *mongo.Database, envID bson.ObjectID, appID string) error {
	_, err := store.Of(db, EnvMeta).Modify(ctx,
		bson.M{"$addToSet": bson.M{"appIDs": appID}}, store.ByID(envID), store.KeepUpdatedAt(),
	)
	return errors.Wrapf(err, "add app %s to environment %s", appID, envID.Hex())
}

// RemoveApp removes an app from an environment.
// 当应用从当前环境删除时, 会调用此方法；不刷新 updatedAt
func RemoveApp(ctx context.Context, db *mongo.Database, envID bson.ObjectID, appID string) error {
	_, err := store.Of(db, EnvMeta).Modify(ctx,
		bson.M{"$pull": bson.M{"appIDs": appID}}, store.ByID(envID), store.KeepUpdatedAt(),
	)
	return errors.Wrapf(err, "remove app %s from environment %s", appID, envID.Hex())
}

// FindByClusterNamespace 查找已绑定到指定 clusterID + namespace 的环境，排除 excludeEnvID 自身（为零值时不排除）。
// 如果未找到匹配的环境，返回 nil, nil。
func FindByClusterNamespace(
	ctx context.Context,
	db *mongo.Database,
	clusterID, namespace string,
	excludeEnvID bson.ObjectID,
) (*Environment, error) {
	opts := []store.Opt{store.Where("cluster.clusterID", clusterID), store.Where("cluster.namespace", namespace)}
	if !excludeEnvID.IsZero() {
		opts = append(opts, store.Where("_id", bson.M{"$ne": excludeEnvID}))
	}
	env, err := store.Of(db, EnvMeta).Get(ctx, opts...)
	if errors.Is(err, ErrEnvNotFound) {
		return nil, nil
	}
	return env, errors.Wrapf(err, "find environment by cluster %s namespace %s", clusterID, namespace)
}

// GetEnvTypeMap 查询指定空间下的标准环境列表，返回 envName→envType 映射
func GetEnvTypeMap(ctx context.Context, db *mongo.Database, workspaceID string) (map[string]string, error) {
	envs, err := store.Of(db, EnvMeta).List(ctx, StdEnvsIn(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "list standard environments in workspace %s", workspaceID)
	}
	return lo.SliceToMap(envs, func(env Environment) (string, string) { return env.Name, env.Type }), nil
}

type envCountByWorkspace struct {
	WorkspaceID string `bson:"_id"`
	Count       int    `bson:"count"`
}

// CountByWorkspaceIDs returns standard environment counts grouped by workspace ID.
func CountByWorkspaceIDs(ctx context.Context, db *mongo.Database, workspaceIDs []string) (map[string]int, error) {
	if len(workspaceIDs) == 0 {
		return map[string]int{}, nil
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"workspaceID": bson.M{"$in": lo.Uniq(workspaceIDs)},
			"kind":        EnvironmentKindStandard,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$workspaceID",
			"count": bson.M{"$sum": 1},
		}}},
	}
	results, err := store.Aggregate[envCountByWorkspace](ctx, store.Of(db, EnvMeta), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate environment counts by workspace IDs")
	}

	counts := make(map[string]int, len(results))
	for _, item := range results {
		counts[item.WorkspaceID] = item.Count
	}
	return counts, nil
}

func setEnvDefaults(env *Environment) error {
	if env.Kind == "" {
		env.Kind = EnvironmentKindStandard
	}
	if env.AppIDs == nil {
		env.AppIDs = make([]string, 0)
	}
	return nil
}

func fillEnvStatus(env *Environment) error {
	env.Status = getEnvStatusByCluster(env.Cluster)
	return nil
}

func getEnvStatusByCluster(cluster BizCluster) EnvStatus {
	if cluster.ClusterID != "" && cluster.Namespace != "" {
		return EnvStatusReady
	}
	return EnvStatusNotReady
}
