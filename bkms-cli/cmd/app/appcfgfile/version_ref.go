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

package appcfgfile

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/appcfgfile"
)

func registerVersionRefFlags(cmd *cobra.Command, version *int64, versionID *string) {
	cmd.Flags().Int64Var(version, "version", 0, "history version number")
	cmd.Flags().StringVar(versionID, "version-id", "", "history version record ID")
}

func parseVersionRefOptions(
	cmd *cobra.Command,
	version int64,
	versionID string,
) (handler.VersionRefOptions, error) {
	versionChanged := cmd.Flags().Changed("version")
	versionIDChanged := cmd.Flags().Changed("version-id")
	if versionChanged == versionIDChanged {
		return handler.VersionRefOptions{}, errors.New("exactly one of --version or --version-id is required")
	}

	opts := handler.VersionRefOptions{
		VersionID: versionID,
	}
	if versionChanged {
		opts.Version = &version
	}
	return opts, nil
}
