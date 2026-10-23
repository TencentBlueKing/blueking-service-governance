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

package appmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/mongo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// AnnotationKeyDeployID 部署 ID 的 Annotation Key，用于在主工作负载及其 PodTemplate 中标记当前部署 ID
const AnnotationKeyDeployID = "bkms.tencent.com/deploy-id"

// Deployer appmodel 应用部署器
type Deployer struct {
	builderService          *workload.BuilderService
	db                      *mongo.Database
	polarisEnvStateManager  *polaris.PolarisEnvStateManager
	hostPortEnvStateManager *hostport.EnvStateManager

	app *bkmsapp.Application

	podManager *podManager
}

// NewDeployer 创建 appmodel 应用部署器
func NewDeployer(
	builderService *workload.BuilderService,
	db *mongo.Database,
	polarisEnvStateManager *polaris.PolarisEnvStateManager,
	hostPortEnvStateManager *hostport.EnvStateManager,
	app *bkmsapp.Application,
) *Deployer {
	return &Deployer{
		builderService:          builderService,
		db:                      db,
		polarisEnvStateManager:  polarisEnvStateManager,
		hostPortEnvStateManager: hostPortEnvStateManager,
		app:                     app,
		podManager:              &podManager{},
	}
}

// Deploy 执行部署操作，返回部署 ID，执行错误。
// strategy 不为 nil 时按发布策略分批更新（下发 Rollout CR），为 nil 时一次更新全部实例（先清理遗留的 Rollout）。
func (d *Deployer) Deploy(
	ctx context.Context,
	appModel *appmodel.AppModel,
	env *bkmsenv.Environment,
	imageRegistry *registry.ImageRegistry,
	trafficLaneName, imageTag, updateStrategy string,
	replicas int32,
	buildAutoDeployInfo *BuildAutoDeployInfo,
	strategy *releasestrategy.ReleaseStrategy,
) (deployID string, retErr error) {
	// 获取待下发的主工作负载 & 额外 k8s 资源
	buildResult, err := d.buildWorkload(
		ctx, appModel, env, imageRegistry, trafficLaneName, imageTag, updateStrategy, replicas,
	)
	if err != nil {
		return "", err
	}
	extraObjs := buildResult.ExtraObjects

	clusterID, namespace := env.Cluster.ClusterID, env.Cluster.Namespace
	// 主工作负载不支持分批发布时，在创建部署记录、下发任何资源之前拒绝
	rollout, err := newWorkloadRollout(
		cluster.NewConfig(clusterID), namespace, buildResult.WorkloadKind, buildResult.Name(), strategy,
	)
	if err != nil {
		return "", err
	}

	// 找出之前部署中下发、本次不再需要的资源；以最新的部署记录为准，必须在创建本次部署记录之前计算
	resourceKeys := d.buildResourceKeys(buildResult.WorkloadKind, buildResult.Name(), extraObjs)
	expiredResources, err := d.getExpiredResources(ctx, env.Name, trafficLaneName, resourceKeys)
	if err != nil {
		return "", errors.Wrap(err, "get expired resources")
	}
	deployID, err = d.createDeployRecord(
		ctx, env, trafficLaneName, imageTag, updateStrategy,
		replicas, buildAutoDeployInfo, buildResult, resourceKeys, strategy,
	)
	if err != nil {
		return "", errors.Wrap(err, "create deploy record")
	}

	// 部署记录已创建，后续步骤失败时将记录标记为 failed
	createdDeployID := deployID
	defer func() {
		if retErr != nil {
			d.markRecordFailed(ctx, createdDeployID, retErr)
		}
	}()

	// 确保待部署环境中已经存在对应的 ImagePullSecret
	if err = d.syncImagePullSecret(ctx, env); err != nil {
		return "", err
	}
	// 先下发额外资源（如 ConfigMap 等）到集群中
	if err = d.upsertExtraObjects(ctx, clusterID, namespace, extraObjs); err != nil {
		return "", errors.Wrap(err, "upsert extra objects")
	}
	// 按发布方式处理 Rollout 并下发主工作负载
	if err = d.applyMainWorkload(ctx, clusterID, namespace, deployID, buildResult, rollout); err != nil {
		return "", err
	}

	// 持久化已下发资源清单；异步执行，不能随请求结束而取消
	go saveResourceSnapshots(
		context.WithoutCancel(ctx),
		d.db,
		appModel.AppID,
		deployID,
		buildResult.MainWorkload,
		rollout.SnapshotObjects(extraObjs),
		buildResult.SensitiveEnvVarValues,
	)

	// 删除之前部署中下发的，现在已经不再需要的资源
	if err = d.deleteDeployedResources(ctx, clusterID, namespace, expiredResources); err != nil {
		return "", errors.Wrap(err, "delete expired resources")
	}

	// 更新应用模型部署组件
	if err = appmodel.UpdateAppModel(ctx, d.db, appModel); err != nil {
		return "", errors.Wrap(err, "update app model")
	}

	d.reconcileEnvStates(ctx, env, buildResult)
	return deployID, nil
}

