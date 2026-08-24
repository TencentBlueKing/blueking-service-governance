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

package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// 单条 SSE 写入的超时；每次写前重置，只约束这一次 Write，不承担连接总时长
// 连接总时长由 Manager.maxAge 硬限制；这里防止客户端连着但不读时写缓冲写满、goroutine 永久阻塞
const chunkWriteTimeout = 30 * time.Second

// sseStream 一条 SSE 连接的写入端，负责逐条 Flush 与单次写超时
type sseStream struct {
	w       http.ResponseWriter
	flusher http.Flusher
	rc      *http.ResponseController
}

// newSSEStream 绑定响应写入端；拿不到 Flusher 说明当前环境不支持流式，直接判定建流失败
func newSSEStream(ctx context.Context, w http.ResponseWriter) (*sseStream, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported")
	}

	s := &sseStream{w: w, flusher: flusher, rc: http.NewResponseController(w)}

	// 探测一次写超时能否设置：不支持时整条流会受 http.Server.WriteTimeout 限制被定时掐断
	// 这里只降级并告警，不阻断建流；后续每次写不再重复告警
	if err := s.resetWriteDeadline(); err != nil {
		log.WarnAttrs(ctx, "set write deadline failed, instance watch may be cut by server write timeout",
			slog.String("err", err.Error()),
		)
	}

	return s, nil
}

// writeHeaders 写入 SSE 响应头；必须在 Watch 建立成功之后、首个事件之前调用
func (s *sseStream) writeHeaders() {
	s.w.Header().Set("Content-Type", "text/event-stream")
	s.w.Header().Set("Cache-Control", "no-cache")
	s.w.Header().Set("Connection", "keep-alive")
}

// writeHeartbeat 写 SSE 注释心跳 `:`，浏览器忽略，仅用来保活
func (s *sseStream) writeHeartbeat() error {
	return s.write([]byte(":\n\n"))
}

// writeEvent 按 `event: message` + `data: {json}` 写出一条投影事件
func (s *sseStream) writeEvent(event serializer.AppInstanceWatchEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return errors.Wrapf(err, "marshal %s watch event", event.Type)
	}

	if err = s.write(fmt.Appendf(nil, "event: message\ndata: %s\n\n", data)); err != nil {
		return err
	}

	metrics.InstanceWatchEventPushed(event.Type)
	return nil
}

// write 落盘一段 SSE payload 并立即 Flush，逐条下发而不是攒在缓冲里
func (s *sseStream) write(payload []byte) error {
	// 每次写前重置写超时，覆盖 http.Server 设的整条响应 deadline
	_ = s.resetWriteDeadline()

	// 写失败基本都是客户端已断开或写超时，此时流无法继续，由调用方收流
	if _, err := s.w.Write(payload); err != nil {
		return errors.Wrap(err, "write sse payload")
	}

	s.flusher.Flush()
	return nil
}

func (s *sseStream) resetWriteDeadline() error {
	return s.rc.SetWriteDeadline(time.Now().Add(chunkWriteTimeout))
}
