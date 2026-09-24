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
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bscpapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// NewListBscpProjectsCmd 创建 list-bscp-projects 命令，列出 workspace 对应 bizID 下的 BSCP 项目。
func NewListBscpProjectsCmd() *cobra.Command {
	var srvCfg string
	var workspaceID string
	var operator string

	cmd := &cobra.Command{
		Use:   "list-bscp-projects",
		Short: "List BSCP projects under a workspace",
		Long:  "List BSCP projects under the business of a workspace, to confirm the target project to bind.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListBscpProjects(cmd.Context(), srvCfg, workspaceID, operator)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")
	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace ID")
	_ = cmd.MarkFlagRequired("workspace")
	cmd.Flags().StringVar(&operator, "operator", "", "operator username (bk_username)")
	_ = cmd.MarkFlagRequired("operator")

	return cmd
}

// runListBscpProjects 从 workspace 取 bizID，再列出该 bizID 下的 BSCP 项目。
func runListBscpProjects(ctx context.Context, srvCfg, workspaceID, operator string) error {
	cfg, err := config.Load(ctx, srvCfg)
	if err != nil {
		return errors.Wrap(err, "load config")
	}

	// 交互式读取 access token（密文输入，不回显），避免经命令行参数明文传递
	accessToken, err := readAccessToken()
	if err != nil {
		return err
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
	bizID := ws.BkSystems.BkCCBizID
	if bizID == "" {
		return errors.Errorf("workspace %s missing BkCCBizID", workspaceID)
	}

	client, err := bscpapi.NewConfigClient(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "create bscp config client")
	}

	projects, err := client.ListProjects(ctx, bizID)
	if err != nil {
		return errors.Wrapf(err, "list projects for biz %s", bizID)
	}

	log.Infof(ctx, "found %d project(s) under workspace %s (biz %s)", len(projects), workspaceID, bizID)
	for _, p := range projects {
		log.Infof(ctx, "project: ID=%d, Key=%s, Name=%s, IsDefault=%v, EnvCount=%d, AppCount=%d",
			p.ID, p.Spec.Key, p.Spec.Name, p.Spec.IsDefault, p.Spec.EnvCount, p.Spec.AppCount)
	}

	return nil
}

// readAccessToken 交互式读取
func readAccessToken() (string, error) {
	_, _ = fmt.Fprint(os.Stderr, "Enter access token: ")
	token, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", errors.Wrap(err, "read access token")
	}

	tokenStr := strings.TrimSpace(string(token))
	if tokenStr == "" {
		return "", errors.New("access token must not be empty")
	}
	return tokenStr, nil
}