// Uninstall 执行卸载操作，将最新部署的相关资源从集群中删除
func (d *Deployer) Uninstall(ctx context.Context, env *bkmsenv.Environment, trafficLaneName string) error {
	envName := env.Name
	// 获取最新部署记录
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "get latest deploy record")
	}
	// 检查是否为可卸载的状态（部署中/卸载中也可以卸载，适用于出错的情况）
	if record.Status == StatusUninstalled {
		return errors.Errorf("deploy record status is %s, cannot uninstall", record.Status)
	}

	// 将部署记录状态更新为卸载中
	record.Status = StatusUninstalling
	record.Updater = auth.MustGetUser(ctx).ID
	record.UpdatedAt = time.Now()
	if err = UpdateRecord(ctx, d.db, record); err != nil {
		return errors.Wrap(err, "update deploy record status as uninstalling")
	}
	if err = d.syncBuildAutoDeployStatus(ctx, record); err != nil {
		return errors.Wrap(err, "sync build auto deploy status as uninstalling")
	}

	// 分批发布下发的 Rollout 不在 ResourceKeys 中；最新记录可能是之后的实例更新记录，不能以发布方式判断是否存在
	if err = deleteRollout(ctx, cluster.NewConfig(record.ClusterID), record); err != nil {
		return errors.Wrap(err, "delete rollout")
	}

	// 删除本次部署所管理的资源
	if err = d.deleteDeployedResources(
		ctx, record.ClusterID, record.Namespace, record.ResourceKeys,
	); err != nil {
		return errors.Wrap(err, "delete deploy resource")
	}

	// 将部署记录状态更新为卸载完成
	record.UpdatedAt = time.Now()
	record.Status = StatusUninstalled
	record.EndedAt = time.Now()
	if err = UpdateRecord(ctx, d.db, record); err != nil {
		return errors.Wrap(err, "update deploy record status as uninstalled")
	}
	if err = d.syncBuildAutoDeployStatus(ctx, record); err != nil {
		return errors.Wrap(err, "sync build auto deploy status as uninstalled")
	}
	if err = d.polarisEnvStateManager.ReconcileAfterUninstall(ctx, d.app, envName); err != nil {
		log.Errorf(
			ctx,
			"reconcile polaris env states after uninstall failed, app=%s env=%s: %v",
			d.app.ID,
			envName,
			err,
		)
	}
	if err = d.hostPortEnvStateManager.ReconcileAfterUninstall(ctx, d.app, env); err != nil {
		log.Errorf(
			ctx,
			"reconcile hostport env states after uninstall failed, app=%s env=%s: %v",
			d.app.ID,
			envName,
			err,
		)
	}
	return nil
}

func (d *Deployer) syncBuildAutoDeployStatus(ctx context.Context, record *Record) error {
	if record == nil {
		return nil
	}
	operator, err := autodeploy.NewOperator(d.db)
	if err != nil {
		return errors.Wrap(err, "init build auto deploy operator")
	}
	patch := autodeploy.StatusPatch{
		Stage:   autodeploy.StageDeploy,
		Status:  string(record.Status),
		Message: record.Message,
	}
	if record.Status.IsStable() {
		endedAt := record.EndedAt
		patch.EndedAt = &endedAt
	}
	err = operator.UpdateStatus(ctx, autodeploy.Locator{
		AppID:    d.app.ID,
		DeployID: record.ID.Hex(),
	}, patch)
	if err != nil && !errors.Is(err, autodeploy.ErrRecordNotFound) {
		return errors.Wrap(err, "update build auto deploy record")
	}
	return nil
}

