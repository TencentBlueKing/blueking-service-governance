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
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/redis"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// backfillImageBuiltAtTolerance 比对镜像构建时间与构建起止时间时的容差，吸收构建机与平台之间的时钟偏差
const backfillImageBuiltAtTolerance = 5 * time.Minute

type backfillBuildImageDigestOptions struct {
	WorkspaceID string
	AppIDs      []string
	Execute     bool
}

// backfillBuildImageDigestItem 应用制品仓库中一个镜像 tag 的补刷结果
type backfillBuildImageDigestItem struct {
	Tag        string
	Digest     string
	BuildID    string
	BuildNum   int64
	Written    bool
	SkipReason string
}

type backfillBuildImageDigestSummary struct {
	AppID    string
	AppName  string
	RepoName string
	Items    []backfillBuildImageDigestItem
}

// NewBackfillBuildImageDigestCmd 创建为指定应用的存量构建记录补刷产物镜像 digest 的一次性命令，
// 补刷后制品列表即可关联到这些构建的源材料
func NewBackfillBuildImageDigestCmd() *cobra.Command {
	var srvCfg string
	opts := backfillBuildImageDigestOptions{}

	cmd := &cobra.Command{
		Use:   "backfill_build_image_digest",
		Short: "Preview or backfill image digests of existing build records for specified apps in one workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := auth.WithMaintenanceUser(cmd.Context())
			cfg, err := config.Load(ctx, srvCfg)
			if err != nil {
				return errors.Wrap(err, "load config")
			}
			if err = log.InitDefaultLogger(cfg.Logging); err != nil {
				return errors.Wrap(err, "init logger")
			}

			database.InitClient(ctx, cfg.Mongo)
			redis.InitClient(ctx, cfg.Redis)
			storereg.Init(ctx)

			summaries, err := runBackfillBuildImageDigest(ctx, storereg.G().DB, opts)
			if err != nil {
				return err
			}
			writeBackfillBuildImageDigestOutput(cmd.OutOrStdout(), summaries, opts.Execute)
			return nil
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&opts.WorkspaceID, "workspace-id", "", "target workspace ID")
	cmd.Flags().StringSliceVar(&opts.AppIDs, "app-ids", nil, "target app IDs in the workspace, comma separated")
	cmd.Flags().BoolVar(&opts.Execute, "execute", false, "actually write image digests; default is preview only")
	_ = cmd.MarkFlagRequired("srvCfg")
	_ = cmd.MarkFlagRequired("workspace-id")
	_ = cmd.MarkFlagRequired("app-ids")

	return cmd
}

