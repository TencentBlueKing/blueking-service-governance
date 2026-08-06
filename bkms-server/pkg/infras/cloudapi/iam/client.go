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

// Package iam 提供蓝鲸权限中心（bk-iam）网关 HTTP 客户端封装。
//
// 本包是权限管理体系的最底层（L1），仅负责与蓝鲸 IAM 网关的协议交互，不包含
// 任何业务领域语义。上层（如 pkg/bkintegrations/bkiam）会基于本包封装角色 / 范围
// 等更高层的概念。
package iam

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	iamsdk "github.com/TencentBlueKing/iam-go-sdk"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
)

// 蓝鲸 IAM 网关相关常量
const (
	// ConflictCode IAM 网关返回的资源冲突错误码
	ConflictCode = 1902409
	// NeverExpireTimestamp 永不过期的时间戳（伪，其实是 2100.01.01 08:00:00，与权限中心保持一致）
	NeverExpireTimestamp = 4102444800

	// iamGatewayName IAM 网关在 bkapi 中的名称
	iamGatewayName = "bk-iam"
	// defaultIAMStage IAM 网关默认 stage
	defaultIAMStage = "prod"
	// apiNamePlaceholder 蓝鲸网关 API 地址模板中的网关名称占位符
	apiNamePlaceholder = "{api_name}"
	// defaultRequestTimeout IAM 网关请求默认超时时间
	defaultRequestTimeout = 60 * time.Second
)

// UserRoleManager 用户角色管理接口，封装分级管理员与用户组相关操作。
type UserRoleManager interface {
	// CreateGradeManager 创建分级管理员，返回分级管理员 ID。
	CreateGradeManager(
		ctx context.Context,
		name string,
		description string,
		members []string,
		authScopes []types.AuthorizationScope,
	) (*int, error)
	// UpdateGradeManager 更新 id 为 gradeManagerID 的分级管理员信息。
	UpdateGradeManager(
		ctx context.Context,
		gradeManagerID int,
		name string,
		description string,
		authScopes []types.AuthorizationScope,
	) error
	// DeleteGradeManager 删除 id 为 gradeManagerID 的分级管理员。
	DeleteGradeManager(ctx context.Context, gradeManagerID int) error
	// GetGradeManagerByName 根据 name 查询分级管理员 ID。
	GetGradeManagerByName(ctx context.Context, name string) (*int, error)
	// AddGradeManagerMembers 将 members 添加到 id 为 gradeManagerID 的分级管理员成员中。
	AddGradeManagerMembers(ctx context.Context, gradeManagerID int, members []string) error
	// DeleteGradeManagerMembers 从 id 为 gradeManagerID 的分级管理员中删除指定 members。
	DeleteGradeManagerMembers(ctx context.Context, gradeManagerID int, members []string) error

	// CreateUserGroups 在分级管理员下批量创建用户组。
	CreateUserGroups(
		ctx context.Context,
		gradeManagerID int,
		groups ...types.UserGroupParam,
	) ([]types.UserGroup, error)
	// DeleteUserGroup 删除用户组。
	DeleteUserGroup(ctx context.Context, userGroupID int) error
	// GrantUserGroupPolicies 给指定的用户组授权。
	GrantUserGroupPolicies(
		ctx context.Context,
		userGroupID int,
		authScopes []types.AuthorizationScope,
	) error
	// AddUserGroupMembers 添加用户组成员，其中 expiredAt 为过期时间戳。
	AddUserGroupMembers(
		ctx context.Context,
		userGroupID int,
		members []string,
		expiredAt int,
	) error
	// DeleteUserGroupMembers 删除某个用户组的成员。
	DeleteUserGroupMembers(ctx context.Context, userGroupID int, members []string) error
	// ListUserGroupMembers 获取用户组成员。
	ListUserGroupMembers(ctx context.Context, userGroupID int) ([]types.UserMember, error)
}

// Authenticator 鉴权接口，基于 iam-go-sdk 完成对资源 / 动作的鉴权判断。
type Authenticator interface {
	// IsAllowed 查询对某个资源是否有某个操作权限。
	IsAllowed(request iamsdk.Request) (allowed bool, err error)
	// BatchResourceMultiActionsAllowed 对同类多个资源的多个操作进行鉴权。
	BatchResourceMultiActionsAllowed(
		request iamsdk.MultiActionRequest,
		resourcesList []iamsdk.Resources,
	) (results map[string]map[string]bool, err error)
}

