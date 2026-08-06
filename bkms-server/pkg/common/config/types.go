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

package config

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
)

// --------------------------- 蓝鲸平台接入 ---------------------------

// BkAppConfig 蓝鲸应用配置（可从开发者中心获取）
// 该配置将用于在调用蓝鲸 API 时进行身份认证
type BkAppConfig struct {
	Code   string
	Secret string
}

// AccountConfig 是和用户账号鉴权、用户 Token 获取相关的配置项集合
type AccountConfig struct {
	// AuthBaseURL 是 auth 服务的基础 URL，通常由蓝鲸网关服务提供，usertoken 令牌相关功能使用
	AuthBaseURL string `validate:"required,url"`
	// LoginURL 是蓝鲸登录服务的基础 URL，auth 令牌认证、usertoken 拼接登录地址时会用到
	LoginURL string `validate:"required,url"`
	// AuthEnvName 是用户 token 使用的环境，不同环境的 token 互相隔离，默认值为 "prod"
	AuthEnvName string
	// BackendType 是用户认证使用的后端类型，比如 bk_ticket 或 bk_token，默认值为 "bk_token"
	BackendType string `validate:"omitempty,oneof=bk_ticket bk_token"`
}

// BkPlatUrlsConfig 蓝鲸平台地址配置
type BkPlatUrlsConfig struct {
	// 蓝鲸网关 API 地址模板，主要用于生成第三方网关的 Base URL
	// 格式如：https://{api_name}.apigw.example.com
	// 其中的 {api_name} 会由 apigw sdk 替换为实际的 API 网关名称
	BkApiUrlTmpl string
	// 组件 API 基础地址，用于调用蓝鲸组件 API（如：cmsi、tof 等）
	CompApiBaseUrl string
}

// BkApiStagesConfig 蓝鲸 API 版本信息，不指定时均默认为 "prod"
type BkApiStagesConfig struct {
	BSCP        string
	BkCI        string
	BCS         string
	KubeInsight string
	BkCC        string
	BkHCM       string
	BkIAM       string
	BkDBM       string
}

// BkIAMSystemIDsConfig 接入蓝鲸权限中心（IAM）时使用的各业务系统 ID 配置
type BkIAMSystemIDsConfig struct {
	// Bkms 是 bkms-server 自身在权限中心注册的系统 ID
	Bkms string
	// BkCI 是蓝盾在权限中心注册的系统 ID
	BkCI string
	// BCS 是蓝鲸容器服务在权限中心注册的系统 ID
	BCS string
	// BkMonitor 是蓝鲸监控在权限中心注册的系统 ID
	BkMonitor string
	// BkLog 是蓝鲸日志在权限中心注册的系统 ID
	BkLog string
	// BkRepo 是蓝鲸制品库在权限中心注册的系统 ID
	BkRepo string
	// BSCP 是蓝鲸服务配置平台在权限中心注册的系统 ID
	BSCP string
	// BkCC 是蓝鲸配置平台在权限中心注册的系统 ID
	BkCC string
}

// --------------------------- 外部依赖服务（蓝鲸 SaaS & 公司系统） ---------------------------

// BCSConfig BCS 配置
type BCSConfig struct {
	BaseUrl string
	Token   string
}

// BKCIProjInitConfig 蓝盾项目初始化默认配置
type BKCIProjInitConfig struct {
	Type        int
	Description string
}

// BKCIPipelineTmplConfig 蓝盾流水线模板
type BKCIPipelineTmplConfig struct {
	BaseDir string
	// BuilderImageCode 蓝盾流水线模板使用的构建机镜像标识，为空时由模板加载逻辑决定默认值
	BuilderImageCode string
	// BuilderImageVersion 蓝盾流水线模板使用的构建机镜像版本，为空时由模板加载逻辑决定默认值
	BuilderImageVersion string
}

// BKCIConfig 蓝盾配置
type BKCIConfig struct {
	PipelineTmpl BKCIPipelineTmplConfig
}

// InitBKRepoConfig 蓝盾制品库仓库初始化配置
// 注意：Type 不应该有重复的（目前 DOCKER + HELM）
type InitBKRepoConfig struct {
	Name         string
	Type         string
	IsPublic     bool
	Description  string
	EndpointTmpl string
}

// BKRepoConfig 蓝盾制品库配置
type BKRepoConfig struct {
	BaseUrl string
	// 管理账号配置，需由制品库管理员提供
	Username string
	Password string
	// 仓库初始化配置
	InitRepos []InitBKRepoConfig
}

