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

// Package cmd defines the commands.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/cmd/migration"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

var rootCmd = &cobra.Command{
	Use:   "bkms-server",
	Short: "bkms server",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info(cmd.Context(), "welcome to use bkms-server, use `bkms-server -h` for help")
	},
}

func init() {
	rootCmd.AddCommand(NewMigrateCmd())
	rootCmd.AddCommand(migration.NewLoadBuiltinComponentCmd())
	rootCmd.AddCommand(migration.NewBackfillDefaultAlertStrategiesCmd())
	rootCmd.AddCommand(migration.NewMigrateComponentPatchCmd())
	rootCmd.AddCommand(migration.NewMigrateTkeRouteEniComponentCmd())
	rootCmd.AddCommand(migration.NewMigrateIAMSystemModelCmd())
	rootCmd.AddCommand(migration.NewCleanupExpiredWorkspaceTempAdminsCmd())
	rootCmd.AddCommand(migration.NewCleanupOrphanAppConfigFileVersionsCmd())
	rootCmd.AddCommand(migration.NewUpsertRuntimeImageCmd())
	rootCmd.AddCommand(migration.NewRefreshWorkspaceBkmonitorPermsCmd())
}

// Execute ...
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
