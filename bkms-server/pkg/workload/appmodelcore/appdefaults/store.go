package appdefaults

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CollectionName is the MongoDB collection storing workspace application defaults.
const CollectionName = "workspace_app_spec_rules"

// RuleStore persists workspace application default rules.
type RuleStore interface {
	List(ctx context.Context, workspaceID string) ([]Rule, error)
	ListByConfigType(ctx context.Context, workspaceID string, configType ConfigType) ([]Rule, error)
	Get(ctx context.Context, workspaceID string, configType ConfigType, id bson.ObjectID) (*Rule, error)
	Create(ctx context.Context, rule *Rule) error
	Update(ctx context.Context, workspaceID string, configType ConfigType, rule *Rule) error
	Delete(ctx context.Context, workspaceID string, configType ConfigType, id bson.ObjectID) (*Rule, error)
	DeleteByWorkspace(ctx context.Context, workspaceID string) error
	Drop(ctx context.Context) error
}

var _ RuleStore = (*RuleStoreMongo)(nil)

// RuleStoreMongo is the MongoDB RuleStore implementation.
type RuleStoreMongo struct {
	collection *mongo.Collection
}

// NewRuleStoreMongo creates a MongoDB-backed rule store.
func NewRuleStoreMongo(client *mongo.Client, dbName string) (*RuleStoreMongo, error) {
	collection := client.Database(dbName).Collection(CollectionName)
	// A workspace section has at most one rule for each environment type.
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{
			{Key: "workspaceID", Value: 1},
			{Key: "configType", Value: 1},
			{Key: "envType", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return &RuleStoreMongo{collection: collection}, nil
}

// List lists all rules in a workspace.
func (s *RuleStoreMongo) List(ctx context.Context, workspaceID string) ([]Rule, error) {
	return s.list(ctx, bson.M{"workspaceID": workspaceID})
}

// ListByConfigType lists one section's rules in a workspace.
func (s *RuleStoreMongo) ListByConfigType(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
) ([]Rule, error) {
	return s.list(ctx, bson.M{
		"workspaceID": workspaceID,
		"configType":  configType,
	})
}

func (s *RuleStoreMongo) list(ctx context.Context, filter bson.M) ([]Rule, error) {
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list application default rules: %w", err)
	}
	defer cursor.Close(ctx)

	rules := make([]Rule, 0)
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("decode application default rules: %w", err)
	}
	return rules, nil
}

// Get gets one rule by workspace, config type, and ID.
func (s *RuleStoreMongo) Get(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	id bson.ObjectID,
) (*Rule, error) {
	rule := new(Rule)
	err := s.collection.FindOne(ctx, bson.M{
		"_id":         id,
		"workspaceID": workspaceID,
		"configType":  configType,
	}).Decode(rule)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	return rule, nil
}

// Create creates a rule.
func (s *RuleStoreMongo) Create(ctx context.Context, rule *Rule) error {
	rule.ID = bson.NewObjectID()
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	if _, err := s.collection.InsertOne(ctx, rule); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: create rule", ErrRuleConflict)
		}
		return err
	}
	return nil
}

// Update replaces the editable environment type and configuration of a rule.
func (s *RuleStoreMongo) Update(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	rule *Rule,
) error {
	rule.UpdatedAt = time.Now()
	// Replace the complete document so removed configuration cannot survive as
	// stale BSON. The caller supplies the existing rule's immutable identity.
	result, err := s.collection.ReplaceOne(ctx, bson.M{
		"_id":         rule.ID,
		"workspaceID": workspaceID,
		"configType":  configType,
	}, rule)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("%w: update rule", ErrRuleConflict)
		}
		return err
	}
	if result.MatchedCount == 0 {
		return ErrRuleNotFound
	}
	return nil
}

// Delete deletes and returns one rule by workspace, config type, and ID.
func (s *RuleStoreMongo) Delete(
	ctx context.Context,
	workspaceID string,
	configType ConfigType,
	id bson.ObjectID,
) (*Rule, error) {
	rule := new(Rule)
	err := s.collection.FindOneAndDelete(ctx, bson.M{
		"_id":         id,
		"workspaceID": workspaceID,
		"configType":  configType,
	}).Decode(rule)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	return rule, nil
}

// DeleteByWorkspace deletes all rules in a workspace.
func (s *RuleStoreMongo) DeleteByWorkspace(ctx context.Context, workspaceID string) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{"workspaceID": workspaceID})
	return err
}

// Drop drops the collection. It is intended for tests only.
func (s *RuleStoreMongo) Drop(ctx context.Context) error {
	return s.collection.Drop(ctx)
}
