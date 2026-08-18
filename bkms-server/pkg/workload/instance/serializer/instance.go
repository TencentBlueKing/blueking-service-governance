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

// Package serializer 定义应用实例相关接口的输入输出序列化结构。
package serializer

import (
	"fmt"
	"slices"
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/timex"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	podstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/pod"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	instancelogsvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// -----------------------------------------------------------------------------
// 通用输入

// EmptyOutput 空响应，用于无返回内容的接口。
type EmptyOutput struct{}

// AppEnvURIInput 按应用和环境限定的路径参数。
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// AppInstanceURIInput 按单个应用实例限定的路径参数。
type AppInstanceURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 实例 ID
	InstanceID string `uri:"instanceID" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// 实例管理

// 分页页码从 1 开始；校验走 Validate 而非 binding:"gte=1"，因为该约束只在分页模式生效
const minListAppInstancesPage int64 = 1

// 分页页码上限；能翻到多少条取决于 pageSize，按最小的 5 算保底也有 5 万条，按最大的 100 算到百万
// 单个应用环境不会有这个量级的实例，超出只可能是脏参数
// 除了给出明确的 400 而不是静默的空列表，这个上界也是 ProjectionRange 里
// (page-1)*pageSize 不溢出 int64 的前提，放宽时要一并复核
const maxListAppInstancesPage int64 = 10000

// 分页 pageSize 仅允许这些固定值；取值约束只在分页模式生效，所以不写 binding:"oneof=..."
var allowedListAppInstancesPageSizes = []int64{5, 10, 20, 50, 100}

// ListAppInstancesQueryInput 查询应用实例列表的请求参数。
// page/pageSize 不用 gin binding 做 required/gte/oneof：
// 全量（all=true）禁止带分页参数，分页模式才必填且 pageSize 受限；
// binding 标签无法按 all 切换必填，也无法表达 all=true 与 page/pageSize 互斥。
type ListAppInstancesQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 为 true 时一次返回匹配的全部实例投影；禁止同时带 page 或 pageSize
	All bool `form:"all"`
	// 页码，从 1 开始；分页模式必填，all=true 时禁止出现
	Page *int64 `form:"page"`
	// 每页数量，仅支持固定枚举值；分页模式必填，all=true 时禁止出现
	PageSize *int64 `form:"pageSize"`
}

// Validate 校验全量与分页参数互斥，以及分页模式下的必填与取值
// 必须在 Bind 之后单独调用：Bind 只做解析，条件校验放这里才能让 all=true 不带分页通过
func (q *ListAppInstancesQueryInput) Validate() error {
	if q.All {
		if q.Page != nil || q.PageSize != nil {
			return bkerrs.New(
				bkerrs.ErrCodeInvalidArgument,
				"all=true cannot be used together with page or pageSize",
			)
		}
		return nil
	}
	if q.Page == nil {
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, "page is required")
	}
	if q.PageSize == nil {
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, "pageSize is required")
	}
	if *q.Page < minListAppInstancesPage || *q.Page > maxListAppInstancesPage {
		return bkerrs.Errorf(
			bkerrs.ErrCodeInvalidArgument,
			"page must be between %d and %d", minListAppInstancesPage, maxListAppInstancesPage,
		)
	}
	if !slices.Contains(allowedListAppInstancesPageSizes, *q.PageSize) {
		return bkerrs.New(bkerrs.ErrCodeInvalidArgument, "pageSize must be one of 5, 10, 20, 50, 100")
	}
	return nil
}

// ProjectionRange 返回需要投影的 [start, end) 下标
// 全量模式投影全部匹配 Pod；分页模式只投影当前页，整页越过尾部时被 min 收敛成空区间
// 分页模式必须已通过 Validate：除了 Page/PageSize 非空，page 上界还保证下面的乘法不会溢出 int64
func (q *ListAppInstancesQueryInput) ProjectionRange(total int64) (start, end int64) {
	if q.All {
		return 0, total
	}

	page := *q.Page
	pageSize := *q.PageSize

	start = min((page-1)*pageSize, total)
	end = min(start+pageSize, total)

	return start, end
}

// -----------------------------------------------------------------------------
// 实例 Watch 契约（传输形态 SSE/WS 待定；本阶段仅声明 API DTO，不实现推送）
// 领域事件类型见 instance/watch.EventType

// WatchAppInstancesQueryInput 订阅应用实例投影变更的查询参数。
type WatchAppInstancesQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// AppInstanceWatchEvent 实例 Watch 推送的平台投影事件（非原生 Pod JSON）
// DELETED 时 Object 可仅含定位字段（至少 id）
// Type 取值对齐 watch.EventType：ADDED / MODIFIED / DELETED
type AppInstanceWatchEvent struct {
	// 事件类型
	// Enums: ADDED, MODIFIED, DELETED
	Type string `json:"type" enums:"ADDED,MODIFIED,DELETED"`
	// 实例投影；字段集合对齐 AppInstanceOutputObj（含 polarisInfos）
	Object *AppInstanceOutputObj `json:"object"`
}

// PolarisInstanceInfoOutputObj 关联到应用实例的北极星实例状态。
type PolarisInstanceInfoOutputObj struct {
	// 北极星命名空间
	ServiceNamespace string `json:"serviceNamespace"`
	// 北极星服务名
	ServiceName string `json:"serviceName"`
	// 实例 IP（等于 Pod IP）
	IP string `json:"ip"`
	// 实例端口（等于应用监听的服务端口）
	Port uint32 `json:"port"`
	// 健康状态
	IsHealthy bool `json:"isHealthy"`
	// 权重
	Weight int64 `json:"weight,string"`
	// 隔离状态
	IsIsolated bool `json:"isIsolated"`
	// 是否启用健康检查
	EnableHealthCheck bool `json:"enableHealthCheck"`
	// 元数据
	Metadata map[string]string `json:"metadata"`
}

// AppInstanceResourcesObj is the main-container CPU/memory quantities from the live Pod.
type AppInstanceResourcesObj struct {
	// CPU limits（Kubernetes quantity 字符串），可选：未配置时不返回该字段
	CPULimits string `json:"cpuLimits,omitempty"`
	// CPU requests，可选：未配置时不返回该字段
	CPURequests string `json:"cpuRequests,omitempty"`
	// Memory limits，可选：未配置时不返回该字段
	MemoryLimits string `json:"memoryLimits,omitempty"`
	// Memory requests，可选：未配置时不返回该字段
	MemoryRequests string `json:"memoryRequests,omitempty"`
}

// AppInstanceOutputObj 单个应用实例（即一个 Pod）。
type AppInstanceOutputObj struct {
	// 实例 ID（即 k8s pod 的 name）
	ID string `json:"id"`
	// 部署记录 ID
	DeployID string `json:"deployID"`
	// Pod IP
	IP string `json:"ip"`
	// 镜像
	Image string `json:"image"`
	// 重启次数
	RestartCount int64 `json:"restartCount,string"`
	// 状态，由 pod.status.phase 等解析获得
	Status string `json:"status"`
	// 状态详情，一般为 pod.status.reason
	Message string `json:"message"`
	// 健康状态，即 k8s 探针检查结果
	IsHealthy bool `json:"isHealthy"`
	// 存在时间，格式如：2d1h，24m29s
	Age string `json:"age"`
	// 节点 IP，Pod 所在节点的 IP 地址
	NodeIP string `json:"nodeIP"`
	// 主容器资源规格（集群 Pod 实际值）
	Resources AppInstanceResourcesObj `json:"resources"`
	// 北极星实例状态列表（一个 Pod 可能注册到多个北极星服务）
	PolarisInfos []*PolarisInstanceInfoOutputObj `json:"polarisInfos"`
}

// FromPodManifest 从 Kubernetes Pod manifest 填充实例输出字段。
func (o *AppInstanceOutputObj) FromPodManifest(
	manifest map[string]any,
	deployID string,
) (*AppInstanceOutputObj, error) {
	podName := mapx.GetStr(manifest, "metadata.name")
	if podName == "" {
		return nil, fmt.Errorf("pod name is empty")
	}

	containers := mapx.GetList(manifest, "spec.containers")
	if len(containers) == 0 {
		return nil, fmt.Errorf("pod %s containers is empty", podName)
	}

	var image string
	if cMap, ok := containers[0].(map[string]any); ok {
		image = mapx.GetStr(cMap, "image")
	}

	resources := extractMainContainerResources(containers)

	var restartCount int64
	for _, cs := range mapx.GetList(manifest, "status.containerStatuses") {
		if csMap, ok := cs.(map[string]any); ok {
			if cnt := mapx.GetInt64(csMap, "restartCount"); cnt > restartCount {
				restartCount = cnt
			}
		}
	}

	message := mapx.GetStr(manifest, "status.message")
	if message == "" {
		message = mapx.GetStr(manifest, "status.reason")
	}

	isHealthy := podstatus.IsReady(manifest)

	*o = AppInstanceOutputObj{
		ID:           podName,
		DeployID:     deployID,
		IP:           mapx.GetStr(manifest, "status.podIP"),
		NodeIP:       mapx.GetStr(manifest, "status.hostIP"),
		Image:        image,
		RestartCount: restartCount,
		Status:       podstatus.NewParser(manifest).Parse().Code,
		Message:      message,
		IsHealthy:    isHealthy,
		Age:          timex.CalcAge(mapx.GetStr(manifest, "metadata.creationTimestamp")),
		Resources:    resources,
	}
	return o, nil
}

// extractMainContainerResources 从 Pod containers 中读取主容器的 CPU/内存。
// 找不到主容器时返回零值，不阻断实例列表。
func extractMainContainerResources(containers []any) AppInstanceResourcesObj {
	for _, c := range containers {
		cMap, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if mapx.GetStr(cMap, "name") != defaults.WorkloadMainContainerName {
			continue
		}
		limits := mapx.GetMap(cMap, "resources.limits")
		requests := mapx.GetMap(cMap, "resources.requests")
		return AppInstanceResourcesObj{
			CPULimits:      mapx.GetStr(limits, string(corev1.ResourceCPU)),
			CPURequests:    mapx.GetStr(requests, string(corev1.ResourceCPU)),
			MemoryLimits:   mapx.GetStr(limits, string(corev1.ResourceMemory)),
			MemoryRequests: mapx.GetStr(requests, string(corev1.ResourceMemory)),
		}
	}
	return AppInstanceResourcesObj{}
}

// MergePolarisInfoToAppInstances 将北极星实例信息合并到应用实例输出对象中。
func MergePolarisInfoToAppInstances(
	appInstances []*AppInstanceOutputObj,
	svcInstances []*polaris.PolarisServiceInstances,
) {
	type polarisMatch struct {
		svc  *polaris.PolarisServiceInstances
		inst *polarisInfra.Instance
	}
	ipIndex := make(map[string][]polarisMatch)
	for _, svc := range svcInstances {
		for _, inst := range svc.Instances {
			ipIndex[inst.IP] = append(ipIndex[inst.IP], polarisMatch{svc: svc, inst: inst})
		}
	}

	for _, instance := range appInstances {
		matches, ok := ipIndex[instance.IP]
		if !ok {
			continue
		}
		for _, m := range matches {
			if int64(m.inst.Port) != int64(m.svc.ServicePort) {
				continue
			}
			instance.PolarisInfos = append(instance.PolarisInfos, &PolarisInstanceInfoOutputObj{
				ServiceNamespace:  m.svc.ServiceNamespace,
				ServiceName:       m.svc.ServiceName,
				IP:                m.inst.IP,
				Port:              m.inst.Port,
				IsHealthy:         m.inst.IsHealthy,
				Weight:            int64(m.inst.Weight),
				IsIsolated:        m.inst.IsIsolated,
				EnableHealthCheck: m.inst.EnableHealthCheck,
				Metadata:          m.inst.Metadata,
			})
		}
	}
}

// SkippedAppInstanceObj 无法投影为 AppInstanceOutputObj 而被跳过的实例。
type SkippedAppInstanceObj struct {
	// 实例 ID（即 k8s pod 的 name）；解析前无 name 时为空字符串
	ID string `json:"id"`
	// 跳过原因
	Reason string `json:"reason"`
}

// PaginatedAppInstancesOutputObj 分页或全量查询应用实例列表的输出载荷。
type PaginatedAppInstancesOutputObj struct {
	// 结果数量；全量为成功投影条数，分页为 LabelSelector 匹配的 Pod 总数
	Count int64 `json:"count,string"`
	// 查询结果，只含成功投影
	Results []*AppInstanceOutputObj `json:"results"`
	// 本次响应中跳过的实例数（仅全量模式可能非 0）
	SkippedCount int64 `json:"skippedCount,string"`
	// 无法投影的实例列表；分页模式为空数组，无跳过项时亦为空数组
	Skipped []*SkippedAppInstanceObj `json:"skipped"`
}

// ListAppInstancesOutput 查询应用实例列表的响应。
type ListAppInstancesOutput struct {
	// 分页查询结果
	Data *PaginatedAppInstancesOutputObj `json:"data"`
}

// UpdateAppInstancesInput 更新应用实例的输入参数。
type UpdateAppInstancesInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
	// 部署使用的镜像 Tag
	ImageTag string `json:"imageTag" binding:"required,min=1"`
	// 更新策略，可选值：RollingUpdate, InplaceUpdate
	UpdateStrategy string `json:"updateStrategy" binding:"required,oneof=RollingUpdate InplaceUpdate"`
	// 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs" binding:"dive,required"`
}

