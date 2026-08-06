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

package labels

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appspec"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewEditCmd returns a Command instance for 'appspec labels edit' sub command.
func NewEditCmd() *cobra.Command {
	var appID, envName, specFile string

	cmd := &cobra.Command{
		Use:    "edit",
		Short:  "Edit labels configuration from a YAML file",
		PreRun: cmdutil.CommonPreRun,
		Long: `Edit the Kubernetes labels configuration for the application from a YAML file.

When --env is omitted, this command edits the default application-level labels.
When --env is provided, this command edits the labels for that specific environment.`,
		Example: `  # YAML file format (labels.yaml):
  labels:
    key1: value1
    key2: value2

  # Edit default labels config
  bkms-cli app appspec labels edit --app my-app -f labels.yaml

  # Edit env-specific labels config
  bkms-cli app appspec labels edit --app my-app --env prod -f labels.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFile == "" {
				return errors.New("-f is required for edit")
			}

			if err := appspec.EditHandler(cmd.Context(), appID, envName, specFile, client.AppSpecSectionLabels); err != nil {
				return errors.Wrap(err, "edit labels")
			}

			if envName == "" {
				fmt.Printf("Successfully updated default labels for app %s\n", appID)
			} else {
				fmt.Printf("Successfully updated labels for app %s in env %s\n", appID, envName)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "environment name (optional, omit for default config)")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "YAML spec file path (required)")

	_ = cmd.MarkFlagRequired("app")

	return cmd
}
