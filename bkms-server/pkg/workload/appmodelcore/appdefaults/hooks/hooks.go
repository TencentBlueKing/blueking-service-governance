// Package hooks registers application-default lifecycle hooks.
package hooks

import (
	"context"
	"fmt"

	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
)

// CleanupRulesByWorkspaceHookName identifies the callback that removes a
// workspace's environment-type rules during workspace deletion.
const CleanupRulesByWorkspaceHookName = "appdefaults.cleanup_rules_by_workspace"

// RegisterPreDeleteHooks registers application-default cleanup before workspace deletion.
func RegisterPreDeleteHooks(store appdefaults.RuleStore) {
	bkmsworkspace.RegisterPreDeleteHook(CleanupRulesByWorkspaceHookName, NewCleanupRulesByWorkspaceHook(store))
}

// NewCleanupRulesByWorkspaceHook removes all rules before deleting a workspace.
func NewCleanupRulesByWorkspaceHook(store appdefaults.RuleStore) bkmsworkspace.PreDeleteHook {
	return func(ctx context.Context, workspaceID string) error {
		if err := store.DeleteByWorkspace(ctx, workspaceID); err != nil {
			return fmt.Errorf("delete application default rules: %w", err)
		}
		return nil
	}
}
