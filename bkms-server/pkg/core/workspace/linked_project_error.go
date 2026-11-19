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

package workspace

import (
	"cmp"
	"context"
	"net"
	"net/http"

	"github.com/pkg/errors"

	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	bkmonitorapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// classifyLinkedProjectError 将平台调用错误归类，并返回平台原始错误信息与请求 ID
func classifyLinkedProjectError(err error) (reason LinkedProjectFailureReason, detail, requestID string) {
	detail = err.Error()

	var ciErr *bkciapi.APIError
	if errors.As(err, &ciErr) {
		reason = classifyLinkedProjectHTTPStatus(ciErr.HTTPStatus)
		if ciErr.Status == bkciapi.ProjectNotExistStatus {
			reason = LinkedProjectFailureNotFound
		}
		return reason, cmp.Or(ciErr.Message, detail), ciErr.RequestID
	}

	var monitorErr *bkmonitorapi.APIError
	if errors.As(err, &monitorErr) {
		reason = classifyLinkedProjectHTTPStatus(monitorErr.HTTPStatus)
		switch {
		case monitorErr.IsNotFound():
			reason = LinkedProjectFailureNotFound
		// 监控接口只做应用鉴权，401/403 说明 BKMS 应用未获授权，用户无法自行处理
		case reason == LinkedProjectFailurePermission:
			reason = LinkedProjectFailureUnknown
		}
		return reason, cmp.Or(monitorErr.Message, detail), monitorErr.RequestID
	}

	if errors.Is(err, bkciapi.ObjectNotFound) || errors.Is(err, bkmonitorapi.ErrSpaceNotFound) {
		return LinkedProjectFailureNotFound, detail, ""
	}

	// 网络错误（超时、连接失败等）
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) {
		return LinkedProjectFailureUnavailable, detail, ""
	}
	return LinkedProjectFailureUnknown, detail, ""
}

func classifyLinkedProjectHTTPStatus(status int) LinkedProjectFailureReason {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return LinkedProjectFailurePermission
	case status == http.StatusNotFound:
		return LinkedProjectFailureNotFound
	case status == http.StatusTooManyRequests, status >= http.StatusInternalServerError:
		return LinkedProjectFailureUnavailable
	default:
		return LinkedProjectFailureUnknown
	}
}
