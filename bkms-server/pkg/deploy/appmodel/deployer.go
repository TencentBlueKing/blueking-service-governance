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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// AnnotationKeyDeployID 部署 ID 的 Annotation Key，用于在 GameDeployment 及其 PodTemplate 中标记当前部署 ID
const AnnotationKeyDeployID = "bkms.tencent.com/deploy-id"

// Deployer appmodel 应用部署器
type Deployer struct {
	recordStore            RecordStore
	resourceSnapshotStore  ResourceSnapshotStore
	buildDeployStore       autodeploy.RecordStore
	appModelStore          appmodel.AppModelStore
	builderService         *workload.BuilderService
	appSpecStore           appspec.AppSpecStore
	buildConfigStore       build.ConfigStore
	appConfigFileStore     appcfg.AppConfigFileStore
	polarisEnvStateManager *polaris.PolarisEnvStateManager

	app *bkmsapp.Application

	podManager *podManager
}

// NewDeployer 创建 appmodel 应用部署器
func NewDeployer(
	recordStore RecordStore,
	resourceSnapshotStore ResourceSnapshotStore,
	buildDeployStore autodeploy.RecordStore,
	appModelStore appmodel.AppModelStore,
	builderService *workload.BuilderService,
	appSpecStore appspec.AppSpecStore,
	buildConfigStore build.ConfigStore,
	appConfigFileStore appcfg.AppConfigFileStore,
	polarisEnvStateManager *polaris.PolarisEnvStateManager,
	app *bkmsapp.Application,
) *Deployer {
	return &Deployer{
		recordStore:            recordStore,
		resourceSnapshotStore:  resourceSnapshotStore,
		buildDeployStore:       buildDeployStore,
		appModelStore:          appModelStore,
		builderService:         builderService,
		appSpecStore:           appSpecStore,
		buildConfigStore:       buildConfigStore,
		appConfigFileStore:     appConfigFileStore,
		polarisEnvStateManager: polarisEnvStateManager,
		app:                    app,
		podManager:             &podManager{},
	}
}

