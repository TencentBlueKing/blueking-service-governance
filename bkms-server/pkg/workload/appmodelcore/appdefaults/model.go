// Package appdefaults manages workspace-scoped defaults used only when an
// AppModel application is created.
package appdefaults

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

var (
	// ErrRuleNotFound is returned when a workspace default rule does not exist.
	ErrRuleNotFound = errors.New("application default rule not found")
	// ErrRuleConflict is returned when the section already has a rule for the
	// environment type.
	ErrRuleConflict = errors.New("application default rule already exists for this environment type")
	// ErrInvalidRule is returned when a rule is incomplete or invalid.
	ErrInvalidRule = errors.New("invalid application default rule")
)

// ConfigType identifies the AppSpec section controlled by a rule.
type ConfigType = appspec.AppSpecSectionID

// RuleDefinition contains the complete configurable content used to create or
// replace a rule.
type RuleDefinition struct {
	EnvType string
	Spec    *appspec.AppSpec
}

// Rule stores one complete AppSpec section for a workspace environment type.
type Rule struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	WorkspaceID string        `bson:"workspaceID"`
	ConfigType  ConfigType    `bson:"configType"`
	EnvType     string        `bson:"envType"`

	// Spec contains exactly the section identified by ConfigType. Identity
	// fields stay empty because a rule is a workspace template, not an app spec.
	Spec *appspec.AppSpec `bson:"spec"`

	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// applyTo places the rule's section into an AppSpec under construction.
func (r Rule) applyTo(target *appspec.AppSpec) {
	switch r.ConfigType {
	case appspec.AppSpecSectionResources:
		target.Resources = r.Spec.Resources
	case appspec.AppSpecSectionDevMode:
		target.DevMode = r.Spec.DevMode
	}
}
