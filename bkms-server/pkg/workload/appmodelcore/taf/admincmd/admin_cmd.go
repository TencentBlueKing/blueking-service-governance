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

// Package admincmd TAF 管理命令
package admincmd

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/TarsCloud/TarsGo/tars"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/adminf"
	"github.com/TarsCloud/TarsGo/tars/util/conf"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

const (
	// maxConcurrentRequests 最大并发请求数
	maxConcurrentRequests = 10
	// defaultAdminProto 默认传输协议
	defaultAdminProto = "tcp"
	// defaultAdminServant 默认 Servant 名称
	defaultAdminServant = "AdminObj"
	// requestTimeout SDK 请求超时（毫秒）
	requestTimeout = 5000
)

var allowedAdminIPs = []string{"0.0.0.0", "127.0.0.1"}

// InstanceExecuteTafAdminCmdResult 实例执行 TAF 管理命令结果
type InstanceExecuteTafAdminCmdResult struct {
	// 实例 ID
	InstanceID string

	// 命令执行是否成功
	Success bool

	// 命令执行结果详情
	Detail string
}

// AdminServiceStores AdminService 所需的存储依赖
type AdminServiceStores struct {
	TafDeployRecordStore appmodeldeploy.RecordStore
	AppConfigFileStore   appcfg.AppConfigFileStore
	EnvStore             envmodel.EnvironmentStore
	AppStore             bkmsapp.ApplicationStore
	AppModelStore        appmodel.AppModelStore
	EnvVarsReader        *envvars.UnifiedEnvVarsReader
}

// TafAdminService TAF Admin 服务，封装所有 TAF Admin 相关的功能
type TafAdminService struct {
	// Stores 存储依赖
	Stores *AdminServiceStores

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
	// PodIPMap Pod 名称到 Pod IP 的映射（用于校验 instanceIDs 是否存在）
	PodIPMap map[string]string

	// AdminPort admin 端口（从 TAF 配置文件动态解析）
	AdminPort int32

	// PodClient Pod 客户端，可复用
	PodClient *k8sclient.PodClient
}

// NewAdminService 创建 TafAdminService 实例（轻量构造函数，仅做参数校验）
func NewAdminService(
	app *bkmsapp.Application,
	envName string,
	instanceIDs []string,
	stores *AdminServiceStores,
) (*TafAdminService, error) {
	// 目前只支持 TAF 应用
	if app.Type != bkmsapp.AppTypeTAF {
		return nil, errors.New("only support taf app")
	}
	if len(instanceIDs) == 0 {
		return nil, errors.New("instanceIDs is required")
	}

	return &TafAdminService{
		App:         app,
		EnvName:     envName,
		InstanceIDs: instanceIDs,
		Stores:      stores,
	}, nil
}

// Init 初始化服务，执行 I/O 操作（获取环境、应用模型、解析配置、验证实例）
func (s *TafAdminService) Init(ctx context.Context) error {
	env, err := s.Stores.EnvStore.GetByName(ctx, s.App.WorkspaceID, s.App.ID, s.EnvName)
	if err != nil {
		return errors.Wrapf(err, "get env by name '%s'", s.EnvName)
	}
	appModel, err := s.Stores.AppModelStore.GetAppModel(ctx, s.App.ID)
	if err != nil {
		return errors.Wrapf(err, "get app model by id '%s'", s.App.ID)
	}
	s.Env = env
	s.AppModel = appModel
	s.ClusterID = env.Cluster.ClusterID
	s.Namespace = env.Cluster.Namespace
	s.PodClient = k8sclient.NewPodClient(cluster.NewConfig(env.Cluster.ClusterID))

	// 从 TAF 配置文件解析 admin 端口
	_, adminPort, err := s.GetAdminConfig(ctx)
	if err != nil {
		return err
	}
	s.AdminPort = adminPort

	if s.PodIPMap, err = s.GetPodNameIPMap(ctx); err != nil {
		return err
	}
	for _, instanceID := range s.InstanceIDs {
		if _, exists := s.PodIPMap[instanceID]; !exists {
			return errors.Errorf(
				"pod '%s' not found in namespace '%s'",
				instanceID,
				s.Namespace,
			)
		}
	}

	return nil
}

