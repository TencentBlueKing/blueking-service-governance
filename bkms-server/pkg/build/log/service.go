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

package log

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

type buildLogClient interface {
	GetInitBuildLog(ctx context.Context, projectCode, pipelineID, buildID string) (*bkciapi.BuildLog, error)
	GetMoreBuildLogs(
		ctx context.Context, projectCode, pipelineID, buildID string, start, batchSize int64,
	) (*bkciapi.BuildLog, error)
	DownloadBuildLogs(ctx context.Context, projectCode, pipelineID, buildID string) (io.ReadCloser, error)
}

// emptyPollIntervals 定义空轮询退避间隔。
// 仅当 BKCI 侧已拉到当前尾部且本次没有新日志时生效；一旦拉到新日志会回到首档。
var emptyPollIntervals = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

const (
	// streamLogBatchSize 用于 SSE 增量拉取，控制单次请求体积
	streamLogBatchSize int64 = 1000
	// emptyPollLogInterval 控制连续空轮询的日志采样频率
	emptyPollLogInterval = 5
)

// Service 构建日志服务
type Service struct{}

// NewService 创建构建日志服务
func NewService() *Service {
	return &Service{}
}

// StreamLogs 流式读取构建日志，每次拉到新日志时调用 onChunk 回调。
// 阻塞直至日志全部输出或 ctx 取消。
func (s *Service) StreamLogs(
	ctx context.Context,
	client buildLogClient,
	query *BuildLogQuery,
	onChunk func(chunk *bkciapi.BuildLog),
) error {
	// GetInitBuildLog 获取首屏日志，bkci决定返回的日志数量，根据 hasMore 和 finished 判断是否需要后续 getMore 拉取
	initLog, err := client.GetInitBuildLog(ctx, query.ProjectCode, query.PipelineID, query.BuildID)
	if err != nil {
		return errors.Wrap(err, "get init build log")
	}
	if len(initLog.Logs) > 0 {
		onChunk(initLog)
	}
	if initLog.IsComplete() {
		return nil
	}

	cursor := initLog.MaxLineNo() + 1
	emptyPollCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		chunk, err := client.GetMoreBuildLogs(
			ctx, query.ProjectCode, query.PipelineID, query.BuildID, cursor, streamLogBatchSize,
		)
		if err != nil {
			return errors.Wrap(err, "get more build log")
		}

		if len(chunk.Logs) > 0 {
			onChunk(chunk)
			cursor = chunk.MaxLineNo() + 1
			emptyPollCount = 0
		}

		if chunk.IsComplete() {
			return nil
		}

		// 无论 BKCI 是否还有更多日志，只要本次没有拿到新日志且构建未结束，就退避等待。
		// 这样可以避免出现“空日志但 HasMore=true”时用相同 cursor 紧密轮询。
		if len(chunk.Logs) == 0 {
			emptyPollCount++
			if shouldLogEmptyPoll(emptyPollCount) {
				log.Debugf(
					ctx,
					"build log stream waiting for more logs, project=%s pipeline=%s build=%s cursor=%d has_more=%t finished=%t empty_polls=%d",
					query.ProjectCode,
					query.PipelineID,
					query.BuildID,
					cursor,
					chunk.HasMore,
					chunk.Finished,
					emptyPollCount,
				)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(calcEmptyPollInterval(emptyPollCount)):
			}
		}
	}
}

func calcEmptyPollInterval(emptyPollCount int) time.Duration {
	if emptyPollCount <= 1 {
		return emptyPollIntervals[0]
	}
	if emptyPollCount >= len(emptyPollIntervals) {
		return emptyPollIntervals[len(emptyPollIntervals)-1]
	}
	return emptyPollIntervals[emptyPollCount-1]
}

func shouldLogEmptyPoll(emptyPollCount int) bool {
	return emptyPollCount > 0 && emptyPollCount%emptyPollLogInterval == 0
}

// OpenDownloadStream 返回构建日志下载流
// 下载场景只聚合当前可拉取到的完整日志内容，不等待构建最终结束
func (s *Service) OpenDownloadStream(
	ctx context.Context,
	client buildLogClient,
	query *BuildLogQuery,
) (io.ReadCloser, error) {
	reader, err := client.DownloadBuildLogs(ctx, query.ProjectCode, query.PipelineID, query.BuildID)
	if err != nil {
		return nil, errors.Wrap(err, "download build log")
	}
	return reader, nil
}
