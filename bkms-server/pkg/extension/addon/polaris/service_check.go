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

package polaris

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	polariscap "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/capability/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/framework/trpc/trpcconf"
)

// serviceCheckTarget 待校验的北极星配置。
type serviceCheckTarget struct {
	// AppID 应用 ID。
	AppID string
	// ConfigName 北极星配置名，用于警告文案。
	ConfigName string
	// PolarisName 北极星服务名。
	PolarisName string
	// EnvNames 逐个校验的环境。
	EnvNames []string
}

// checkServiceName 逐个环境校验北极星服务名是否出现在 tRPC 配置的服务名中，返回不一致或读取失败的警告。
// 应用不支持该能力时不校验。
func checkServiceName(
	ctx context.Context,
	provider appcfg.MountableFileProvider,
	mod runtimemod.Mod[polariscap.ServiceCheckProps],
	target serviceCheckTarget,
) (warnings []string) {
	if !mod.Supported {
		return nil
	}
	for _, envName := range target.EnvNames {
		if warning := checkServiceNameInEnv(ctx, provider, target, envName); warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings
}

// checkServiceNameInEnv 校验单个环境，一致时返回空字符串
func checkServiceNameInEnv(
	ctx context.Context,
	provider appcfg.MountableFileProvider,
	target serviceCheckTarget,
	envName string,
) string {
	serviceNames, err := getTrpcServiceNames(ctx, provider, target.AppID, envName)
	if err != nil {
		if envName == "" {
			return i18n.LocalizeOr(
				ctx,
				"extension.polaris.serviceCheck.readConfigFailed",
				map[string]string{"configName": target.ConfigName, "err": err.Error()},
				fmt.Sprintf("[%s] 读取 tRPC 配置文件失败: %v", target.ConfigName, err),
			)
		}
		return i18n.LocalizeOr(
			ctx,
			"extension.polaris.serviceCheck.readConfigFailedInEnv",
			map[string]string{"configName": target.ConfigName, "envName": envName, "err": err.Error()},
			fmt.Sprintf("[%s] 环境 '%s': 读取 tRPC 配置文件失败: %v", target.ConfigName, envName, err),
		)
	}

	if !lo.Contains(serviceNames, target.PolarisName) {
		serviceNamesStr := fmt.Sprintf("%v", serviceNames)
		if envName == "" {
			return i18n.LocalizeOr(
				ctx,
				"extension.polaris.serviceCheck.serviceNameMismatch",
				map[string]string{
					"configName":   target.ConfigName,
					"polarisName":  target.PolarisName,
					"serviceNames": serviceNamesStr,
				},
				fmt.Sprintf("[%s] 北极星服务名 '%s' 推荐与 tRPC 配置中的服务名 %v 一致",
					target.ConfigName, target.PolarisName, serviceNames),
			)
		}
		return i18n.LocalizeOr(
			ctx,
			"extension.polaris.serviceCheck.serviceNameMismatchInEnv",
			map[string]string{
				"configName":   target.ConfigName,
				"envName":      envName,
				"polarisName":  target.PolarisName,
				"serviceNames": serviceNamesStr,
			},
			fmt.Sprintf("[%s] 环境 '%s': 北极星服务名 '%s' 推荐与 tRPC 配置中的服务名 %v 一致",
				target.ConfigName, envName, target.PolarisName, serviceNames),
		)
	}
	return ""
}

// getTrpcServiceNames 获取指定应用和环境下 tRPC 配置文件中的所有服务名
func getTrpcServiceNames(
	ctx context.Context,
	provider appcfg.MountableFileProvider,
	appID, envName string,
) ([]string, error) {
	frameworkFile, err := provider.GetFrameworkMountableFile(ctx, appID, envName)
	if err != nil {
		return nil, err
	}
	return trpcconf.ParseServiceNames(frameworkFile.Content)
}
