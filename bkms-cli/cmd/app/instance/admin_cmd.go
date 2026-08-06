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

package instance

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/instance"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/params"
)

// NewListAdminCmdsCmd returns a Command instance for 'app instance list-admin-cmds' sub command
func NewListAdminCmdsCmd() *cobra.Command {
	var appID, envName, instanceIDsStr, workspaceID, outputFormat string

	cmd := &cobra.Command{
		Use:   "list-admin-cmds",
		Short: "List admin commands for application instances",
		Long: `List available admin commands for the specified application instances.

This command queries the admin command list from application instances
via the admin port. Only Trpc type applications are supported.`,
		Example: `  # List admin commands
  bkms-cli app instance list-admin-cmds --app myapp --env test --instance-ids pod1,pod2`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			instanceIDs, err := params.MustGetSplitString(instanceIDsStr, ",")
			if err != nil {
				return errors.Wrap(err, "get instance IDs")
			}

			// 检查 app 是否为 trpc 类型
			app, err := client.New().GetAppMinimal(cmd.Context(), workspaceID, appID)
			if err != nil {
				return errors.Wrap(err, "get app info")
			}
			if app.Type != constant.AppTypeTrpc {
				return errors.Errorf("list-admin-cmds is only supported for Trpc app, got app type: %s", app.Type)
			}

			results, err := handler.ListTrpcAdminCmds(cmd.Context(), appID, envName, instanceIDs)
			if err != nil {
				return errors.Wrap(err, "list admin cmds")
			}

			items := make([]adminCmdItem, len(results))
			for i, cmd := range results {
				items[i] = adminCmdItem{Command: cmd}
			}
			formatted, err := output.FormatData(cmd.Context(), items, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVar(&instanceIDsStr, "instance-ids", "", "instance IDs, comma separated")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("instance-ids")

	return cmd
}

// NewExecAdminCmdCmd returns a Command instance for 'app instance exec-admin-cmd' sub command
func NewExecAdminCmdCmd() *cobra.Command {
	var (
		appID, envName, instanceIDsStr, workspaceID, outputFormat string
		method, urlPath, paramsJSON, body, command                string
	)

	cmd := &cobra.Command{
		Use:   "exec-admin-cmd",
		Short: "Execute admin command on application instances",
		Long: `Execute an admin command on application instances.

Automatically routes to Trpc or Taf admin command API based on the
application type:
  - Trpc: requires --method, --url; optional --params, --body
  - Taf: requires --command (e.g., "taf.viewversion", "taf.setloglevel DEBUG")`,
		Example: `  # Execute Trpc admin command (auto-detected by app type)
  bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 --method GET --url /cmds

  # Execute Trpc admin command with params
  bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 \
    --method POST --url /config --params '{"key":"val"}'

  # Execute Taf admin command (auto-detected by app type)
  bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 --command "taf.viewversion"`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			instanceIDs, err := params.MustGetSplitString(instanceIDsStr, ",")
			if err != nil {
				return errors.Wrap(err, "get instance IDs")
			}

			// 先获取 appType 来校验必填参数
			app, err := client.New().GetAppMinimal(cmd.Context(), workspaceID, appID)
			if err != nil {
				return errors.Wrap(err, "get app info")
			}

			opts := handler.ExecAdminCmdOptions{
				InstanceIDs: instanceIDs,
				Method:      method,
				URL:         urlPath,
				Body:        body,
				Command:     command,
			}

			switch app.Type {
			case constant.AppTypeTrpc:
				if method == "" {
					return errors.New("--method is required for Trpc app")
				}
				if urlPath == "" {
					return errors.New("--url is required for Trpc app")
				}
				if paramsJSON != "" {
					if parseErr := json.Unmarshal([]byte(paramsJSON), &opts.Params); parseErr != nil {
						return errors.Wrap(parseErr, "parse --params")
					}
				}
			case constant.AppTypeTaf:
				if command == "" {
					return errors.New("--command is required for Taf app")
				}
			default:
				return errors.Errorf("unsupported app type for admin cmd: %s", app.Type)
			}

			results, err := handler.ExecAdminCmd(cmd.Context(), workspaceID, appID, envName, opts)
			if err != nil {
				return errors.Wrap(err, "exec admin cmd")
			}

			formatted, err := output.FormatData(cmd.Context(), results, outputFormat)
			if err != nil {
				return errors.Wrap(err, "format output")
			}
			fmt.Println(formatted)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVar(&instanceIDsStr, "instance-ids", "", "instance IDs, comma separated")
	cmd.Flags().StringVar(&method, "method", "", "HTTP method for Trpc admin cmd (GET/POST/PUT)")
	cmd.Flags().StringVar(&urlPath, "url", "", "URL path for Trpc admin cmd")
	cmd.Flags().StringVar(&paramsJSON, "params", "", "query params for Trpc admin cmd, JSON string")
	cmd.Flags().StringVar(&body, "body", "", "request body for Trpc admin cmd")
	cmd.Flags().StringVar(&command, "command", "", "command for Taf admin cmd (e.g. taf.viewversion)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", output.FlagUsage)

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("instance-ids")

	return cmd
}

// adminCmdItem 用于将 []string 格式化为表格输出
type adminCmdItem struct {
	Command string `json:"command"`
}