// ScaleAppInstancesInput 扩缩容应用实例的输入参数。
type ScaleAppInstancesInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
	// 目标实例数量
	TargetReplicas int32 `json:"targetReplicas" binding:"gte=0"`
}

// BatchDeleteAppInstancesInput 批量删除应用实例的输入参数。
type BatchDeleteAppInstancesInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
	// 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs" binding:"required,min=1,dive,required"`
}

// UpdateAppInstancePolarisInput 更新应用实例北极星注解的输入参数。
type UpdateAppInstancePolarisInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
	// 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs" binding:"required,min=1,dive,required"`
	// 北极星权重（可选，如 100），不设置表示不修改
	Weight *int32 `json:"weight"`
	// 北极星隔离状态（可选），不设置表示不修改
	Isolate *bool `json:"isolate"`
}

// -----------------------------------------------------------------------------
// 终端控制台

// CreateAppInstanceWebConsoleInput 创建应用实例终端控制台的输入参数。
type CreateAppInstanceWebConsoleInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
}

// WebConsoleInfoOutputObj 终端控制台连接信息。
type WebConsoleInfoOutputObj struct {
	// 访问链接
	URL string `json:"url"`
}

// CreateAppInstanceWebConsoleOutput 创建应用实例终端控制台的响应。
type CreateAppInstanceWebConsoleOutput struct {
	Data *WebConsoleInfoOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// 实例日志

// ListAppInstanceLogsQueryInput 查询应用实例日志的请求参数。
type ListAppInstanceLogsQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 是否获取重启前日志
	Previous bool `form:"previous"`
	// 日志行数（从尾部起算），最大 2000
	TailLines int64 `form:"tailLines" binding:"required,gte=100,lte=2000"`
}

