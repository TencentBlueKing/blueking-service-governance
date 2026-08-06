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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	pfwhitelist "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/portforward"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

// NewPortForwardWhitelistCmd 创建 portforward-whitelist 父命令，用于管理 port-forward 白名单。
func NewPortForwardWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portforward-whitelist",
		Short: "Manage port-forward whitelist (add/rm/list environment IDs)",
	}

	cmd.AddCommand(newPFWhitelistAddCmd())
	cmd.AddCommand(newPFWhitelistRmCmd())
	cmd.AddCommand(newPFWhitelistListCmd())

	return cmd
}

func newPFWhitelistAddCmd() *cobra.Command {
	var srvCfg string

	cmd := &cobra.Command{
		Use:   "add <envID> [envID...]",
		Short: "Add environment IDs to port-forward whitelist",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}
			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)
			reg := storereg.G()

			svc := pfwhitelist.NewService(reg.PortForwardWhitelistStore, reg.EnvStore)
			if err = svc.Add(ctx, args); err != nil {
				log.Fatalf("add to port-forward whitelist failed: %v", err)
			}
			log.Info(ctx, "successfully added environment IDs to port-forward whitelist")
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func newPFWhitelistRmCmd() *cobra.Command {
	var srvCfg string

	cmd := &cobra.Command{
		Use:   "rm <envID> [envID...]",
		Short: "Remove environment IDs from port-forward whitelist",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}
			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)
			reg := storereg.G()

			svc := pfwhitelist.NewService(reg.PortForwardWhitelistStore, reg.EnvStore)
			if err = svc.Remove(ctx, args); err != nil {
				log.Fatalf("remove from port-forward whitelist failed: %v", err)
			}
			log.Info(ctx, "successfully removed environment IDs from port-forward whitelist")
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func newPFWhitelistListCmd() *cobra.Command {
	var srvCfg string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all environment IDs in port-forward whitelist",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}
			database.InitClient(ctx, cfg.Mongo)
			storereg.Init(ctx)
			reg := storereg.G()

			svc := pfwhitelist.NewService(reg.PortForwardWhitelistStore, reg.EnvStore)
			envIDs, err := svc.List(ctx)
			if err != nil {
				log.Fatalf("list port-forward whitelist failed: %v", err)
			}

			if len(envIDs) == 0 {
				fmt.Println("(empty)")
				return
			}
			for _, id := range envIDs {
				fmt.Println(id)
			}
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func init() {
	rootCmd.AddCommand(NewPortForwardWhitelistCmd())
}
