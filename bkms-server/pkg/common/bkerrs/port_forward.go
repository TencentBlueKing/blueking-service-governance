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

package bkerrs

import (
	"errors"

	"github.com/coder/websocket"
)

// 端口转发自定义 WebSocket Close Code（4000-4999 为应用私有范围）。
// CLI 端通过 websocket.CloseStatus(err) 提取 close code 判断断开原因。
const (
	// WsCloseCodePodConnectionLost Server→Pod TCP 连接断开（含 connection reset）。
	WsCloseCodePodConnectionLost websocket.StatusCode = 4001
	// WsCloseCodePodConnectionTimeout Server→Pod 连接超时。
	WsCloseCodePodConnectionTimeout websocket.StatusCode = 4002
	// WsCloseCodePodEOF Pod 主动关闭了连接。
	WsCloseCodePodEOF websocket.StatusCode = 4004
	// WsCloseCodeServerInternalError Server 内部错误。
	WsCloseCodeServerInternalError websocket.StatusCode = 4010
)

// 端口转发 sentinel error。
// Handler 层根据这些错误选择对应的 WebSocket close code。
var (
	// ErrPodConnectionLost Server→Pod TCP 连接断开（connection reset by peer）。
	ErrPodConnectionLost = errors.New("pod connection lost")
	// ErrPodConnectionTimeout Server→Pod 连接超时。
	ErrPodConnectionTimeout = errors.New("pod connection timed out")
	// ErrPodEOF Pod 主动关闭了连接（EOF）。
	ErrPodEOF = errors.New("pod closed connection")
)

// PortForwardCloseCodeForError 根据端口转发错误选择对应的 WebSocket close code 和 reason。
// 返回值可直接用于 wsConn.Close(code, reason)。
func PortForwardCloseCodeForError(err error) (websocket.StatusCode, string) {
	switch {
	case err == nil:
		return websocket.StatusNormalClosure, "port-forward completed"
	case errors.Is(err, ErrPodConnectionLost):
		return WsCloseCodePodConnectionLost, "pod connection lost"
	case errors.Is(err, ErrPodConnectionTimeout):
		return WsCloseCodePodConnectionTimeout, "pod connection timed out"
	case errors.Is(err, ErrPodEOF):
		return WsCloseCodePodEOF, "pod closed connection"
	default:
		return WsCloseCodeServerInternalError, "server internal error"
	}
}
