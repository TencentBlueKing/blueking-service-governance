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

package instance

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/coder/websocket"
)

// copyBidirectional 在本地 TCP 连接和 Server WebSocket 隧道之间进行双向数据复制。
// 任一方向结束后关闭双方连接，等待两个 goroutine 退出。
func copyBidirectional(ctx context.Context, localConn net.Conn, tunnel io.ReadWriteCloser) error {
	slog.Debug("bidirectional copy started",
		"local_addr", localConn.LocalAddr().String(),
	)

	copyDone := make(chan error, 2)

	// Local → Server
	go func() {
		n, err := io.Copy(tunnel, localConn)
		slog.Debug("local→server copy finished", "bytes", n, "error", err)
		closeWrite(tunnel)
		copyDone <- err
	}()
	// Server → Local
	go func() {
		n, err := io.Copy(localConn, tunnel)
		slog.Debug("server→local copy finished", "bytes", n, "error", err)
		closeWrite(tunnel)
		copyDone <- err
	}()

	var copyErrors [2]error
	select {
	case copyErrors[0] = <-copyDone:
		// 某个方向的 io.Copy 先返回（对端 EOF 或网络异常），进入后续关闭流程。
	case <-ctx.Done():
		// 外部取消（如用户 Ctrl+C），关闭连接并等待两个 goroutine 退出后直接返回。
		slog.Debug("bidirectional copy context canceled, closing streams")
		closeLocalAndTunnel(localConn, tunnel)
		<-copyDone
		<-copyDone
		slog.Debug("bidirectional copy stopped due to context cancellation")
		return nil
	}

	// 某个方向先结束，关闭双方连接触发另一个方向退出。
	slog.Debug("one direction ended, closing both streams")
	closeLocalAndTunnel(localConn, tunnel)
	copyErrors[1] = <-copyDone

	// 返回第一个有意义的异常。
	for _, copyErr := range copyErrors {
		if copyErr = normalizeCopyError(copyErr); copyErr != nil {
			slog.Debug("bidirectional copy finished with error",
				"error", copyErr.Error(),
			)
			return copyErr
		}
	}
	slog.Debug("bidirectional copy finished successfully")
	return nil
}

// normalizeCopyError 过滤双向复制中的正常关闭错误，仅保留真正的异常。
// 优先通过 Server 发送的自定义 WebSocket close code 判断断开原因，
// fallback 到字符串匹配以兼容网络层异常（如 TCP reset、超时等）。
func normalizeCopyError(err error) error {
	if err == nil {
		return nil
	}

	// --- 正常关闭，不视为错误 ---
	if isNormalCloseError(err) {
		return nil
	}

	// --- 优先通过 WebSocket close code 判断（Server 端发送的自定义码） ---
	if closeCode := websocket.CloseStatus(err); closeCode != -1 {
		if sentinelErr := errorFromCloseCode(closeCode); sentinelErr != nil {
			return sentinelErr
		}
	}

	// --- Fallback：网络层异常（CLI↔Server 链路断开，无法收到 close frame） ---
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "StatusAbnormalClosure"):
		return ErrServerConnectionLost
	case strings.Contains(errMsg, "connection reset by peer"):
		return ErrConnectionReset
	case strings.Contains(errMsg, "i/o timeout") || strings.Contains(errMsg, "deadline exceeded"):
		return ErrConnectionTimeout
	case errors.Is(err, io.EOF) || strings.Contains(errMsg, "EOF"):
		return ErrServerConnectionLost
	}

	return err
}

// errorFromCloseCode 根据 Server 发送的自定义 close code 映射为对应的 sentinel error。
// 返回 nil 表示该 close code 不是已知的自定义码。
func errorFromCloseCode(code websocket.StatusCode) error {
	switch code {
	case CloseCodePodConnectionLost:
		return ErrPodConnectionLost
	case CloseCodePodConnectionTimeout:
		return ErrPodConnectionTimeout
	case CloseCodePodEOF:
		return ErrPodEOF
	case CloseCodeServerInternalError:
		return ErrServerInternalError
	default:
		return nil
	}
}

// isNormalCloseError 判断是否为正常关闭产生的错误（不应向用户展示）。
func isNormalCloseError(err error) bool {
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// Server 正常关闭（StatusNormalClosure）。
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		return true
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "status = StatusNormalClosure") {
		return true
	}
	if strings.Contains(errMsg, "failed to get reader") && strings.Contains(errMsg, "context canceled") {
		return true
	}
	return false
}

func closeWrite(conn io.ReadWriteCloser) {
	if closer, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = closer.CloseWrite()
	}
}

// closeLocalAndTunnel 关闭本地连接和远程隧道，关闭失败时记录 debug 日志并区分来源。
func closeLocalAndTunnel(localConn net.Conn, tunnel io.ReadWriteCloser) {
	if err := localConn.Close(); err != nil {
		slog.Debug("close local connection failed", "error", err.Error())
	}
	if err := tunnel.Close(); err != nil {
		slog.Debug("close server tunnel failed", "error", err.Error())
	}
}