// Deploy 执行部署操作，返回部署 ID，执行错误
func (d *Deployer) Deploy(
	ctx context.Context,
	appModel *appmodel.AppModel,
	env *bkmsenv.Environment,
	imageRegistry *registry.ImageRegistry,
	trafficLaneName, imageTag, updateStrategy string,
	replicas int32,
) (deployID string, retErr error) {
	// 前置准备 & 预设配置
	// 设置环境下的部署规格，调整 replicas 副本数
	if err := appspec.SetReplicas(ctx, d.appSpecStore, d.app.ID, env.Name, replicas); err != nil {
		return "", errors.Wrap(err, "setting replicas for the env")
	}
	d.setupAppModel(appModel, imageRegistry, trafficLaneName, imageTag, updateStrategy)
	// 校验应用模型
	if err := d.validateAppModel(appModel); err != nil {
		return "", errors.Wrap(err, "validate app model")
	}

	// 获取待下发的主工作负载 & 额外 k8s 资源
	builder := workload.NewBuilder(d.builderService, d.app, appModel)
	buildResult, err := builder.Build(ctx, env)
	if err != nil {
		return "", errors.Wrap(err, "build app model workload")
	}
	gameDeploy := buildResult.GameDeployment
	extraObjs := buildResult.ExtraObjects

	// 构建当前部署的 资源引用信息 并创建部署记录
	resourceKeys := d.buildResourceKeys(gameDeploy, extraObjs)
	// 删除之前部署中下发的，现在已经不再需要的资源
	expiredResources, err := d.getExpiredResources(ctx, env.Name, trafficLaneName, resourceKeys)
	if err != nil {
		return "", errors.Wrap(err, "get expired resources")
	}
	deployID, err = d.createDeployRecord(
		ctx, env, trafficLaneName, imageTag, updateStrategy,
		replicas, gameDeploy.Spec.Selector.MatchLabels, resourceKeys,
	)
	if err != nil {
		return "", errors.Wrap(err, "create deploy record")
	}

	// 部署记录已创建，后续步骤失败时将记录标记为 failed
	createdDeployID := deployID
	defer func() {
		if retErr != nil {
			record, getErr := d.recordStore.Get(ctx, d.app.ID, createdDeployID)
			if getErr != nil {
				log.Errorf(ctx, "mark record failed: get deploy record %s: %v", createdDeployID, getErr)
				return
			}
			record.Status = StatusFailed
			record.Message = retErr.Error()
			record.EndedAt = time.Now()
			if updateErr := d.recordStore.Update(ctx, record); updateErr != nil {
				log.Errorf(ctx, "mark record failed: update deploy record %s: %v", createdDeployID, updateErr)
			}
		}
	}()

	buildCfg, err := d.buildConfigStore.Get(ctx, d.app.ID)
	if err != nil {
		return "", errors.Wrap(err, "get build config")
	}

	// 确保待部署环境中已经存在对应的 ImagePullSecret
	if err = secret.NewImagePullSecretSyncer(env, d.app.ID, buildCfg).Sync(ctx); err != nil {
		return "", errors.Wrap(err, "sync image pull secret")
	}

	clusterID, namespace := env.Cluster.ClusterID, env.Cluster.Namespace
	// 先下发额外资源（如 ConfigMap 等）到集群中
	if err = d.upsertExtraObjects(ctx, clusterID, namespace, extraObjs); err != nil {
		return "", errors.Wrap(err, "upsert extra objects")
	}
	// 在 GameDeployment 中注入 deployID，确保每次部署都能触发 Pod 滚动更新
	d.injectDeployID(gameDeploy, deployID)
	// 下发主工作负载资源（GameDeployment）到集群中
	if err = d.upsertGameDeployment(ctx, clusterID, namespace, gameDeploy); err != nil {
		return "", errors.Wrap(err, "upsert game deployment")
	}

	// 持久化已下发资源清单
	go saveResourceSnapshots(
		ctx,
		d.resourceSnapshotStore,
		appModel.AppID,
		deployID,
		gameDeploy,
		extraObjs,
		buildResult.SensitiveEnvVarValues,
	)

	// 删除之前部署中下发的，现在已经不再需要的资源
	if err = d.deleteDeployedResources(ctx, clusterID, namespace, expiredResources); err != nil {
		return "", errors.Wrap(err, "delete expired resources")
	}

	// 更新应用模型部署组件
	if err = d.appModelStore.UpdateAppModel(ctx, appModel); err != nil {
		return "", errors.Wrap(err, "update app model")
	}

	// 部署已带上最新 PolarisConfig，记录集群中的关键字段并清理已离开 scope 的环境信息。
	if err = d.polarisEnvStateManager.ReconcileAfterDeploy(ctx, d.app, env); err != nil {
		log.Errorf(ctx, "reconcile polaris config after deploy failed, app=%s env=%s: %v",
			d.app.ID, env.Name, err)
	}

	return deployID, nil
}

