// Package hooks registers application-default lifecycle hooks.
package hooks

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
)

// CleanupRulesByWorkspaceHookName identifies the callback that removes a
// workspace's environment-type rules during workspace deletion.
const CleanupRulesByWorkspaceHookName = "appdefaults.cleanup_rules_by_workspace"

// RegisterPreDeleteHooks registers application-default cleanup before workspace deletion.
func RegisterPreDeleteHooks(db *mongo.Database) {
	bkmsworkspace.RegisterPreDeleteHook(CleanupRulesByWorkspaceHookName, NewCleanupRulesByWorkspaceHook(db))
}

// NewCleanupRulesByWorkspaceHook removes all rules before deleting a workspace.
func NewCleanupRulesByWorkspaceHook(db *mongo.Database) bkmsworkspace.PreDeleteHook {
	return func(ctx context.Context, workspaceID string) error {
		rules := store.Of(db, appdefaults.RuleMeta)
		if _, err := rules.DeleteMany(ctx, appdefaults.ByWorkspace(workspaceID)); err != nil {
			return errors.Wrapf(err, "delete application default rules of workspace %s", workspaceID)
		}
		return nil
	}
}