// UpdateInstances 更新应用实例镜像
//   - 灰度更新：指定部分 Pod 原地更新，不新建部署记录，返回空 deployID
//   - 全量更新：未指定 Pod 或指定了全部运行态 Pod，视为一次部署，新建部署记录并返回 deployID
func (d *Deployer) UpdateInstances(
	ctx context.Context,
	envName, trafficLaneName, imageTag, updateStrategy string,
	podNames []string,
) (deployID string, err error) {
	// 获取最新的部署记录
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return "", errors.Wrap(err, "get latest deploy record")
	}
	// 检查是否为可更新的状态
	if record.Status != StatusDeployed {
		return "", errors.Errorf("deploy record status is %s, cannot update", record.Status)
	}
	kind, _ := record.MainWorkload()
	driver, err := k8sworkload.GetMain(kind)
	if err != nil {
		return "", errors.Wrap(err, "get main workload driver")
	}
	if !driver.Capabilities().InplaceUpdate {
		return "", errors.Errorf("workload %s does not support inplace or grayscale instance update", kind)
	}

	// 获取最新的 GameDeployment
	gameDeploy, err := d.getLatestGameDeployment(ctx, record)
	if err != nil {
		return "", errors.Wrap(err, "get latest game deployment")
	}
	clusterID, namespace := record.ClusterID, record.Namespace
	// 分批发布结束后 Rollout 与标签仍保留在集群中，修改 Pod 模板前不清理会按旧策略再分批一次
	rollouts := newRolloutManager(cluster.NewConfig(clusterID))
	if err = rollouts.Cleanup(ctx, namespace, gameDeploy.Name); err != nil {
		return "", errors.Wrap(err, "clean up rollout")
	}
	// workloadPods 为工作负载下的全部 Pod，podNames 为请求指定的 Pod
	var workloadPods []unstructured.Unstructured
	if len(podNames) > 0 {
		if workloadPods, err = d.podManager.ListPods(ctx, clusterID, namespace, record.LabelSelector); err != nil {
			return "", errors.Wrap(err, "list pods for instances update")
		}
		// 指定了全部运行态 Pod，按全量更新处理
		if d.podManager.coversAllRunningPods(workloadPods, podNames) {
			podNames = nil
		}
	}
	patchBuilder := NewGameDeploymentJSONPatchBuilder(gameDeploy)
	imagePatch := patchBuilder.BuildMainContainerImagePatch(imageTag)
	patches := []map[string]any{imagePatch}
	// wantPartition 本次更新后 partition 应有的值，全量更新为 0
	var wantPartition int64
	if len(podNames) > 0 {
		// 灰度更新：调低待灰度 Pod 的 DeletionCost 使其优先更新，并通过分区保留其余 Pod 为旧版本
		shouldGrayscalePodNames := d.podManager.FilterShouldGrayscalePodNames(workloadPods, podNames, imageTag)
		record.DeletionCost--
		if err = d.podManager.SetPodDeletionCost(
			ctx, clusterID, namespace, shouldGrayscalePodNames, record.DeletionCost,
		); err != nil {
			return "", errors.Wrap(err, "set pod deletion cost")
		}

		// 分区 = 旧版本 Pod 数 - 本次待灰度 Pod 数（灰度进行中时旧版本数需扣除已更新的 Pod）
		var partition int
		if gameDeploy.Status.Canary.Revision != gameDeploy.Status.CurrentRevision {
			partition = int(*gameDeploy.Spec.Replicas-gameDeploy.Status.UpdatedReplicas) - len(shouldGrayscalePodNames)
		} else {
			partition = int(*gameDeploy.Spec.Replicas) - len(shouldGrayscalePodNames)
		}
		partition = max(0, partition)
		wantPartition = int64(partition)
		patches = append(patches, patchBuilder.BuildInplaceUpdatePatch(partition)...)
	} else if updateStrategy == string(tkex.InPlaceGameDeploymentUpdateStrategyType) {
		patches = append(patches, patchBuilder.BuildInplaceUpdatePatch(0)...)
	} else {
		updateStrategy = string(tkex.RollingGameDeploymentUpdateStrategyType)
		patches = append(patches, patchBuilder.BuildRollingUpdatePatch()...)
	}

	live, err := d.patchWorkload(ctx, clusterID, namespace, gameDeploy.Name, gvr.GameDeploy, patches)
	if err != nil {
		log.Errorf(
			ctx, "patch record %s game deployment %s with patches %+v failed, err: %v",
			record.ID.Hex(), gameDeploy.Name, patches, err,
		)
		return "", errors.Wrap(err, "patch game deployment")
	}
	// patches 中的 partition 可能被尚未同步 Rollout 删除的 webhook 覆盖，需按返回的工作负载纠正。
	// 不以本次是否删除了 Rollout 为条件：删除后的步骤失败时，重试已看不到遗留的 Rollout
	if err = rollouts.RestorePartition(ctx, live, wantPartition); err != nil {
		return "", errors.Wrapf(err, "game deployment %s/%s already patched", namespace, gameDeploy.Name)
	}

	if len(podNames) > 0 {
		// 灰度只回写 DeletionCost，不改 ImageTag
		record.Updater = auth.MustGetUser(ctx).ID
		record.UpdatedAt = time.Now()
		if err = UpdateRecord(ctx, d.db, record); err != nil {
			return "", errors.Wrap(err, "update deploy record")
		}
		return "", nil
	}

	// 全量更新：Patch 成功后再建记录，避免 Patch 失败留下失败记录阻塞后续更新
	replicas := lo.FromPtrOr(gameDeploy.Spec.Replicas, record.Replicas)
	deployID, err = d.createInstancesUpdateRecord(ctx, record, imageTag, updateStrategy, replicas)
	if err != nil {
		// 工作负载已 Patch 但记录未创建，重试同一操作即可恢复
		return "", errors.Wrapf(
			err, "game deployment %s/%s already patched with image tag %s (strategy %s), "+
				"but create instances update deploy record failed",
			namespace, gameDeploy.Name, imageTag, updateStrategy,
		)
	}
	// 资源清单快照沿用上一次部署，并同步主工作负载的镜像与更新策略
	go copyInstancesUpdateResourceSnapshots(
		context.WithoutCancel(ctx),
		d.db,
		d.app.ID,
		record.ID.Hex(),
		deployID,
		ResourceKey{Kind: k8skind.GameDeploy, Name: gameDeploy.Name},
		cast.ToString(imagePatch["value"]),
		updateStrategy,
	)
	return deployID, nil
}

