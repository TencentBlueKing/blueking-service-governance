package workspace

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// The name of the MongoDB collection for storing workspace component data.
const workspaceComponentCollectionName = "workspace_components"

// ErrComponentNotFound is returned when a component is not found in the store.
var ErrComponentNotFound = errors.New("component not found")

// ComponentUpdateData 定义了更新 Component 时允许修改的数据
type ComponentUpdateData struct {
	Version       *string
	Properties    map[string]any
	Name          *string
	ScopeType     *component.ScopeType
	ScopeEnvNames []string
}

// WorkspaceCompsStore stores workspace components config data.
type WorkspaceCompsStore interface {
	// Get gets a component by its ID.
	Get(ctx context.Context, compID bson.ObjectID) (*Component, error)
	// GetByName gets a component by its name and workspace ID.
	GetByName(ctx context.Context, workspaceID, name string) (*Component, error)
	// ListByWorkspace retrieves a workspace components config record by its workspace ID.
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Component, error)
	// DeleteByWorkspace deletes a workspace components config record by its workspace ID.
	DeleteByWorkspace(ctx context.Context, workspaceID string) error

	// Add adds multiple components to a workspace.
	// Zero value fields (Name, CreatedAt, UpdatedAt, Properties) will be auto-initialized
	// before insertion
	Add(ctx context.Context, comps ...*Component) error
	// Remove deletes multiple components from a workspace.
	Remove(ctx context.Context, compIDs ...bson.ObjectID) error
	// Update updates a component in a workspace.
	Update(ctx context.Context, compID bson.ObjectID, updateData *ComponentUpdateData) error

	// === Hooks ===
	// SetComponentHooks sets hooks for component changes, which will be automatically triggered
	// after components are successfully added/removed.
	SetComponentHooks(hooks *ComponentHooks)
}

var _ WorkspaceCompsStore = &WorkspaceCompsStoreMongo{}

// ComponentHooks 空间组件变更时的可选回调，在组件成功添加/删除后自动触发。
// 用于维护 ComponentDef 引用计数等跨集合的副作用。
// 当回调返回 error 时，调用方会将其作为操作错误返回给上层。
type ComponentHooks struct {
	// AfterAdd 在组件成功添加后调用
	AfterAdd func(ctx context.Context, comps []*Component) error
	// AfterRemove 在组件成功删除后调用（Remove 和 DeleteByWorkspace 共用）
	AfterRemove func(ctx context.Context, comps []*Component) error
}

// WorkspaceCompsStoreMongo implements WorkspaceCompsConfigStore interface with mongodb
type WorkspaceCompsStoreMongo struct {
	collection     *mongo.Collection
	componentHooks *ComponentHooks
}

// SetComponentHooks 设置组件变更的回调函数
func (s *WorkspaceCompsStoreMongo) SetComponentHooks(hooks *ComponentHooks) {
	s.componentHooks = hooks
}

// NewWorkspaceCompsStoreMongo creates a new DbWorkspaceCompsStore
func NewWorkspaceCompsStoreMongo(client *mongo.Client, dbName string) (WorkspaceCompsStore, error) {
	coll := client.Database(dbName).Collection(workspaceComponentCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + name
	return &WorkspaceCompsStoreMongo{collection: coll}, nil
}

// Get returns the component with the given ID, or ErrComponentNotFound if it does not exist.
func (s *WorkspaceCompsStoreMongo) Get(ctx context.Context, compID bson.ObjectID) (*Component, error) {
	filter := bson.M{"_id": compID}
	var comp Component
	err := s.collection.FindOne(ctx, filter).Decode(&comp)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}
	return &comp, nil
}

// GetByName gets a component by its name and workspace ID.
func (s *WorkspaceCompsStoreMongo) GetByName(ctx context.Context, workspaceID, name string) (*Component, error) {
	filter := bson.M{"workspaceID": workspaceID, "name": name}
	var comp Component
	err := s.collection.FindOne(ctx, filter).Decode(&comp)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}
	return &comp, nil
}

// ListByWorkspace retrieves workspace components by workspace ID.
func (s *WorkspaceCompsStoreMongo) ListByWorkspace(
	ctx context.Context,
	workspaceID string,
) ([]*Component, error) {
	filter := bson.M{"workspaceID": workspaceID}
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var comps []*Component
	if err = cursor.All(ctx, &comps); err != nil {
		return nil, errors.Wrap(err, "decode components")
	}

	// 规范化 Properties 中的 BSON 类型为 Go 原生类型
	for i := range comps {
		comps[i].NormalizeProperties()
	}

	return comps, nil
}