func runBackfillBuildImageDigest(
	ctx context.Context, db *mongo.Database, opts backfillBuildImageDigestOptions,
) ([]*backfillBuildImageDigestSummary, error) {
	if opts.WorkspaceID == "" {
		return nil, errors.New("workspace-id is required")
	}
	appIDs := lo.Uniq(lo.Compact(opts.AppIDs))
	if len(appIDs) == 0 {
		return nil, errors.New("app-ids is required")
	}

	// 先校验全部应用的归属，避免写到一半才发现参数有误
	apps := make([]*bkmsapp.Application, 0, len(appIDs))
	for _, appID := range appIDs {
		app, err := store.Of(db, bkmsapp.AppMeta).Get(ctx, bkmsapp.ByID(appID))
		if err != nil {
			return nil, errors.Wrapf(err, "get app %s", appID)
		}
		if app.WorkspaceID != opts.WorkspaceID {
			return nil, errors.Errorf("app %s does not belong to workspace %s", app.ID, opts.WorkspaceID)
		}
		apps = append(apps, app)
	}

	summaries := make([]*backfillBuildImageDigestSummary, 0, len(apps))
	for _, app := range apps {
		summary, err := backfillAppBuildImageDigest(ctx, db, app, opts.Execute)
		if err != nil {
			return nil, errors.Wrapf(err, "backfill build image digest of app %s", app.ID)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// backfillAppBuildImageDigest 以应用制品仓库当前的镜像快照为准，为产出各镜像的构建记录补写 digest。
// 只处理每个 tag 最近结束的成功构建：更早构建的产物已被同名 tag 覆盖，不在仓库中
func backfillAppBuildImageDigest(
	ctx context.Context, db *mongo.Database, app *bkmsapp.Application, execute bool,
) (*backfillBuildImageDigestSummary, error) {
	info, err := snapshot.NewService(db).ResolveRepoKeyForApp(ctx, app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "resolve app image repo")
	}
	images, err := store.Of(db, snapshot.ImageMeta).List(ctx, snapshot.InRepo(info.RepoKey), store.Asc("tag"))
	if err != nil {
		return nil, errors.Wrap(err, "list image snapshots")
	}

	artifactOf := func(tag string) string { return info.RepoName + ":" + tag }
	records := store.Of(db, build.RecordMeta)
	latest, err := records.ListLatestBy(
		ctx, []string{"artifact"},
		build.ByApp(app.ID),
		store.Where("status", build.StatusSuccess),
		store.Where("artifact", bson.M{"$in": lo.Map(images, func(img snapshot.Image, _ int) string {
			return artifactOf(img.Tag)
		})}),
		store.Desc("endedAt"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "list latest succeeded build records by artifact")
	}
	latestByArtifact := lo.KeyBy(latest, func(r build.Record) string { return r.Artifact })

	summary := &backfillBuildImageDigestSummary{AppID: app.ID, AppName: app.Name, RepoName: info.RepoName}
	for _, img := range images {
		item := backfillBuildImageDigestItem{Tag: img.Tag, Digest: img.Digest}
		record, found := latestByArtifact[artifactOf(img.Tag)]
		if found {
			item.BuildID, item.BuildNum = record.BuildID, record.Num
		}
		item.SkipReason = backfillImageDigestSkipReason(img, record, found)
		if item.SkipReason == "" && execute {
			err = build.UpdateRecordImageDigest(ctx, records, app.ID, record.BuildID, img.Digest)
			switch {
			case errors.Is(err, build.ErrRecordNotFound):
				item.SkipReason = "build digest recorded concurrently"
			case err != nil:
				return nil, errors.Wrapf(err, "write image digest of tag %s", img.Tag)
			default:
				item.Written = true
			}
		}
		summary.Items = append(summary.Items, item)
	}
	return summary, nil
}

// backfillImageDigestSkipReason 返回不补写该镜像的原因，可补写时返回空字符串
func backfillImageDigestSkipReason(img snapshot.Image, record build.Record, found bool) string {
	switch {
	case !found:
		return "no succeeded build of this tag"
	case record.ImageDigest != "":
		return "build digest already recorded"
	case img.Digest == "" || img.DetailSyncPending:
		return "image detail not synced"
	case img.BuiltAt == nil || img.BuiltAt.IsZero():
		return "image built time unknown"
	}
	// 镜像构建时间落在构建起止时间内，才认为它由这次构建产出；tag 被外部覆盖或复用了旧镜像时都不满足
	from := record.StartedAt.Add(-backfillImageBuiltAtTolerance)
	to := record.EndedAt.Add(backfillImageBuiltAtTolerance)
	if img.BuiltAt.Before(from) || img.BuiltAt.After(to) {
		return fmt.Sprintf(
			"image built at %s, outside build time %s ~ %s",
			img.BuiltAt.Format(time.RFC3339),
			record.StartedAt.Format(time.RFC3339),
			record.EndedAt.Format(time.RFC3339),
		)
	}
	return ""
}

func writeBackfillBuildImageDigestOutput(w io.Writer, summaries []*backfillBuildImageDigestSummary, execute bool) {
	pendingStatus := lo.Ternary(execute, "written", "to-write (preview)")
	for _, summary := range summaries {
		_, _ = fmt.Fprintf(w, "app: %s (%s)\n", summary.AppID, summary.AppName)
		_, _ = fmt.Fprintf(w, "repo: %s\n", summary.RepoName)
		matched := 0
		for _, item := range summary.Items {
			status := pendingStatus
			if item.SkipReason != "" {
				status = "skipped (" + item.SkipReason + ")"
			} else {
				matched++
			}
			_, _ = fmt.Fprintf(
				w, "  tag: %s, build: #%d %s, digest: %s, status: %s\n",
				item.Tag, item.BuildNum, item.BuildID, item.Digest, status,
			)
		}
		_, _ = fmt.Fprintf(w, "images: %d, %s: %d\n\n", len(summary.Items), pendingStatus, matched)
	}
}
