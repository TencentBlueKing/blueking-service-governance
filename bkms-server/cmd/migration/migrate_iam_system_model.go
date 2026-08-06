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

// Package migration migrate_iam_system_model 命令用于将 bkms IAM 系统模型
// （资源类型、操作、实例选择等）一次性注册到蓝鲸权限中心（IAM）。
package migration

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/migrate"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	// iamSchemaMigrationsCollection 是 golang-migrate 用来记录 IAM 系统模型
	// 迁移版本号的集合名。与 bkms 业务集合无关，只是迁移历史簿记。
	iamSchemaMigrationsCollection = "iam_schema_migrations"
)

// NewMigrateIAMSystemModelCmd 将 bkms IAM 系统模型（资源类型、操作、实例选择等）
// 一次性注册到蓝鲸权限中心。
func NewMigrateIAMSystemModelCmd() *cobra.Command {
	var srvCfg, bkmsHost string

	cmd := &cobra.Command{
		Use:   "migrate_iam_system_model",
		Short: "Register bkms IAM system model (resource types, actions, etc.) to BlueKing IAM",
		Run: func(cmd *cobra.Command, _ []string) {
			ctx := cmd.Context()
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("load config: %v", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}

			database.InitClient(ctx, cfg.Mongo)

			iamGatewayURL, err := iam.BuildIAMGatewayURL(cfg.BkPlatUrls.BkApiUrlTmpl, cfg.BkApiStages.BkIAM)
			if err != nil {
				log.Fatalf("%v", errors.Wrap(err, "build iam gateway url"))
			}

			mongoCfg := migrate.MongoConfig{
				User:       cfg.Mongo.Username,
				Password:   cfg.Mongo.Password,
				Host:       cfg.Mongo.Host,
				Port:       cfg.Mongo.Port,
				Database:   cfg.Mongo.Database,
				Collection: iamSchemaMigrationsCollection,
			}

			iamCfg := migrate.Config{
				BkApiGatewayURL: iamGatewayURL,
				BkmsSystemID:    cfg.BkIAMSystemIDs.Bkms,
				AppCode:         cfg.BkApp.Code,
				AppSecret:       cfg.BkApp.Secret,
				BkmsHost:        bkmsHost,
			}

			if err = migrate.Migrate(mongoCfg, iamCfg); err != nil {
				log.Fatalf("%v", errors.Wrap(err, "migrate iam system model"))
			}
			log.Info(ctx, "migrate iam system model completed successfully")
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&bkmsHost, "bkms-host", "",
		"externally reachable bkms-server URL rendered into IAM system provider_config.host")
	_ = cmd.MarkFlagRequired("srvCfg")
	_ = cmd.MarkFlagRequired("bkms-host")
	return cmd
}