// IAMClient 蓝鲸 IAM 网关客户端接口，组合了 UserRoleManager 与 Authenticator。
type IAMClient interface {
	UserRoleManager
	Authenticator
}

// BKIAMClient 蓝鲸 IAM 网关客户端默认实现。
//
// 通过 define.BkApiClient 调用网关 HTTP 接口；通过嵌入的 *iamsdk.IAM
// 提供 IsAllowed / BatchResourceMultiActionsAllowed 等鉴权能力。
type BKIAMClient struct {
	define.BkApiClient
	*iamsdk.IAM
}

// 编译期接口实现检查
var _ IAMClient = (*BKIAMClient)(nil)

// NewIAMClient 创建蓝鲸 IAM 网关客户端。
//
// 客户端的鉴权信息由 config.G.BkApp 提供，网关地址由
// config.G.BkPlatUrls.BkApiUrlTmpl 与 config.G.BkApiStages.BkIAM 共同决定。
func NewIAMClient() (IAMClient, error) {
	authorization, err := json.Marshal(map[string]string{
		"bk_app_code":   config.G.BkApp.Code,
		"bk_app_secret": config.G.BkApp.Secret,
	})
	if err != nil {
		return nil, errors.Wrap(err, "marshal iam authorization")
	}

	apiClient, err := bkapi.NewBkApiClient(
		iamGatewayName,
		bkapi.ClientConfig{
			BkApiUrlTmpl: config.G.BkPlatUrls.BkApiUrlTmpl,
			Stage:        config.G.BkApiStages.BkIAM,
			ClientOptions: []define.BkApiClientOption{
				bkapi.OptSetRequestHeader("x-bkapi-authorization", string(authorization)),
				bkapi.OptJsonResultProvider(),
				bkapi.OptJsonBodyProvider(),
				bkapi.OptTimeout(defaultRequestTimeout),
			},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "new bkapi iam client")
	}

	return newIAMClient(apiClient)
}

// newIAMClient 基于已构造好的 BkApiClient 组装 IAMClient（便于单测注入 mock）。
func newIAMClient(client define.BkApiClient) (IAMClient, error) {
	iamGatewayURL, err := BuildIAMGatewayURL(config.G.BkPlatUrls.BkApiUrlTmpl, config.G.BkApiStages.BkIAM)
	if err != nil {
		return nil, err
	}

	return &BKIAMClient{
		BkApiClient: client,
		IAM: iamsdk.NewAPIGatewayIAM(
			config.G.BkIAMSystemIDs.Bkms,
			config.G.BkApp.Code,
			config.G.BkApp.Secret,
			iamGatewayURL,
		),
	}, nil
}

// BuildIAMGatewayURL 将 bkapi 模板（apiURLTmpl，例如 http://{api_name}.apigw.example.com）
// 渲染为 iam-go-sdk 需要的具体 IAM 网关地址：
//   - 将模板中的 {api_name} 占位符替换为 bk-iam；
//   - 将 stage 作为子路径拼接到末尾（stage 为空时回退为 prod）。
//
// 该函数被本包内的 IAM 网关客户端使用，也供外部命令（如 IAM 系统模型迁移命令）
// 复用，避免重复实现同一套渲染逻辑。
func BuildIAMGatewayURL(apiURLTmpl, stage string) (string, error) {
	gatewayURL := strings.ReplaceAll(apiURLTmpl, apiNamePlaceholder, iamGatewayName)
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = defaultIAMStage
	}

	parsedURL, err := url.Parse(gatewayURL)
	if err != nil {
		return "", errors.Wrap(err, "parse iam gateway url")
	}
	parsedURL.Path, err = url.JoinPath(parsedURL.Path, stage)
	if err != nil {
		return "", errors.Wrap(err, "join iam gateway stage")
	}
	return parsedURL.String(), nil
}

// executeOperation 统一发起 IAM 网关请求。业务 span 提供精确的操作名，transport 层自动传播 trace context。
func (c *BKIAMClient) executeOperation(ctx context.Context, op define.Operation, result any) error {
	ctx, span := apm.StartClientSpan(ctx, "iam", op.FullName())
	resp, err := op.SetContext(ctx).SetResult(result).Request()
	apm.EndClientSpan(span, resp, &err)
	return err
}