// Scale 对应用实例进行扩缩容操作
func (d *Deployer) Scale(ctx context.Context, envName, trafficLaneName string, replicas int32) error {
	// 获取最新的部署记录
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "get latest deploy record")
	}
	kind, name := record.MainWorkload()
	if name == "" {
		return errors.Errorf("main workload not found in deploy record %s", record.ID.Hex())
	}
	driver, err := k8sworkload.Get(kind)
	if err != nil {
		return errors.Wrap(err, "get workload driver")
	}

	patches := []map[string]any{replicasJSONPatch(replicas)}
	if _, err = d.patchWorkload(ctx, record.ClusterID, record.Namespace, name, driver.GVR(), patches); err != nil {
		log.Errorf(
			ctx, "patch record %s %s %s with patches %+v failed, err: %v",
			record.ID.Hex(), kind, name, patches, err,
		)
		return errors.Wrapf(err, "patch %s", kind)
	}

	// 更新部署记录
	record.Updater = auth.MustGetUser(ctx).ID
	record.UpdatedAt = time.Now()
	record.Replicas = replicas
	if err = UpdateRecord(ctx, d.db, record); err != nil {
		return errors.Wrap(err, "update deploy record replicas")
	}

	// 设置副本数量
	if err = appspec.SetReplicas(ctx, d.db, d.app.ID, envName, replicas); err != nil {
		return errors.Wrap(err, "setting replicas for the env")
	}
	return nil
}

// BatchDeleteInstances 批量删除应用实例（Pod）
// 删除需要分两种情况处理：
// 1. 处于终止状态的 Pod（不受 GameDeployment 管理），直接删除
// 2. 处于运行中的 Pod，添加到 GameDeployment 的 podsToDelete 中，并相应减少副本数量
func (d *Deployer) BatchDeleteInstances(
	ctx context.Context, envName, trafficLaneName string, podNames []string,
) error {
	if len(podNames) == 0 {
		return errors.Errorf("pod names to delete is empty")
	}

	// 获取最新的部署记录
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "get latest deploy record")
	}

	// 查询 Pod 状态并分类为终止态和运行态
	terminatedPods, runningPods, err := d.podManager.ClassifyPodsByStatus(
		ctx, record.ClusterID, record.Namespace, podNames, record.LabelSelector,
	)
	if err != nil {
		return errors.Wrap(err, "classify pods by status")
	}

	// 直接删除终止态的 Pod
	if len(terminatedPods) > 0 {
		log.Infof(ctx, "deleting %d terminated pods directly: %v", len(terminatedPods), terminatedPods)
		if err = d.podManager.BatchDeleteTerminatedPods(
			ctx, record.ClusterID, record.Namespace, terminatedPods,
		); err != nil {
			return errors.Wrap(err, "batch delete terminated pods")
		}
	}

	// 如果没有运行态的 Pod 需要删除，直接返回
	if len(runningPods) == 0 {
		log.Infof(ctx, "no running pods to delete, all pods selected are terminated")
		return nil
	}
	kind, _ := record.MainWorkload()
	driver, err := k8sworkload.GetMain(kind)
	if err != nil {
		return errors.Wrap(err, "get main workload driver")
	}
	if !driver.Capabilities().SelectedPodDeletion {
		return errors.Errorf("workload %s does not support deleting selected running pods", kind)
	}

	// 通过 GameDeployment 删除运行态的 Pod
	log.Infof(ctx, "deleting %d running pods via GameDeployment: %v", len(runningPods), runningPods)
	return d.deleteRunningPodsByGameDeployment(ctx, record, runningPods)
}

