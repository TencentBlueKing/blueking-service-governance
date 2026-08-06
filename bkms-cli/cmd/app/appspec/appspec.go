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

// Package appspec provides the appspec command group.
package appspec

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/annotations"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/labels"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/lifecycle"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/probe"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/resources"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/startcommand"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app/appspec/updatestrategy"
)

// NewCmd creates the appspec command group.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appspec",
		Short: "Manage application deployment spec",
		Long: `Manage application deployment spec (AppSpec) sections.

AppSpec defines how an application is deployed, including start command, resource limits,
update strategy, lifecycle hooks, health probes, labels and annotations.`,
		Example: `  # View all sections (default config)
  bkms-cli app appspec view --app my-app

  # View all sections (env effective config)
  bkms-cli app appspec view --app my-app --env prod

  # View all sections in JSON format
  bkms-cli app appspec view --app my-app -o json`,
		DisableFlagsInUseLine: true,
	}

	// Register subcommands
	cmd.AddCommand(NewViewCmd())
	cmd.AddCommand(startcommand.NewCmd())
	cmd.AddCommand(lifecycle.NewCmd())
	cmd.AddCommand(probe.NewCmd())
	cmd.AddCommand(resources.NewCmd())
	cmd.AddCommand(updatestrategy.NewCmd())
	cmd.AddCommand(labels.NewCmd())
	cmd.AddCommand(annotations.NewCmd())

	return cmd
}
