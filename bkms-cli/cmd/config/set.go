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
	"strings"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewCmdSet updates API and distribution endpoints in the local config file.
func NewCmdSet() *cobra.Command {
	var bkmsBaseURL string
	var updateSource config.UpdateSource
	var ifUnset bool

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set bkms-cli API and update endpoints",
		Long: `Set the service URL or an update source in the local config file.
The update latest URL and download URL template must be supplied together.
With --if-unset, existing settings are preserved; update URLs are treated as a pair.`,
		Example: `  bkms-cli config set --bkms-base-url https://bkms.example.com
  bkms-cli config set --if-unset --bkms-base-url https://bkms.example.com
  bkms-cli config set \
    --update-latest-url https://artifacts.example.com/bkms-cli/latest.txt \
    --update-download-url-template 'https://artifacts.example.com/bkms-cli/v{version}/{archive}'`,
		DisableFlagsInUseLine: true,
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			hasBaseURL := cmd.Flags().Changed("bkms-base-url")
			hasUpdate := cmd.Flags().Changed("update-latest-url")
			if !hasBaseURL && !hasUpdate {
				return clierr.Usagef("provide --bkms-base-url or both update URL options")
			}
			if hasBaseURL && strings.TrimSpace(bkmsBaseURL) == "" {
				return clierr.Usagef("--bkms-base-url must not be empty")
			}
			if hasUpdate {
				updateSource.LatestVersionURL = strings.TrimSpace(updateSource.LatestVersionURL)
				updateSource.DownloadURLTemplate = strings.TrimSpace(updateSource.DownloadURLTemplate)
				if err := updateSource.Validate(); err != nil {
					return clierr.Usage(err)
				}
			}

			updated, err := config.G.SetEndpoints(bkmsBaseURL, updateSource, ifUnset)
			if err != nil {
				return err
			}
			if !updated {
				console.Info("config unchanged")
				return nil
			}

			console.Info("config updated")
			if hasBaseURL {
				console.Info("  bkmsBaseUrl: %s", config.G.BkmsBaseURL)
			}
			if hasUpdate {
				console.Info("%s", strings.TrimSuffix(config.G.Update.String(), "\n"))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&bkmsBaseURL, "bkms-base-url", "", "bkms service base URL")
	cmd.Flags().
		StringVar(&updateSource.LatestVersionURL, "update-latest-url", "", "URL of the latest stable version file")
	cmd.Flags().StringVar(&updateSource.DownloadURLTemplate, "update-download-url-template", "",
		"download URL template using {version} and {archive}")
	cmd.MarkFlagsRequiredTogether("update-latest-url", "update-download-url-template")
	cmd.Flags().BoolVar(&ifUnset, "if-unset", false, "only set when currently empty")
	return cmd
}
