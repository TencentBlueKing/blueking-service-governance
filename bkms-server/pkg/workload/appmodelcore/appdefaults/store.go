package appdefaults

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CollectionName is the MongoDB collection storing workspace application defaults.
const CollectionName = "workspace_app_spec_rules"

// RuleStore persists workspace application default rules.
type RuleStore = store.GenericStore[Rule]

// RuleMeta is the storage metadata of workspace application default rules.
// Indexes are managed by db/migrations, see 000006: the unique index on
// workspace + section + environment type reports overlapping rules as ErrRuleConflict.
var RuleMeta = store.Meta[Rule]{
	Collection: CollectionName,
	NotFound:   ErrRuleNotFound,
	Duplicate:  ErrRuleConflict,
}

// ByWorkspace matches all rules in a workspace.
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByConfigType matches one section's rules in a workspace.
func ByConfigType(workspaceID string, configType ConfigType) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("configType", configType))
}

// ByID matches one rule by workspace, config type, and ID. Workspace and config
// type are part of the filter so a rule ID from another section resolves as not found.
func ByID(workspaceID string, configType ConfigType, id bson.ObjectID) store.Opt {
	return store.Combine(ByConfigType(workspaceID, configType), store.ByID(id))
}
