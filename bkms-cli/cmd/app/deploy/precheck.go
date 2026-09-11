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

package deploy

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	deployhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewPrecheckCmd returns a Command instance for 'app deploy precheck' sub command
func NewPrecheckCmd() *cobra.Command {
	var appID, envName, outputFormat string

	cmd := &cobra.Command{
		Use:   "precheck",
		Short: "Run pre-deployment checks",
		Long: `Check that environment variables referenced in application config files,
component properties, and Polaris service labels are defined, and that required
cluster addons are installed in the specified environment.

Exit code 0 means both checks passed.
Exit code 1 means a check failed or could not be completed.

Only supported for trpc and taf application types.`,
		Example: `  # Check before deploying to prod
  bkms-cli app deploy precheck --app myapp --env prod

  # Output as JSON for scripting
  bkms-cli app deploy precheck --app myapp --env prod -o json`,
		PreRunE: cmdutil.ResolveAppPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeployPrecheck(cmd, appID, envName, outputFormat)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&envName, "env", "", "environment name (required)")
	output.AddFormatFlag(cmd, &outputFormat)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func runDeployPrecheck(cmd *cobra.Command, appID, envName, outputFormat string) error {
	result, err := deployhandler.Precheck(cmd.Context(), appID, envName)
	if err != nil {
		return err
	}

	if outputFormat != "" && outputFormat != string(output.FormatTable) {
		formatted, fmtErr := output.FormatData(cmd.Context(), result, outputFormat)
		if fmtErr != nil {
			return errors.Wrap(fmtErr, "format output")
		}
		console.Info("%s", formatted)
		if !result.Passed {
			return clierr.Reportedf("precheck failed")
		}
		return nil
	}

	if result.Passed {
		console.Info("✓ Pre-check passed for app %s in env %s", appID, envName)
		return nil
	}

	if err := printPrecheckFailure(cmd.Context(), result); err != nil {
		return errors.Wrap(err, "print precheck findings")
	}
	return clierr.Reportedf("precheck failed")
}

func printPrecheckFailure(ctx context.Context, result *client.DeployPrecheckResult) error {
	parts := make([]string, 0, 2)
	if n := len(result.UndefinedVars); n > 0 {
		parts = append(parts, fmt.Sprintf("%d undefined environment variable(s)", n))
	}
	if n := len(result.MissingRequiredClusterAddons); n > 0 {
		parts = append(parts, fmt.Sprintf("%d missing required cluster addon(s)", n))
	}
	console.Warn("✗ Pre-check FAILED: %s found", strings.Join(parts, ", "))

	if len(result.UndefinedVars) > 0 {
		type envVarRow struct {
			Key          string
			ReferencedBy string
		}
		rows := lo.Map(result.UndefinedVars, func(v client.UndefinedEnvVar, _ int) envVarRow {
			refs := lo.Map(v.Sources, func(s client.EnvVarSource, _ int) string {
				if s.Name != "" {
					return fmt.Sprintf("%s:%s", s.Type, s.Name)
				}
				return s.Type
			})
			return envVarRow{Key: v.Key, ReferencedBy: strings.Join(refs, ", ")}
		})
		table, err := output.FormatData(ctx, rows, string(output.FormatTable))
		if err != nil {
			return errors.Wrap(err, "format undefined variables")
		}
		console.Info("\n%s", table)
		console.Tips("Fix: use 'bkms-cli envvar create' to define missing variables, then re-run precheck.")
	}

	if len(result.MissingRequiredClusterAddons) > 0 {
		type addonRow struct {
			MissingClusterAddons string
		}
		rows := lo.Map(result.MissingRequiredClusterAddons, func(addon client.ClusterAddonReference, _ int) addonRow {
			return addonRow{MissingClusterAddons: cmp.Or(addon.DisplayName, addon.Name)}
		})
		table, err := output.FormatData(ctx, rows, string(output.FormatTable))
		if err != nil {
			return errors.Wrap(err, "format missing cluster addons")
		}
		console.Info("\n%s", table)
		console.Tips("Fix: install the missing cluster addons from the environment page, then re-run precheck.")
	}
	return nil
}
