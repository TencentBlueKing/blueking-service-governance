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

// Package polaris provides polaris update command
package polaris

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewUpdateCmd returns a Command instance for 'app polaris update' sub command
func NewUpdateCmd() *cobra.Command {
	var appID, configName, specFile string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a polaris config for an application",
		Long: `Update an existing polaris config from a YAML spec file (partial update).

Only fields present in the YAML file will be updated; omitted fields remain unchanged.
This allows you to modify specific aspects of the config without affecting others.

polarisName, polarisNamespace and direct cannot be updated after create.
Per-environment instance weight defaults to 100 and cannot be set via this command.

Effect timing:
  - instanceKey, servicePort and polarisToken: require a redeployment to take effect
  - All other fields (keepNotReadyPod, enableHealthCheck, serviceLabels,
    scopeEnvNames): take effect immediately without redeployment when the environment
    is already deployed and the redeploy-required fields are unchanged

Updatable YAML spec file fields:

  [Require redeployment]
  instanceKey:        Instance key used as env var prefix
                      (letters/digits/underscore, must start with letter)
  servicePort:        The port your application listens on (1-65535)
  polarisToken:       Polaris access token

  [Take effect immediately]
  keepNotReadyPod:    Keep not-ready pods in polaris instance list with 0 weight (bool).
                      If false, not-ready pods will be deregistered from polaris immediately
  enableHealthCheck:  Enable polaris health check for registered instances (bool).
                      When enabled, polaris will actively probe instance health
  serviceLabels:      Labels applied to ALL registered polaris instances (map[string]string).
                      When provided, fully replaces existing labels (not merged)
  scopeEnvNames:      Replace the environments where this config takes effect ([]string).
                      An empty list clears the scope`,
		Example: `  # Update service port
  bkms-cli app polaris update --app my-app --name polaris-xxxxx -f update.yaml

  # Example update.yaml (change port):
  # servicePort: 9090

  # Example update.yaml (change scope environments):
  # scopeEnvNames:
  #   - prod
  #   - staging

  # Example update.yaml (update labels):
  # serviceLabels:
  #   version: v2
  #   region: shenzhen`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 读取并解析 YAML 规格文件
			if _, err := os.Stat(specFile); err != nil {
				return errors.Wrapf(err, "polaris spec file %s not found", specFile)
			}
			fileContent, err := os.ReadFile(specFile)
			if err != nil {
				return errors.Wrapf(err, "read polaris spec file %s failed", specFile)
			}

			// 解析为 map 以保持灵活性，仅传递 YAML 中存在的字段实现部分更新
			var body map[string]any
			if err = yaml.Unmarshal(fileContent, &body); err != nil {
				return errors.Wrapf(err, "parse polaris spec file %s failed, please check YAML syntax", specFile)
			}
			// CLI 不支持修改直连模式，忽略 YAML 中的 direct 以免误改
			delete(body, "direct")

			// 调用后端 API 更新北极星配置
			err = client.New().PatchAppPolarisConfig(cmd.Context(), appID, configName, body)
			if err != nil {
				return errors.Wrap(err, "update app polaris config")
			}

			console.Info("✓ Polaris config updated successfully")
			console.Info("  Name: %s", configName)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().
		StringVar(&configName, "name", "", "polaris config name from list (e.g. polaris-xxxxx), not polarisName")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "polaris update spec file path (YAML)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
