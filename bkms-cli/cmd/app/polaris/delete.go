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

// Package polaris provides polaris delete command
package polaris

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// NewDeleteCmd returns a Command instance for 'app polaris delete' sub command
func NewDeleteCmd() *cobra.Command {
	var appID, configName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a polaris config for an application",
		Long: `Delete a polaris config from the specified application.

This will remove the polaris service registration config. After deletion,
the application will no longer register instances to the corresponding polaris
service in the affected environments.

Note: The deletion takes effect only after a new deployment is triggered.
The registered instances will be deregistered from polaris on next deployment.`,
		Example: `  # Delete a polaris config by name
  bkms-cli app polaris delete --app my-app --name polaris-config-abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 调用后端 API 删除北极星配置
			err := client.New().DeleteAppPolarisConfig(cmd.Context(), appID, configName)
			if err != nil {
				return errors.Wrap(err, "delete app polaris config")
			}

			fmt.Printf("✓ Polaris config deleted successfully\n")
			fmt.Printf("  Name: %s\n", configName)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&configName, "name", "", "polaris config name to delete")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
