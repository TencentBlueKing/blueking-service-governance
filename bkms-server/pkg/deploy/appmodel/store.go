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

package appmodel

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
)

// collectionName 应用部署记录表名
const collectionName = "app_model_deploy_records"

// ErrDeployRecordNotFound 部署记录不存在
var ErrDeployRecordNotFound = errors.New("app model deploy record not found")

// RecordStore 部署记录存储接口
type RecordStore interface {
	// Create 创建应用部署记录
	Create(ctx context.Context, record *Record) (string, error)

	// Update 更新应用部署记录
	Update(ctx context.Context, record *Record) error

	// List 列表应用部署记录（支持分页）
	List(
		ctx context.Context,
		appID, envName, trafficLaneName, keyword string,
		page, pageSize int64,
	) ([]Record, int64, error)

	// ListByImageTag 按 appID 和 imageTag 查询部署记录（支持分页，按 createdAt 降序）
	ListByImageTag(ctx context.Context, appID, imageTag string, page, pageSize int64) ([]Record, int64, error)

	// Get 通过 ID 获取应用部署记录
	Get(ctx context.Context, appID, id string) (*Record, error)

	// GetLatest 获取最新应用部署记录
	GetLatest(ctx context.Context, appID, envName, trafficLaneName string) (*Record, error)

	// GetLatestByStatuses 获取指定状态集合中的最新应用部署记录
	GetLatestByStatuses(
		ctx context.Context,
		appID, envName, trafficLaneName string,
		statuses []Status,
	) (*Record, error)

	// ListImageTagDeployedEnvs 获取指定应用各镜像标签的已部署环境列表（去重，仅统计部署成功的记录，默认查询所有环境）
	ListImageTagDeployedEnvs(ctx context.Context, appID string) ([]deploytypes.ImageTagEnvPair, error)

	// HasActiveDeployments 检查是否存在活跃的部署（未卸载的部署）
	HasActiveDeployments(ctx context.Context, appID string) (bool, error)
}

var _ RecordStore = &RecordStoreMongo{}

// RecordStoreMongo RecordStore 实现（基于 MongoDB）
type RecordStoreMongo struct {
	collection *mongo.Collection
}

// NewRecordStoreMongo 新建 RecordStoreMongo 实例
func NewRecordStoreMongo(client *mongo.Client, dbName string) (*RecordStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	return &RecordStoreMongo{collection: coll}, nil
}

// Create 创建应用部署记录
func (s *RecordStoreMongo) Create(ctx context.Context, record *Record) (string, error) {
	timeNow := time.Now()
	record.CreatedAt, record.UpdatedAt = timeNow, timeNow
	ret, err := s.collection.InsertOne(ctx, record)
	if err != nil {
		return "", err
	}
	if oid, ok := ret.InsertedID.(bson.ObjectID); ok {
		return oid.Hex(), nil
	}
	return "", errors.New("failed to get inserted ID")
}

// Update 更新应用部署记录
func (s *RecordStoreMongo) Update(ctx context.Context, record *Record) error {
	filter := bson.M{"_id": record.ID}
	updateDoc := bson.M{"$set": bson.M{
		"replicas":     record.Replicas,
		"deletionCost": record.DeletionCost,
		"message":      record.Message,
		"status":       record.Status,
		"extras":       record.Extras,
		"endedAt":      record.EndedAt,
		"updater":      record.Updater,
		"updatedAt":    time.Now(),
	}}
	ret, err := s.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return errors.Wrapf(err, "update app model app deploy record %s", record.ID.Hex())
	}
	// id 匹配不到数据的情况
	if ret.MatchedCount == 0 {
		return ErrDeployRecordNotFound
	}
	return nil
}

// List 列出应用部署记录（支持分页）
func (s *RecordStoreMongo) List(
	ctx context.Context,
	appID, envName, trafficLaneName, keyword string,
	page, pageSize int64,
) ([]Record, int64, error) {
	filter := bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
	}
	// 支持 keyword 参数
	if keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword = regexp.QuoteMeta(keyword)
		// 模糊匹配：Image Tag、更新人
		filter["$or"] = []bson.M{
			{"imageTag": bson.M{"$regex": keyword, "$options": "i"}},
			{"updater": bson.M{"$regex": keyword, "$options": "i"}},
		}
	}

	records, total, err := s.list(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "list app model app %s deploy records", appID)
	}
	return records, total, nil
}

// ListByImageTag 按 appID 和 imageTag 查询部署记录（支持分页，按 createdAt 降序）
func (s *RecordStoreMongo) ListByImageTag(
	ctx context.Context, appID, imageTag string, page, pageSize int64,
) ([]Record, int64, error) {
	filter := bson.M{
		"appID":    appID,
		"imageTag": imageTag,
		// 经讨论确认该功能仅关注部分状态
		"status": bson.M{
			"$in": []Status{
				// 部署成功
				StatusDeployed,
				// 部署中
				StatusDeploying,
				// 部署失败（含超时、中断）
				StatusFailed,
				StatusPollingTimeout,
				StatusPollingBroken,
			},
		},
	}

	records, total, err := s.list(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "list app model app %s image tag %s deploy records", appID, imageTag)
	}
	return records, total, nil
}

