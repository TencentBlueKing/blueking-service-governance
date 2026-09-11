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

package env

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// NewListCmd 创建 app env list 子命令。
func NewListCmd() *cobra.Command {
	var appID, kind, outputFormat string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List standard and application-owned feature environments",
		Long:  "List available environments, optionally filtered by kind: standard or feature. Status indicates environment readiness; use 'app deploy list' to inspect deployments.",
		Example: `  bkms-cli app env list --app my-app
  bkms-cli app env list --app my-app --kind feature -o json`,
		Args: cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// 命令层负责参数规范化和用法错误，在解析应用前完成校验。
			kind = strings.TrimSpace(kind)
			if kind != "" && kind != "standard" && kind != "feature" {
				return clierr.Usage(errors.Errorf("invalid --kind %q: expected standard or feature", kind))
			}
			return cmdutil.ResolveAppPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			envs, err := handler.ListAppEnvs(cmd.Context(), client.New(), appID, kind)
			if err != nil {
				return err
			}
			formatted, err := output.FormatData(cmd.Context(), envs, outputFormat)
			if err != nil {
				return err
			}
			console.Info("%s", formatted)
			return nil
		},
	}
	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&kind, "kind", "", "Filter by environment kind: standard | feature (default: all)")
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")
	return cmd
}