// GetAdminConfig 从 TAF 配置文件解析 admin IP 和端口号
//
// TAF 配置格式示例:
//
//	<taf>
//	  <application>
//	    <server>
//	      local=tcp -h 9.165.155.166 -p 17064 -t 30000
//	    </server>
//	  </application>
//	</taf>
//
// 解析 local 字段中的 -p 参数获取端口号, -h 参数获取监听 IP
func (s *TafAdminService) GetAdminConfig(ctx context.Context) (string, int32, error) {
	// 获取配置文件内容
	_, configContent, err := appcfg.GetEnvContent(ctx, s.Stores.AppConfigFileStore, s.App.ID, s.Env.Name)
	if err != nil {
		return "", 0, errors.Wrap(err, "get taf config content")
	}

	// 使用 TarsGo conf 包解析 TAF 配置
	tafConf := conf.New()
	if err = tafConf.InitFromString(configContent); err != nil {
		return "", 0, errors.Wrap(err, "parse taf config")
	}

	// 获取 local 字段（尝试 /taf 和 /tars 两种根路径）
	localEndpoint := tafConf.GetString("/taf/application/server<local>")
	if localEndpoint == "" {
		localEndpoint = tafConf.GetString("/tars/application/server<local>")
	}
	if localEndpoint == "" {
		return "", 0, errors.New("'local' field not found in config file")
	}

	// 从 endpoint 解析监听 IP 和端口号
	ip, port, err := s.parseEndpoint(ctx, localEndpoint)
	if err != nil {
		return "", 0, errors.Wrapf(err, "parse port from endpoint '%s'", localEndpoint)
	}

	if !slices.Contains(allowedAdminIPs, ip) {
		return "", 0, errors.Errorf("admin server binding IP can only be %s,"+
			" binding other IPs will cause inaccessibility, current IP: %s", strings.Join(allowedAdminIPs, ", "), ip,
		)
	}

	return ip, port, nil
}

// parseEndpoint 从 TAF endpoint 字符串中解析监听 IP 和端口号
// 格式: "tcp -h 9.165.155.166 -p 17064 -t 30000"
func (s *TafAdminService) parseEndpoint(ctx context.Context, endpoint string) (ip string, port int32, err error) {
	renderedEndpoint, err := s.matchConfigValue(ctx, endpoint)
	if err != nil {
		return "", 0, errors.Wrapf(err, "render endpoint '%s'", endpoint)
	}
	parts := strings.Fields(renderedEndpoint)
	for i, part := range parts {
		if part == "-p" && i+1 < len(parts) {
			parsePort, err := strconv.ParseInt(parts[i+1], 10, 32)
			if err != nil {
				return "", 0, errors.Wrapf(err, "invalid port number '%s'", parts[i+1])
			}
			port = int32(parsePort)
		}

		if part == "-h" && i+1 < len(parts) {
			ip = parts[i+1]
		}
	}
	if ip == "" {
		return "", 0, errors.New("ip (-h) not found in endpoint string")
	}
	if port == 0 {
		return "", 0, errors.New("port (-p) not found in endpoint string")
	}
	return ip, port, nil
}

