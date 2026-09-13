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

// Package plainfiles 提供 plain 配置文件在运行时（workload）层的集成能力。
//
// BuildPlainConfigFiles 是面向 plain 配置文件的上层包装，由 Builder 公共步骤调用。
// 内部的 env var 收集与模板渲染由 runtimerender.RenderConfigContents 公共管线完成。
package plainfiles

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
)

// BuildPlainConfigFiles 获取当前环境实际生效的 plain 配置文件，
// 完成挂载路径校验后调用 RenderConfigContents 收集环境变量引用并渲染模板变量。
//
// 该函数由 Builder 公共步骤调用，独立于任何框架 plugin，因此所有 workload 类型
// （trpc、taf、standard）都能自动获得 plain 文件支持。
func BuildPlainConfigFiles(
	ctx context.Context,
	cfgProvider appcfg.MountableFileProvider,
	appID string,
	envName string,
	envVars map[string]string,
	collector *envvarrefs.Collector,
) ([]runtimerender.ConfigFileParams, error) {
	if cfgProvider == nil {
		return nil, nil
	}

	plainFiles, err := cfgProvider.ListPlainMountableFiles(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "listing env plain config files")
	}

	for _, item := range plainFiles {
		if item.MountDir == "" {
			return nil, errors.Errorf("plain config file %s has empty mount dir", item.Name)
		}
	}

	return runtimerender.RenderConfigContents(plainFiles, envVars, collector)
}
