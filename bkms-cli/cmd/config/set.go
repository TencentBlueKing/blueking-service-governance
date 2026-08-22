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

package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCmdSet updates API endpoint settings in the local config file.
func NewCmdSet() *cobra.Command {
	var bkmsBaseURL string
	var bcsAPIHost string
	var ifUnset bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set bkms-cli API endpoints",
		Long: `Update API endpoint fields in the local config file.

At least one of --bkms-base-url or --bcs-api-host must be provided.
Unspecified fields are left unchanged.

With --if-unset, a field is written only when it is currently empty.`,
		Example: "  bkms-cli config set --bkms-base-url https://bkms.example.com\n" +
			"  bkms-cli config set --bkms-base-url https://bkms.example.com --bcs-api-host https://bcs-api.example.com\n" +
			"  bkms-cli config set --if-unset --bkms-base-url https://bkms.example.com",
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if bkmsBaseURL == "" && bcsAPIHost == "" {
				return errors.New("at least one of --bkms-base-url or --bcs-api-host is required")
			}

			updated, err := config.G.SetEndpoints(bkmsBaseURL, bcsAPIHost, ifUnset)
			if err != nil {
				return err
			}
			if !updated.Changed() {
				console.Info("config unchanged (--if-unset and values already set)")
				return nil
			}

			console.Info("config updated")
			if updated.BkmsBaseURLUpdated {
				console.Info("  bkmsBaseUrl: %s", config.G.BkmsBaseURL)
			}
			if updated.BcsAPIHostUpdated {
				console.Info("  bcsApiHost: %s", config.G.BcsAPIHost)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&bkmsBaseURL, "bkms-base-url", "", "bkms service base URL")
	cmd.Flags().StringVar(&bcsAPIHost, "bcs-api-host", "", "BCS API gateway host")
	cmd.Flags().BoolVar(&ifUnset, "if-unset", false, "only set fields that are currently empty")
	return cmd
}
