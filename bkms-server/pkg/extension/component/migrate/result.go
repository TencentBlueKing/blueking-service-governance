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

// Package migrate migrates legacy component output into patchers and specs arrays.
package migrate

// Action describes the operation for one component definition.
type Action string

const (
	// ActionMigrate replaces legacy output with patchers and specs.
	ActionMigrate Action = "migrate"
	// ActionSkip leaves an already migrated component definition unchanged.
	ActionSkip Action = "skip"
	// ActionError marks a component definition that cannot be migrated safely.
	ActionError Action = "error"
)

// Change is one component definition entry in a migration result.
type Change struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`
	Action   Action   `yaml:"action"`
	Patchers []string `yaml:"patchers,omitempty"`
	Specs    []string `yaml:"specs,omitempty"`
	Error    string   `yaml:"error,omitempty"`
}

// Summary counts processed migration actions.
type Summary struct {
	Migrated int `yaml:"migrated"`
	Skipped  int `yaml:"skipped"`
	Failed   int `yaml:"failed"`
}

// Result describes a dry run or an applied component patch migration.
type Result struct {
	DryRun  bool     `yaml:"dryRun"`
	Summary Summary  `yaml:"summary"`
	Changes []Change `yaml:"changes"`
}

func componentDefKey(name, version string) string {
	return name + ":" + version
}
