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

// Package publish 开发模式文件发布命令
package publish

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/publish"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/params"
)

// NewCmd 创建 publish 命令
func NewCmd() *cobra.Command {
	var appID, envName, file, instances, bcsToken, workspaceID string
	var publishAll bool

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish binary file to dev mode container",
		Long: `Publish binary file to dev mode container.

This command uploads a binary file to the dev mode container for the specified app and environment.
Dev mode must be enabled for the target environment.
Use --bcs-token to provide BCS API token on first use, it will be saved to config for subsequent calls.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional. Otherwise, you must specify it explicitly.`,
		Example: `
# First use, provide BCS token (will be saved to config)
bkms-cli app publish --app myapp --env stage -f /path/to/binary --instance-ids pod1,pod2 --bcs-token <token>

# Or write token directly in ~/.bkms/config.yaml, same effect:
#   bcs:
#     token: <token>

# Subsequent use, token is read from config
bkms-cli app publish --app myapp --env stage -f /path/to/binary --instance-ids pod1,pod2

# Publish to all Running instances
bkms-cli app publish --app myapp --env stage -f /path/to/binary --all

# Specify workspace explicitly
bkms-cli app publish --workspace ws-demo --app myapp --env stage -f /path/to/binary --instance-ids pod1,pod2`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			specifiedInstanceIDs := params.NormalizeInstIDs(instances, ",")

			if publishAll && len(specifiedInstanceIDs) > 0 {
				return errors.New("--all and --instance-ids cannot be used together")
			}
			if !publishAll && len(specifiedInstanceIDs) == 0 {
				return errors.New("instance-ids is required unless --all is specified")
			}

			// 发布二进制到线上实例
			publisher := publish.NewPublisher(cmd.Context(), client.New(), workspaceID, appID, envName)
			if err := publisher.PreCheck(); err != nil {
				return err
			}

			var (
				targetInstanceIDs []string
				err               error
			)

			switch {
			case publishAll:
				targetInstanceIDs, err = publisher.GetAllRunningInstanceIDs()
			default:
				targetInstanceIDs, err = publisher.GetSpecifiedInstanceIDs(specifiedInstanceIDs)
			}
			if err != nil {
				return err
			}
			if err := publisher.Publish(file, targetInstanceIDs, bcsToken); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "Application ID (required)")
	cmd.Flags().StringVar(&envName, "env", "", "Environment name (required)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to the binary file to publish (required)")
	cmd.Flags().
		StringVar(&instances, "instance-ids", "", "Instance IDs (pod names) to publish to, comma-separated")
	cmd.Flags().BoolVar(&publishAll, "all", false, "Publish to all Running instances")

	cmd.Flags().
		StringVar(&bcsToken, "bcs-token", "", "BCS API access token (only required on first use, will be saved to config)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
