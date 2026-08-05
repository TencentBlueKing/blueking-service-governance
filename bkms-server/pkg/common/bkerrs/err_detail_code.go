package bkerrs

// ErrDetailCode 错误详情码，用于标记某类具体的错误，如删除工作空间失败、创建环境失败等，推荐开发者按需定义
// 重要：在业务逻辑中所有 bkerrs.NewDetail() 调用，都应使用该处枚举值之一，以便后续统一管理，单元测试则不做限制！
type ErrDetailCode string

const (
	// ErrDetailCodeIAMNoPermission 无 IAM 权限
	ErrDetailCodeIAMNoPermission ErrDetailCode = "IAM_NO_PERMISSION"

	// ErrDetailCodeAPMConfigMissing APM 配置缺失
	ErrDetailCodeAPMConfigMissing ErrDetailCode = "APM_CONFIG_MISSING"

	// ErrDetailCodeTrpcAdminPrecheckFailed trpc admin 配置预检查失败
	ErrDetailCodeTrpcAdminPrecheckFailed ErrDetailCode = "TRPC_ADMIN_PRECHECK_FAILED"

	// ErrDetailCodeAppConfigFileVersionConflict 应用配置文件版本冲突
	ErrDetailCodeAppConfigFileVersionConflict ErrDetailCode = "APP_CONFIG_FILE_VERSION_CONFLICT"

	// ErrDetailCodeNotFullyReleased BSCP 服务未全量发布
	ErrDetailCodeNotFullyReleased ErrDetailCode = "BSCP_NOT_FULLY_RELEASED"

	// ErrDetailCodeImageRepositoryAuthRequired 删除镜像时镜像仓库鉴权缺失或失败
	ErrDetailCodeImageRepositoryAuthRequired ErrDetailCode = "IMAGE_REPOSITORY_AUTH_REQUIRED"

	// ErrDetailCodeBSCPNoPermission BSCP 服务无权限
	ErrDetailCodeBSCPNoPermission ErrDetailCode = "BSCP_NO_PERMISSION"

	// ErrDetailCodeComponentNotInstalled 环境所在集群未安装所需的组件（通过 module 区分具体组件）
	ErrDetailCodeComponentNotInstalled ErrDetailCode = "COMPONENT_NOT_INSTALLED"

	// ErrDetailCodeBuildLogUnavailable 构建日志已过期或已清理
	ErrDetailCodeBuildLogUnavailable ErrDetailCode = "BUILD_LOG_UNAVAILABLE"
)
