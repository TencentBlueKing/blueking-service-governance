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

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	diag "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
	instanceserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

const (
	// maxLogInstances 异常实例按重启次数只取前几个拉日志
	maxLogInstances = 5
	// logTailLines 每个实例最多取的日志行数，从末尾截取
	logTailLines = int64(200)
	// maxEvents 本次部署开始后最多拉取的事件条数
	maxEvents = 200
)

// latestDeploy 当前环境最新一次部署，以及列出实例、拉日志和事件所需的集群位置
type latestDeploy struct {
	deployID      string
	status        string
	message       string
	clusterID     string
	namespace     string
	labelSelector map[string]string
	// resourceKeys 本次部署创建的工作负载，用来关联环境事件
	resourceKeys []appmodeldeploy.ResourceKey
	// startedAtUnix 部署开始时间，事件只取这之后的
	startedAtUnix int64
	// canListPods 有命名空间和标签选择器才能列 Pod；Helm 记录没有选择器，为 false
	canListPods bool
}

// collectSnapshot 拉取最新部署、实例、日志、事件和框架配置，组成规则用的快照
func (d *Diagnoser) collectSnapshot(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	trafficLaneName string,
) (*diag.Snapshot, error) {
	latest, err := d.resolveLatestDeploy(ctx, app, env.Name, trafficLaneName)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve latest deploy for app %s env %s", app.ID, env.Name)
	}

	snap := &diag.Snapshot{
		AppType:   app.Type,
		Deploy:    buildDeployOutput(latest),
		Instances: []*diag.Instance{},
		Events:    []diag.Event{},
		Framework: d.loadFrameworkConfig(ctx, app),
	}
	if latest == nil || !latest.canListPods {
		return snap, nil
	}

	instances, err := d.listInstances(ctx, latest)
	if err != nil {
		return nil, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list instances for app %s env %s", app.ID, env.Name,
		)
	}
	snap.Instances = instances
	// 日志或事件失败本波次仍给结论，partial 恒为 false
	d.fillInstanceLogs(ctx, latest, snap.Instances)
	snap.Events = d.listRelatedEvents(ctx, env, latest, snap.Instances)
	return snap, nil
}

func buildDeployOutput(latest *latestDeploy) serializer.DeployOutputObj {
	if latest == nil {
		return serializer.DeployOutputObj{}
	}
	return serializer.DeployOutputObj{
		DeployID: latest.deployID,
		Status:   latest.status,
		Message:  latest.message,
	}
}

// resolveLatestDeploy 按应用类型取该环境最新一次部署，没有记录时返回 nil
func (d *Diagnoser) resolveLatestDeploy(
	ctx context.Context,
	app *bkmsapp.Application,
	envName, trafficLaneName string,
) (*latestDeploy, error) {
	if bkmsapp.IsAppModelType(app.Type) {
		return d.resolveAppModelDeploy(ctx, app.ID, envName, trafficLaneName)
	}
	if bkmsapp.IsHelmBasedType(app.Type) {
		return d.resolveHelmDeploy(ctx, app.ID, envName, trafficLaneName)
	}
	return nil, nil
}

// resolveAppModelDeploy 读取应用模型的最新部署，并带上列 Pod 用的命名空间和标签
func (d *Diagnoser) resolveAppModelDeploy(
	ctx context.Context,
	appID, envName, trafficLaneName string,
) (*latestDeploy, error) {
	record, err := d.registry.AppModelDeployRecordStore.GetLatest(ctx, appID, envName, trafficLaneName)
	if err != nil {
		if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest app model deploy record")
	}
	startedAt := int64(0)
	if !record.StartedAt.IsZero() {
		startedAt = record.StartedAt.Unix()
	}
	return &latestDeploy{
		deployID:      record.ID.Hex(),
		status:        string(record.Status),
		message:       record.Message,
		clusterID:     record.ClusterID,
		namespace:     record.Namespace,
		labelSelector: record.LabelSelector,
		resourceKeys:  record.ResourceKeys,
		startedAtUnix: startedAt,
		canListPods:   len(record.LabelSelector) > 0 && record.Namespace != "",
	}, nil
}

// resolveHelmDeploy 读取 Helm 最新部署；记录没有标签选择器，因此不能列 Pod
func (d *Diagnoser) resolveHelmDeploy(
	ctx context.Context,
	appID, envName, trafficLaneName string,
) (*latestDeploy, error) {
	record, err := d.registry.HelmDeployRecordStore.GetLatest(ctx, appID, envName, trafficLaneName)
	if err != nil {
		if errors.Is(err, helmdeploy.ErrLatestDeployRecordNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest helm deploy record")
	}
	// Helm 记录没有 LabelSelector，本波次只带部署状态，实例列表留给后续规则全集
	return &latestDeploy{
		deployID:  record.ID.Hex(),
		status:    string(record.Status),
		message:   record.Message,
		clusterID: record.ClusterID,
		namespace: record.Namespace,
	}, nil
}

// loadFrameworkConfig 读取 tRPC / TAF 的配置文件名、路径和启动命令，读失败则留空
func (d *Diagnoser) loadFrameworkConfig(ctx context.Context, app *bkmsapp.Application) diag.Framework {
	if !bkmsapp.IsAppModelType(app.Type) {
		return diag.Framework{}
	}
	am, err := d.registry.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return diag.Framework{}
	}
	return frameworkFromAppModel(app.Type, am)
}

