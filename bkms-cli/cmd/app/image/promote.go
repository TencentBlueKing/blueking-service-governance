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

package image

import (
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	imagehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/clierr"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewPromoteCmd returns a Command instance for 'app image promote' sub command
func NewPromoteCmd() *cobra.Command {
	var appID string
	var tags []string

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote application image tags for production deployment",
		Long: `Promote one or more image tags of an application.

Only promoted image tags can be deployed to production environments.
Promotion cannot be revoked. The tag must exist in the application image list;
if an image was just pushed to the registry, run 'app image refresh' first.

When multiple tags are given, each tag is promoted independently: a failed tag
does not stop the remaining ones, and the command exits non-zero if any tag fails.`,
		Example: `  # Promote a single image tag
  bkms-cli app image promote --app demo --tag v1.0.0

  # Promote multiple image tags
  bkms-cli app image promote --app demo --tag v1.0.0,v1.0.1
  bkms-cli app image promote --app demo --tag v1.0.0 --tag v1.0.1`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// 先做本地参数校验，避免无效输入还要先请求服务端解析应用
			tags = imagehandler.NormalizeTags(tags)
			if len(tags) == 0 {
				return clierr.Usagef("--tag must contain at least one non-empty image tag")
			}
			return cmdutil.ResolveAppPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPromote(cmd, appID, tags)
		},
	}

	cmdutil.AddAppFlags(cmd, &appID)
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "image tags to promote, comma-separated or repeated (required)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("tag")

	return cmd
}

func runPromote(cmd *cobra.Command, appID string, tags []string) error {
	results := imagehandler.PromoteImages(cmd.Context(), client.New(), appID, tags)
	for _, r := range results {
		if r.Err != nil {
			console.Error("✗ %s: %v", r.Tag, r.Err)
			continue
		}
		console.Info("✓ Promoted image tag: %s", r.Tag)
	}

	failed := imagehandler.FailedPromoteResults(results)
	if len(failed) > 0 {
		return clierr.Reportedf("failed to promote %d of %d image tag(s)", len(failed), len(results))
	}
	return nil
}