// GenRepoEndpoint 获取制品库仓库访问地址（镜像源 / Helm 仓库）
func (c *BKRepoConfig) GenRepoEndpoint(projectID, repoType string) (string, error) {
	for _, cfg := range c.InitRepos {
		if cfg.Type == repoType {
			return fmt.Sprintf(cfg.EndpointTmpl, projectID, cfg.Name), nil
		}
	}
	return "", errors.Errorf("%s repository not found", repoType)
}

// BkMonitorConfig 蓝鲸监控配置
type BkMonitorConfig struct {
	// Endpoint 网关地址
	Endpoint string
	// GatewayEndpoint 新版 bk-monitor 网关地址，供新增接口（如 user group）使用
	GatewayEndpoint string

	// APMEndpoint 蓝鲸监控 APM gRPC 上报地址，APMHttpEndpoint 为空时作为兼容配置使用
	// 带 http/https scheme 时仍会兼容走 OTLP HTTP exporter，否则走 OTLP gRPC exporter
	// FIXME: 后续要切成 k8s svc 地址，待蓝鲸监控完成整体链路迁移
	APMEndpoint string

	// APMHttpEndpoint 蓝鲸监控 APM HTTP 上报地址
	// bkms-server 自身 trace 优先使用该地址走 OTLP HTTP exporter，上报给业务应用时也作为 HTTP 采集地址下发
	APMHttpEndpoint string

	// APMToken 蓝鲸监控 APM 上报 Token
	APMToken string

	// APMServiceName 蓝鲸监控 APM 服务名称，为空时默认使用 bkms.${cobra 子命令名}（如 bkms.webserver、bkms.worker）
	APMServiceName string
}

// BSCPConfig BSCP 服务配置
type BSCPConfig struct {
	// FeedAddr BSCP 服务订阅地址（Feed Server 地址）
	FeedAddr string
}

// TxCMDBConfig Tx CMDB 配置
type TxCMDBConfig struct {
	// BaseUrl 服务地址
	BaseUrl string

	// AppID 应用 ID
	AppID string

	// AppKey 应用密钥（用于签名）
	AppKey string
}

// PolarisConfig 北极星 SDK 配置
type PolarisConfig struct {
	// 当两者都配置时优先使用 JoinPoint, 两者都空置时使用 SDK 的内置接入点

	// Address 北极星后台服务 IP:Port 地址（逗号分隔），与 JoinPoint 二选一
	Address string
	// JoinPoint 北极星接入点(如 default)，与 Address 二选一
	JoinPoint string
}

// --------------------------- 基础设施依赖 ---------------------------

// MongoConfig 数据库配置
type MongoConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

// GetURI 返回 MongoDB 的连接 URI，以供 driver 使用
func (c MongoConfig) GetURI() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%s", c.Username, url.QueryEscape(c.Password), c.Host, c.Port)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host            string
	Port            string
	DB              int
	Password        string
	DialTimeout     int
	ReadTimeout     int
	WriteTimeout    int
	PoolSize        int
	MinIdleConns    int
	ConnMaxIdleTime int
}

// RabbitMQConfig RabbitMQ 配置
type RabbitMQConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Vhost    string
	Queue    string
	Prefetch int
}

// GetURI 返回 RabbitMQ URI
func (c RabbitMQConfig) GetURI() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", c.Username, url.QueryEscape(c.Password), c.Host, c.Port, c.Vhost)
}

// AsynqConfig 通用异步任务框架(taskq)的完整配置
// 各任务可在代码中用 asynq 原生 Option(asynq.MaxRetry/asynq.Timeout 等)覆盖部分默认值。
// 注意: 当前 server 仅消费 Queue 字段指定的单一队列, 不支持任务级队列覆盖。
type AsynqConfig struct {
	// Redis taskq 专用的独立 Redis 连接配置
	Redis RedisConfig
	// Concurrency asynq server 的并发 worker 数, <=0 时由 asynq 取默认值。
	Concurrency int
	// Queue 默认队列名, 任务未在代码中指定 asynq.Queue 时的兜底。
	Queue string
	// MaxRetry 默认最大重试次数, 任务未在代码中指定 asynq.MaxRetry 时的兜底。
	MaxRetry int
	// RetryInterval "进行中"类错误(ErrFixedRetry)的固定重试间隔(秒), 全局生效。
	RetryInterval int
}

// --------------------------- 服务运行时 ---------------------------