// LogEntryOutputObj 单条应用实例日志。
type LogEntryOutputObj struct {
	// 日志时间戳
	Timestamp string `json:"timestamp"`
	// 日志内容
	Content string `json:"content"`
}

// FromModel 从日志模型填充输出字段。
func (o *LogEntryOutputObj) FromModel(entry *instancelogsvc.LogEntry) *LogEntryOutputObj {
	*o = LogEntryOutputObj{Timestamp: entry.Timestamp, Content: entry.Content}
	return o
}

// ListAppInstanceLogsOutput 查询应用实例日志的响应。
type ListAppInstanceLogsOutput struct {
	Data []*LogEntryOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// 事件

// ListEventsQueryInput 查询应用环境事件列表的请求参数。
type ListEventsQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 事件级别（可选过滤参数，可选值：Normal, Warning）
	Level string `form:"level"`
	// 起始时间戳（可选过滤参数，如：1772223278）
	StartedAt int64 `form:"startedAt"`
	// 结束时间戳（可选过滤参数，如：1772223278）
	EndedAt int64 `form:"endedAt"`
	// 页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量，仅支持固定枚举值
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// EventEntryOutputObj 单条事件记录。
type EventEntryOutputObj struct {
	// BCS 集群 ID
	ClusterID string `json:"clusterID"`
	// 命名空间
	Namespace string `json:"namespace"`
	// 事件级别
	Level string `json:"level"`
	// 事件内容
	Content string `json:"content"`
	// 事件类型
	Type string `json:"type"`
	// 组件名称
	ComponentName string `json:"componentName"`
	// 关联的资源类型，如：Deployment, Pod，Node 等
	ResourceKind string `json:"resourceKind"`
	// 关联的资源名称，如：nginx-ingress-2695bd-58877d456b
	ResourcesName string `json:"resourcesName"`
	// 事件创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// PaginatedEventsOutputObj 分页查询事件列表的输出载荷。
type PaginatedEventsOutputObj struct {
	// 结果数量
	Count int64 `json:"count,string"`
	// 查询结果
	Results []*EventEntryOutputObj `json:"results"`
}

// ListEventsOutput 查询应用环境事件列表的响应。
type ListEventsOutput struct {
	// 分页查询结果
	Data *PaginatedEventsOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// 管理命令

// ListTrpcAdminCmdsQueryInput 查询 Trpc 管理命令列表的请求参数。
type ListTrpcAdminCmdsQueryInput struct {
	// 实例 ID 列表
	InstanceIDs []string `form:"instanceIDs" binding:"required,min=1,dive,required"`
}

// ListTrpcAdminCmdsOutputObjs 查询 Trpc 管理命令列表的输出载荷。
type ListTrpcAdminCmdsOutputObjs struct {
	// 结果数量
	Count int64 `json:"count,string"`
	// 查询结果
	Results []string `json:"results"`
}

// ListTrpcAdminCmdsOutput 查询 Trpc 管理命令列表的响应。
type ListTrpcAdminCmdsOutput struct {
	Data *ListTrpcAdminCmdsOutputObjs `json:"data"`
}

// ExecuteTrpcAdminCmdInput 执行 Trpc 管理命令的输入参数。
type ExecuteTrpcAdminCmdInput struct {
	// 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs" binding:"required,min=1,dive,required"`
	// HTTP 方法，限定 GET, POST, PUT
	Method string `json:"method" binding:"required,oneof=GET POST PUT"`
	// 访问的 url
	URL string `json:"url" binding:"required,min=1"`
	// url 查询参数，选填
	Params map[string]string `json:"params"`
	// 请求体，选填
	Body string `json:"body"`
}

// InstanceExecuteTrpcAdminCmdResultOutputObj 单个实例执行 Trpc 管理命令的结果。
type InstanceExecuteTrpcAdminCmdResultOutputObj struct {
	// 实例 ID
	InstanceID string `json:"instanceID"`
	// 命令执行是否成功
	Success bool `json:"success"`
	// 命令执行结果详情
	Detail string `json:"detail"`
}

// ExecuteTrpcAdminCmdOutputObjs 执行 Trpc 管理命令的输出载荷。
type ExecuteTrpcAdminCmdOutputObjs struct {
	// 结果数量
	Count int64 `json:"count,string"`
	// 查询结果
	Results []*InstanceExecuteTrpcAdminCmdResultOutputObj `json:"results"`
}

// ExecuteTrpcAdminCmdOutput 执行 Trpc 管理命令的响应。
type ExecuteTrpcAdminCmdOutput struct {
	Data *ExecuteTrpcAdminCmdOutputObjs `json:"data"`
}

// ExecuteTafAdminCmdInput 执行 TAF 管理命令的输入参数。
type ExecuteTafAdminCmdInput struct {
	// 实例 ID 列表
	InstanceIDs []string `json:"instanceIDs" binding:"required,min=1,dive,required"`
	// 执行的命令（如 "taf.viewversion", "taf.setloglevel DEBUG"）
	Command string `json:"command" binding:"required,min=1"`
}

// InstanceExecuteTafAdminCmdResultOutputObj 单个实例执行 TAF 管理命令的结果。
type InstanceExecuteTafAdminCmdResultOutputObj struct {
	// 实例 ID
	InstanceID string `json:"instanceID"`
	// 命令执行是否成功
	Success bool `json:"success"`
	// 命令执行结果详情
	Detail string `json:"detail"`
}

// ExecuteTafAdminCmdOutputObjs 执行 TAF 管理命令的输出载荷。
type ExecuteTafAdminCmdOutputObjs struct {
	// 结果数量
	Count int64 `json:"count,string"`
	// 查询结果
	Results []*InstanceExecuteTafAdminCmdResultOutputObj `json:"results"`
}

// ExecuteTafAdminCmdOutput 执行 TAF 管理命令的响应。
type ExecuteTafAdminCmdOutput struct {
	Data *ExecuteTafAdminCmdOutputObjs `json:"data"`
}

// -----------------------------------------------------------------------------
// 端口转发

// PortForwardQueryInput 应用实例端口转发查询参数。
type PortForwardQueryInput struct {
	// 目标 Pod 端口号
	RemotePort int32 `form:"remotePort" binding:"required,min=1,max=65535"`
	// CLI 本地监听端口号，用于审计
	LocalPort int32 `form:"localPort" binding:"required,min=1,max=65535"`
}
