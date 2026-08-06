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

package helm

import (
	"context"

	"helm.sh/helm/v3/pkg/action"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// NewHelmDebugLogger 创建 Helm SDK DebugLog 函数
// 将 Helm 内部日志转发到内部统一日志封装，附带 releaseName 和 operationType 上下文，
// 并携带调用方传入的 ctx 中的 trace_id / span_id，方便一站串联部署链路。
func NewHelmDebugLogger(ctx context.Context, releaseName, operationType string) action.DebugLog {
	logCtx := context.WithoutCancel(ctx)
	prefix := "[helm-sdk] release=" + releaseName + " op=" + operationType + " "
	return func(format string, v ...any) {
		log.Debugf(logCtx, prefix+format, v...)
	}
}