// deleteRunningPodsByGameDeployment 通过 GameDeployment 删除运行态的 Pod
// 注意：该函数参数中的 podNames 是运行态的 Pod 名称列表，不能包含终止态的 Pod
func (d *Deployer) deleteRunningPodsByGameDeployment(
	ctx context.Context, record *Record, podNames []string,
) error {
	// 获取最新的 GameDeployment
	gameDeploy, err := d.getLatestGameDeployment(ctx, record)
	if err != nil {
		return errors.Wrap(err, "get latest game deployment")
	}

	// 通过计算差值，得出目标的副本数量，该副本数量不得为负数
	newReplicas := max(0, *gameDeploy.Spec.Replicas-int32(len(podNames))) //nolint:gosec // G115: integer overflow
	patchBuilder := NewGameDeploymentJSONPatchBuilder(gameDeploy)
	patches := []map[string]any{
		patchBuilder.BuildReplicasPatch(newReplicas),
		patchBuilder.BuildPodsToDeletePatch(podNames),
	}
	if _, err = d.patchWorkload(
		ctx,
		record.ClusterID,
		record.Namespace,
		gameDeploy.Name,
		gvr.GameDeploy,
		patches,
	); err != nil {
		log.Errorf(
			ctx, "patch record %s game deployment %s with patches %+v failed, err: %v",
			record.ID.Hex(), gameDeploy.Name, patches, err,
		)
		return errors.Wrap(err, "patch game deployment")
	}

	// 更新部署记录
	record.Replicas = newReplicas
	record.Updater = auth.MustGetUser(ctx).ID
	record.UpdatedAt = time.Now()
	if err = UpdateRecord(ctx, d.db, record); err != nil {
		return errors.Wrap(err, "update deploy record replicas")
	}

	// 设置副本数量
	if err = appspec.SetReplicas(ctx, d.db, record.AppID, record.EnvName, newReplicas); err != nil {
		return errors.Wrap(err, "setting replicas for the env")
	}
	return nil
}

// injectDeployID 向主工作负载的资源级与 PodTemplate 级 Annotations 注入 deployID，触发滚动更新。
func (d *Deployer) injectDeployID(result *workload.BuildResult, deployID string) {
	injectDeployIDAnno(result.ObjectMeta(), deployID)
	injectDeployIDAnno(result.TemplateMeta(), deployID)
}

func injectDeployIDAnno(meta *metav1.ObjectMeta, deployID string) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[AnnotationKeyDeployID] = deployID
}

// setupAppModel 设置工作负载名称 & 镜像
func (d *Deployer) setupAppModel(
	appModel *appmodel.AppModel,
	imageRegistry *registry.ImageRegistry,
	trafficLaneName, imageTag, updateStrategy string,
) {
	// 设置工作负载名称
	appModel.Workload.Name = d.app.Name
	// 如果指定泳道，则将泳道名作为工作负载名称前缀以做区分
	if trafficLaneName != "" {
		appModel.Workload.Name = fmt.Sprintf("%s-%s", trafficLaneName, d.app.Name)
	}

	// 根据目前用户选中的镜像仓库 + 镜像 tag 生成镜像地址
	appModel.Workload.Image = fmt.Sprintf("%s/%s:%s", imageRegistry.Registry, d.app.Name, imageTag)
	// 设置应用实例更新策略
	appModel.UpdateStrategy.Type = updateStrategy
}

// validateAppModel 校验应用模型
func (d *Deployer) validateAppModel(appModel *appmodel.AppModel) error {
	// 目前 app model 应用必须设置资源配额
	if len(appModel.Workload.Resources) == 0 {
		return errors.New("workload resources unconfigured!")
	}
	return nil
}

// upsertExtraObjects 批量更新或创建额外资源（如 ConfigMap、Secret 等）
func (d *Deployer) upsertExtraObjects(
	ctx context.Context,
	clusterID, namespace string,
	extraObjs []unstructured.Unstructured,
) error {
	clusterCfg := cluster.NewConfig(clusterID)
	for idx, obj := range extraObjs {
		// 资源合法性检查
		apiVersion, kind := obj.GetAPIVersion(), obj.GetKind()
		if apiVersion == "" || kind == "" {
			return errors.Errorf("invalid extra object %d: %s", idx, obj.GetName())
		}
		// 动态获取资源 GVR（preferred version）
		resGVR, err := discovery.GetGroupVersionResource(clusterCfg, kind, apiVersion)
		if err != nil {
			return errors.Wrapf(err, "get GVR for kind %s", kind)
		}
		// 创建或更新资源
		client := k8sclient.NewWithGVR(clusterCfg, *resGVR)
		if _, err = client.Upsert(ctx, namespace, obj.Object, metav1.PatchOptions{}); err != nil {
			return errors.Wrapf(err, "upsert apiVersion %s kind %s resource %s", apiVersion, kind, obj.GetName())
		}
	}
	return nil
}

