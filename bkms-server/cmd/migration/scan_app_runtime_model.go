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

package migration

import (
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// NewScanAppRuntimeModelCmd 只读扫描 Application / AppModel 技术栈冲突。
//
// 输出计数与脱敏样本，不打印配置正文或凭据。
func NewScanAppRuntimeModelCmd() *cobra.Command {
	var srvCfg string
	sampleLimit := 20

	cmd := &cobra.Command{
		Use:   "scan_app_runtime_model",
		Short: "Dry-run scan of Application/AppModel tech-stack conflicts",
		Run: func(cmd *cobra.Command, _ []string) {
			ctx := auth.WithMaintenanceUser(cmd.Context())
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				log.Fatalf("failed to load config: %s", err)
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				log.Fatalf("init logger: %v", err)
			}
			database.InitClient(ctx, cfg.Mongo)

			summary, err := ScanAppRuntimeModel(ctx, sampleLimit)
			if err != nil {
				log.Fatalf("scan app runtime model failed: %v", err)
			}
			writeScanAppRuntimeModelOutput(cmd.OutOrStdout(), summary)
		},
	}
	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().IntVar(&sampleLimit, "sample-limit", 20, "number of conflict samples to print")
	_ = cmd.MarkFlagRequired("srvCfg")
	return cmd
}

func writeScanAppRuntimeModelOutput(w io.Writer, summary appruntime.InventorySummary) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(summary)
}

// ScanAppRuntimeModel reads applications and app models and returns a desensitized report.
func ScanAppRuntimeModel(ctx context.Context, sampleLimit int) (appruntime.InventorySummary, error) {
	appColl := database.Client().Database(database.Name()).Collection("applications")
	modelColl := database.Client().Database(database.Name()).Collection(appmodel.CollectionName)

	appCursor, err := appColl.Find(ctx, bson.M{})
	if err != nil {
		return appruntime.InventorySummary{}, errors.Wrap(err, "list applications")
	}
	defer appCursor.Close(ctx)

	var apps []*bkmsapp.Application
	if err = appCursor.All(ctx, &apps); err != nil {
		return appruntime.InventorySummary{}, errors.Wrap(err, "decode applications")
	}

	modelCursor, err := modelColl.Find(ctx, bson.M{})
	if err != nil {
		return appruntime.InventorySummary{}, errors.Wrap(err, "list app models")
	}
	defer modelCursor.Close(ctx)

	var models []*appmodel.AppModel
	if err = modelCursor.All(ctx, &models); err != nil {
		return appruntime.InventorySummary{}, errors.Wrap(err, "decode app models")
	}
	byAppID := make(map[string]*appmodel.AppModel, len(models))
	for _, m := range models {
		byAppID[m.AppID] = m
	}

	snapshots := make([]appruntime.Snapshot, 0, len(apps))
	reports := make([]appruntime.Report, 0, len(apps))
	for _, app := range apps {
		snap := app.RuntimeSnapshot()
		if model, ok := byAppID[app.ID]; ok {
			model.Workload.FillSnapshot(&snap)
		} else if appruntime.IsAppModelType(app.Type) {
			snap.AppModelMissing = true
		}
		snapshots = append(snapshots, snap)
		reports = append(reports, appruntime.Inspect(snap))
	}
	return appruntime.Summarize(reports, snapshots, sampleLimit), nil
}
