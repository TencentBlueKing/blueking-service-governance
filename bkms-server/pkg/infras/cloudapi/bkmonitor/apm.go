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

// Package bkmonitor api client，如：蓝鲸监控的 apm、蓝鲸监控的 metadata
// 监控不支持 APM 的物理删除，因此，无需删除方法。
package bkmonitor

import (
	"context"
	"net/http"
	"strings"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

const (
	// headerBkapiUserName 蓝鲸 API 网关用户名请求头
	headerBkapiUserName = "X-Bkapi-User-Name"
)

// CreateApmApp 创建 APM 应用
// bkBizID 是指蓝鲸监控下容器项目的业务 ID
// envName 是 apm app 的名称
func (c *ApiClient) CreateApmApp(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	req := NewDefaultCreateApmAppReq(bkBizID, bcsProjectCode, envName, description, operator)
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, err := c.handleOperation(ctx, c.newCreateApmAppOperation(req)); err != nil {
		// 检查是否是重复创建错误
		if strings.Contains(err.Error(), "已被创建") {
			metrics.CreateEnvApmFailed(workspaceID, envName, "already_exists")
			return nil, ErrApmAppDuplicate
		}
		metrics.CreateEnvApmFailed(workspaceID, envName, "create_failed")
		return nil, errors.Wrapf(err, "create apm app failed, bk_biz_id: %d, app_name: %s", req.BkBizID, req.AppName)
	}

	// 创建成功后查询应用详情
	result, err := c.GetApmApp(ctx, req.BkBizID, 0, req.AppName)
	if err != nil {
		metrics.CreateEnvApmFailed(workspaceID, envName, "query_after_create_failed")
		return nil, errors.Wrap(err, "get apm app after creation failed")
	}

	return result, nil
}

// newCreateApmAppOperation 创建 APM 应用操作
func (c *ApiClient) newCreateApmAppOperation(req *CreateApmAppReq) define.Operation {
	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "create_apm_app",
			Method: http.MethodPost,
			Path:   "/apm/create_application/",
		},
		bkapi.OptSetRequestBody(req),
		bkapi.OptSetRequestHeader(headerBkapiUserName, req.Operator),
	)
}

// GetApmApp 获取 APM 应用详情
// bkBizID 是指蓝鲸监控下容器项目的业务 ID，必填
// apmAppID、 envName 二选一
func (c *ApiClient) GetApmApp(ctx context.Context, bkBizID, apmAppID int64, envName string) (*ApmApp, error) {
	req := NewGetApmAppReq(bkBizID, apmAppID, envName)
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.newGetApmAppOperation(req))
	if err != nil {
		return nil, errors.Wrapf(ErrApmAppNotFound, "get apm app failed: %v", err)
	}

	result := new(ApmApp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrapf(err, "decode apm app failed, envName: %s", req.AppName)
	}

	return result, nil
}

// newGetApmAppOperation 获取 APM 应用详情操作
func (c *ApiClient) newGetApmAppOperation(req *GetApmAppReq) define.Operation {
	params := map[string]string{
		"bk_biz_id": cast.ToString(req.BkBizID),
	}
	if req.AppName != "" {
		params["app_name"] = req.AppName
	} else {
		params["application_id"] = cast.ToString(req.ApmAppID)
	}

	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "get_apm_app",
			Method: http.MethodGet,
			Path:   "/detail_apm_application/",
		},
	).SetQueryParams(params)
}

// GetOrCreate 创建或获取 APM 应用
// bkBizID 是指蓝鲸监控下容器项目的业务 ID
// envName 是 apm app 的名称
func (c *ApiClient) GetOrCreate(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	if result, err := c.GetApmApp(ctx, bkBizID, 0, envName); err == nil {
		return result, nil
	}

	return c.CreateApmApp(ctx, bkBizID, bcsProjectCode, envName, description, operator, workspaceID)
}

// ListApmApp 列出 APM 应用
func (c *ApiClient) ListApmApp(ctx context.Context, bkBizID int64) ([]*ApmApp, error) {
	req := NewListApmAppReq(bkBizID)
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.newListApmAppOperation(req))
	if err != nil {
		return nil, errors.Wrapf(err, "list apm app failed, bk_biz_id: %d", req.BkBizID)
	}

	result := make([]*ApmApp, 0)
	if err = mapstructure.Decode(mapx.GetList(resp, "data"), &result); err != nil {
		return nil, errors.Wrapf(err, "read apm app list failed, bk_biz_id: %d", req.BkBizID)
	}

	return result, nil
}

// newListApmAppOperation 列出 APM 应用操作
func (c *ApiClient) newListApmAppOperation(req *ListApmAppReq) define.Operation {
	params := map[string]string{
		"bk_biz_id": cast.ToString(req.BkBizID),
	}

	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_apm_app",
			Method: http.MethodGet,
			Path:   "/list_apm_application/",
		},
	).SetQueryParams(params)
}