// getGameDeployment 获取最新一次部署的 GameDeployment 资源
func (d *Deployer) getLatestGameDeployment(ctx context.Context, record *Record) (*tkex.GameDeployment, error) {
	// 获取 GameDeployment 资源名称
	var gameDeployName string
	for _, key := range record.ResourceKeys {
		if key.Kind == k8skind.GameDeploy {
			gameDeployName = key.Name
			break
		}
	}
	if gameDeployName == "" {
		return nil, errors.Errorf("game deployment not found in deploy record %s", record.ID.Hex())
	}

	// 获取 GameDeployment 资源
	client := k8sclient.NewWithGVR(cluster.NewConfig(record.ClusterID), gvr.GameDeploy)
	result, err := client.Get(ctx, record.Namespace, gameDeployName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get game deployment")
	}

	// 转换 unstructured 为 tkex.GameDeployment
	var gameDeploy tkex.GameDeployment
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, &gameDeploy); err != nil {
		return nil, errors.Wrap(err, "convert game deployment from unstructured")
	}
	return &gameDeploy, nil
}

// upsertWorkload 更新或创建主工作负载，返回集群中更新后的工作负载（含 admission webhook 的修改）
func (d *Deployer) upsertWorkload(
	ctx context.Context,
	clusterID, namespace string,
	resGVR schema.GroupVersionResource,
	obj any,
) (*unstructured.Unstructured, error) {
	manifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, errors.Wrap(err, "convert workload to unstructured")
	}

	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), resGVR)
	live, err := client.Upsert(ctx, namespace, manifest, metav1.PatchOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "upsert workload")
	}
	return live, nil
}

// patchWorkload 通过 patch 的方式更新主工作负载，返回集群中更新后的工作负载（含 admission webhook 的修改）
func (d *Deployer) patchWorkload(
	ctx context.Context,
	clusterID, namespace, name string,
	resGVR schema.GroupVersionResource,
	patches []map[string]any,
) (*unstructured.Unstructured, error) {
	patches = lo.Filter(patches, func(p map[string]any, _ int) bool {
		return len(p) > 0
	})
	patchesByte, err := json.Marshal(patches)
	if err != nil {
		return nil, errors.Wrap(err, "marshal workload patches")
	}
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), resGVR)
	live, err := client.Patch(ctx, namespace, name, types.JSONPatchType, patchesByte, metav1.PatchOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "patch workload")
	}
	return live, nil
}

// buildResourceKeys 构建资源引用信息（用于后续资源管理和追踪）。
// workloadKind / workloadName 是主工作负载的 Kind 与名称。
func (d *Deployer) buildResourceKeys(
	workloadKind, workloadName string,
	extraObjs []unstructured.Unstructured,
) ResourceKeys {
	resources := ResourceKeys{{Kind: workloadKind, Name: workloadName}}
	for _, obj := range extraObjs {
		resources = append(resources, ResourceKey{
			Kind: obj.GetKind(), Name: obj.GetName(),
		})
	}
	return resources
}

// getExpiredResources 对比检查 & 删除过期的资源
// 其中 currentResourceKeys 为当前部署需要保留的资源列表
func (d *Deployer) getExpiredResources(
	ctx context.Context,
	envName, trafficLaneName string,
	currentResourceKeys ResourceKeys,
) (ResourceKeys, error) {
	// 获取最新部署记录
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		// 如果是首次部署，是没有历史记录的，可以直接跳过
		if errors.Is(err, ErrDeployRecordNotFound) {
			log.Infof(
				ctx, "app %s env %s traffic lane %s no previous deploy record, not expired resources",
				d.app.ID, envName, trafficLaneName,
			)
			return nil, nil
		}
		return nil, errors.Wrap(err, "get latest deploy record")
	}

	// 对资源进行对比，统计出需要删除的资源信息
	return record.ResourceKeys.Diff(currentResourceKeys), nil
}

