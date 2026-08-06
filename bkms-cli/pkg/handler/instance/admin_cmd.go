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

// Package instance 管理命令，根据 appType 分流处理 Trpc/Taf 管理命令。
package instance

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// ExecAdminCmdOptions 执行管理命令的统一参数
type ExecAdminCmdOptions struct {
	// Trpc 专用
	Method string // HTTP 方法 (GET/POST/PUT)
	URL    string // 访问的 URL
	Params map[string]string
	Body   string
	// Taf 专用
	Command string // 执行的命令（如 "taf.viewversion"）
	// 通用
	InstanceIDs []string
}

// ListTrpcAdminCmds 查询 Trpc 管理命令列表
func ListTrpcAdminCmds(ctx context.Context, appID, envName string, instanceIDs []string) ([]string, error) {
	return client.New().ListTrpcAdminCmds(ctx, appID, envName, instanceIDs)
}

// ExecAdminCmd 执行管理命令，根据 appType 自动路由
func ExecAdminCmd(
	ctx context.Context,
	workspaceID, appID, envName string,
	opts ExecAdminCmdOptions,
) ([]client.AdminCmdResult, error) {
	app, err := client.New().GetAppMinimal(ctx, workspaceID, appID)
	if err != nil {
		return nil, err
	}

	switch app.Type {
	case constant.AppTypeTrpc:
		return client.New().ExecuteTrpcAdminCmd(ctx, appID, envName, client.ExecuteTrpcAdminCmdOptions{
			InstanceIDs: opts.InstanceIDs,
			Method:      opts.Method,
			URL:         opts.URL,
			Params:      opts.Params,
			Body:        opts.Body,
		})
	case constant.AppTypeTaf:
		return client.New().ExecuteTafAdminCmd(ctx, appID, envName, client.ExecuteTafAdminCmdOptions{
			InstanceIDs: opts.InstanceIDs,
			Command:     opts.Command,
		})
	default:
		return nil, errors.Errorf("unsupported app type for admin cmd: %s", app.Type)
	}
}
