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

package deploydiagnosis

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime/runtimemod/mods"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/podquery"
	instanceserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

const (
	// maxLogInstances 异常实例按重启次数只取前几个拉日志
	maxLogInstances = 5
	// logTailLines 每个实例最多取的日志行数，从末尾截取
	logTailLines = int64(200)
	// maxEvents 每类事件（Pod / 工作负载）最多拉取的条数
	maxEvents = 200
)

// collectSnapshot 拉取最新部署、实例、日志、事件和框架配置，组成规则用的快照
// 只有列实例失败会中断；日志和事件失败只记入 PartialReasons
func (d *Diagnoser) collectSnapshot(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
) (*rules.Snapshot, error) {
	snap := &rules.Snapshot{
		Instances:       []*rules.Instance{},
		Events:          []rules.Event{},
		PartialReasons:  []string{},
		FrameworkConfig: d.loadFrameworkConfig(ctx, app),
	}

	latest, err := d.statusSvc.GetLatestDeploy(ctx, app, env.Name, trafficLaneName)
	if err != nil {
		// 没有部署记录、或应用类型没有部署记录（如未知类型）时，交给引擎给出 no_deploy
		if errors.Is(err, deploystatus.ErrDeployRecordNotFound) || errors.Is(err, deploystatus.ErrUnsupportedAppType) {
			return snap, nil
		}
		return nil, errors.Wrap(err, "get latest deploy")
	}
	snap.Deploy = buildDeploy(latest)

	// 进行中不会跑规则，省掉列实例、拉日志和事件
	if latest.Phase == deploystatus.DeployPhaseInProgress {
		return snap, nil
	}
	// 没有部署记录（构建阶段就失败）或缺少标签选择器（Helm）时列不出该次部署的 Pod
	target := latest.Target
	if target == nil || target.Namespace == "" || len(target.LabelSelector) == 0 {
		return snap, nil
	}

	pods, err := podquery.NewPodLister(podquery.Location{
		ClusterID:     target.ClusterID,
		Namespace:     target.Namespace,
		LabelSelector: target.LabelSelector,
	}).List(ctx)
	if err != nil {
		return nil, bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeInternalServerError,
			"list instances of deploy %s",
			target.DeployID,
		)
	}
	instances, err := buildInstances(pods.Items, target.DeployID)
	if err != nil {
		return nil, bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeInternalServerError,
			"parse instances of deploy %s",
			target.DeployID,
		)
	}
	snap.Instances = instances
	snap.InstancesListed = true

	if d.fillInstanceLogs(ctx, target, snap.AbnormalInstances()) {
		snap.PartialReasons = append(snap.PartialReasons, PartialReasonLogs)
	}
	events, eventsPartial := d.listRelatedEvents(ctx, env, target, pods.Items)
	snap.Events = events
	if eventsPartial {
		snap.PartialReasons = append(snap.PartialReasons, PartialReasonEvents)
	}
	return snap, nil
}

// buildDeploy 把最新部署转成快照里的摘要，DeployID 取实例所属的部署记录而不是最新那条记录
func buildDeploy(latest *deploystatus.LatestDeploy) *rules.Deploy {
	deploy := &rules.Deploy{
		BuildID: latest.BuildID,
		Stage:   latest.Stage,
		Status:  latest.Status,
		Phase:   latest.Phase,
		Message: latest.Message,
	}
	if latest.Target != nil {
		deploy.DeployID = latest.Target.DeployID
	}
	return deploy
}

