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

// Package admincmd trpc 管理命令
package admincmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

const (
	// maxConcurrentRequests 最大并发请求数
	maxConcurrentRequests = 10
)

// trpc 框架语言类型常量
const (
	languageGo     = "go"
	languageCpp    = "cpp"
	languageJava   = "java"
	languageNode   = "node"
	languagePython = "python"
)

// getTrpcAdminCmdsResponse trpc admin cmds 接口响应结构
type getTrpcAdminCmdsResponse struct {
	// Cmds 管理命令列表
	Cmds []string `json:"cmds"`

	// ErrorCode 错误码
	ErrorCode int `json:"errorcode"`

	// Message 消息
	Message string `json:"message"`
}

// execTrpcAdminCmdResponse trpc admin 命令执行响应结构
type execTrpcAdminCmdResponse struct {
	ErrorCode int `json:"errorcode"`

	Message string `json:"message"`
}

// AdminConfig trpc admin 配置
type AdminConfig struct {
	Server struct {
		Admin struct {
			IP        string `yaml:"ip"`
			Port      string `yaml:"port"`
			AdminIP   string `yaml:"admin_ip"`
			AdminPort string `yaml:"admin_port"`
		} `yaml:"admin"`

		// 兼容多种类型的 tRPC 配置文件（go-tRPC、java-tRPC等）
		IP        string `yaml:"ip"`
		Port      string `yaml:"port"`
		AdminIP   string `yaml:"admin_ip"`
		AdminPort string `yaml:"admin_port"`
	} `yaml:"server"`
}

// InstanceExecuteTrpcAdminCmdResult 实例执行 Trpc 管理命令结果
type InstanceExecuteTrpcAdminCmdResult struct {
	// 实例 ID
	InstanceID string

	// 命令执行是否成功
	Success bool

	// 命令执行结果详情
	Detail string
}

// TrpcAdminService TRPC Admin 服务，封装所有 TRPC Admin 相关的功能
type TrpcAdminService struct {
	// TrpcDeployRecordStore 部署记录存储
	TrpcDeployRecordStore appmodeldeploy.RecordStore
	// AppConfigFileStore 配置文件存储
	AppConfigFileStore appcfg.AppConfigFileStore
	// EnvStore 环境存储
	EnvStore envmodel.EnvironmentStore
	// AppStore 应用存储
	AppStore bkmsapp.ApplicationStore
	// AppModelStore 应用模型存储
	AppModelStore appmodel.AppModelStore
	// EnvVarsReader 统一环境变量读取器
	EnvVarsReader *envvars.UnifiedEnvVarsReader

	// App 应用
	App *bkmsapp.Application
	// AppModel 应用模型
	AppModel *appmodel.AppModel
	// Env 环境
	Env *envmodel.Environment

	// EnvName 环境名称
	EnvName string
	// ClusterID 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
	// InstanceIDs 验证通过的实例 ID 列表
	InstanceIDs []string
	// PodIPMap Pod 名称到 Pod IP 的映射
	PodIPMap map[string]string

	// PodClient Pod 客户端，可复用
	PodClient *k8sclient.PodClient
}

