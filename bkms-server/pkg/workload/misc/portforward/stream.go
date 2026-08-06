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

// Package portforward 提供应用实例 Pod 端口转发服务。
package portforward

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// IdleTimeout 空闲连接自动关闭超时时间（30 分钟无数据传输则关闭）。
const IdleTimeout = 30 * time.Minute

// CopyResult 双向复制的统计结果。
type CopyResult struct {
	// BytesSent 客户端 → Pod 方向传输的字节数。
	BytesSent int64
	// BytesReceived Pod → 客户端方向传输的字节数。
	BytesReceived int64
	// Err 复制过程中的错误（nil 表示正常结束）。
	Err error
}

// CopyBidirectional 在客户端流和 Pod 流之间进行双向数据复制，支持空闲超时和字节数统计。
// 任一方向结束（或 ctx 取消、或空闲超时）后关闭双方连接，等待两个 goroutine 退出。
func CopyBidirectional(ctx context.Context, clientStream, podStream io.ReadWriteCloser) CopyResult {
	// 空闲超时控制：任一方向有数据传输则重置计时器。
	idleTimer := time.NewTimer(IdleTimeout)
	defer idleTimer.Stop()

	idleCtx, idleCancel := context.WithCancelCause(ctx)
	defer idleCancel(nil)

	// 监控空闲超时
	go func() {
		select {
		case <-idleTimer.C:
			log.WarnAttrs(ctx, "port-forward idle timeout reached, closing connection",
				slog.Duration("idle_timeout", IdleTimeout),
			)
			idleCancel(errors.New("idle timeout"))
		case <-idleCtx.Done():
		}
	}()

	// 字节数统计
	var bytesSent, bytesReceived atomic.Int64

	// 使用 idleAwareReader 包装，每次读取数据时重置空闲计时器。
	clientReader := &idleAwareReader{r: clientStream, idleTimer: idleTimer, counter: &bytesSent}
	podReader := &idleAwareReader{r: podStream, idleTimer: idleTimer, counter: &bytesReceived}

	copyDone := make(chan error, 2)

	// Client → Pod
	go func() {
		_, err := io.Copy(podStream, clientReader)
		closeWrite(podStream)
		copyDone <- err
	}()
	// Pod → Client
	go func() {
		_, err := io.Copy(clientStream, podReader)
		closeWrite(clientStream)
		copyDone <- err
	}()

	var copyErrors [2]error
	select {
	case copyErrors[0] = <-copyDone:
		// 某个方向的 io.Copy 先返回（对端 EOF 或网络异常），进入后续关闭流程。
	case <-idleCtx.Done():
		// 外部取消（如 keepAlive 超时或空闲超时），关闭连接并等待两个 goroutine 退出后直接返回。
		closeStreams(ctx, clientStream, podStream)
		<-copyDone
		<-copyDone
		return CopyResult{
			BytesSent:     bytesSent.Load(),
			BytesReceived: bytesReceived.Load(),
			Err:           nil,
		}
	}
	// 某个方向先结束，关闭双方连接触发另一个方向退出。
	closeStreams(ctx, clientStream, podStream)
	copyErrors[1] = <-copyDone

	for _, copyErr := range copyErrors {
		if copyErr = normalizeCopyError(copyErr); copyErr != nil {
			return CopyResult{
				BytesSent:     bytesSent.Load(),
				BytesReceived: bytesReceived.Load(),
				Err:           copyErr,
			}
		}
	}
	return CopyResult{
		BytesSent:     bytesSent.Load(),
		BytesReceived: bytesReceived.Load(),
		Err:           nil,
	}
}

// idleAwareReader 在每次 Read 时重置空闲计时器并统计字节数。
type idleAwareReader struct {
	r         io.Reader
	idleTimer *time.Timer
	counter   *atomic.Int64
}

func (r *idleAwareReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.idleTimer.Reset(IdleTimeout)
		r.counter.Add(int64(n))
	}
	return n, err
}

// closeStreams 关闭客户端和 Pod 双方连接，关闭失败时记录 warn 日志并区分来源。
func closeStreams(ctx context.Context, clientStream, podStream io.ReadWriteCloser) {
	if err := clientStream.Close(); err != nil {
		log.WarnAttrs(ctx, "close client stream failed", slog.String("error", err.Error()))
	}
	if err := podStream.Close(); err != nil {
		log.WarnAttrs(ctx, "close pod stream failed", slog.String("error", err.Error()))
	}
}

func closeWrite(conn io.ReadWriteCloser) {
	if closer, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = closer.CloseWrite()
	}
}

func normalizeCopyError(err error) error {
	if err == nil || errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
		return nil
	}
	// WebSocket NetConn 在连接正常关闭时可能返回 context.Canceled。
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}

	// 对 Pod 端异常进行分类，返回 sentinel error 供 handler 层选择 close code。
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "connection reset by peer"):
		return bkerrs.ErrPodConnectionLost
	case strings.Contains(errMsg, "i/o timeout") || strings.Contains(errMsg, "deadline exceeded"):
		return bkerrs.ErrPodConnectionTimeout
	case errors.Is(err, io.EOF):
		return bkerrs.ErrPodEOF
	}
	return err
}

func classifyTargetOpenError(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "connection refused"):
		return "connection_refused"
	case strings.Contains(message, "network is unreachable"), strings.Contains(message, "no route to host"):
		return "network_unreachable"
	case strings.Contains(message, "timeout"), strings.Contains(message, "i/o timeout"):
		return "timeout"
	default:
		return "error"
	}
}
