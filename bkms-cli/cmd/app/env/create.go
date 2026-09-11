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

// NewCreateCmd 创建 app env create 子命令，仅创建应用专属特性环境。
func NewCreateCmd() *cobra.Command {
	var appID, source, displayName, outputFormat string
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a feature environment from a standard environment",
		Long:    "Clone a standard environment and its application configuration. The server generates the environment name and an isolated namespace.",
		Example: "  bkms-cli app env create --app my-app --source-env staging --display-name 'New feature' -o json",
		Args:    cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// 在解析应用前规范化并校验必填参数，包含显式传入空白字符串的情况。
			source = strings.TrimSpace(source)
			displayName = strings.TrimSpace(displayName)
			if source == "" {
				return clierr.Usage(errors.New("--source-env must not be empty"))
			}
			if displayName == "" {
				return clierr.Usage(errors.New("--display-name must not be empty"))
			}
			return cmdutil.ResolveAppPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := handler.CreateFeatureEnv(cmd.Context(), client.New(), appID, source, displayName)
			if err != nil {
				return err
			}
			formatted, err := output.FormatData(cmd.Context(), env, outputFormat)
			if err != nil {
				return errors.Wrapf(
					err,
					"feature environment %s (%s) created, but formatting output failed",
					env.Name,
					env.ID,
				)
			}
			console.Info("%s", formatted)
			return nil
		},
	}
	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringVar(&source, "source-env", "", "Source standard environment name or ID (required)")
	cmd.Flags().StringVar(&displayName, "display-name", "", "Feature environment display name (required)")
	output.AddFormatFlag(cmd, &outputFormat)
	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("source-env")
	_ = cmd.MarkFlagRequired("display-name")
	return cmd
}
