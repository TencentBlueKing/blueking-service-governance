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

// Package migration 提供一次性数据迁移与运维命令。
package migration

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// NewBindBscpProjectCmd 创建 bind-bscp-project 命令，按 projectKey 将 workspace 绑定到 BSCP 项目。
func NewBindBscpProjectCmd() *cobra.Command {
	var srvCfg string
	var workspaceID string
	var projectKey string
	var operator string
	var accessToken string
	var execute bool

	cmd := &cobra.Command{
		Use:   "bind-bscp-project",
		Short: "Bind a workspace to a BSCP project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBindBscpProject(cmd.Context(), srvCfg, workspaceID, projectKey, operator, accessToken, execute)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")
	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	_ = cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&projectKey, "projectKey", "", "BSCP project key")
	_ = cmd.MarkFlagRequired("projectKey")
	cmd.Flags().StringVar(&operator, "operator", "", "operator username (bk_username)")
	_ = cmd.MarkFlagRequired("operator")
	cmd.Flags().StringVar(&accessToken, "access-token", "", "operator access token")
	_ = cmd.MarkFlagRequired("access-token")
	cmd.Flags().BoolVar(&execute, "execute", false, "actually execute (default is dry-run)")

	return cmd
}

// runBindBscpProject 查询 BSCP 项目并写回 workspace 的 BkBSCPProjectID / BkBSCPProjectKey。
func runBindBscpProject(
	ctx context.Context,
	srvCfg, workspaceID, projectKey, operator, accessToken string,
	execute bool,
) error {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return errors.Wrap(err, "load config")
	}
	ctx = auth.WithUser(ctx, auth.User{
		ID:   operator,
		Cred: auth.UserCredential{AccessToken: accessToken},
	})
	if err = log.InitDefaultLogger(cfg.Logging); err != nil {
		return errors.Wrap(err, "init logger")
	}

	database.InitClient(ctx, cfg.Mongo)
	storereg.Init(ctx)
	reg := storereg.G()

	ws, err := reg.WorkspaceStore.Get(ctx, workspaceID)
	if err != nil {
		return errors.Wrapf(err, "get workspace %s", workspaceID)
	}
	if ws.BkSystems.BkCCBizID == "" {
		return errors.Errorf("workspace %s missing BkCCBizID", workspaceID)
	}

	client, err := bscpapi.NewConfigClient(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "create bscp config client")
	}

	project, err := client.GetProjectByKey(ctx, ws.BkSystems.BkCCBizID, projectKey)
	if err != nil {
		return errors.Wrapf(err, "get bscp project by key %s", projectKey)
	}

	projectIDStr := cast.ToString(project.ID)
	if !execute {
		log.Infof(ctx, "[DRY-RUN] would bind workspace %s to BSCP project %s (key %s)",
			workspaceID, projectIDStr, project.Spec.Key)
		return nil
	}

	ws.BkSystems.BkBSCPProjectID = strings.TrimSpace(projectIDStr)
	ws.BkSystems.BkBSCPProjectKey = strings.TrimSpace(project.Spec.Key)
	if err = reg.WorkspaceStore.Update(ctx, ws); err != nil {
		return errors.Wrap(err, "update workspace bk systems")
	}

	log.Infof(ctx, "workspace %s bound to BSCP project %s (key %s)", workspaceID, projectIDStr, project.Spec.Key)
	return nil
}
