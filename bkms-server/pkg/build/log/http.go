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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	httpresp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/http"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

// WriteDownloadBuildLogs 通用日志下载逻辑（应用构建与 Helm Chart 构建共用）。
func WriteDownloadBuildLogs(
	c *gin.Context,
	svc *Service,
	client buildLogClient,
	query *BuildLogQuery,
) error {
	ctx := c.Request.Context()
	reader, err := svc.OpenDownloadStream(ctx, client, query)
	if err != nil {
		return errors.Wrap(err, "open build log download stream")
	}
	defer reader.Close()
	filename := query.DownloadFilename()

	c.DataFromReader(
		http.StatusOK,
		-1,
		httpresp.AttachmentContentType,
		reader,
		map[string]string{
			"Content-Disposition": httpresp.BuildAttachmentDisposition(filename),
		},
	)
	return nil
}

// WriteSSEBuildLogs 通用 SSE 日志写入逻辑（应用构建与 Helm Chart 构建共用）
func WriteSSEBuildLogs(
	c *gin.Context,
	svc *Service,
	client buildLogClient,
	query *BuildLogQuery,
	buildErrorOutput func(error, *BuildLogQuery) any,
) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx := c.Request.Context()
	flusher, _ := c.Writer.(http.Flusher)

	err := svc.StreamLogs(ctx, client, query, func(chunk *bkciapi.BuildLog) {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", data)
		if flusher != nil {
			flusher.Flush()
		}
	})
	if err != nil {
		errData, _ := json.Marshal(buildErrorOutput(err, query))
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errData)
	}

	fmt.Fprint(c.Writer, "event: done\ndata: {}\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}
