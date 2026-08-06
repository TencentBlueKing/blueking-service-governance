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

// Package root defines the root command and global behavior of bkms-cli.
package root

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/auth"
	cfgCmd "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/version"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/cmd/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/logx"
)

// NewRootCmd ...
func NewRootCmd() *cobra.Command {
	var logLevel string

	rootCmd := &cobra.Command{
		Use:   "bkms-cli",
		Short: "bkms cli",
		Long:  "Work seamlessly with bkms from the command line.",
		Run: func(cmd *cobra.Command, args []string) {
			if !config.G.UserIsInitialized() {
				console.Info(
					"Welcome to bkms-cli!\n\n" +
						"No initialized user info found, run `bkms-cli login` to get started.",
				)
			} else {
				console.Info("Hello %s, welcome to use bkms-cli, use `bkms-cli -h` for help", config.G.Username)
			}
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 加载全局配置（若配置文件不存在，会自动创建默认配置）
			if _, err := config.G.Load(); err != nil {
				return errors.Wrapf(err, "load config")
			}
			if err := logx.SetLevel(logLevel); err != nil {
				return errors.Wrap(err, "set log level")
			}
			// 如果某命令不需要用户认证，直接返回
			if !cmdutil.IsAuthRequired(cmd) {
				return nil
			}
			// 用户认证
			user, err := client.New().ValidateAccessToken(config.G.AccessToken)
			if err != nil {
				return errors.Wrapf(err, "user unauthorized, please use `bkms-cli login` to login")
			}
			config.G.Username = user
			return nil
		},
	}

	// 用户登录
	rootCmd.AddCommand(auth.NewLoginCmd())
	// 用户登出
	rootCmd.AddCommand(auth.NewLogoutCmd())
	// 配置管理
	rootCmd.AddCommand(cfgCmd.NewCmd())
	// 版本信息
	rootCmd.AddCommand(version.NewCmd())

	// 工作空间相关命令
	rootCmd.AddCommand(workspace.NewCmd())
	// 环境相关命令
	rootCmd.AddCommand(env.NewCmd())
	// 应用相关命令
	rootCmd.AddCommand(app.NewCmd())

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")

	return rootCmd
}

// ExecuteContext bkms-cli command with context
func ExecuteContext(ctx context.Context) {
	if err := NewRootCmd().ExecuteContext(ctx); err != nil {
		console.Error(err.Error())
		os.Exit(1)
	}
}
