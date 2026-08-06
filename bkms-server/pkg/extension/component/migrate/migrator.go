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

package migrate

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// Migrator performs the component patch storage migration.
type Migrator struct {
	db *mongo.Database
}

// New creates a component patch storage migrator.
func New(db *mongo.Database) *Migrator {
	return &Migrator{db: db}
}

// Run processes component definitions in name and version order, stopping on the first error.
func (m *Migrator) Run(ctx context.Context, dryRun bool) (Result, error) {
	items, err := m.listCurrentComponentDefs(ctx)
	if err != nil {
		return Result{DryRun: dryRun}, err
	}

	result := Result{DryRun: dryRun, Changes: make([]Change, 0, len(items))}
	collection := m.db.Collection(componentDefsCollection)
	for _, item := range items {
		change := convertComponentDef(item)
		if change.Action != ActionError && !dryRun {
			if change.Action == ActionMigrate {
				updateResult, updateErr := collection.UpdateOne(ctx,
					componentDefCASFilter(item),
					bson.M{
						"$set":   bson.M{"patchers": change.Patchers, "specs": change.Specs},
						"$unset": bson.M{"output": ""},
					},
				)
				if updateErr != nil {
					change.Action = ActionError
					change.Error = fmt.Sprintf("migrate %s: %v",
						componentDefKey(change.Name, change.Version), updateErr)
				} else if updateResult.MatchedCount != 1 {
					change.Action = ActionError
					change.Error = fmt.Sprintf("migrate %s: stale component definition",
						componentDefKey(change.Name, change.Version))
				}
			}
		}

		result.Changes = append(result.Changes, change)
		switch change.Action {
		case ActionMigrate:
			result.Summary.Migrated++
		case ActionSkip:
			result.Summary.Skipped++
		case ActionError:
			result.Summary.Failed++
			return result, errors.New(change.Error)
		}
	}
	return result, nil
}

func convertComponentDef(item currentComponentDef) Change {
	change := Change{Name: item.Name, Version: item.Version}
	if item.Output == nil {
		change.Action = ActionSkip
		change.Patchers = item.Patchers
		change.Specs = item.Specs
		if item.Patchers == nil || item.Specs == nil {
			change.Action = ActionError
			change.Error = fmt.Sprintf("component %s must contain patchers and specs arrays",
				componentDefKey(item.Name, item.Version))
		} else if err := validateComponentFragments(
			item.Name,
			item.Version,
			item.Patchers,
			item.Specs,
		); err != nil {
			change.Action = ActionError
			change.Error = err.Error()
		}
		return change
	}

	change.Action = ActionMigrate
	var patchers, specs []string
	var err error
	if item.Name == importPolarisComponentName && item.Version == component.DefaultComponentDefVersion {
		patchers, specs = importPolarisFragments()
	} else {
		patchers, specs, err = ConvertLegacyOutput(*item.Output)
	}
	change.Patchers = patchers
	change.Specs = specs
	if err == nil {
		err = validateComponentFragments(item.Name, item.Version, patchers, specs)
	}
	if err != nil {
		change.Action = ActionError
		change.Error = err.Error()
	}
	return change
}

func validateComponentFragments(name, version string, patchers, specs []string) error {
	migrationDef := &component.ComponentDef{
		Name:     name,
		Version:  version,
		Patchers: patchers,
		Specs:    specs,
	}
	if err := component.ValidateComponentDef(migrationDef); err != nil {
		return fmt.Errorf("component %s: %w", componentDefKey(name, version), err)
	}
	return nil
}

func componentDefCASFilter(item currentComponentDef) bson.M {
	return bson.M{
		"name":    item.Name,
		"version": item.Version,
		"output":  *item.Output,
	}
}
