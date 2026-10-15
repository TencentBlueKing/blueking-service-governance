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

package build

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// buildRecordCollName 构建记录表名
const buildRecordCollName = "build_records"

// buildCounterCollName 构建计数器表名
const buildCounterCollName = "build_counters"

var (
	// ErrConfigNotFound 构建配置未找到
	ErrConfigNotFound = errors.New("build config not found")
	// ErrRecordNotFound 构建记录未找到。调用方据此区分「记录不存在」与 DB 瞬时故障
	ErrRecordNotFound = errors.New("build record not found")
)

// ConfigStore 构建配置存储
type ConfigStore = store.GenericStore[Config]

// ConfigMeta 构建配置的存储元数据，镜像凭证密码在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID
var ConfigMeta = store.Meta[Config]{
	Collection: "build_configs",
	NotFound:   ErrConfigNotFound,
	Encode:     []func(*Config) error{encryptImagePassword},
	Decode:     []func(*Config) error{decryptImagePassword},
}

func encryptImagePassword(cfg *Config) (err error) {
	if cfg.Image != nil {
		cfg.Image.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, cfg.Image.Password)
	}
	return errors.Wrap(err, "encrypt image password")
}

func decryptImagePassword(cfg *Config) (err error) {
	if cfg.Image != nil {
		cfg.Image.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, cfg.Image.Password)
	}
	return errors.Wrap(err, "decrypt image password")
}

// ByApp 按应用 ID 匹配构建配置
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// UpdateConfig 更新已存在的应用构建配置的全部可变字段（appID 与 createdAt 不可变），
// cfg 中为空的可选配置会从库中清除；配置不存在时返回 ErrConfigNotFound
func UpdateConfig(ctx context.Context, configs *ConfigStore, cfg *Config) error {
	if err := configs.Update(ctx, cfg, store.UpdateOmit("appID"), ByApp(cfg.AppID)); err != nil {
		return errors.Wrapf(err, "update app %s build config", cfg.AppID)
	}
	return nil
}

// RecordStore 构建记录存储接口
type RecordStore interface {
	// Create 创建新的构建记录
	Create(ctx context.Context, record *Record) error

	// Update 更新已存在的应用构建记录
	Update(ctx context.Context, record *Record) error

	// List 获取应用构建记录列表（支持分页）
	List(ctx context.Context, appID, keyword string, page, pageSize int64) ([]Record, int64, error)

	// Get 使用 AppID 和 BuildID 获取构建记录
	Get(ctx context.Context, appID, buildID string) (*Record, error)
}

var _ RecordStore = &RecordStoreMongo{}

// RecordStoreMongo RecordStore 实现（基于 MongoDB）
type RecordStoreMongo struct {
	collection  *mongo.Collection
	counterColl *mongo.Collection
}

// NewRecordStoreMongo 创建 RecordStoreMongo 实例
func NewRecordStoreMongo(client *mongo.Client, dbName string) (*RecordStoreMongo, error) {
	db := client.Database(dbName)
	coll := db.Collection(buildRecordCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + buildID
	counterColl := db.Collection(buildCounterCollName)
	return &RecordStoreMongo{collection: coll, counterColl: counterColl}, nil
}

// buildCounter 构建计数器文档
type buildCounter struct {
	AppID string `bson:"_id"`
	Seq   int64  `bson:"seq"`
}

// nextNum 获取指定 AppID 的下一个构建序号（原子操作，并发安全）
func (s *RecordStoreMongo) nextNum(ctx context.Context, appID string) (int64, error) {
	filter := bson.M{"_id": appID}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var counter buildCounter
	if err := s.counterColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&counter); err != nil {
		return 0, errors.Wrapf(err, "get next build num for app %s", appID)
	}
	return counter.Seq, nil
}

// Create 创建新的构建记录
func (s *RecordStoreMongo) Create(ctx context.Context, record *Record) error {
	// 获取该 AppID 的下一个构建序号
	num, err := s.nextNum(ctx, record.AppID)
	if err != nil {
		return err
	}
	record.Num = num

	timeNow := time.Now()
	record.CreatedAt, record.UpdatedAt = timeNow, timeNow
	if _, err = s.collection.InsertOne(ctx, record); err != nil {
		// Check if it's a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("build record with the same appID & buildID already exists")
		}
		return err
	}
	return nil
}

// Update 更新已存在的应用构建记录
func (s *RecordStoreMongo) Update(ctx context.Context, record *Record) error {
	filter := bson.M{
		"appID":   record.AppID,
		"buildID": record.BuildID,
	}
	updateDoc := bson.M{"$set": bson.M{
		"num":       record.Num,
		"status":    record.Status,
		"extras":    record.Extras,
		"endedAt":   record.EndedAt,
		"updatedAt": time.Now(),
	}}
	ret, err := s.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return errors.Wrapf(err, "update app %s build record %s failed", record.AppID, record.BuildID)
	}
	if ret.MatchedCount == 0 {
		return errors.Wrapf(ErrRecordNotFound, "app %s build record %s", record.AppID, record.BuildID)
	}
	return nil
}

// List 获取应用构建记录列表（支持分页）
func (s *RecordStoreMongo) List(
	ctx context.Context, appID, keyword string, page, pageSize int64,
) ([]Record, int64, error) {
	filter := bson.M{"appID": appID}
	// 支持 keyword 参数
	if keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword = regexp.QuoteMeta(keyword)
		// 模糊匹配：制品信息、操作人
		filter["$or"] = []bson.M{
			{"artifact": bson.M{"$regex": keyword, "$options": "i"}},
			{"operator": bson.M{"$regex": keyword, "$options": "i"}},
		}
	}

	// 先统计总数
	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "count app %s build records", appID)
	}

	// 分页参数
	opts := options.Find().
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize).
		SetSort(bson.D{{"createdAt", -1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "find app %s build records", appID)
	}
	defer cursor.Close(ctx)

	var records []Record
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, errors.Wrapf(err, "decode app %s build records", appID)
	}
	return records, total, nil
}

// Get 使用 AppID 和 BuildID 获取构建记录
func (s *RecordStoreMongo) Get(ctx context.Context, appID, buildID string) (*Record, error) {
	var record Record
	filter := bson.M{"appID": appID, "buildID": buildID}
	err := s.collection.FindOne(ctx, filter).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// 应用构建记录不存在则返回错误
			return nil, errors.Wrapf(ErrRecordNotFound, "app %s build record %s", appID, buildID)
		}
		return nil, err
	}
	return &record, nil
}
