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

// Package migration load_builtin_component 命令用于从指定目录加载内置组件定义到数据库
package migration

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// NewLoadBuiltinComponentCmd 从指定目录加载内置组件定义到数据库
func NewLoadBuiltinComponentCmd() *cobra.Command {
	var srvCfg, folderPath string

	cmd := &cobra.Command{
		Use:   "load_builtin_component",
		Short: "Load builtin component definitions from folder to database",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			// 加载配置
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}

			// 初始化数据库客户端
			database.InitClient(ctx, cfg.Mongo)

			// 执行加载内置组件操作
			if err = runLoadBuiltinComponent(ctx, folderPath); err != nil {
				log.Fatalf("load builtin component failed: %v", err)
			}
			log.Info(ctx, "load builtin component completed successfully")
		},
	}

	// 配置文件路径
	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&folderPath, "folderPath", "", "folder path containing component definition files")
	_ = cmd.MarkFlagRequired("srvCfg")
	_ = cmd.MarkFlagRequired("folderPath")
	return cmd
}

// runLoadBuiltinComponent 执行加载内置组件操作
func runLoadBuiltinComponent(ctx context.Context, folderPath string) error {
	// 初始化 ComponentDefStore
	store, err := component.NewComponentDefStoreMongo(database.Client(), database.Name())
	if err != nil {
		return err
	}

	// 从指定目录加载组件定义
	return component.LoadBuiltinFromFolder(ctx, store, folderPath)
}