// Uninstall 执行卸载操作，将最新部署的相关资源从集群中删除
func (d *Deployer) Uninstall(ctx context.Context, envName, trafficLaneName string) error {
	// 获取最新部署记录
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
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
	if err = d.recordStore.Update(ctx, record); err != nil {
		return errors.Wrap(err, "update deploy record status as uninstalling")
	}
	if err = d.syncBuildAutoDeployStatus(ctx, record); err != nil {
		return errors.Wrap(err, "sync build auto deploy status as uninstalling")
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
	if err = d.recordStore.Update(ctx, record); err != nil {
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
	return nil
}

func (d *Deployer) syncBuildAutoDeployStatus(ctx context.Context, record *Record) error {
	if d.buildDeployStore == nil || record == nil {
		return nil
	}
	operator, err := autodeploy.NewOperator(d.buildDeployStore)
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

// UpdateInstances 更新部分/全量应用实例
// 使用场景：
// 1. 灰度更新功能：指定几个 Pod，执行仅镜像更新操作（InplaceUpdate）
// 2. 全量更新功能：执行全量 Pod 镜像更新操作（RollingUpdate / InplaceUpdate）
func (d *Deployer) UpdateInstances(
	ctx context.Context,
	envName, trafficLaneName, imageTag, updateStrategy string,
	podNames []string,
) error {
	// 获取最新的部署记录
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "get latest deploy record")
	}
	// 检查是否为可更新的状态
	if record.Status != StatusDeployed {
		return errors.Errorf("deploy record status is %s, cannot update", record.Status)
	}

	// 获取最新的 GameDeployment
	gameDeploy, err := d.getLatestGameDeployment(ctx, record)
	if err != nil {
		return errors.Wrap(err, "get latest game deployment")
	}

	clusterID, namespace := record.ClusterID, record.Namespace
	// 通过 Patch 的方式更新 GameDeployment 更新策略
	patchBuilder := NewGameDeploymentJSONPatchBuilder(gameDeploy)
	patches := []map[string]any{
		patchBuilder.BuildMainContainerImagePatch(imageTag),
	}
	// 如果指定实例，则表示是灰度更新，需要设置 DeletionCost + 分区
	if len(podNames) > 0 {
		// 查询 Pod 状态并分类为终止态和运行态，只有运行态的，且镜像 TAG 与灰度镜像不同的需要参与灰度 & 分区设置
		shouldGrayscalePodNames, cErr := d.podManager.FilterShouldGrayscalePodNames(
			ctx, record.ClusterID, record.Namespace, imageTag, podNames, record.LabelSelector,
		)
		if cErr != nil {
			return errors.Wrap(cErr, "classify pods by status")
		}

		// 先执行 Pod 的 Patch，设置 deletionCost 为上一次的值 -1 确保这部分 Pods 优先被更新
		record.DeletionCost--
		if err = d.podManager.SetPodDeletionCost(
			ctx, clusterID, namespace, shouldGrayscalePodNames, record.DeletionCost,
		); err != nil {
			return errors.Wrap(err, "set pod deletion cost")
		}

		// 计算分区，即不需要更新的 Pod 数量（本次灰度排除在外的）
		// 处于灰度更新中时，分区应该用老版本的 Pod 数量，减去待灰度的 Pod 数量
		var partition int
		if gameDeploy.Status.Canary.Revision != gameDeploy.Status.CurrentRevision {
			partition = int(*gameDeploy.Spec.Replicas-gameDeploy.Status.UpdatedReplicas) - len(shouldGrayscalePodNames)
		} else {
			partition = int(*gameDeploy.Spec.Replicas) - len(shouldGrayscalePodNames)
		}
		partition = max(0, partition)
		patches = append(patches, patchBuilder.BuildInplaceUpdatePatch(partition)...)
	} else if updateStrategy == string(tkex.InPlaceGameDeploymentUpdateStrategyType) {
		// 全量原地更新，不需要提前设置 DeletionCost，也不需要分区
		patches = append(patches, patchBuilder.BuildInplaceUpdatePatch(0)...)
	} else {
		// 默认设置更新策略为滚动更新
		patches = append(patches, patchBuilder.BuildRollingUpdatePatch())
	}

	if err = d.patchGameDeployment(ctx, clusterID, namespace, gameDeploy.Name, patches); err != nil {
		log.Errorf(
			ctx, "patch record %s game deployment %s with patches %+v failed, err: %v",
			record.ID.Hex(), gameDeploy.Name, patches, err,
		)
		return errors.Wrap(err, "patch game deployment")
	}

	// 更新部署记录，灰度部分镜像 Tag 不会影响到部署记录的 ImageTag
	record.Updater = auth.MustGetUser(ctx).ID
	record.UpdatedAt = time.Now()
	if err = d.recordStore.Update(ctx, record); err != nil {
		return errors.Wrap(err, "update deploy record")
	}
	return nil
}

// Scale 对应用实例进行扩缩容操作
func (d *Deployer) Scale(ctx context.Context, envName, trafficLaneName string, replicas int32) error {
	// 获取最新的部署记录
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
	if err != nil {
		return errors.Wrap(err, "get latest deploy record")
	}
	// 获取最新的 GameDeployment
	gameDeploy, err := d.getLatestGameDeployment(ctx, record)
	if err != nil {
		return errors.Wrap(err, "get latest game deployment")
	}

	// 通过 Patch 的方式更新 GameDeployment 副本数量
	patches := []map[string]any{
		NewGameDeploymentJSONPatchBuilder(gameDeploy).BuildReplicasPatch(replicas),
	}
	if err = d.patchGameDeployment(ctx, record.ClusterID, record.Namespace, gameDeploy.Name, patches); err != nil {
		log.Errorf(
			ctx, "patch record %s game deployment %s with patches %+v failed, err: %v",
			record.ID.Hex(), gameDeploy.Name, patches, err,
		)
		return errors.Wrap(err, "patch game deployment")
	}

	// 更新部署记录
	record.Updater = auth.MustGetUser(ctx).ID
	record.UpdatedAt = time.Now()
	record.Replicas = replicas
	if err = d.recordStore.Update(ctx, record); err != nil {
		return errors.Wrap(err, "update deploy record replicas")
	}

	// 设置副本数量
	if err = appspec.SetReplicas(ctx, d.appSpecStore, d.app.ID, envName, replicas); err != nil {
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
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
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
	if err = d.patchGameDeployment(
		ctx, record.ClusterID, record.Namespace, gameDeploy.Name, patches,
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
	if err = d.recordStore.Update(ctx, record); err != nil {
		return errors.Wrap(err, "update deploy record replicas")
	}

	// 设置副本数量
	if err = appspec.SetReplicas(ctx, d.appSpecStore, record.AppID, record.EnvName, newReplicas); err != nil {
		return errors.Wrap(err, "setting replicas for the env")
	}
	return nil
}

// injectDeployID 向 GameDeployment 的两个层级 Annotations 中注入 deployID
// 1. GameDeployment.ObjectMeta.Annotations —— 资源级别标记，便于运维查询和追溯
// 2. GameDeployment.Spec.Template.ObjectMeta.Annotations —— PodTemplate 级别标记，触发 Pod 滚动更新
func (d *Deployer) injectDeployID(gameDeploy *tkex.GameDeployment, deployID string) {
	// 注入到 GameDeployment 资源级别的 Annotations
	if gameDeploy.Annotations == nil {
		gameDeploy.Annotations = make(map[string]string)
	}
	gameDeploy.Annotations[AnnotationKeyDeployID] = deployID

	// 注入到 PodTemplate 级别的 Annotations，确保 PodTemplate 变化触发滚动更新
	if gameDeploy.Spec.Template.Annotations == nil {
		gameDeploy.Spec.Template.Annotations = make(map[string]string)
	}
	gameDeploy.Spec.Template.Annotations[AnnotationKeyDeployID] = deployID
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

// upsertGameDeployment 更新或创建 GameDeployment 资源
func (d *Deployer) upsertGameDeployment(
	ctx context.Context,
	clusterID, namespace string,
	gameDeploy *tkex.GameDeployment,
) error {
	// 使用 k8s runtime 提供的转换器，避免 json 转换导致 int 变 float64 的问题
	manifest, err := runtime.DefaultUnstructuredConverter.ToUnstructured(gameDeploy)
	if err != nil {
		return errors.Wrap(err, "convert game deployment to unstructured")
	}

	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.GameDeploy)
	if _, err = client.Upsert(ctx, namespace, manifest, metav1.PatchOptions{}); err != nil {
		return errors.Wrap(err, "upsert game deployment")
	}
	return nil
}

// patchGameDeployment 通过 patch 的方式更新 GameDeployment 资源
func (d *Deployer) patchGameDeployment(
	ctx context.Context,
	clusterID, namespace, gameDeployName string,
	patches []map[string]any,
) error {
	// 过滤掉空 patch
	patches = lo.Filter(patches, func(p map[string]any, _ int) bool {
		return len(p) > 0
	})
	// patch 序列化
	patchesByte, err := json.Marshal(patches)
	if err != nil {
		return errors.Wrap(err, "marshal game deployment patches")
	}
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.GameDeploy)
	// patch GameDeployment 资源
	if _, err = client.Patch(
		ctx, namespace, gameDeployName, types.JSONPatchType, patchesByte, metav1.PatchOptions{},
	); err != nil {
		return errors.Wrap(err, "patch game deployment")
	}
	return nil
}

// buildResourceKeys 构建资源引用信息（用于后续资源管理和追踪）
func (d *Deployer) buildResourceKeys(
	gameDeploy *tkex.GameDeployment,
	extraObjs []unstructured.Unstructured,
) ResourceKeys {
	// 主工作负载
	resources := ResourceKeys{{Kind: gameDeploy.Kind, Name: gameDeploy.Name}}
	// 其他资源
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
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
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
	labelSelector map[string]string,
	resourceKeys ResourceKeys,
) (string, error) {
	timeNow := time.Now()
	operator := auth.MustGetUser(ctx).ID
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
		LabelSelector:   labelSelector,
		ResourceKeys:    resourceKeys,
		Message:         "",
		Status:          StatusDeploying,
		Creator:         operator,
		Updater:         operator,
		StartedAt:       timeNow,
		CreatedAt:       timeNow,
		UpdatedAt:       timeNow,
	}
	// 创建部署记录
	deployID, err := d.recordStore.Create(ctx, &record)
	if err != nil {
		return "", errors.Wrapf(err, "create deploy record")
	}
	return deployID, nil
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
	record, err := d.recordStore.GetLatest(ctx, d.app.ID, envName, trafficLaneName)
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
