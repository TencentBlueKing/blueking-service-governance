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

package handler

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	pfwhitelist "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/portforward"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/misc/portforward"
)

// 注意：websocket.Accept 与 Gin 中间件的兼容性要求：
//   - 不要在 Upgrade 前由中间件写响应头或响应体
//   - 不要对该路由启用强制压缩中间件
//   - 不要对该路由设置短超时（如 http.TimeoutHandler）

// PortForward 应用实例端口转发。
//
//	@ID			PortForward
//	@Summary	应用实例端口转发
//	@Tags		instance
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path	string	true	"应用 ID"
//	@Param		envName		path	string	true	"部署环境名称"
//	@Param		instanceID	path	string	true	"实例 ID"
//	@Param		remotePort	query	int		true	"目标 Pod 端口号"
//	@Param		localPort	query	int		true	"CLI 本地监听端口号"
//	@Router		/apps/{appID}/envs/{envName}/instances/{instanceID}/port-forward/connect [get]
func (h *Handler) PortForward(c *gin.Context) {
	var uriInput serializer.AppInstanceURIInput
	var queryInput serializer.PortForwardQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, err := h.validateEditableAppModel(ctx, uriInput.AppID)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err = perm.NewManager().HasDeployEnvPerm(ctx, app.WorkspaceID, uriInput.EnvName); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, "check deploy env perm"))
		return
	}

	// 校验环境类型：禁止在正式环境使用 port-forward
	envInfo, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, uriInput.AppID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get environment info"))
		return
	}
	if bkmsenv.IsProductionType(bkmsenv.Type(envInfo.Type)) {
		bkerrs.AbortWithErr(c, bkerrs.New(
			bkerrs.ErrCodeInvalidRequest,
			"port-forward is not allowed in production environment",
		))
		return
	}

	// 校验 port-forward 白名单权限
	pfService := pfwhitelist.NewService(h.registry.PortForwardWhitelistStore, h.registry.EnvStore)
	if err = pfService.CheckPermission(ctx, envInfo.ID.Hex()); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.New(
			bkerrs.ErrCodeNoPermission, pfwhitelist.ErrPermissionDenied.Error(),
		))
		return
	}

	// 先连接目标 Pod，失败时直接返回 HTTP 错误
	svc := portforward.NewService(h.registry.AppModelDeployRecordStore)
	podStream, err := svc.OpenTargetStream(
		ctx,
		uriInput.AppID,
		uriInput.EnvName,
		uriInput.InstanceID,
		queryInput.RemotePort,
	)
	if err != nil {
		c.Header("X-Port-Forward-Error", "target unreachable")
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "connect target pod"))
		return
	}
	defer podStream.Close()

	// 目标连接成功，升级 WebSocket。
	wsConn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		// CLI 不是浏览器，不校验 Origin。
		InsecureSkipVerify: true,
	})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidArgument, "upgrade port-forward websocket"))
		return
	}

	startedAt := time.Now()
	metrics.PortForwardSessionStarted()
	log.InfoAttrs(
		ctx,
		"port-forward session started",
		slog.String("app_id", uriInput.AppID),
		slog.String("env_name", uriInput.EnvName),
		slog.String("instance_id", uriInput.InstanceID),
		slog.Int("remote_port", int(queryInput.RemotePort)),
		slog.Int("local_port", int(queryInput.LocalPort)),
	)

	auditData := map[string]any{
		"instanceID": uriInput.InstanceID,
		"remotePort": queryInput.RemotePort,
		"localPort":  queryInput.LocalPort,
		"startedAt":  startedAt,
	}
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeExecute,
		audit.ResourceTypeInstance,
		uriInput.InstanceID,
		audit.WithAttribute(audit.AttributePortForward),
		audit.WithDataAfter(auditData),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)

	// 独立 context 控制 WebSocket 生命周期，避免 Gin 请求 context 提前取消。
	connCtx, connCancel := context.WithCancel(context.Background())
	defer connCancel()
	tunnelConn := websocket.NetConn(connCtx, wsConn, websocket.MessageBinary)

	// 心跳探测 CLI 连接存活，超时则 connCancel 触发关闭。
	go keepAliveWebSocket(connCtx, connCancel, wsConn)

	// FIXME: 后续考虑引入 yamux/smux 实现多路复用
	// 双向复制：CLI(WebSocket) ↔ Pod(TCP)。
	// 使用 contextCancelWriter 包装：Close() 时取消 connCtx 以解除 WebSocket Read 阻塞，
	// 但不直接发送 close frame，保留 handler 根据错误类型选择自定义 close code 的能力。
	copyResult := portforward.CopyBidirectional(connCtx, &contextCancelWriter{tunnelConn, connCancel}, podStream)

	result := audit.ResultSuccess
	metricResult := metrics.StatusOK
	if copyResult.Err != nil {
		result = audit.ResultFailed
		metricResult = metrics.StatusErr
	}
	logAttrs := []slog.Attr{
		slog.String("app_id", uriInput.AppID),
		slog.String("env_name", uriInput.EnvName),
		slog.String("instance_id", uriInput.InstanceID),
		slog.Int("remote_port", int(queryInput.RemotePort)),
		slog.String("outcome", metricResult),
		slog.Duration("duration", time.Since(startedAt)),
		slog.Int64("bytes_sent", copyResult.BytesSent),
		slog.Int64("bytes_received", copyResult.BytesReceived),
	}
	if copyResult.Err != nil {
		logAttrs = append(logAttrs, slog.String("error", copyResult.Err.Error()))
	}
	metrics.PortForwardSessionFinished(startedAt, metricResult)
	log.InfoAttrs(ctx, "port-forward session finished", logAttrs...)
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeExecute,
		audit.ResourceTypeInstance,
		uriInput.InstanceID,
		audit.WithAttribute(audit.AttributePortForward),
		audit.WithResult(result),
		audit.WithDataAfter(map[string]any{
			"instanceID":    uriInput.InstanceID,
			"remotePort":    queryInput.RemotePort,
			"localPort":     queryInput.LocalPort,
			"endedAt":       time.Now(),
			"result":        result,
			"bytesSent":     copyResult.BytesSent,
			"bytesReceived": copyResult.BytesReceived,
		}),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)

	code, reason := bkerrs.PortForwardCloseCodeForError(copyResult.Err)
	if err = wsConn.Close(code, reason); err != nil {
		log.DebugAttrs(ctx, "close port-forward websocket", slog.String("error", err.Error()))
	}
}

// contextCancelWriter 包装 WebSocket tunnelConn，Close() 仅取消 context 以解除阻塞的 Read/Write，
// 不发送 close frame，保留 handler 后续使用自定义 close code 关闭连接的能力。
type contextCancelWriter struct {
	io.ReadWriteCloser
	cancel context.CancelFunc
}

func (c *contextCancelWriter) Close() error {
	c.cancel()
	return nil
}

const (
	// portForwardPingInterval 心跳发送间隔，需小于中间代理的空闲超时（通常 60s）。
	portForwardPingInterval = 25 * time.Second
	// portForwardPingTimeout 等待 pong 响应的超时时间。
	portForwardPingTimeout = 10 * time.Second
)

// keepAliveWebSocket 定期向 CLI 发送 ping，检测 CLI↔Server 连接存活。
// ping 超时则取消 context 触发连接关闭。
func keepAliveWebSocket(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	ticker := time.NewTicker(portForwardPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, portForwardPingTimeout)
			err := conn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				cancel()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
