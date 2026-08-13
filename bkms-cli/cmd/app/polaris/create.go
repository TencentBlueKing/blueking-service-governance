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

// Package polaris provides polaris create command
package polaris

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCreateCmd returns a Command instance for 'app polaris create' sub command
func NewCreateCmd() *cobra.Command {
	var appID, specFile string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a polaris config for an application",
		Long: `Create a new polaris config for an application from a YAML spec file.

The YAML spec file structure is consistent with the backend API request body.

Note: After creating a polaris config, you need to trigger a deployment for the
config to take effect in the cluster. polarisName and polarisNamespace cannot
be changed after create.

YAML spec file fields:

  [Required]
  instanceKey:        Instance key used as env var prefix. e.g. "my_polaris" generates
                      my_polaris_polarisToken and my_polaris_serviceport
                      (letters/digits/underscore, must start with letter)
  polarisName:        Polaris service name to register with
  polarisNamespace:   Polaris namespace: Test | Production | Development | Pre-release
  polarisToken:       Polaris access token (required when createNewService is false;
                      when createNewService is true, the platform creates the service and fills this automatically)
  servicePort:        The port your application listens on, will be registered to polaris (1-65535)
  operator:           Owner of the polaris service (required when createNewService is true).
                      Multiple owners are comma-separated, e.g. "zhangsan,lisi"

  [Optional]
  scopeEnvNames:      List of environment names where this config takes effect ([]string).
                      Omitted or empty means the config applies to no environments
  createNewService:   If true, the platform will create a new polaris service and fill in the token
                      automatically; if false (default), you must provide an existing polarisToken
  keepNotReadyPod:    Keep not-ready pods in polaris instance list with 0 weight (bool, default true).
                      If false, not-ready pods will be deregistered from polaris immediately
  enableHealthCheck:  Enable polaris health check for registered instances (bool, default false).
                      When enabled, polaris will actively probe instance health
  serviceLabels:      Labels applied to ALL registered polaris instances (map[string]string).
                      Can be used for polaris routing rules and traffic management`,
		Example: `  # Create a polaris config from a YAML spec file:
  bkms-cli app polaris create --app my-app -f polaris.yaml

  # Example polaris.yaml (use existing polaris service):
  # scopeEnvNames:
  #   - prod
  # instanceKey: my_polaris
  # polarisName: my-service
  # polarisNamespace: Production
  # polarisToken: "xxxx"
  # servicePort: 8080

  # Example polaris.yaml (platform creates new service):
  # createNewService: true
  # scopeEnvNames:
  #   - test
  # instanceKey: auto_polaris
  # polarisName: my-new-service
  # polarisNamespace: Test
  # servicePort: 9090
  # operator: zhangsan,lisi`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 读取并解析 YAML 规格文件
			if _, err := os.Stat(specFile); err != nil {
				return errors.Wrapf(err, "polaris spec file %s not found", specFile)
			}
			fileContent, err := os.ReadFile(specFile)
			if err != nil {
				return errors.Wrapf(err, "read polaris spec file %s failed", specFile)
			}

			// 解析为 map 以保持灵活性，直接传递给后端 API
			var body map[string]any
			if err = yaml.Unmarshal(fileContent, &body); err != nil {
				return errors.Wrapf(err, "parse polaris spec file %s failed, please check YAML syntax", specFile)
			}
			// CLI 固定直连模式，不接受 YAML 中的 direct 字段
			body["direct"] = true

			// 调用后端 API 创建北极星配置
			name, err := client.New().CreateAppPolarisConfig(cmd.Context(), appID, body)
			if err != nil {
				return errors.Wrap(err, "create app polaris config")
			}

			console.Info("✓ Polaris config created successfully")
			console.Info("  Name: %s", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "polaris spec file path (YAML)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
