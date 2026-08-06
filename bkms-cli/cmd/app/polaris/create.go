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
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
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
config to take effect in the cluster.

YAML spec file fields:

  [Required]
  scopeEnvNames:      List of environment names where this config takes effect ([]string)
  instanceKey:        Instance key used as env var prefix, e.g. "my_polaris" generates
                      env vars like my_polaris_polarisToken (letters/digits/underscore, must start with letter)
  polarisName:        Polaris service name to register with
  polarisNamespace:   Polaris namespace: Test | Production | Development | Pre-release
  polarisToken:       Polaris access token (required when createNewService is false;
                      when createNewService is true, the platform creates the service and fills this automatically)
  servicePort:        The port your application listens on, will be registered to polaris (1-65535)

  [Optional]
  createNewService:   If true, the platform will create a new polaris service and fill in the token
                      automatically; if false (default), you must provide an existing polarisToken
  direct:             Register Pod IP directly to polaris service instead of ClusterIP (bool, default false).
                      When enabled, each pod's IP:port is registered as an individual polaris instance
  keepNotReadyPod:    Keep not-ready pods in polaris service instance list with 0 weight (bool, default true).
                      If false, not-ready pods will be deregistered from polaris immediately
  enableHealthCheck:  Enable polaris health check for registered instances (bool, default false).
                      When enabled, polaris will actively probe instance health
  weight:             Default weight applied to ALL registered instances of this service (int, default 10).
                      Higher weight means more traffic routed to the instance
  serviceLabels:      Labels applied to ALL registered polaris instances (map[string]string).
                      Can be used for polaris routing rules and traffic management
  operator:           Operator/owner of the polaris service (only effective when createNewService is true)`,
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
  # operator: zhangsan`,
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
			// scopeType 固定为 "environment"，用户无需填写
			body["scopeType"] = "environment"

			// 调用后端 API 创建北极星配置
			name, err := client.New().CreateAppPolarisConfig(cmd.Context(), appID, body)
			if err != nil {
				return errors.Wrap(err, "create app polaris config")
			}

			fmt.Printf("✓ Polaris config created successfully\n")
			fmt.Printf("  Name: %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVarP(&specFile, "file", "f", "", "polaris spec file path (YAML)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
