package polaris

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// CollectConfigWarnings 为单个 PolarisConfig 收集校验 warnings。
// 校验逻辑：
// 1. 检查应用是否为 tRPC 类型，不是则跳过
// 2. 对 ScopeEnvNames 中每个环境分别校验
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

	for _, envName := range config.ScopeEnvNames {
		warning := validateServiceNameInEnv(ctx, config, appConfigFileStore, envName)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return warnings
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
