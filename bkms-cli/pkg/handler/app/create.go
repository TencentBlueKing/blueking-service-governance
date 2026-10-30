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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// CreateApp 创建应用，读取 YAML 文件并调用后端 API
// 返回创建成功的应用信息，由上层决定如何输出
func CreateApp(ctx context.Context, workspaceID, specFile string) (*client.AppMinimal, error) {
	if _, err := os.Stat(specFile); err != nil {
		return nil, errors.Wrapf(err, "app spec file %s not found", specFile)
	}
	file, err := os.ReadFile(specFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read app spec file %s failed", specFile)
	}
	spec := new(AppCreateSpec)
	if err = yaml.Unmarshal(file, spec); err != nil {
		return nil, errors.Wrapf(err, "parse app spec file %s failed, please check YAML syntax", specFile)
	}
	if err = spec.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate app spec")
	}

	// 应用 ID 固定由服务端返回的后缀与 name 拼接生成，不允许用户指定
	suffix, err := client.New().GetAppIDAutoSuffix(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get app id auto suffix")
	}
	appID, err := buildAppID(spec.Name, suffix)
	if err != nil {
		return nil, err
	}

	// 旧的 type: trpc/taf 写法在 Validate 中已归一化为 framework + language
	if spec.Type == constant.AppTypeTrpc || spec.Type == constant.AppTypeTaf {
		console.Tips("'type: %s' is deprecated, please use 'framework: %s' with 'frameworkConfigFile' instead",
			spec.Type, spec.Framework)
	}

	// 构建后端 API 请求体
	legacyBody, bkmsBody, err := buildCreateAppRequest(appID, spec)
	if err != nil {
		return nil, errors.Wrap(err, "build create app request")
	}

	// 调用后端 API 创建应用
	cli := client.New()
	if bkmsBody != nil {
		return cli.CreateBKMSApp(ctx, workspaceID, bkmsBody)
	}
	return cli.CreateApp(ctx, workspaceID, legacyBody)
}