// NewAdminService 创建 TrpcAdminService 实例
// 在 new 方法中，会对参数进行校验
func NewAdminService(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
	instanceIDs []string,
	trpcDeployRecordStore appmodeldeploy.RecordStore,
	appConfigFileStore appcfg.AppConfigFileStore,
	envStore envmodel.EnvironmentStore,
	appStore bkmsapp.ApplicationStore,
	appModelStore appmodel.AppModelStore,
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appDepsVarReader *depenvvars.Reader,
	polarisVarReader *polarisenvvars.Reader,
) (*TrpcAdminService, error) {
	s := &TrpcAdminService{
		App:                   app,
		EnvName:               envName,
		InstanceIDs:           instanceIDs,
		TrpcDeployRecordStore: trpcDeployRecordStore,
		AppConfigFileStore:    appConfigFileStore,
		EnvStore:              envStore,
		AppStore:              appStore,
		AppModelStore:         appModelStore,
		EnvVarsReader:         envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
	}

	// 目前只支持 Trpc 应用
	if app.Type != bkmsapp.AppTypeTRPC {
		return nil, errors.New("only support trpc app")
	}
	if len(instanceIDs) == 0 {
		return nil, errors.New("instanceIDs is required")
	}
	env, err := s.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return nil, errors.Wrapf(err, "get env by name '%s'", envName)
	}
	appModel, err := s.AppModelStore.GetAppModel(ctx, s.App.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "get app model by id '%s'", app.ID)
	}
	s.Env = env
	s.AppModel = appModel
	s.ClusterID = env.Cluster.ClusterID
	s.Namespace = env.Cluster.Namespace
	s.PodClient = k8sclient.NewPodClient(cluster.NewConfig(env.Cluster.ClusterID))

	if s.PodIPMap, err = s.GetPodIPMap(ctx); err != nil {
		return nil, err
	}
	for _, instanceID := range s.InstanceIDs {
		if _, exists := s.PodIPMap[instanceID]; !exists {
			return nil, errors.Errorf(
				"pod '%s' not found in namespace '%s'",
				instanceID,
				s.Namespace,
			)
		}
	}

	return s, nil
}

// GetPodIPMap 获取 Pod IP 映射
func (s *TrpcAdminService) GetPodIPMap(ctx context.Context) (map[string]string, error) {
	record, err := s.TrpcDeployRecordStore.GetLatest(ctx, s.App.ID, s.EnvName, "")
	if err != nil {
		return nil, errors.Wrapf(err, "deploy record not found")
	}
	label := labels.SelectorFromSet(record.LabelSelector).String()

	pods, err := s.PodClient.List(ctx, s.Namespace, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, errors.Wrapf(
			err, "list namespace %s labelsSelector [%s] pods", s.Namespace, label,
		)
	}

	podIPMap := make(map[string]string, len(pods.Items))
	for _, pod := range pods.Items {
		podName := mapx.GetStr(pod.Object, "metadata.name")
		podIP := mapx.GetStr(pod.Object, "status.podIP")
		if podName != "" {
			podIPMap[podName] = podIP
		}
	}

	return podIPMap, nil
}

// ListTrpcAdminCmds 查询 Trpc 管理命令
func (s *TrpcAdminService) ListTrpcAdminCmds(ctx context.Context) ([]string, error) {
	instanceID := s.InstanceIDs[0]

	cfg, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "get admin config")
	}
	portConfig, err := s.GetAdminPort(cfg)
	if err != nil {
		return nil, err
	}

	// 解析 Port
	adminPort, err := s.ResolvePort(ctx, portConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve port")
	}

	// 注意： 通过 k8s port-forward(spdy 协议) 只能访问 127.0.0.1 绑定的服务
	// HTTP 请求目标地址必须使用 127.0.0.1，不能使用 pod IP
	cmdsUrl := fmt.Sprintf("http://127.0.0.1:%d/cmds", adminPort)

	// 向 pod 发送请求获取 admin cmds
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cmdsUrl, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "create http request")
	}

	resp, err := s.PodClient.SendHTTPRequest(ctx, s.Namespace, instanceID, adminPort, httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "send http request to pod '%s'", instanceID)
	}
	defer resp.Body.Close()

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read response body")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("get admin cmds failed, status: %d, body: %s", resp.StatusCode, string(body))
	}
	// 解析 admin cmds 响应
	cmdsResp := new(getTrpcAdminCmdsResponse)
	if err = json.Unmarshal(body, &cmdsResp); err != nil {
		return nil, errors.Wrapf(err, "unmarshal admin cmds response")
	}

	// 检查响应中的错误码
	if cmdsResp.ErrorCode != 0 {
		return nil, errors.Errorf(
			"get admin cmds failed, errorcode: %d, message: %s",
			cmdsResp.ErrorCode,
			cmdsResp.Message,
		)
	}

	return cmdsResp.Cmds, nil
}