// createDeployRecord 创建部署记录
func (d *Deployer) createDeployRecord(
	ctx context.Context,
	env *bkmsenv.Environment,
	trafficLaneName, imageTag, updateStrategy string,
	replicas int32,
	buildAutoDeployInfo *BuildAutoDeployInfo,
	buildResult *workload.BuildResult,
	resourceKeys ResourceKeys,
	strategy *releasestrategy.ReleaseStrategy,
) (string, error) {
	timeNow := time.Now()
	operator := auth.MustGetUser(ctx).ID
	releaseMode := ReleaseModeAll
	var strategySnapshot *ReleaseStrategySnapshot
	if strategy != nil {
		releaseMode = ReleaseModeBatch
		strategySnapshot = &ReleaseStrategySnapshot{
			ID:      strategy.ID.Hex(),
			Name:    strategy.Name,
			Batches: strategy.Batches,
		}
	}
	record := Record{
		WorkspaceID:     d.app.WorkspaceID,
		AppID:           d.app.ID,
		EnvName:         env.Name,
		TrafficLaneName: trafficLaneName,
		ClusterID:       env.Cluster.ClusterID,
		Namespace:       env.Cluster.Namespace,
		ImageTag:        imageTag,
		UpdateStrategy:  updateStrategy,
		Replicas:        replicas,
		DeletionCost:    defaults.PodDeletionCost,
		LabelSelector:   buildResult.Selector(),
		WorkloadKind:    buildResult.WorkloadKind,
		ResourceKeys:    resourceKeys,
		ReleaseMode:     releaseMode,
		ReleaseStrategy: strategySnapshot,
		Message:         "",
		Status:          StatusDeploying,
		Extras:          NewBuildAutoDeployExtras(buildAutoDeployInfo),
		Creator:         operator,
		Updater:         operator,
		StartedAt:       timeNow,
		CreatedAt:       timeNow,
		UpdatedAt:       timeNow,
	}
	// 创建部署记录
	if err := store.Of(d.db, RecordMeta).Create(ctx, &record); err != nil {
		return "", errors.Wrapf(err, "create deploy record")
	}
	return record.ID.Hex(), nil
}

// createInstancesUpdateRecord 为全量实例更新创建部署记录
// 全量更新无论采用何种更新策略（RollingUpdate / InplaceUpdate），都视为一次部署，需写入部署记录；
// 除镜像 Tag、更新策略、副本数外，其余部署信息均沿用当前部署记录
func (d *Deployer) createInstancesUpdateRecord(
	ctx context.Context,
	base *Record,
	imageTag, updateStrategy string,
	replicas int32,
) (string, error) {
	timeNow := time.Now()
	operator := auth.MustGetUser(ctx).ID
	record := Record{
		WorkspaceID:     base.WorkspaceID,
		AppID:           base.AppID,
		EnvName:         base.EnvName,
		TrafficLaneName: base.TrafficLaneName,
		ClusterID:       base.ClusterID,
		Namespace:       base.Namespace,
		ImageTag:        imageTag,
		UpdateStrategy:  updateStrategy,
		Replicas:        replicas,
		DeletionCost:    base.DeletionCost,
		LabelSelector:   base.LabelSelector,
		WorkloadKind:    base.WorkloadKind,
		ResourceKeys:    base.ResourceKeys,
		Message:         "",
		Status:          StatusDeploying,
		Creator:         operator,
		Updater:         operator,
		StartedAt:       timeNow,
		CreatedAt:       timeNow,
		UpdatedAt:       timeNow,
	}
	if err := store.Of(d.db, RecordMeta).Create(ctx, &record); err != nil {
		return "", errors.Wrapf(err, "create deploy record based on %s", base.ID.Hex())
	}
	return record.ID.Hex(), nil
}

// deleteDeployedResources 删除指定的某次部署所管理的资源
func (d *Deployer) deleteDeployedResources(
	ctx context.Context,
	clusterID, namespace string,
	resourceKeys ResourceKeys,
) error {
	clusterCfg := cluster.NewConfig(clusterID)
	for _, key := range resourceKeys {
		// 动态获取资源 GVR（preferred version）
		resGVR, err := discovery.GetGroupVersionResource(clusterCfg, key.Kind, "")
		if err != nil {
			return errors.Wrapf(err, "get GVR for kind %s", key.Kind)
		}

		// 删除 k8s 资源
		client := k8sclient.NewWithGVR(clusterCfg, *resGVR)
		if err = client.Delete(ctx, namespace, key.Name, metav1.DeleteOptions{}); err != nil {
			return errors.Wrapf(err, "delete kind %s resource %s", key.Kind, key.Name)
		}
	}
	return nil
}

// UpdateInstancePolaris 通过部署记录定位 Pod 并更新北极星注解（权重 / 隔离）
func (d *Deployer) UpdateInstancePolaris(
	ctx context.Context,
	envName, trafficLaneName string,
	instanceIDs []string,
	weight *int32,
	isolate *bool,
) error {
	record, err := GetLatest(ctx, d.db, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "deploy record not found")
	}

	annotations := make(map[string]string)
	if weight != nil {
		annotations[polaris.AnnotationKeyWeight] = cast.ToString(*weight)
	}
	if isolate != nil {
		annotations[polaris.AnnotationKeyIsolate] = cast.ToString(*isolate)
	}

	if err = d.podManager.SetPodPolarisAnnotations(
		ctx,
		record.ClusterID,
		record.Namespace,
		instanceIDs,
		annotations,
	); err != nil {
		return errors.Wrapf(err, "set polaris annotations for app %s instances", d.app.ID)
	}
	return nil
}