// loadFrameworkConfig 读取框架配置文件的名称、挂载目录和启动命令，任一项读失败只留空、不中断诊断
// 读取的模型：AppConfigFileDef（文件名、挂载目录）、AppModel（启动命令）
//
// FIXME: 读的是应用当前配置而非该次部署的资源快照，配置已改但未重新部署时，路径比对可能对不上正在跑的实例
func (d *Diagnoser) loadFrameworkConfig(ctx context.Context, app *bkmsapp.Application) rules.FrameworkConfig {
	// 与框架配置文件接口同一判定：Helm 类应用和 blank 框架都没有框架配置文件
	if !mods.Of(app).FrameworkFile.Supported {
		return rules.FrameworkConfig{}
	}
	fc := rules.FrameworkConfig{Supported: true}

	// 文件名和挂载目录以 framework def 为准，AppModel 上的旧字段不再随配置修改同步
	def, err := appcfg.GetFrameworkDef(ctx, store.Of(d.registry.DB, appcfg.AppConfigFileDefMeta), app.ID)
	if err != nil {
		logging.Warnf(ctx, "load framework config def for diagnosis app %s: %v", app.ID, err)
	} else if def != nil {
		fc.FileName = def.Name
		fc.FilePath = def.MountDir
	}

	am, err := store.Of(d.registry.DB, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
	if err != nil {
		logging.Warnf(ctx, "load app model for diagnosis app %s: %v", app.ID, err)
		return fc
	}
	fc.Command = buildStartCommand(am.Workload)
	return fc
}

// buildStartCommand 把启动命令和参数拼成一行展示
func buildStartCommand(w appmodel.Workload) string {
	return strings.TrimSpace(strings.Join(append(append([]string{}, w.Command...), w.Args...), " "))
}

// buildInstances 用实例列表同一套口径投影 Pod，任一 Pod 解析失败则整单失败
func buildInstances(pods []unstructured.Unstructured, deployID string) ([]*rules.Instance, error) {
	instances := make([]*rules.Instance, 0, len(pods))
	for _, pod := range pods {
		obj, err := new(instanceserializer.AppInstanceOutputObj).FromPodManifest(pod.Object, deployID)
		if err != nil {
			return nil, errors.Wrapf(err, "parse pod %s", mapx.GetStr(pod.Object, "metadata.name"))
		}
		instances = append(instances, &rules.Instance{
			ID:           obj.ID,
			Status:       obj.Status,
			RestartCount: obj.RestartCount,
			IsHealthy:    obj.IsHealthy,
			LogSource:    rules.LogSourceNone,
			LogLines:     []string{},
		})
	}
	return instances, nil
}

// fillInstanceLogs 并发给重启次数最高的若干异常实例拉日志，返回 true 表示日志未取全
func (d *Diagnoser) fillInstanceLogs(
	ctx context.Context,
	target *deploystatus.DeployTarget,
	abnormal []*rules.Instance,
) bool {
	sort.SliceStable(abnormal, func(i, j int) bool {
		return abnormal[i].RestartCount > abnormal[j].RestartCount
	})
	// 超出上限的实例不再拉日志，响应里标成 logs 未取全
	truncated := len(abnormal) > maxLogInstances
	if truncated {
		abnormal = abnormal[:maxLogInstances]
	}

	logs := instancelog.NewPodLogManager(target.ClusterID, target.Namespace, defaults.WorkloadMainContainerName)
	// 每个协程只写自己的实例和自己的结果位，不需要加锁
	complete := make([]bool, len(abnormal))
	var wg sync.WaitGroup
	for i, inst := range abnormal {
		wg.Go(func() { complete[i] = d.fillOneInstanceLogs(ctx, logs, inst) })
	}
	wg.Wait()
	return truncated || lo.Contains(complete, false)
}

// fillOneInstanceLogs 优先取重启前日志，没有再取当前日志；任一接口失败返回 false，空日志不算失败
func (d *Diagnoser) fillOneInstanceLogs(ctx context.Context, logs *instancelog.LogManager, inst *rules.Instance) bool {
	sawError := false
	// 有重启才值得读退出前日志，它最能说明进程为什么退出
	if inst.RestartCount > 0 {
		lines, ok := d.readLogLines(ctx, logs, inst.ID, true)
		if !ok {
			sawError = true
		} else if len(lines) > 0 {
			inst.LogSource = rules.LogSourcePrevious
			inst.LogHint = rules.LogHintPrevious
			inst.LogLines = lines
			return true
		}
	}

	lines, ok := d.readLogLines(ctx, logs, inst.ID, false)
	if !ok {
		return false
	}
	if len(lines) > 0 {
		inst.LogSource = rules.LogSourceCurrent
		inst.LogHint = rules.LogHintCurrent
		inst.LogLines = lines
	}
	// previous 失败但 current 成功时，最有价值的日志没取到，仍算未取全
	return !sawError
}

// readLogLines 读取主容器末尾日志并去掉空行；失败时打日志并返回未取到，不中断诊断
func (d *Diagnoser) readLogLines(
	ctx context.Context,
	logs *instancelog.LogManager,
	podName string,
	previous bool,
) ([]string, bool) {
	entries, err := logs.ListLogs(ctx, podName, previous, logTailLines)
	if err != nil {
		logging.Warnf(ctx, "list logs for pod %s previous=%t: %v", podName, previous, err)
		return nil, false
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Content != "" {
			lines = append(lines, entry.Content)
		}
	}
	return lines, true
}

// listRelatedEvents 拉取本次部署开始后的 Pod 事件和工作负载事件；失败或条数超过一页时 partial 为 true，已取到的仍返回
func (d *Diagnoser) listRelatedEvents(
	ctx context.Context,
	env *envmodel.Environment,
	target *deploystatus.DeployTarget,
	pods []unstructured.Unstructured,
) ([]rules.Event, bool) {
	lister, err := podquery.NewEventLister(ctx, env)
	if err != nil {
		logging.Warnf(ctx, "create event lister for diagnosis deploy %s: %v", target.DeployID, err)
		return []rules.Event{}, true
	}

	podRefs := podquery.BuildPodRefs(pods)
	podNames := make(map[string]struct{}, len(podRefs))
	for _, ref := range podRefs {
		podNames[ref.Name] = struct{}{}
	}
	filter := podquery.EventFilter{Page: 1, PageSize: maxEvents}
	if !target.StartedAt.IsZero() {
		filter.StartedAt = target.StartedAt.Unix()
	}

	events := make([]rules.Event, 0)
	partial := false
	// 两类分开查：合在一起时工作负载事件可能占满第一页，把更关键的 Pod 事件挤掉
	for _, refs := range [][]podquery.ResourceRef{podRefs, podquery.BuildWorkloadRefs(target.ResourceKeys)} {
		if len(refs) == 0 {
			continue
		}
		page, err := lister.List(ctx, target.Namespace, refs, filter)
		if err != nil {
			logging.Warnf(ctx, "list events for diagnosis deploy %s: %v", target.DeployID, err)
			partial = true
			continue
		}
		events = append(events, buildEvents(page, podNames)...)
		partial = partial || page.Count > int64(len(page.Data))
	}
	return events, partial
}

// buildEvents 把事件页转成快照事件，并按资源类型决定挂到实例还是分组
func buildEvents(page *clusterresources.PaginatedEvents, podNames map[string]struct{}) []rules.Event {
	events := make([]rules.Event, 0, len(page.Data))
	for _, e := range page.Data {
		events = append(events, rules.Event{
			InstanceID:   resolveEventInstanceID(e.ResourceKind, e.ResourcesName, podNames),
			ResourceKind: e.ResourceKind,
			ResourceName: e.ResourcesName,
			CreatedAt:    e.CreatedAt,
			Level:        e.Level,
			Content:      e.Content,
		})
	}
	return events
}

// resolveEventInstanceID Pod 事件挂到实例；类型缺失时仅当名称就是本次的 Pod 才挂上，其余视为工作负载事件
func resolveEventInstanceID(kind, name string, podNames map[string]struct{}) string {
	if kind == k8skind.Po {
		return name
	}
	if _, ok := podNames[name]; kind == "" && ok {
		return name
	}
	return ""
}
