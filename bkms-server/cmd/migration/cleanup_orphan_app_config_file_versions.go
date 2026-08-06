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

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	appConfigFilesCollectionName        = "app_config_files"
	appConfigFileVersionsCollectionName = "app_config_file_versions"
	defaultSampleLimit                  = 20
	defaultDeleteBatchSize              = 500
)

// orphanAppConfigFileVersionSample 仅用于命令行预览输出，帮助执行人确认待处理数据是否符合预期。
type orphanAppConfigFileVersionSample struct {
	ID              bson.ObjectID `bson:"_id"`
	AppID           string        `bson:"appID"`
	AppConfigFileID bson.ObjectID `bson:"appConfigFileID"`
	EnvName         string        `bson:"envName"`
	Name            string        `bson:"name"`
	Version         int64         `bson:"version"`
	Creator         string        `bson:"creator"`
	CreatedAt       any           `bson:"createdAt"`
	IsDeleted       bool          `bson:"isDeleted"`
}

// orphanVersionID 用于流式读取待删除版本记录的主键，避免一次性将全量文档加载到内存。
type orphanVersionID struct {
	ID bson.ObjectID `bson:"_id"`
}

// cleanupOrphanAppConfigFileVersionsOptions 描述本次治理任务的执行参数。
type cleanupOrphanAppConfigFileVersionsOptions struct {
	AppID       string
	Execute     bool
	SampleLimit int64
	BatchSize   int
}

// cleanupOrphanAppConfigFileVersionsSummary 汇总预览和执行结果，便于统一输出。
type cleanupOrphanAppConfigFileVersionsSummary struct {
	OrphanCount  int64
	Samples      []orphanAppConfigFileVersionSample
	DeletedCount int64
}

// NewCleanupOrphanAppConfigFileVersionsCmd 创建一个一次性 Mongo 数据治理命令。
//
// 背景：
//   - 历史上删除 app_config_files 记录时，可能未同步清理 app_config_file_versions
//   - 从而留下 appConfigFileID 已失效的“孤儿版本记录”
//   - 这些脏数据会污染按环境维度查看的版本列表，甚至导致“当前版本”展示错乱
//
// 处理策略：
//   - 默认仅预览：输出总量与样本数据，不执行删除
//   - 显式传入 --execute 后，按批次删除所有孤儿版本记录
//   - 可通过 --appID 仅清理某个应用，便于灰度或定向治理
//
// 示例：
//
//	bkms-server cleanup_orphan_app_config_file_versions --srvCfg ./biz.yaml
//	bkms-server cleanup_orphan_app_config_file_versions --srvCfg ./biz.yaml --execute
//	bkms-server cleanup_orphan_app_config_file_versions --srvCfg ./biz.yaml --appID app-xxx --execute
func NewCleanupOrphanAppConfigFileVersionsCmd() *cobra.Command {
	var srvCfg string
	opts := cleanupOrphanAppConfigFileVersionsOptions{
		SampleLimit: defaultSampleLimit,
		BatchSize:   defaultDeleteBatchSize,
	}

	cmd := &cobra.Command{
		Use:   "cleanup_orphan_app_config_file_versions",
		Short: "预览或清理 app config file orphan version records",
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

			summary, err := runCleanupOrphanAppConfigFileVersions(ctx, opts)
			if err != nil {
				log.Fatalf("cleanup orphan app config file versions failed: %v", err)
			}
			writeCleanupOrphanAppConfigFileVersionsOutput(cmd.OutOrStdout(), summary, opts.Execute)
		},
	}

	cmd.Flags().StringVar(&srvCfg, "srvCfg", "", "server config file")
	cmd.Flags().StringVar(&opts.AppID, "appID", "", "only process a specific app")
	cmd.Flags().BoolVar(&opts.Execute, "execute", false, "actually delete records; default is preview only")
	cmd.Flags().Int64Var(&opts.SampleLimit, "sample-limit", defaultSampleLimit, "number of sample documents to print")
	cmd.Flags().IntVar(&opts.BatchSize, "batch-size", defaultDeleteBatchSize, "delete batch size when execute=true")
	_ = cmd.MarkFlagRequired("srvCfg")

	return cmd
}

func writeCleanupOrphanAppConfigFileVersionsOutput(
	w io.Writer,
	summary *cleanupOrphanAppConfigFileVersionsSummary,
	execute bool,
) {
	_, _ = fmt.Fprintf(w, "orphan version count: %d\n", summary.OrphanCount)
	if !execute {
		if len(summary.Samples) == 0 {
			_, _ = fmt.Fprintln(w, "sample: (empty)")
		} else {
			_, _ = fmt.Fprintln(w, "sample documents:")
			for idx, item := range summary.Samples {
				line, err := bson.MarshalExtJSON(item, false, false)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "%d. %s\n", idx+1, line)
			}
		}
		_, _ = fmt.Fprintln(w, "preview only. rerun with --execute to delete.")
		return
	}

	_, _ = fmt.Fprintf(w, "deleted orphan version count: %d\n", summary.DeletedCount)
}

