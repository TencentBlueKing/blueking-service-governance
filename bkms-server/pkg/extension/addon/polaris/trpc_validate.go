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

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// CollectConfigWarnings 为单个 PolarisConfig 收集校验 warnings。
// 校验逻辑：
// 1. 检查应用是否为 tRPC 类型，不是则跳过
// 2. 根据 ScopeType 确定校验环境：
//   - global/空 → 使用应用级别配置（envName=""）
//   - environment → 对 ScopeEnvNames 中每个环境都校验
//
// 3. 对每个环境调用 appcfg.GetTrpcServiceNames 获取服务名列表
// 4. 检查 PolarisName 是否在服务名列表中，不在则生成 warning
func CollectConfigWarnings(
	ctx context.Context,
	appModelStore appmodel.AppModelStore,
	appConfigFileStore appcfg.AppConfigFileStore,
	config *PolarisConfig,
) (warnings []string) {
	// 获取应用模型
	model, err := appModelStore.GetAppModel(ctx, config.AppID)
	if err != nil {
		log.Warnf(ctx, "get app model for polaris validation failed: app=%s, err=%v", config.AppID, err)
		return nil
	}

	// 仅对 tRPC 类型的应用进行校验
	if model.Workload.Type != appmodel.WorkloadTypeTrpc {
		return nil
	}

	// 确定需要校验的环境列表
	envNames := getValidationEnvNames(config)

	for _, envName := range envNames {
		warning := validateServiceNameInEnv(ctx, config, appConfigFileStore, envName)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings
}

// getValidationEnvNames 根据 PolarisConfig 的 ScopeType 确定需要校验的环境列表
func getValidationEnvNames(config *PolarisConfig) []string {
	switch config.ScopeType {
	case component.ScopeTypeEnvironment:
		return config.ScopeEnvNames
	default:
		// global 或空类型，使用应用级别配置（envName=""）
		return []string{appcfg.EnvNameDefault}
	}
}

// validateServiceNameInEnv 校验指定环境下 PolarisConfig 的服务名是否与 tRPC 配置匹配
func validateServiceNameInEnv(
	ctx context.Context,
	config *PolarisConfig,
	appConfigFileStore appcfg.AppConfigFileStore,
	envName string,
) string {
	serviceNames, err := appcfg.GetTrpcServiceNames(ctx, appConfigFileStore, config.AppID, envName)
	if err != nil {
		if envName == "" {
			return fmt.Sprintf("[%s] 读取 tRPC 配置文件失败: %v", config.Name, err)
		}
		return fmt.Sprintf("[%s] 环境 '%s': 读取 tRPC 配置文件失败: %v", config.Name, envName, err)
	}

	// 检查 PolarisName 是否在 tRPC 服务名列表中
	if !lo.Contains(serviceNames, config.PolarisName) {
		if envName == "" {
			return fmt.Sprintf("[%s] 北极星服务名 '%s' 推荐与 tRPC 配置中的服务名 %v 一致",
				config.Name, config.PolarisName, serviceNames)
		}
		return fmt.Sprintf("[%s] 环境 '%s': 北极星服务名 '%s' 推荐与 tRPC 配置中的服务名 %v 一致",
			config.Name, envName, config.PolarisName, serviceNames)
	}
	return ""
}
