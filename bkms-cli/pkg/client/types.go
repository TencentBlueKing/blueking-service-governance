// Package client provides API client interface and core domain models
package client

import (
	"context"
	"io"
)

// Client API 调用入口
type Client interface {
	// ---------- 鉴权 ----------

	// ValidateAccessToken 通过 AccessToken 获取用户名信息，校验 token 有效性
	ValidateAccessToken(accessToken string) (string, error)
	// ExchangeBkTicketForToken 使用 bk_ticket 兑换 access_token
	ExchangeBkTicketForToken(username, bkTicket string) (string, error)

	// ---------- 工作空间 ----------

	// ListWorkspaces 获取工作空间列表
	ListWorkspaces(ctx context.Context, keyword string) ([]Workspace, error)
	// GetWorkspace 获取工作空间详情
	GetWorkspace(ctx context.Context, id string) (*Workspace, error)

	// ---------- 环境 ----------

	// ListEnvs 获取环境列表
	ListEnvs(ctx context.Context, workspaceID string) ([]Env, error)

	// ---------- 应用 ----------

	// GetAppIDAutoSuffix 获取应用 ID 自动后缀
	GetAppIDAutoSuffix(ctx context.Context) (string, error)
	// ListApps 获取应用列表
	ListApps(ctx context.Context, workspaceID string) ([]AppMinimal, error)
	// GetAppMinimal 获取应用，过滤 ListApps 结果
	GetAppMinimal(ctx context.Context, workspaceID, appID string) (*AppMinimal, error)
	// CreateApp 创建应用
	CreateApp(ctx context.Context, workspaceID string, body any) (*AppMinimal, error)
	// CreateAppBuild 执行应用构建
	CreateAppBuild(ctx context.Context, appID string, opts BuildOptions) error
	// ListBuildRecords 获取最近的应用构建记录（10 条）
	ListBuildRecords(ctx context.Context, appID, keyword string) ([]BuildRecord, error)
	// ListAppImages 获取应用镜像列表
	ListAppImages(ctx context.Context, appID, keyword string) ([]Image, error)
	// GetEnvEffectiveDevMode 获取应用在某个环境下实际生效的开发模式配置
	GetEnvEffectiveDevMode(ctx context.Context, appID, envName string) (*DevModeConfig, error)

	// ---------- 应用配置文件 ----------

	// ListAppConfigFiles 获取应用配置文件列表
	ListAppConfigFiles(ctx context.Context, appID, envName string) ([]AppConfigFile, error)
	// GetAppConfigFileDetails 获取应用配置文件详情
	GetAppConfigFileDetails(ctx context.Context, appID, fileID string) (*AppConfigFileDetails, error)
	// ListAppConfigFileVersions 获取应用配置文件历史版本列表
	ListAppConfigFileVersions(
		ctx context.Context, appID string, opts ListAppConfigFileVersionsOptions,
	) (*PaginatedAppConfigFileVersions, error)
	// GetAppConfigFileVersion 获取应用配置文件某个历史版本详情
	GetAppConfigFileVersion(ctx context.Context, appID, versionID string) (*AppConfigFileVersion, error)
	// DeleteAppConfigFileVersion 删除应用配置文件某个历史版本
	DeleteAppConfigFileVersion(ctx context.Context, appID, versionID string) error
	// RollbackAppConfigFileVersion 回滚到应用配置文件某个历史版本
	RollbackAppConfigFileVersion(
		ctx context.Context, appID, versionID string, opts RollbackAppConfigFileVersionOptions,
	) (*AppConfigFile, error)
	// UpdateAppConfigFileContent 更新应用配置文件 Content
	UpdateAppConfigFileContent(
		ctx context.Context, appID, fileID string, opts AppConfigFileContentOptions,
	) (*AppConfigFileContentUpdateResult, error)
	// UpdateAppConfigFileOverlayContent 更新应用配置文件 OverlayContent
	UpdateAppConfigFileOverlayContent(
		ctx context.Context, appID, fileID string, opts AppConfigFileContentOptions,
	) (*AppConfigFileContentUpdateResult, error)

	// ---------- 部署 ----------

	// CreateAppHelmDeploy 执行 Helm 应用部署
	CreateAppHelmDeploy(ctx context.Context, appID, envName string, opts HelmDeployOptions) error
	// ListHelmDeployRecords 获取最近的应用 Helm 部署记录（10 条）
	ListHelmDeployRecords(
		ctx context.Context, appID, envName, trafficLaneName, keyword string,
	) ([]HelmDeployRecord, error)

	// --- Trpc ---
	// CreateAppTrpcDeploy 执行 Trpc 应用部署
	// image 镜像，完整镜像地址（必填）。
	// replicas 副本数/实例数量，发布实例数量必须大于或等于 1（必填）。
	CreateAppTrpcDeploy(ctx context.Context, appID, envName string, opts AppModelDeployOptions) error
	// ListTrpcDeployRecords 获取最近的应用 Trpc 部署记录
	// appID,envName 应用/环境 必填
	ListTrpcDeployRecords(
		ctx context.Context, appID, envName, keyword, trafficLaneName string,
	) ([]AppModelDeployRecord, error)

	// --- TAF ---
	// CreateAppTafDeploy 执行 TAF 应用部署
	CreateAppTafDeploy(ctx context.Context, appID, envName string, opts AppModelDeployOptions) error
	// ListTafDeployRecords 获取最近的应用 TAF 部署记录
	ListTafDeployRecords(
		ctx context.Context, appID, envName, keyword, trafficLaneName string,
	) ([]AppModelDeployRecord, error)

	// --- 通用 ---
	// GrayscaleUpdateInstance 灰度更新 AppModel 实例
	GrayscaleUpdateInstance(
		ctx context.Context, appID, envName, imageTag string, instanceIDs []string, updateStrategy string) error
	// BatchUpdateInstance 全量更新 AppModel 实例
	BatchUpdateInstance(ctx context.Context, appID, envName, imageTag, updateStrategy string) error
	// ListAppInstances 获取应用实例列表
	ListAppInstances(
		ctx context.Context,
		appID, envName string,
		opts ListAppInstancesOptions,
	) (*PaginatedInstances, error)

	// --- 管理命令 ---
	// ListTrpcAdminCmds 查询 Trpc 管理命令列表
	ListTrpcAdminCmds(ctx context.Context, appID, envName string, instanceIDs []string) ([]string, error)
	// ExecuteTrpcAdminCmd 执行 Trpc 管理命令
	ExecuteTrpcAdminCmd(
		ctx context.Context,
		appID, envName string,
		opts ExecuteTrpcAdminCmdOptions,
	) ([]AdminCmdResult, error)
	// ExecuteTafAdminCmd 执行 Taf 管理命令
	ExecuteTafAdminCmd(
		ctx context.Context,
		appID, envName string,
		opts ExecuteTafAdminCmdOptions,
	) ([]AdminCmdResult, error)

	// ---------- 北极星配置 ----------

	// ListAppPolarisConfigs 获取应用的北极星配置列表
	ListAppPolarisConfigs(ctx context.Context, appID string) ([]PolarisConfig, error)
	// CreateAppPolarisConfig 创建应用的北极星配置
	CreateAppPolarisConfig(ctx context.Context, appID string, body any) (string, error)
	// DeleteAppPolarisConfig 删除应用的北极星配置
	DeleteAppPolarisConfig(ctx context.Context, appID, configName string) error
	// PatchAppPolarisConfig 更新应用的北极星配置（部分更新）
	PatchAppPolarisConfig(ctx context.Context, appID, configName string, body any) error

	// ---------- AppSpec ----------

	// GetAppDetail 获取应用详情（包含类型和启动命令）
	GetAppDetail(ctx context.Context, appID string) (*AppDetail, error)

	// GetAppSpecDefaultSection 获取应用默认 section 配置，result 传入具体类型指针接收结果
	GetAppSpecDefaultSection(ctx context.Context, appID string, section AppSpecSectionName, result any) error
	// GetAppSpecEnvEffectiveSection 获取应用环境生效 section 配置，result 传入具体类型指针接收结果
	GetAppSpecEnvEffectiveSection(
		ctx context.Context,
		appID, envName string,
		section AppSpecSectionName,
		result any,
	) error

	// SetAppSpecDefaultSection 设置应用默认 section 配置
	SetAppSpecDefaultSection(ctx context.Context, appID string, section AppSpecSectionName, body any) error
	// SetAppSpecEnvSection 设置应用环境 section 覆盖配置
	SetAppSpecEnvSection(ctx context.Context, appID, envName string, section AppSpecSectionName, body any) error
	// DeleteAppSpecEnvSection 删除应用环境 section 覆盖配置（恢复默认）
	DeleteAppSpecEnvSection(ctx context.Context, appID, envName string, section AppSpecSectionName) error
	// UpdateAppStartCommand 更新应用启动命令
	UpdateAppStartCommand(ctx context.Context, appID, appType string, body any) error

	// --- 实例端口转发 ---
	// CheckPortForwardPermission 预检 port-forward 权限（仅当 403 时返回错误）。
	CheckPortForwardPermission(
		ctx context.Context,
		appID, envName, instanceID string,
		remotePort, localPort int,
	) error
	// OpenPortForwardTunnel 打开应用实例端口转发隧道。
	OpenPortForwardTunnel(
		ctx context.Context,
		appID, envName string,
		opts PortForwardTunnelOptions,
	) (io.ReadWriteCloser, error)
}
