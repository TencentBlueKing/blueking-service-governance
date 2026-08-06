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
package bkmonitor

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/httpresp"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

const (
	// legacyGatewayName 旧版蓝鲸监控网关名（bkmonitorv3），用于 APM、metadata 等存量接口。
	legacyGatewayName = "bkmonitorv3"
	// newGatewayName 新版蓝鲸监控网关名（bk-monitor），告警策略等新增接口走此网关，后续均切换到新网关。
	newGatewayName = "bk-monitor"
)

// Client 蓝鲸监控 API 客户端接口
type Client interface {
	// CreateApmApp 创建 APM 应用
	CreateApmApp(
		ctx context.Context,
		bkBizID int64,
		bcsProjectCode, envName, description, operator, workspaceID string,
	) (*ApmApp, error)
	// GetApmApp 获取 APM 应用详情
	GetApmApp(ctx context.Context, bkBizID, apmAppID int64, envName string) (*ApmApp, error)
	// GetOrCreate 创建或获取 APM 应用
	GetOrCreate(
		ctx context.Context,
		bkBizID int64,
		bcsProjectCode, envName, description, operator, workspaceID string,
	) (*ApmApp, error)
	// ListApmApp 列出 APM 应用
	ListApmApp(ctx context.Context, bkBizID int64) ([]*ApmApp, error)
	// ListMetadataSpaceByUID 根据 space_uid 获取空间
	ListMetadataSpaceByUID(ctx context.Context, uid string) (*Space, error)
	// GetMetadataSpaceDetail 获取空间详情
	GetMetadataSpaceDetail(ctx context.Context, bcsProjectCode string) (*Space, error)
	// SearchUserGroups 查询告警组列表
	SearchUserGroups(ctx context.Context, req *SearchUserGroupsReq) ([]*UserGroup, error)
	// SearchUserGroupDetail 查询告警组详情
	SearchUserGroupDetail(ctx context.Context, req *SearchUserGroupDetailReq) (*UserGroupDetail, error)
	// SaveUserGroup 保存（创建/更新）告警组
	SaveUserGroup(ctx context.Context, req *SaveUserGroupReq) (*UserGroupDetail, error)
	// TimeSeriesUnifyQuery 统一时序数据查询
	TimeSeriesUnifyQuery(ctx context.Context, req *TimeSeriesUnifyQueryReq) (*TimeSeriesUnifyQueryResp, error)
}

// MonitorClient 蓝鲸监控新版 bk-monitor 网关客户端接口。
type MonitorClient interface {
	Client
	// DeleteUserGroup 删除告警组
	DeleteUserGroup(ctx context.Context, req *DeleteUserGroupReq) error
	// SearchAlarmStrategy 查询告警策略列表
	SearchAlarmStrategy(ctx context.Context, req *SearchAlarmStrategyReq) (*SearchAlarmStrategyResp, error)
	// SaveAlarmStrategy 创建或更新告警策略
	SaveAlarmStrategy(ctx context.Context, req *SaveAlarmStrategyReq) (*SaveAlarmStrategyResp, error)
	// SwitchAlarmStrategy 批量启停告警策略
	SwitchAlarmStrategy(ctx context.Context, req *SwitchAlarmStrategyReq) error
	// DeleteAlarmStrategy 批量删除告警策略
	DeleteAlarmStrategy(ctx context.Context, req *DeleteAlarmStrategyReq) error
	// SearchAlert 查询告警事件列表
	SearchAlert(ctx context.Context, req *SearchAlertReq) (*SearchAlertResp, error)
	// GetAlertDetail 查询告警详情
	GetAlertDetail(ctx context.Context, req *AlertDetailReq) (map[string]any, error)
}

// ApiClient 蓝鲸监控 APM API 客户端
type ApiClient struct {
	define.BkApiClient
}

// New 创建蓝鲸监控 APM API 客户端实例
func New(operator string) (Client, error) {
	if config.G.Development.UseStubBkMonitor {
		log.InfoNoContext("use stub bkmonitor client according to config")
		return NewStub(operator), nil
	}

	return newAPIClient(operator, config.G.BkMonitor.Endpoint, legacyGatewayName)
}

// NewMonitorClient 创建新版 bk-monitor 网关客户端，供新增监控接口复用。
func NewMonitorClient(operator string) (MonitorClient, error) {
	if config.G.Development.UseStubBkMonitor {
		log.InfoNoContext("use stub bkmonitor client according to config")
		return NewStub(operator), nil
	}

	client, err := newAPIClient(operator, config.G.BkMonitor.GatewayEndpoint, newGatewayName)
	if err != nil {
		return nil, err
	}
	return &MonitorGatewayClient{ApiClient: client}, nil
}

// generateAuthInfo 生成鉴权信息
func generateAuthInfo() (string, error) {
	authorization, err := json.Marshal(map[string]string{
		"bk_app_code":   config.G.BkApp.Code,
		"bk_app_secret": config.G.BkApp.Secret,
	})
	if err != nil {
		return "", err
	}
	return string(authorization), nil
}

func newAPIClient(operator, endpoint, gatewayName string) (*ApiClient, error) {
	authInfo, err := generateAuthInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate auth info")
	}
	client, err := bkapi.NewBkApiClient("", buildClientConfig(endpoint, operator, authInfo))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s api client", gatewayName)
	}
	return &ApiClient{client}, nil
}

// buildClientConfig 构建客户端配置
func buildClientConfig(endpoint, operator, authInfo string) bkapi.ClientConfig {
	return bkapi.ClientConfig{
		Endpoint: endpoint,
		ClientOptions: []define.BkApiClientOption{
			bkapi.OptSetRequestHeader("x-bkapi-authorization", authInfo),
			bkapi.OptSetRequestHeader("X-Bkapi-User-Name", operator),
			bkapi.OptJsonResultProvider(),
			bkapi.OptJsonBodyProvider(),
			bkapi.OptTimeout(60 * time.Second),
		},
	}
}

// handleOperation 发起请求并检查结果，返回响应数据 & 错误
func (c *ApiClient) handleOperation(
	ctx context.Context, op define.Operation,
) (result map[string]any, err error) {
	started := time.Now()
	defer metrics.ReportClientRequestMetric("bkmonitorv3", op.FullName(), started, &err)

	ctx, span := apm.StartClientSpan(ctx, "bkmonitorv3", op.FullName())
	resp, err := op.SetContext(ctx).SetResult(&result).Request()
	defer apm.EndClientSpan(span, resp, &err)
	if err != nil {
		return nil, errors.Wrap(err, "api request failed")
	}
	defer resp.Body.Close()

	if !httpresp.IsSuccess(resp) {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("call bkmonitorv3 api %s failed, http code: %d, err: %s",
			op.FullName(), resp.StatusCode, errMsg)
	}

	if !mapx.GetBool(result, "result") {
		return nil, errors.Errorf(
			"api error, code: %d, message: %s, request_id: %s",
			cast.ToInt64(
				mapx.Get(result, "code", 0),
			),
			mapx.GetStr(result, "message"),
			mapx.GetStr(result, "requestID"),
		)
	}

	return result, nil
}