// list 通用的分页查询方法
func (s *RecordStoreMongo) list(
	ctx context.Context,
	filter bson.M,
	page, pageSize int64,
) ([]Record, int64, error) {
	// 先统计总数
	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 分页参数
	opts := options.Find().
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize).
		SetSort(bson.D{{"createdAt", -1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var records []Record
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

// Get 获取应用部署记录
func (s *RecordStoreMongo) Get(ctx context.Context, appID, id string) (*Record, error) {
	// ID 类型转换
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid record id %s", id)
	}

	var record Record
	err = s.collection.FindOne(ctx, bson.M{"appID": appID, "_id": objID}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDeployRecordNotFound
		}
		return nil, errors.Wrapf(err, "find app model deploy record %s", id)
	}
	return &record, nil
}

// GetLatest 获取最新应用部署记录
func (s *RecordStoreMongo) GetLatest(
	ctx context.Context,
	appID, envName, trafficLaneName string,
) (*Record, error) {
	return s.getLatestByFilter(ctx, bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
	})
}

// GetLatestByStatuses 获取指定状态集合中的最新应用部署记录
func (s *RecordStoreMongo) GetLatestByStatuses(
	ctx context.Context,
	appID, envName, trafficLaneName string,
	statuses []Status,
) (*Record, error) {
	return s.getLatestByFilter(ctx, bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
		"status":          bson.M{"$in": statuses},
	})
}

func (s *RecordStoreMongo) getLatestByFilter(ctx context.Context, filter bson.M) (*Record, error) {
	// 排序参数
	opts := options.FindOne().SetSort(bson.D{{"createdAt", -1}})

	var record Record
	err := s.collection.FindOne(ctx, filter, opts).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDeployRecordNotFound
		}
		return nil, errors.Wrapf(err, "find app model app latest deploy record (filter: %v)", filter)
	}
	return &record, nil
}

// ListImageTagDeployedEnvs 获取指定应用各镜像标签的已部署环境列表（去重，仅统计部署成功的记录，默认查询所有环境）
func (s *RecordStoreMongo) ListImageTagDeployedEnvs(
	ctx context.Context,
	appID string,
) ([]deploytypes.ImageTagEnvPair, error) {
	pipeline := bson.A{
		// Stage 1: 按 appID 和 status 过滤，只保留成功部署的记录
		bson.M{"$match": bson.M{"appID": appID, "status": string(StatusDeployed)}},
		// Stage 2: 按 (imageTag, envName) 复合键分组去重
		bson.M{"$group": bson.M{
			"_id": bson.M{"imageTag": "$imageTag", "envName": "$envName"},
		}},
		// Stage 3: 将分组 _id 中的字段提升为顶层字段，输出扁平结构
		bson.M{"$project": bson.M{
			"imageTag": "$_id.imageTag",
			"envName":  "$_id.envName",
			"_id":      0,
		}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrapf(err, "aggregate distinct image tag envs for app %s", appID)
	}
	defer cursor.Close(ctx)

	var results []deploytypes.ImageTagEnvPair
	if err = cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrapf(err, "decode distinct image tag envs for app %s", appID)
	}
	return results, nil
}

// HasActiveDeployments 检查是否存在活跃的部署（已部署、部署中或卸载中）
// 判断逻辑：按环境+泳道分组，获取每组最新记录，检查最新记录是否为活跃状态
func (s *RecordStoreMongo) HasActiveDeployments(ctx context.Context, appID string) (bool, error) {
	activeStatuses := bson.A{
		string(StatusDeployed),
		string(StatusDeploying),
		string(StatusUninstalling),
	}

	// 使用聚合查询：按 (envName, trafficLaneName) 分组，获取每组最新记录
	pipeline := bson.A{
		// Stage 1: 按 appID 过滤
		bson.M{"$match": bson.M{"appID": appID}},
		// Stage 2: 按 (envName, trafficLaneName) 分组，获取每组最新记录
		bson.M{"$sort": bson.M{"createdAt": -1}},
		bson.M{"$group": bson.M{
			"_id":    bson.M{"envName": "$envName", "trafficLaneName": "$trafficLaneName"},
			"status": bson.M{"$first": "$status"},
		}},
		// Stage 3: 检查最新记录是否为活跃状态
		bson.M{"$match": bson.M{"status": bson.M{"$in": activeStatuses}}},
		// Stage 4: 限制返回 1 条记录（只要有一条活跃部署即可）
		bson.M{"$limit": 1},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return false, errors.Wrapf(err, "aggregate active deployments for app %s", appID)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return false, errors.Wrapf(err, "decode active deployments for app %s", appID)
	}

	return len(results) > 0, nil
}