// buildWorkload 按本次部署的参数生成待下发的主工作负载与额外 k8s 资源。
// 会先保存环境的副本数：构建时从环境的部署规格读取副本数
func (d *Deployer) buildWorkload(
	ctx context.Context,
	appModel *appmodel.AppModel,
	env *bkmsenv.Environment,
	imageRegistry *registry.ImageRegistry,
	trafficLaneName, imageTag, updateStrategy string,
	replicas int32,
) (*workload.BuildResult, error) {
	if err := appspec.SetReplicas(ctx, d.db, d.app.ID, env.Name, replicas); err != nil {
		return nil, errors.Wrap(err, "setting replicas for the env")
	}
	d.setupAppModel(appModel, imageRegistry, trafficLaneName, imageTag, updateStrategy)
	if err := d.validateAppModel(appModel); err != nil {
		return nil, errors.Wrap(err, "validate app model")
	}
	buildResult, err := workload.NewBuilder(d.builderService, d.app, appModel).Build(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "build app model workload")
	}
	return buildResult, nil
}

// syncImagePullSecret 确保待部署环境中已经存在应用镜像仓库对应的 ImagePullSecret
func (d *Deployer) syncImagePullSecret(ctx context.Context, env *bkmsenv.Environment) error {
	buildCfg, err := store.Of(d.db, build.ConfigMeta).Get(ctx, build.ByApp(d.app.ID))
	if err != nil {
		return errors.Wrap(err, "get build config")
	}
	return errors.Wrap(
		secret.NewImagePullSecretSyncer(d.db, env, d.app.ID, buildCfg).Sync(ctx), "sync image pull secret",
	)
}

// applyMainWorkload 下发主工作负载。Rollout 的存在与否决定了本次更新是否分批，
// 因此更新前按发布方式下发或清理 Rollout，更新后确认 partition 符合本次的发布方式
func (d *Deployer) applyMainWorkload(
	ctx context.Context,
	clusterID, namespace, deployID string,
	buildResult *workload.BuildResult,
	rollout *workloadRollout,
) error {
	// 在主工作负载中注入 deployID，确保每次部署都能触发 Pod 滚动更新
	d.injectDeployID(buildResult, deployID)
	if err := rollout.EnsureRollout(ctx, buildResult.ObjectMeta()); err != nil {
		return err
	}
	liveWorkload, err := d.upsertWorkload(ctx, clusterID, namespace, buildResult.GVR(), buildResult.MainWorkload)
	if err != nil {
		return errors.Wrap(err, "upsert main workload")
	}
	return rollout.EnsurePartition(ctx, liveWorkload)
}

// reconcileEnvStates 部署完成后调谐北极星、HostPort 的环境状态：部署已带上最新 PolarisConfig，
// 记录集群中的关键字段并清理已离开 scope 的环境信息。失败只打日志，不影响部署结果
func (d *Deployer) reconcileEnvStates(
	ctx context.Context, env *bkmsenv.Environment, buildResult *workload.BuildResult,
) {
	if err := d.polarisEnvStateManager.ReconcileAfterDeploy(ctx, d.app, env); err != nil {
		log.Errorf(ctx, "reconcile polaris config after deploy failed, app=%s env=%s: %v", d.app.ID, env.Name, err)
	}
	if err := d.hostPortEnvStateManager.ReconcileAfterDeploy(
		ctx, d.app, env, buildResult.HostPortAppliedPorts,
	); err != nil {
		log.Errorf(ctx, "reconcile hostport after deploy failed, app=%s env=%s: %v", d.app.ID, env.Name, err)
	}
}

// markRecordFailed 将部署记录标记为失败，cause 作为失败信息；标记失败只打日志。
// 部署可能正是因 ctx 取消而失败，标记时不能随之取消，否则记录会一直停留在部署中
func (d *Deployer) markRecordFailed(ctx context.Context, deployID string, cause error) {
	ctx = context.WithoutCancel(ctx)
	record, err := GetRecord(ctx, d.db, d.app.ID, deployID)
	if err != nil {
		log.Errorf(ctx, "mark record failed: get deploy record %s: %v", deployID, err)
		return
	}
	record.Status = StatusFailed
	record.Message = cause.Error()
	record.EndedAt = time.Now()
	if err = UpdateRecord(ctx, d.db, record); err != nil {
		log.Errorf(ctx, "mark record failed: update deploy record %s: %v", deployID, err)
	}
}

func replicasJSONPatch(replicas int32) map[string]any {
	return map[string]any{
		"op":    "replace",
		"path":  "/spec/replicas",
		"value": replicas,
	}
}
