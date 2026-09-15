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

// Package bscp 提供蓝鲸 bscp 服务的 API 调用封装
package bscp

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var (
	// stubBizs 本地开发时返回的固定业务列表
	stubBizs = []Biz{
		{ID: "100001", Name: "stub-biz-a"},
		{ID: "100002", Name: "stub-biz-b"},
	}

	// stubServices 本地开发时返回的固定服务列表
	stubServices = []Service{
		{
			ID:         "1001",
			Name:       "stub-service-file",
			Alias:      "Stub 文件服务",
			Desc:       "本地开发用 stub 文件型服务",
			ConfigType: ConfigTypeFile,
			DataType:   DataTypeAny,
		},
		{
			ID:         "1002",
			Name:       "stub-service-kv",
			Alias:      "Stub 键值服务",
			Desc:       "本地开发用 stub 键值型服务",
			ConfigType: ConfigTypeKV,
			DataType:   DataTypeString,
		},
	}

	// stubVersions 本地开发时返回的固定版本列表
	stubVersions = Versions{
		{
			ID:              "2001",
			Name:            "v1.0.0",
			Desc:            "stub 初始版本",
			IsFullyReleased: true,
			Creator:         "stub-user",
			CreatedAt:       "2024-01-01 00:00:00",
		},
	}
)

// StubApiClient 测试用的 BSCP API 客户端实现，返回模拟数据
type StubApiClient struct {
	user auth.User
}

// NewStub 创建 StubApiClient
func NewStub(user auth.User) *StubApiClient {
	return &StubApiClient{user: user}
}

// ListUserBizs 模拟获取用户有权限的业务列表，返回 stubBizs
func (s *StubApiClient) ListUserBizs(ctx context.Context) ([]Biz, error) {
	log.Infof(ctx, "Stub: ListUserBizs request: user=%s", s.user.ID)
	return stubBizs, nil
}

// GetBiz 模拟获取指定业务信息
func (s *StubApiClient) GetBiz(ctx context.Context, bizID string) (*Biz, error) {
	log.Infof(ctx, "Stub: GetBiz request: bizID=%s", bizID)
	for i := range stubBizs {
		if stubBizs[i].ID == bizID {
			biz := stubBizs[i]
			return &biz, nil
		}
	}
	return &Biz{ID: bizID, Name: fmt.Sprintf("stub-biz-%s", bizID)}, nil
}

// ListBizServices 模拟获取业务下的服务列表，返回 stubServices
func (s *StubApiClient) ListBizServices(ctx context.Context, bizID string) ([]Service, error) {
	log.Infof(ctx, "Stub: ListBizServices request: bizID=%s", bizID)
	return stubServices, nil
}

// GetBizService 模拟获取指定服务
func (s *StubApiClient) GetBizService(ctx context.Context, bizID, svcID string) (*Service, error) {
	log.Infof(ctx, "Stub: GetBizService request: bizID=%s, svcID=%s", bizID, svcID)
	for i := range stubServices {
		if stubServices[i].ID == svcID {
			svc := stubServices[i]
			return &svc, nil
		}
	}
	return &Service{
		ID:         svcID,
		Name:       fmt.Sprintf("stub-service-%s", svcID),
		Alias:      "Stub 服务",
		ConfigType: ConfigTypeFile,
		DataType:   DataTypeAny,
	}, nil
}

// ListServiceVersions 模拟获取服务下版本列表，返回 stubVersions
func (s *StubApiClient) ListServiceVersions(ctx context.Context, bizID, svcID string) (Versions, error) {
	log.Infof(ctx, "Stub: ListServiceVersions request: bizID=%s, svcID=%s", bizID, svcID)
	return stubVersions, nil
}

// ListServiceConfigs 模拟获取服务下的配置项列表
func (s *StubApiClient) ListServiceConfigs(ctx context.Context, bizID, svcID, versionID string) ([]Config, error) {
	log.Infof(ctx, "Stub: ListServiceConfigs request: bizID=%s, svcID=%s, versionID=%s", bizID, svcID, versionID)
	return []Config{
		NewKeyValue("app_name", "stub-app", "应用名称"),
		NewKeyValue("log_level", "info", "日志级别"),
	}, nil
}

// GetServiceConfig 模拟获取指定的配置项
func (s *StubApiClient) GetServiceConfig(ctx context.Context, bizID, svcID, versionID, id string) (Config, error) {
	log.Infof(
		ctx, "Stub: GetServiceConfig request: bizID=%s, svcID=%s, versionID=%s, id=%s",
		bizID, svcID, versionID, id,
	)
	return NewKeyValue("stub-config", "stub-value", "stub 配置项"), nil
}
