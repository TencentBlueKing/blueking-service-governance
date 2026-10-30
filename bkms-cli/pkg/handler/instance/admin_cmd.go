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

// Package instance 管理命令，根据应用框架（framework）分流处理 Trpc/Taf 管理命令。
package instance

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// ListTrpcAdminCmds 查询 Trpc 管理命令列表
func ListTrpcAdminCmds(ctx context.Context, appID, envName string, instanceIDs []string) ([]string, error) {
	return client.New().ListTrpcAdminCmds(ctx, appID, envName, instanceIDs)
}

// ExecAdminCmd 执行管理命令：校验类型必填参数 → 按 framework 路由到 Trpc/Taf API。
// trpc、taf 与 bkmsApp 应用都按 framework 判断；blank 框架没有管理命令。
func ExecAdminCmd(
	ctx context.Context,
	workspaceID, appID, envName string,
	opts ExecAdminCmdOptions,
) ([]client.AdminCmdResult, error) {
	app, err := client.New().GetAppMinimal(ctx, workspaceID, appID)
	if err != nil {
		return nil, errors.Wrap(err, "get app info")
	}

	switch framework := client.FrameworkOf(app.Type, app.Framework); framework {
	case constant.FrameworkTrpc:
		return execTrpcAdminCmd(ctx, appID, envName, opts)
	case constant.FrameworkTaf:
		return execTafAdminCmd(ctx, appID, envName, opts)
	default:
		return nil, errors.Errorf("app %s (framework %q) does not support admin commands", appID, framework)
	}
}

func execTrpcAdminCmd(
	ctx context.Context, appID, envName string, opts ExecAdminCmdOptions,
) ([]client.AdminCmdResult, error) {
	if opts.Method == "" {
		return nil, errors.New("method is required for Trpc app")
	}
	if opts.URL == "" {
		return nil, errors.New("url is required for Trpc app")
	}
	if opts.ParamsJSON != "" {
		if err := json.Unmarshal([]byte(opts.ParamsJSON), &opts.Params); err != nil {
			return nil, errors.Wrap(err, "parse params JSON")
		}
	}
	return client.New().ExecuteTrpcAdminCmd(ctx, appID, envName, client.ExecuteTrpcAdminCmdOptions{
		InstanceIDs: opts.InstanceIDs,
		Method:      opts.Method,
		URL:         opts.URL,
		Params:      opts.Params,
		Body:        opts.Body,
	})
}

func execTafAdminCmd(
	ctx context.Context, appID, envName string, opts ExecAdminCmdOptions,
) ([]client.AdminCmdResult, error) {
	if opts.Command == "" {
		return nil, errors.New("command is required for Taf app")
	}
	return client.New().ExecuteTafAdminCmd(ctx, appID, envName, client.ExecuteTafAdminCmdOptions{
		InstanceIDs: opts.InstanceIDs,
		Command:     opts.Command,
	})
}