// runCleanupOrphanAppConfigFileVersions 执行一次完整的“预览 / 删除”流程。
//
// 执行顺序：
//  1. 统计孤儿版本数量
//  2. 拉取少量样本，供操作者确认
//  3. 若 execute=true，则分批执行物理删除
func runCleanupOrphanAppConfigFileVersions(
	ctx context.Context,
	opts cleanupOrphanAppConfigFileVersionsOptions,
) (*cleanupOrphanAppConfigFileVersionsSummary, error) {
	if opts.SampleLimit <= 0 {
		opts.SampleLimit = defaultSampleLimit
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultDeleteBatchSize
	}

	coll := database.Client().Database(database.Name()).Collection(appConfigFileVersionsCollectionName)
	count, err := countOrphanAppConfigFileVersions(ctx, coll, opts.AppID)
	if err != nil {
		return nil, err
	}
	samples, err := sampleOrphanAppConfigFileVersions(ctx, coll, opts.AppID, opts.SampleLimit)
	if err != nil {
		return nil, err
	}

	summary := &cleanupOrphanAppConfigFileVersionsSummary{
		OrphanCount: count,
		Samples:     samples,
	}
	if !opts.Execute || count == 0 {
		return summary, nil
	}

	deletedCount, err := deleteOrphanAppConfigFileVersions(ctx, coll, opts.AppID, opts.BatchSize)
	if err != nil {
		return nil, err
	}
	summary.DeletedCount = deletedCount
	return summary, nil
}

// orphanAppConfigFileVersionPipeline 构造识别孤儿版本记录的公共聚合管道。
//
// “孤儿”的定义：
//   - app_config_file_versions.appConfigFileID 在 app_config_files._id 中已不存在
func orphanAppConfigFileVersionPipeline(appID string) mongo.Pipeline {
	pipeline := mongo.Pipeline{}
	if appID != "" {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "appID", Value: appID}}}})
	}

	return append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: appConfigFilesCollectionName},
			{Key: "localField", Value: "appConfigFileID"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "matchedFiles"},
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "matchedFiles", Value: bson.D{{Key: "$eq", Value: bson.A{}}}},
		}}},
	)
}

func countOrphanAppConfigFileVersions(ctx context.Context, coll *mongo.Collection, appID string) (int64, error) {
	pipeline := append(orphanAppConfigFileVersionPipeline(appID), bson.D{{Key: "$count", Value: "count"}})
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var ret []struct {
		Count int64 `bson:"count"`
	}
	if err = cursor.All(ctx, &ret); err != nil {
		return 0, err
	}
	if len(ret) == 0 {
		return 0, nil
	}
	return ret[0].Count, nil
}

func sampleOrphanAppConfigFileVersions(
	ctx context.Context,
	coll *mongo.Collection,
	appID string,
	sampleLimit int64,
) ([]orphanAppConfigFileVersionSample, error) {
	pipeline := append(orphanAppConfigFileVersionPipeline(appID),
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 1},
			{Key: "appID", Value: 1},
			{Key: "appConfigFileID", Value: 1},
			{Key: "envName", Value: 1},
			{Key: "name", Value: 1},
			{Key: "version", Value: 1},
			{Key: "creator", Value: 1},
			{Key: "createdAt", Value: 1},
			{Key: "isDeleted", Value: 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}}},
		bson.D{{Key: "$limit", Value: sampleLimit}},
	)

	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var samples []orphanAppConfigFileVersionSample
	if err = cursor.All(ctx, &samples); err != nil {
		return nil, err
	}
	return samples, nil
}

// deleteOrphanAppConfigFileVersions 以游标 + 批量删除的方式清理孤儿版本。
//
//   - 避免先 toArray 全量加载所有 _id，降低内存占用
//   - 便于在生产环境中处理较大规模的历史脏数据
func deleteOrphanAppConfigFileVersions(
	ctx context.Context,
	coll *mongo.Collection,
	appID string,
	batchSize int,
) (int64, error) {
	pipeline := append(
		orphanAppConfigFileVersionPipeline(appID),
		bson.D{{Key: "$project", Value: bson.D{{Key: "_id", Value: 1}}}},
	)
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var (
		totalDeleted int64
		batchIDs     []bson.ObjectID
	)
	flush := func() error {
		if len(batchIDs) == 0 {
			return nil
		}
		result, dErr := coll.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: batchIDs}}}})
		if dErr != nil {
			return dErr
		}
		totalDeleted += result.DeletedCount
		batchIDs = batchIDs[:0]
		return nil
	}

	for cursor.Next(ctx) {
		var item orphanVersionID
		if err = cursor.Decode(&item); err != nil {
			return totalDeleted, err
		}
		batchIDs = append(batchIDs, item.ID)
		if len(batchIDs) >= batchSize {
			if err = flush(); err != nil {
				return totalDeleted, err
			}
		}
	}
	if err = cursor.Err(); err != nil {
		return totalDeleted, err
	}
	if err = flush(); err != nil {
		return totalDeleted, err
	}
	return totalDeleted, nil
}