// ExecuteTrpcAdminCmd 执行 Trpc 管理命令（并发执行，最多 10 个并发）
func (s *TrpcAdminService) ExecuteTrpcAdminCmd(
	ctx context.Context,
	path, method, body string,
	reqParams map[string]string,
) ([]InstanceExecuteTrpcAdminCmdResult, error) {
	cfg, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "get admin config")
	}
	portConfig, err := s.GetAdminPort(cfg)
	if err != nil {
		return nil, err
	}

	// 解析 Port（所有 Pod 使用相同的端口配置）
	adminPort, err := s.ResolvePort(ctx, portConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve port")
	}

	// 构建请求 URL（包含 query params）
	requestPath := path
	if len(reqParams) > 0 {
		params := url.Values{}
		for k, v := range reqParams {
			params.Add(k, v)
		}
		requestPath = fmt.Sprintf("%s?%s", path, params.Encode())
	}

	results := make([]InstanceExecuteTrpcAdminCmdResult, 0, len(s.InstanceIDs))
	resultChan := make(chan *InstanceExecuteTrpcAdminCmdResult, len(s.InstanceIDs))

	// 使用 semaphore 控制并发数量
	semaphore := make(chan bool, maxConcurrentRequests)

	// 启动 goroutine 并行发送请求（受 semaphore 控制）
	for _, podName := range s.InstanceIDs {
		go func(pod string) {
			// 获取信号量（如果已达到最大并发数，会阻塞等待）
			semaphore <- true
			defer func() { <-semaphore }()

			resultChan <- s.executeTrpcAdminCmdOnPod(
				ctx, pod, method, requestPath, body, adminPort,
			)
		}(podName)
	}

	// 收集所有结果
	for i := 0; i < len(s.InstanceIDs); i++ {
		res := <-resultChan
		results = append(results, *res)
	}

	return results, nil
}

// ResolvePort 解析端口号
// portConfig 是 tRPC 的 admin Port 配置，它可能是直接配置的端口号（纯数字组成），也可能是环境变量。
// 返回 int32 类型的端口号
func (s *TrpcAdminService) ResolvePort(
	ctx context.Context,
	portConfig string,
) (int32, error) {
	port, err := strconv.ParseInt(portConfig, 10, 32)
	if err == nil {
		return int32(port), nil
	}

	renderedPort, err := s.MatchConfigValue(ctx, portConfig)
	if err != nil {
		return 0, errors.Wrapf(err, "render port config value")
	}

	port, err = strconv.ParseInt(renderedPort, 10, 32)
	if err == nil {
		return int32(port), nil
	}

	return 0, errors.Wrapf(err, "unknown port config value: %s", portConfig)
}

// MatchConfigValue 查找配置中的变量
// 支持 ${VAR_NAME} 和 $VAR_NAME 两种格式
func (s *TrpcAdminService) MatchConfigValue(ctx context.Context, varName string) (string, error) {
	appEnvVars, err := envvars.BuildAppEnvVars(ctx, s.App, s.AppModel, s.Env, s.EnvVarsReader)
	if err != nil {
		return "", errors.Wrapf(err, "build app env vars")
	}

	return render.RenderShellVars(varName, appEnvVars.ToMap()), nil
}

// GetAdminIP 根据配置和语言类型获取 admin IP 字符串配置
func (s *TrpcAdminService) GetAdminIP(cfg *AdminConfig) string {
	switch s.App.TrpcSpec.Language {
	case languageGo, languagePython:
		return cfg.Server.Admin.IP
	case languageCpp, languageNode:
		return cfg.Server.AdminIP
	case languageJava:
		return cfg.Server.Admin.AdminIP
	default:
		return ""
	}
}

