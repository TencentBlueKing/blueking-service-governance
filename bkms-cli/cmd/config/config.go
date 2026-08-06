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

// Package config provide command to manage bkms-cli config
package config

import (
	"github.com/spf13/cobra"

	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

var configLongDesc = `
Display bkms-cli config files using subcommands like "bkms-cli config view"

The loading order follows these rules:
	
  1.  ${BKMS_CLI_CONFIG} environment variable.
  2.  Use ${HOME}/.bkms/config.yaml.
`

// NewCmd create bkms-cli config command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "config",
		Short:                 "Manage bkms-cli config",
		Long:                  configLongDesc,
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
	}

	// 配置信息查看
	cmd.AddCommand(NewCmdView())
	return cmd
}