// DeleteByWorkspace deletes all components of a workspace by workspace ID.
func (s *WorkspaceCompsStoreMongo) DeleteByWorkspace(ctx context.Context, workspaceID string) error {
	// 如果设置了 Hook，先查出即将被删除的组件
	var removedComps []*Component
	if s.componentHooks != nil && s.componentHooks.AfterRemove != nil {
		removedComponents, err := s.ListByWorkspace(ctx, workspaceID)
		if err != nil {
			return errors.Wrap(err, "list components by workspace")
		}
		removedComps = removedComponents
	}

	_, err := s.collection.DeleteMany(ctx, bson.M{"workspaceID": workspaceID})
	if err != nil {
		return err
	}

	if len(removedComps) > 0 {
		if err := s.componentHooks.AfterRemove(ctx, removedComps); err != nil {
			return errors.Wrap(err, "component hook after delete by workspace")
		}
	}
	return nil
}

// Add inserts one or more components into the workspace and triggers the AfterAdd hook on success.
func (s *WorkspaceCompsStoreMongo) Add(ctx context.Context, comps ...*Component) error {
	if len(comps) == 0 {
		return nil
	}

	// initialize zero value fields
	s.prepCompsDBValue(comps)
	if _, err := s.collection.InsertMany(ctx, comps); err != nil {
		return err
	}

	// 成功后触发 Hook
	if s.componentHooks != nil && s.componentHooks.AfterAdd != nil {
		if err := s.componentHooks.AfterAdd(ctx, comps); err != nil {
			return errors.Wrap(err, "component hook after add")
		}
	}
	return nil
}

// Update updates a component by its ID.
func (s *WorkspaceCompsStoreMongo) Update(
	ctx context.Context,
	compID bson.ObjectID,
	updateData *ComponentUpdateData,
) error {
	if updateData == nil {
		return nil
	}

	filter := bson.M{"_id": compID}
	updateSet := bson.M{}

	needUpdate := false

	if updateData.Version != nil {
		updateSet["version"] = *updateData.Version
		needUpdate = true
	}
	if updateData.Properties != nil {
		updateSet["properties"] = updateData.Properties
		needUpdate = true
	}
	if updateData.ScopeEnvNames != nil {
		updateSet["scopeEnvNames"] = updateData.ScopeEnvNames
		needUpdate = true
	}
	if updateData.ScopeType != nil {
		if !lo.Contains(
			[]component.ScopeType{component.ScopeTypeGlobal, component.ScopeTypeEnvironment}, *updateData.ScopeType,
		) {
			return errors.New("invalid scope type")
		}
		updateSet["scopeType"] = *updateData.ScopeType
		if *updateData.ScopeType == component.ScopeTypeGlobal {
			// 全局生效时，清空生效环境列表
			updateSet["scopeEnvNames"] = []string{}
		}
		needUpdate = true
	}
	if updateData.Name != nil {
		updateSet["name"] = *updateData.Name
		needUpdate = true
	}
	if !needUpdate {
		return nil
	}

	updateSet["updatedAt"] = time.Now()

	update := bson.M{"$set": updateSet}

	ret, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if ret.MatchedCount == 0 {
		return errors.New("component not found")
	}

	return nil
}

// Remove deletes multiple components by their IDs.
func (s *WorkspaceCompsStoreMongo) Remove(ctx context.Context, compIDs ...bson.ObjectID) error {
	if len(compIDs) == 0 {
		return nil
	}

	// 如果设置了 Hook，先查出即将被删除的组件
	var removedComps []*Component
	if s.componentHooks != nil && s.componentHooks.AfterRemove != nil {
		cursor, findErr := s.collection.Find(ctx, bson.M{"_id": bson.M{"$in": compIDs}})
		if findErr == nil {
			_ = cursor.All(ctx, &removedComps)
		}
	}

	filter := bson.M{
		"_id": bson.M{"$in": compIDs},
	}

	ret, err := s.collection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	if ret.DeletedCount == 0 {
		return ErrComponentNotFound
	}

	if len(removedComps) > 0 {
		if err := s.componentHooks.AfterRemove(ctx, removedComps); err != nil {
			return errors.Wrap(err, "component hook after remove")
		}
	}
	return nil
}

// prepCompsDBValue prepares components for database storage.
// 具体准备操作包括:
// - 一些零值字段的初始化
func (s *WorkspaceCompsStoreMongo) prepCompsDBValue(comps []*Component) {
	for _, comp := range comps {
		if comp.Name == "" {
			comp.Name = comp.GenerateName()
		}
		if comp.CreatedAt.IsZero() {
			comp.CreatedAt = time.Now()
		}
		if comp.UpdatedAt.IsZero() {
			comp.UpdatedAt = time.Now()
		}

		if comp.Properties == nil {
			comp.Properties = make(map[string]any)
		}
	}
}