// HTTPServerConfig 业务 HTTP Server 配置
type HTTPServerConfig struct {
	// Address HTTP Server 监听地址
	Address string `validate:"required"`
	// Port HTTP Server 监听端口
	// 端口上限 65535 对应包内常量 MaxPortNumber，struct tag 受语法限制无法引用常量，此处需与该常量保持一致
	Port uint `validate:"required,gt=0,lte=65535"`
	// Mode Gin 运行模式，支持 release / debug / test，默认为 "release"
	Mode string `validate:"omitempty,oneof=release debug test"`
	// ReadHeaderTimeout HTTP Server 读取请求头超时时间，单位：秒
	ReadHeaderTimeout int
	// ReadTimeout HTTP Server 读取完整请求超时时间，单位：秒
	ReadTimeout int
	// WriteTimeout HTTP Server 写响应超时时间，单位：秒
	WriteTimeout int
	// IdleTimeout HTTP Server keep-alive 空闲连接超时时间，单位：秒
	IdleTimeout int
	// ShutdownTimeout 进程退出时等待 HTTP Server 优雅关闭的最长时间，单位：秒
	ShutdownTimeout int
	// EnableSwaggerPath 是否启用 Swagger 文档访问路径（/swagger/*），默认为 false，
	// 仅推荐在开发或测试环境启用
	EnableSwaggerPath bool
}

// MetricsConfig Prometheus Metrics Server 配置
type MetricsConfig struct {
	// Port Metrics HTTP Server 监听端口
	// 端口上限 65535 对应包内常量 MaxPortNumber，struct tag 受语法限制无法引用常量，此处需与该常量保持一致
	Port uint `validate:"required,gt=0,lte=65535"`
}

// LoggingConfig 日志配置。
type LoggingConfig struct {
	// Level 日志级别，支持 debug、info、warn、warning、error。
	Level string
	// HandlerName 日志格式，支持 text 和 json。
	HandlerName string
	// Writers 日志输出目标列表，支持 stdout、stderr、file。
	Writers []LoggingWriterConfig
}

// LoggingWriterConfig 单个日志输出目标配置。
type LoggingWriterConfig struct {
	// WriterName 日志输出目标，支持 stdout、stderr、file。
	WriterName string
	// WriterConfig 日志输出目标配置。
	WriterConfig LoggingWriterFileConfig
}

// LoggingWriterFileConfig file writer 配置。
type LoggingWriterFileConfig struct {
	// Filename 日志文件路径，仅 file writer 生效。
	Filename string
	// MaxSize 单个日志文件容量上限，单位 MB。
	MaxSize int
	// MaxBackups 最大日志文件数，单位：个。
	MaxBackups int
	// MaxAge 最长日志保留天数，单位：天。
	MaxAge int
	// Compress 是否压缩轮转日志。
	Compress bool
}

// EncryptConfig 加密配置
type EncryptConfig struct {
	// 加密密钥
	Secret string
}

// --------------------------- 业务功能配置 ---------------------------

// ClusterAddonConfig 集群插件（Addon）相关配置
type ClusterAddonConfig struct {
	// 内置集群 Addon 定义文件目录路径，为空则跳过加载
	BuiltinAddonDir string
	// Helm 仓库 URL（所有 addon 使用统一仓库）
	HelmRepoURL string
	// Helm 仓库用户名（可选）
	HelmRepoUsername string
	// Helm 仓库密码（可选）
	HelmRepoPassword string
}

// HelmConfig Helm Chart 构建工具链配置
type HelmConfig struct {
	// ToolchainBaseURL Helm, Helmify, Kustomize 等二进制下载基础 URL（流水线中使用，需拼接具体二进制名称）
	ToolchainBaseURL string
	// BuiltinRepoURLTmpl 内置蓝盾制品库 Helm 仓库地址模板，用于拉取 HelmChart（需拼接具体项目 ID）
	BuiltinRepoURLTmpl string
}

// GenBuiltinRepoURL 获取内置蓝盾制品库 Helm 仓库访问地址，用于拉取 Helm Chart
func (c *HelmConfig) GenBuiltinRepoURL(projectID string) (string, error) {
	return fmt.Sprintf(c.BuiltinRepoURLTmpl, projectID), nil
}

// ImageBuildConfig 镜像构建工具链配置
type ImageBuildConfig struct {
	// ToolchainBaseURL 镜像构建工具链下载基础 URL（流水线中使用，需拼接具体工具名称）
	ToolchainBaseURL string
}

// PollConfig 轮询配置
type PollConfig struct {
	// 轮询超时时间（单位：秒）
	Timeout int
	// 轮询间隔（单位：秒）
	Interval int
}

// TaskPollerConfig 轮询器配置
type TaskPollerConfig struct {
	// 部署状态轮询配置
	DeployStatus PollConfig
}

