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
	"regexp"

	"github.com/coder/websocket"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// defaultPortForwardLocalAddress 默认侦听地址
const defaultPortForwardLocalAddress = "127.0.0.1"

// maxConcurrentConnections 最大并发端口转发连接数。
const maxConcurrentConnections = 10

const (
	// CloseCodePodConnectionLost Server→Pod TCP 连接断开（含 connection reset）。
	CloseCodePodConnectionLost websocket.StatusCode = 4001
	// CloseCodePodConnectionTimeout Server→Pod 连接超时。
	CloseCodePodConnectionTimeout websocket.StatusCode = 4002
	// CloseCodePodEOF Pod 主动关闭了连接。
	CloseCodePodEOF websocket.StatusCode = 4004
	// CloseCodeServerInternalError Server 内部错误。
	CloseCodeServerInternalError websocket.StatusCode = 4010
)

// Pod 实例状态常量（对应 Kubernetes Pod Phase）。
const (
	// InstanceStatusPending 实例等待调度中。
	InstanceStatusPending = "Pending"
	// InstanceStatusRunning 实例运行中。
	InstanceStatusRunning = "Running"
	// InstanceStatusSucceeded 实例已成功终止。
	InstanceStatusSucceeded = "Succeeded"
	// InstanceStatusFailed 实例异常终止。
	InstanceStatusFailed = "Failed"
	// InstanceStatusUnknown 实例状态未知（通常为节点通信异常）。
	InstanceStatusUnknown = "Unknown"
)

// podNameRegexp 校验 Kubernetes Pod 名称格式。
var podNameRegexp = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,251}[a-z0-9])?$`)

// 端口转发隧道错误，调用方可通过 errors.Is 判断具体断开原因。
var (
	// ErrPodConnectionLost 表示 Server→Pod 连接断开。
	ErrPodConnectionLost = errors.New("remote pod connection lost")

	// ErrPodConnectionTimeout 表示 Server→Pod 连接超时。
	ErrPodConnectionTimeout = errors.New("remote pod connection timed out")

	// ErrPodEOF 表示 Pod 主动关闭了连接。
	ErrPodEOF = errors.New("remote pod closed connection")

	// ErrServerConnectionLost 表示 CLI↔Server WebSocket 连接异常关闭（如 Server 重启/网络中断）。
	ErrServerConnectionLost = errors.New("connection to server lost unexpectedly (server may have restarted)")

	// ErrConnectionReset 表示 CLI↔Server TCP 连接被重置（网络层中断）。
	ErrConnectionReset = errors.New("connection to server reset (network interruption)")

	// ErrServerInternalError 表示 Server 报告了内部错误。
	ErrServerInternalError = errors.New("server internal error")

	// ErrConnectionTimeout 表示 CLI↔Server 连接超时。
	ErrConnectionTimeout = errors.New("connection to server timed out")
)

// portForwardTunnelClient 是 CLI client 中端口转发隧道能力的最小接口。
type portForwardTunnelClient interface {
	OpenPortForwardTunnel(
		ctx context.Context,
		appID, envName string,
		opts client.PortForwardTunnelOptions,
	) (io.ReadWriteCloser, error)
}

// PortForwardOptions 端口转发命令选项。
type PortForwardOptions struct {
	AppID   string `validate:"required"`
	EnvName string `validate:"required"`
	// InstanceID 目标 Pod 实例 ID。
	InstanceID string `validate:"required"`
	// RemotePort 目标 Pod 端口号。
	RemotePort int `validate:"min=1,max=65535"`
	// LocalPort 本地监听端口号。
	LocalPort int `validate:"min=1,max=65535"`
	// LocalAddress 本地监听地址，默认 127.0.0.1。
	LocalAddress string `validate:"omitempty,ip"`
}
