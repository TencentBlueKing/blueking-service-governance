package portforward

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "plat_admin_port_forward_whitelist"

// Store 持久化 port-forward 白名单记录。
// 每个被允许的环境 ID 对应集合中的一条独立文档（_id = envID）。
type Store interface {
	// Add 将环境 ID 添加到白名单中（已存在则忽略）。
	Add(ctx context.Context, envIDs []string) error
	// Remove 从白名单中移除指定的环境 ID。
	Remove(ctx context.Context, envIDs []string) error
	// List 返回当前白名单中所有环境 ID。
	List(ctx context.Context) ([]string, error)
	// Contains 判断指定 envID 是否存在于白名单中。
	Contains(ctx context.Context, envID string) (bool, error)
}

var _ Store = (*StoreMongo)(nil)

// StoreMongo 是 Store 接口的 MongoDB 实现。
type StoreMongo struct {
	collection *mongo.Collection
}

// NewStoreMongo 创建基于 MongoDB 的 port-forward 白名单 Store。
func NewStoreMongo(client *mongo.Client, dbName string) (*StoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	return &StoreMongo{collection: coll}, nil
}

// Add 将环境 ID 添加到白名单中。每个 envID 作为独立文档插入（_id = envID），已存在则跳过。
func (s *StoreMongo) Add(ctx context.Context, envIDs []string) error {
	if len(envIDs) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(envIDs))
	for _, envID := range envIDs {
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": envID}).
			SetUpdate(bson.M{"$setOnInsert": bson.M{"_id": envID}}).
			SetUpsert(true))
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := s.collection.BulkWrite(ctx, models, opts)
	if err != nil {
		return errors.Wrap(err, "add env IDs to port-forward whitelist")
	}
	return nil
}

// Remove 从白名单中移除指定的环境 ID（删除对应文档）。
func (s *StoreMongo) Remove(ctx context.Context, envIDs []string) error {
	if len(envIDs) == 0 {
		return nil
	}

	filter := bson.M{"_id": bson.M{"$in": envIDs}}
	_, err := s.collection.DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "remove env IDs from port-forward whitelist")
	}
	return nil
}

// List 返回白名单中所有环境 ID。
func (s *StoreMongo) List(ctx context.Context) ([]string, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "list port-forward whitelist")
	}
	defer cursor.Close(ctx)

	var entries []WhitelistEntry
	if err = cursor.All(ctx, &entries); err != nil {
		return nil, errors.Wrap(err, "decode port-forward whitelist entries")
	}

	result := make([]string, 0, len(entries))
	for _, e := range entries {
		result = append(result, e.EnvID)
	}
	return result, nil
}

// Contains 判断指定 envID 是否存在于白名单中。
func (s *StoreMongo) Contains(ctx context.Context, envID string) (bool, error) {
	count, err := s.collection.CountDocuments(ctx, bson.M{"_id": envID})
	if err != nil {
		return false, errors.Wrap(err, "check port-forward whitelist contains env ID")
	}
	return count > 0, nil
}