// --------------------------- 开发环境专用 ---------------------------

// DevConfig 包含开发相关的配置项
type DevConfig struct {
	// 用于本地开发测试， 远程服务调用时返回默认 Mock 数据
	UseStubBkCI        bool
	UseStubBkRepo      bool
	UseStubBkCMDB      bool
	UseStubTxCMDB      bool
	UseStubBCS         bool
	UseStubBkHCM       bool
	UseStubBkMonitor   bool
	UseStubKubeInsight bool
	UseStubBSCP        bool
	UseStubDBM         bool
	// 启用后，允许通过请求头直接设置已认证用户。该配置会绕过真实用户认证，
	// 仅可用于本地开发和测试，禁止在生产环境中打开。
	AllowSetUserInHeader bool
	// 启用后，权限管理器（pkg/infras/perm）返回 stub 实现，总是通过所有鉴权请求；
	// 这是唯一的权限管理器桩开关，仅本地开发使用，禁止在生产环境中打开。
	UseStubPerm bool
	// 启用后，不会使用 bcs 集群，而是直接使用本地 kubeconfig 指向的集群
	// 适合在本地开发，不便访问真实集群时使用；不要在生产环境中打开。
	UseKubeConfigCluster bool
	// StubKubeConfigPath 本地 kubeconfig 文件的路径
	// 仅当 UseKubeConfigCluster 的值为 true 时生效
	// 如该项为空值时，会默认使用 ~/.kube/config
	StubKubeConfigPath string
	// AllowSkipNewerDBMigration 启用后，当数据库记录的 migration 版本高于本二进制内嵌的最大版本时，
	// migrate up 会静默跳过而不报错。用于测试环境：开发分支可能已执行更高 seq 的 migration，
	// 而 main 尚未合入对应文件，导致 main 发布时 golang-migrate 找不到该版本而失败。
	// 禁止在生产环境开启：跳过只保证本次发布不失败，不会补执行本二进制内未执行的 migration
	AllowSkipNewerDBMigration bool
}

// Config SaaS 配置
type Config struct {
	// --------------------------- 蓝鲸平台接入 ---------------------------
	// 蓝鲸应用配置
	BkApp BkAppConfig
	// Account 用户账号认证与用户 Token 获取相关集合
	Account AccountConfig
	// 蓝鲸平台地址配置
	BkPlatUrls BkPlatUrlsConfig
	// 蓝鲸 API 版本信息
	BkApiStages BkApiStagesConfig
	// BkIAMSystemIDs 接入蓝鲸权限中心（IAM）时使用的各业务系统 ID
	BkIAMSystemIDs BkIAMSystemIDsConfig

	// --------------------------- 外部依赖服务（蓝鲸 SaaS & 公司系统） ---------------------------
	// 蓝鲸容器平台（BCS）配置
	BCS BCSConfig
	// BSCP 服务配置
	BSCP BSCPConfig
	// 蓝盾（CI）配置
	BKCI BKCIConfig
	// 蓝盾制品库（Repo）配置
	BKRepo BKRepoConfig
	// BkMonitor 蓝鲸监控配置
	BkMonitor BkMonitorConfig
	// Tx CMDB 配置
	CMDB TxCMDBConfig
	// 北极星 SDK 配置
	Polaris PolarisConfig

	// --------------------------- 基础设施依赖 ---------------------------
	// 数据库配置
	Mongo MongoConfig
	// Redis 配置
	Redis RedisConfig
	// RabbitMQ 配置
	RabbitMQ RabbitMQConfig
	// Asynq 通用异步任务框架（taskq）配置
	Asynq AsynqConfig

	// --------------------------- 服务运行时 ---------------------------
	// HTTPServer 业务 HTTP Server 配置
	HTTPServer HTTPServerConfig
	// Metrics Prometheus Metrics Server 配置
	Metrics MetricsConfig
	// Logging 日志配置
	Logging LoggingConfig
	// 加密配置
	Encrypt EncryptConfig

	// --------------------------- 业务功能配置 ---------------------------
	// ClusterAddons 集群插件（Addons）配置
	ClusterAddons ClusterAddonConfig
	// Helm Helm 相关配置
	Helm HelmConfig
	// ImageBuild 镜像构建相关配置
	ImageBuild ImageBuildConfig
	// 任务轮询器
	TaskPoller TaskPollerConfig

	// --------------------------- 开发环境专用 ---------------------------
	// Development 包含与项目开发相关的各种配置项，仅供开发时使用
	Development DevConfig
}