// GetPodNameIPMap 获取 Pod IP 映射
func (s *TafAdminService) GetPodNameIPMap(ctx context.Context) (map[string]string, error) {
	record, err := s.Stores.TafDeployRecordStore.GetLatest(ctx, s.App.ID, s.EnvName, "")
	if err != nil {
		return nil, errors.Wrapf(err, "deploy record not found")
	}
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()

	pods, err := s.PodClient.List(ctx, s.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, errors.Wrapf(
			err, "list namespace %s labelsSelector [%s] pods", s.Namespace, labelSelector,
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

// ExecuteTafAdminCmd 执行 TAF 管理命令（并发执行，最多 10 个并发）
func (s *TafAdminService) ExecuteTafAdminCmd(
	ctx context.Context,
	command string,
) ([]InstanceExecuteTafAdminCmdResult, error) {
	if command == "" {
		return nil, errors.New("command is required")
	}

	results := make([]InstanceExecuteTafAdminCmdResult, 0, len(s.InstanceIDs))
	resultChan := make(chan *InstanceExecuteTafAdminCmdResult, len(s.InstanceIDs))

	// 使用 semaphore 控制并发数量
	semaphore := make(chan bool, maxConcurrentRequests)
	var wg sync.WaitGroup

	// 启动 goroutine 并行发送请求（受 semaphore 控制）
	for _, podName := range s.InstanceIDs {
		pod := podName
		wg.Go(func() {
			// 获取信号量（如果已达到最大并发数，会阻塞等待）
			semaphore <- true
			defer func() { <-semaphore }()

			resultChan <- s.executeTafAdminCmdOnPod(ctx, pod, command)
		})
	}

	// 等待所有 goroutine 完成后关闭 channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集所有结果
	for res := range resultChan {
		results = append(results, *res)
	}

	return results, nil
}

// executeTafAdminCmdOnPod 在单个 Pod 上通过 TarsGo SDK 执行 TAF admin 命令
//
// 通过 K8s port-forward 建立本地 TCP 代理，将 TarsGo SDK 的请求桥接到 Pod 内的 admin 端口。
// NOTE: TarsGo SDK 内部使用 net.DialTimeout 建立 TCP 连接，无法注入自定义 dialer，
// 因此需要通过本地 TCP 监听器作为中转，将流量桥接到 K8s SPDY data stream。
func (s *TafAdminService) executeTafAdminCmdOnPod(
	ctx context.Context, podName, command string,
) *InstanceExecuteTafAdminCmdResult {
	result := &InstanceExecuteTafAdminCmdResult{
		InstanceID: podName,
		Success:    false,
	}

	// 创建 TCP 端口转发到 Pod
	pf, err := s.PodClient.CreateTCPPortForward(ctx, s.Namespace, podName, s.AdminPort)
	if err != nil {
		log.Errorf(ctx, "[TafAdmin] create port-forward failed, pod='%s': %v", podName, err)
		result.Detail = fmt.Sprintf("create port-forward to pod '%s' failed: %v", podName, err)
		return result
	}
	defer pf.Close()

	// 构建 Servant 字符串，连接到本地代理端口
	// 格式：AdminObj@tcp -h 127.0.0.1 -p <localPort> -t <timeout>
	obj := fmt.Sprintf(
		"%s@%s -h 127.0.0.1 -p %d -t %d",
		defaultAdminServant,
		defaultAdminProto,
		pf.LocalPort,
		requestTimeout,
	)

	// 创建通信器和 Admin 代理对象
	comm := tars.NewCommunicator()
	app := new(adminf.AdminF)
	comm.StringToProxy(obj, app)
	// TarsGo Communicator 的 RPC 调用超时默认是 3000ms（AsyncInvokeTimeout），
	// 与 endpoint 字符串中的 -t 参数无关，需要显式设置。
	app.TarsSetTimeout(requestTimeout)

	// 执行命令
	ret, err := app.Notify(command)
	if err != nil {
		log.Errorf(ctx, "[TafAdmin] Notify failed, pod='%s', command='%s': %v", podName, command, err)
		result.Detail = fmt.Sprintf("execute admin command '%s' failed: %v", command, err)
		return result
	}

	result.Success = true
	result.Detail = ret

	return result
}

// MatchConfigValue 查找配置中的变量
// 支持 ${{env.VAR_NAME}} 格式
func (s *TafAdminService) matchConfigValue(ctx context.Context, varName string) (string, error) {
	appEnvVars, err := envvars.BuildAppEnvVars(ctx, s.App, s.AppModel, s.Env, s.Stores.EnvVarsReader)
	if err != nil {
		return "", errors.Wrapf(err, "build app env vars")
	}

	return render.New(render.SetEnvContext(appEnvVars.ToMap())).Render(varName)
}