// Precheck 预检查 admin 配置
// 检查 server 配置的 IP 是否为 0.0.0.0 或 127.0.0.1（回环地址），
// 只有绑定在这些地址上的服务才能通过 k8s port-forward 访问。
// 返回空字符串表示检查通过，否则返回错误信息。
func (s *TrpcAdminService) Precheck(ctx context.Context) error {
	cfg, err := s.GetAdminConfig(ctx)
	if err != nil {
		return errors.Wrapf(err, "get admin config")
	}

	ipConfig := s.GetAdminIP(cfg)
	if ipConfig == "" {
		return errors.New("admin server IP is not configured in trpc config")
	}

	// 渲染可能包含环境变量的 IP 配置
	resolvedIP, err := s.MatchConfigValue(ctx, ipConfig)
	if err != nil {
		return errors.Wrapf(err, "resolve admin IP config")
	}

	// 检查 IP 是否为 0.0.0.0 或 127.0.0.1
	if resolvedIP != "0.0.0.0" && resolvedIP != "127.0.0.1" {
		return errors.Errorf("admin server binding IP can only be 0.0.0.0 or 127.0.0.1,"+
			" binding other IPs will cause inaccessibility, current IP: %s", resolvedIP,
		)
	}

	return nil
}

// GetAdminPort 根据配置和语言类型获取 admin Port 字符串配置
func (s *TrpcAdminService) GetAdminPort(cfg *AdminConfig) (string, error) {
	switch s.App.TrpcSpec.Language {
	case languageGo, languagePython:
		return cfg.Server.Admin.Port, nil
	case languageCpp, languageNode:
		return cfg.Server.AdminPort, nil
	case languageJava:
		return cfg.Server.Admin.AdminPort, nil
	default:
		return "", errors.Errorf("unsupported trpc language type: %s", s.App.TrpcSpec.Language)
	}
}

// GetAdminConfig 获取并解析 admin 配置
func (s *TrpcAdminService) GetAdminConfig(ctx context.Context) (*AdminConfig, error) {
	_, configContent, err := appcfg.GetEnvContent(ctx, s.AppConfigFileStore, s.App.ID, s.Env.Name)
	if err != nil {
		return nil, err
	}

	cfg := new(AdminConfig)
	if err = yaml.Unmarshal([]byte(configContent), cfg); err != nil {
		return nil, errors.Wrapf(err, "unmarshal trpc config")
	}

	return cfg, nil
}

// executeTrpcAdminCmdOnPod 在单个 pod 内部，通过 http 访问 trpc admin server
func (s *TrpcAdminService) executeTrpcAdminCmdOnPod(
	ctx context.Context, podName, method, requestPath, body string,
	adminPort int32,
) *InstanceExecuteTrpcAdminCmdResult {
	result := &InstanceExecuteTrpcAdminCmdResult{
		InstanceID: podName,
		Success:    false,
	}
	// 注意： 通过 k8s port-forward(spdy 协议) 只能访问 127.0.0.1 绑定的服务
	// HTTP 请求目标地址必须使用 127.0.0.1，不能使用 pod IP
	requestURL := fmt.Sprintf("http://127.0.0.1:%d%s", adminPort, requestPath)

	// 构建 HTTP 请求
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		result.Detail = fmt.Sprintf("create http request failed: %v", err)
		return result
	}

	// 设置 Content-Type（如果有 body）
	if body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := s.PodClient.SendHTTPRequest(ctx, s.Namespace, podName, adminPort, httpReq)
	if err != nil {
		result.Detail = fmt.Sprintf("send http request failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Detail = fmt.Sprintf("read response body failed: %v", err)
		return result
	}
	// 检查 HTTP 状态码
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		result.Detail = fmt.Sprintf("http status code %d, body: %s", resp.StatusCode, string(respBody))
		return result
	}

	// 解析响应，检查 errorCode
	execResp := new(execTrpcAdminCmdResponse)
	if err = json.Unmarshal(respBody, &execResp); err != nil {
		// 如果解析失败，可能不是标准格式，直接返回原始响应
		result.Success = true
		result.Detail = string(respBody)
		return result
	}

	// 检查响应中的错误码
	if execResp.ErrorCode != 0 {
		result.Detail = fmt.Sprintf("errorcode: %d, message: %s", execResp.ErrorCode, execResp.Message)
		return result
	}

	result.Success = true
	result.Detail = string(respBody)

	return result
}