func frameworkFromAppModel(appType string, am *appmodel.AppModel) diag.Framework {
	fw := diag.Framework{
		Command: strings.TrimSpace(strings.Join(append(am.Workload.Command, am.Workload.Args...), " ")),
	}
	switch appType {
	case bkmsapp.AppTypeTRPC:
		fw.FileName = am.Workload.TrpcConfig.FileName
		fw.FilePath = am.Workload.TrpcConfig.FilePath
	case bkmsapp.AppTypeTAF:
		fw.FileName = am.Workload.TafConfig.FileName
		fw.FilePath = am.Workload.TafConfig.FilePath
	}
	return fw
}

// listInstances 按部署记录的命名空间和标签列出 Pod，列失败则整单失败
func (d *Diagnoser) listInstances(ctx context.Context, latest *latestDeploy) ([]*diag.Instance, error) {
	client := k8sclient.NewPodClient(cluster.NewConfig(latest.clusterID))
	selector := labels.SelectorFromSet(latest.labelSelector).String()
	pods, err := client.List(ctx, latest.namespace, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, errors.Wrapf(err, "list namespace %s labelSelector [%s] pods", latest.namespace, selector)
	}
	instances := make([]*diag.Instance, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj, err := new(instanceserializer.AppInstanceOutputObj).FromPodManifest(pod.Object, latest.deployID)
		if err != nil {
			return nil, errors.Wrapf(err, "parse pod %s", mapx.GetStr(pod.Object, "metadata.name"))
		}
		instances = append(instances, &diag.Instance{
			ID:           obj.ID,
			Status:       obj.Status,
			RestartCount: obj.RestartCount,
			IsHealthy:    obj.IsHealthy,
			LogSource:    serializer.LogSourceNone,
			LogLines:     []string{},
		})
	}
	return instances, nil
}

// fillInstanceLogs 只给重启次数最高的若干异常实例拉日志
func (d *Diagnoser) fillInstanceLogs(ctx context.Context, latest *latestDeploy, instances []*diag.Instance) {
	abnormal := make([]*diag.Instance, 0, len(instances))
	for _, inst := range instances {
		if !inst.IsHealthy {
			abnormal = append(abnormal, inst)
		}
	}
	sort.SliceStable(abnormal, func(i, j int) bool {
		return abnormal[i].RestartCount > abnormal[j].RestartCount
	})
	if len(abnormal) > maxLogInstances {
		abnormal = abnormal[:maxLogInstances]
	}

	client := k8sclient.NewPodClient(cluster.NewConfig(latest.clusterID))
	for _, inst := range abnormal {
		d.fillOneInstanceLogs(ctx, client, latest.namespace, inst)
	}
}

// fillOneInstanceLogs 优先取重启前日志，没有再取当前日志
func (d *Diagnoser) fillOneInstanceLogs(
	ctx context.Context,
	client *k8sclient.PodClient,
	namespace string,
	inst *diag.Instance,
) {
	// previous 优先：有重启才值得读退出前日志
	if inst.RestartCount > 0 {
		if lines, ok := d.readLogLines(ctx, client, namespace, inst.ID, true); ok && len(lines) > 0 {
			inst.LogSource = serializer.LogSourcePrevious
			inst.LogHint = previousLogHint
			inst.LogLines = lines
			return
		}
	}
	if lines, ok := d.readLogLines(ctx, client, namespace, inst.ID, false); ok && len(lines) > 0 {
		inst.LogSource = serializer.LogSourceCurrent
		inst.LogHint = currentLogHint
		inst.LogLines = lines
	}
}

// readLogLines 读取主容器末尾日志；失败时返回未取到，不中断诊断
func (d *Diagnoser) readLogLines(
	ctx context.Context,
	client *k8sclient.PodClient,
	namespace, podName string,
	previous bool,
) ([]string, bool) {
	tail := logTailLines
	entries, err := client.ListLogs(ctx, namespace, podName, &corev1.PodLogOptions{
		Container: defaults.WorkloadMainContainerName,
		Previous:  previous,
		TailLines: &tail,
	})
	if err != nil {
		return nil, false
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Content == "" {
			continue
		}
		lines = append(lines, entry.Content)
	}
	return lines, true
}

// listRelatedEvents 拉取本次部署开始后、与工作负载和 Pod 相关的事件；失败则返回空列表
func (d *Diagnoser) listRelatedEvents(
	ctx context.Context,
	env *envmodel.Environment,
	latest *latestDeploy,
	instances []*diag.Instance,
) []diag.Event {
	user, err := auth.GetUser(ctx)
	if err != nil {
		return []diag.Event{}
	}
	client, err := clusterresources.New(user)
	if err != nil {
		return []diag.Event{}
	}

	resourceKinds := make([]string, 0, len(latest.resourceKeys)+len(instances))
	resourceNames := make([]string, 0, len(latest.resourceKeys)+len(instances))
	for _, rk := range latest.resourceKeys {
		resourceKinds = append(resourceKinds, rk.Kind)
		resourceNames = append(resourceNames, rk.Name)
	}
	for _, inst := range instances {
		resourceKinds = append(resourceKinds, k8skind.Po)
		resourceNames = append(resourceNames, inst.ID)
	}

	paginated, err := client.ListEvents(
		ctx,
		env.Cluster.ProjectCode,
		env.Cluster.ClusterID,
		clusterresources.ListEventParams{
			Namespace:     latest.namespace,
			ResourceKinds: resourceKinds,
			ResourceNames: resourceNames,
			StartedAt:     latest.startedAtUnix,
			Page:          1,
			PageSize:      maxEvents,
		},
	)
	if err != nil || paginated == nil {
		return []diag.Event{}
	}

	events := make([]diag.Event, 0, len(paginated.Data))
	for _, e := range paginated.Data {
		events = append(events, diag.Event{
			InstanceID: e.ResourcesName,
			CreatedAt:  e.CreatedAt,
			Level:      e.Level,
			Content:    e.Content,
		})
	}
	return events
}
