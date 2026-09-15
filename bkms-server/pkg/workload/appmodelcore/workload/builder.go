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

package workload

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/secret"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/cfgrender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/plainfiles"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/runtimerender"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

const (
	// tkeRouteEniAnnotationKey is the Pod annotation key that enables TKE Route ENI (VPC-CNI) networking.
	tkeRouteEniAnnotationKey = "tke.cloud.tencent.com/networks"
	// tkeRouteEniAnnotationValue is the annotation value that activates VPC-CNI mode.
	tkeRouteEniAnnotationValue = "tke-route-eni"
)

// Build builds the workload resource for the given application and environment.
//
// Returns:
// - The main workload resource, extra resources, and masking metadata.
// - Undefined environment-variable references are included in BuildResult without failing the build.
// - An optional error if any occurs during the process.
func (b *Builder) Build(
	ctx context.Context,
	env *envmodel.Environment,
) (*BuildResult, error) {
	appModel, err := b.ComputeAppModel(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "computing app model")
	}

	appEnvVars, err := envvars.BuildAppEnvVars(ctx, b.app, appModel, env, b.envVarsReader)
	if err != nil {
		return nil, errors.Wrap(err, "building app env vars")
	}

	varsMap := appEnvVars.ToMap()
	collector := envvarrefs.NewCollector(varsMap)
	extraObjs := make([]unstructured.Unstructured, 0)

	// ── 主容器 spec（镜像、探针、资源配额、生命周期） ──

	containerSpec, err := b.buildContainerSpec(appModel, appEnvVars)
	if err != nil {
		return nil, errors.Wrap(err, "building container spec")
	}
	resourceReq, err := buildResourceRequirements(appModel.Workload.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "building resource requirements")
	}
	if resourceReq != nil {
		containerSpec.Resources = *resourceReq
	}

	// 用户自定义 volume mounts
	volumeMounts, volumes, err := buildVolumeMounts(appModel.Workload.VolumeMounts)
	if err != nil {
		return nil, errors.Wrap(err, "building volume mounts")
	}

	var initContainers []corev1.Container

	// ── Framework plugin 提供的存储、额外资源和 init container ──

	wlPlugin, err := plugin.GetWorkloadPlugin(appModel.Workload.Type)
	if err != nil {
		return nil, errors.Wrap(err, "resolving workload plugin")
	}
	pluginSession, err := wlPlugin.Start(ctx, env, b.app, appModel, plugin.RenderContext{
		EnvVars:   varsMap,
		Collector: collector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "starting workload plugin")
	}

	pluginMounts, pluginVolumes, err := pluginSession.Storage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin storage")
	}
	volumeMounts = append(volumeMounts, pluginMounts...)
	volumes = append(volumes, pluginVolumes...)

	pluginExtraObjs, err := pluginSession.ExtraResources(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin extra resources")
	}
	extraObjs = append(extraObjs, pluginExtraObjs...)

	pluginInits, err := pluginSession.InitContainers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin init containers")
	}
	for i := range pluginInits {
		pluginInits[i].Env = appEnvVars.ToKubeObjs()
	}
	initContainers = append(initContainers, pluginInits...)

	// ── Plain 配置文件 ──

	plainMounts, plainVolumes, plainExtras, plainInits, err := b.buildPlainConfigFiles(
		ctx, appModel, env, varsMap, collector,
	)
	if err != nil {
		return nil, errors.Wrap(err, "building plain config files")
	}
	volumeMounts = append(volumeMounts, plainMounts...)
	volumes = append(volumes, plainVolumes...)
	extraObjs = append(extraObjs, plainExtras...)
	for i := range plainInits {
		plainInits[i].Env = appEnvVars.ToKubeObjs()
	}
	initContainers = append(initContainers, plainInits...)

	// ── Dev mode 组件 ──

	devModeOutput, err := devmode.New(b.devModeConfig).Build()
	if err != nil {
		return nil, errors.Wrap(err, "building dev mode component")
	}
	if devModeOutput != nil {
		if extraObjs, err = AppendAsUnstructured(extraObjs, devModeOutput.ConfigMap); err != nil {
			return nil, errors.Wrap(err, "appending dev mode config map as unstructured")
		}
		volumes = append(volumes, devModeOutput.Volume)
		volumeMounts = append(volumeMounts, devModeOutput.VolumeMount)
		containerSpec.Command = devModeOutput.Command
		containerSpec.Args = nil
	}

	containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, volumeMounts...)

	// ── 组装 GameDeployment ──

	gd, err := b.buildGameDeployment(ctx, env, appModel, containerSpec, volumes, initContainers)
	if err != nil {
		return nil, err
	}

	// ── 后处理：注入 metadata、hostport、BSCP、components、polaris + 校验 ──

	gd, hostPortAppliedPorts, err := b.applyPostProcessing(
		ctx, env, appModel, varsMap, collector, &extraObjs, gd,
	)
	if err != nil {
		return nil, err
	}

	// ── 汇总结果 ──

	result := &BuildResult{
		ExtraObjects:          extraObjs,
		SensitiveEnvVarValues: buildSensitiveEnvVarValues(appEnvVars),
		UndefinedEnvVars:      collector.UndefinedEnvVars(),
		HostPortAppliedPorts:  hostPortAppliedPorts,
	}
	if env.Cluster.IsFederation {
		result.WorkloadKind = kind.Deploy
		result.MainWorkload = gameDeploymentToDeployment(gd)
	} else {
		result.WorkloadKind = kind.GameDeploy
		result.MainWorkload = &gd
	}
	return result, nil
}

// buildGameDeployment 构造 GameDeployment 资源对象（含 strategy、secrets、PodTemplate）。
func (b *Builder) buildGameDeployment(
	ctx context.Context,
	env *envmodel.Environment,
	appModel *appmodel.AppModel,
	containerSpec corev1.Container,
	volumes []corev1.Volume,
	initContainers []corev1.Container,
) (tkex.GameDeployment, error) {
	buildCfg, err := b.buildConfigStore.Get(ctx, b.app.ID)
	if err != nil {
		return tkex.GameDeployment{}, errors.Wrap(err, "get build config")
	}

	secretNames := appModel.Workload.ImagePullSecrets
	if name := secret.ResolveImagePullSecretName(
		env.WorkspaceID, b.app.ID, buildCfg,
	); !lo.Contains(secretNames, name) {
		secretNames = append(secretNames, name)
	}

	gdStrategy := buildUpdateStrategy(appModel.UpdateStrategy)
	name := appModel.Workload.Name
	imagePullSecrets := lo.Map(secretNames, func(name string, _ int) corev1.LocalObjectReference {
		return corev1.LocalObjectReference{Name: name}
	})

	gd := tkex.GameDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind.GameDeploy,
			APIVersion: gvr.GameDeploy.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"controller.kubernetes.io/pod-deletion-cost":    strconv.Itoa(defaults.PodDeletionCost),
				"io.tencent.bcs.dev/update-strategy-type-allow": "true",
			},
			Labels: map[string]string{
				// TODO 未来需要评估是否改成使用 Cascading，即用户需要先缩容 pod 数到 0，才能删除 GameDeployment
				"io.tencent.bcs.dev/deletion-allow": "Always",
			},
		},
		Spec: tkex.GameDeploymentSpec{
			UpdateStrategy: gdStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"controller.kubernetes.io/pod-deletion-cost": strconv.Itoa(defaults.PodDeletionCost),
					},
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets,
					InitContainers:                initContainers,
					Containers:                    []corev1.Container{containerSpec},
					Volumes:                       volumes,
					TerminationGracePeriodSeconds: appModel.Workload.TerminationGracePeriodSeconds,
				},
			},
		},
	}
	if appModel.Replicas != nil {
		gd.Spec.Replicas = appModel.Replicas
	}
	return gd, nil
}

// buildUpdateStrategy 根据 AppModel 配置构造 GameDeployment 更新策略。
func buildUpdateStrategy(strategy *appmodel.UpdateStrategy) tkex.GameDeploymentUpdateStrategy {
	gdStrategy := tkex.GameDeploymentUpdateStrategy{
		MaxUnavailable: lo.ToPtr(intstr.Parse(defaults.MaxUnavailable)),
		MaxSurge:       lo.ToPtr(intstr.Parse(defaults.MaxSurge)),
		Partition:      lo.ToPtr(intstr.FromInt32(0)),
	}
	if strategy == nil {
		return gdStrategy
	}
	if strategy.Type != "" {
		gdStrategy.Type = tkex.GameDeploymentUpdateStrategyType(strategy.Type)
	}
	if strategy.MaxUnavailable != nil {
		gdStrategy.MaxUnavailable = lo.ToPtr(intstr.Parse(*strategy.MaxUnavailable))
	}
	if strategy.MaxSurge != nil {
		gdStrategy.MaxSurge = lo.ToPtr(intstr.Parse(*strategy.MaxSurge))
	}
	return gdStrategy
}

// applyPostProcessing 在 GameDeployment 构造完成后注入各类扩展并做最终校验：
// metadata、TKE ENI、HostPort、BSCP、Components、Polaris、MountPath 冲突检测。
// extraObjs 通过指针传入，方法内会 append components 和 polaris 产出的额外资源。
func (b *Builder) applyPostProcessing(
	ctx context.Context,
	env *envmodel.Environment,
	appModel *appmodel.AppModel,
	varsMap map[string]string,
	collector *envvarrefs.Collector,
	extraObjs *[]unstructured.Unstructured,
	gd tkex.GameDeployment,
) (tkex.GameDeployment, []int32, error) {
	mergeUserMetadata(&gd.Spec.Template.ObjectMeta, appModel.Labels, appModel.Annotations)

	if appModel.TkeRouteEni {
		if gd.Spec.Template.Annotations == nil {
			gd.Spec.Template.Annotations = make(map[string]string)
		}
		gd.Spec.Template.Annotations[tkeRouteEniAnnotationKey] = tkeRouteEniAnnotationValue
	}

	// HostPort（联邦环境）
	var hostPortAppliedPorts []int32
	if env.Cluster.IsFederation {
		appliedPorts, err := hostport.InjectFromStore(
			ctx, b.hostPortStore, b.app.ID,
			&gd.Spec.Template.ObjectMeta,
			gd.Spec.Template.Spec.Containers,
			defaults.WorkloadMainContainerName,
		)
		if err != nil {
			return gd, nil, errors.Wrap(err, "injecting hostport")
		}
		hostPortAppliedPorts = appliedPorts
	}

	// BSCP
	if err := bscpcfg.InjectFromStore(
		ctx, b.bscpCfgStore, b.app.ID, env.Name,
		defaults.WorkloadMainContainerName, &gd.Spec.Template.Spec,
	); err != nil {
		return gd, nil, errors.Wrap(err, "injecting bscp config")
	}

	// Components
	comps, err := b.ListComponents(ctx, env, appModel)
	if err != nil {
		return gd, nil, errors.Wrap(err, "listing components")
	}
	compApplier, err := component.CreateDefaultApplier()
	if err != nil {
		return gd, nil, errors.Wrap(err, "creating component applier")
	}
	evaluatedComps := make([]*component.EvaluatedComponent, 0, len(comps))
	for _, comp := range comps {
		evaluated, evaluateErr := compApplier.Evaluate(ctx, b.app, *comp, env.ID, varsMap, collector)
		if evaluateErr != nil {
			return gd, nil, errors.Wrapf(evaluateErr, "evaluating component %s", comp.Name)
		}
		evaluatedComps = append(evaluatedComps, evaluated)
	}
	gd, extraCompObjs, err := b.applyComponents(gd, evaluatedComps)
	if err != nil {
		return gd, nil, errors.Wrap(err, "applying components")
	}
	*extraObjs = append(*extraObjs, extraCompObjs...)

	// Polaris
	polarisResult, err := b.polarisWorkloadBuilder.Build(
		ctx, b.app, env, varsMap,
		gd.Spec.Template.Spec, collector,
	)
	if err != nil {
		return gd, nil, errors.Wrap(err, "applying polaris configs")
	}
	gd.Spec.Template.Spec = polarisResult.PodSpec
	*extraObjs = append(*extraObjs, polarisResult.ExtraObjects...)

	// 全局 MountPath 唯一性校验
	if err = validateAllMountPaths(&gd); err != nil {
		return gd, nil, err
	}

	return gd, hostPortAppliedPorts, nil
}

// validateAllMountPaths 校验所有容器的 MountPath 唯一性。
func validateAllMountPaths(gd *tkex.GameDeployment) error {
	for i := range gd.Spec.Template.Spec.Containers {
		c := &gd.Spec.Template.Spec.Containers[i]
		if err := validateVolumeMountPaths(c.Name, c.VolumeMounts); err != nil {
			return errors.Wrap(err, "global mount path conflict")
		}
	}
	for i := range gd.Spec.Template.Spec.InitContainers {
		c := &gd.Spec.Template.Spec.InitContainers[i]
		if err := validateVolumeMountPaths(c.Name, c.VolumeMounts); err != nil {
			return errors.Wrap(err, "global mount path conflict")
		}
	}
	return nil
}

// buildPlainConfigFiles 构建 plain 配置文件的 K8s 资源。
//
// 根据 EnableEnvVarRender 开关，plain 文件分为两条路径：
//   - renderParams（EnableEnvVarRender=true）→ cfgrender + runtimerender 管线（ConfigMap + emptyDir + init container）
//   - directParams（EnableEnvVarRender=false）→ 直接 ConfigMap 挂载（无 init container）
//
// 两组产物各自生成 mounts/volumes/extras/inits，合并后返回。
func (b *Builder) buildPlainConfigFiles(
	ctx context.Context,
	appModel *appmodel.AppModel,
	env *envmodel.Environment,
	varsMap map[string]string,
	collector *envvarrefs.Collector,
) ([]corev1.VolumeMount, []corev1.Volume, []unstructured.Unstructured, []corev1.Container, error) {
	buildResult, err := plainfiles.BuildPlainConfigFiles(
		ctx, b.mountableFileProvider, b.app.ID, env.Name, varsMap, collector,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if buildResult == nil {
		return nil, nil, nil, nil, nil
	}

	var (
		allMounts []corev1.VolumeMount
		allVols   []corev1.Volume
		allExtras []unstructured.Unstructured
		allInits  []corev1.Container
	)

	// 需要 env var 渲染的文件走 runtimerender（含 init container）
	if len(buildResult.RenderParams) > 0 {
		plainCfg, renderErr := runtimerender.BuildConfig(runtimerender.ConfigParams{
			WorkloadType:  "plain-cfg",
			ConfigMapName: fmt.Sprintf("%s-plain-cfg", appModel.Workload.Name),
			Files:         buildResult.RenderParams,
		})
		if renderErr != nil {
			return nil, nil, nil, nil, errors.Wrap(renderErr, "runtime render")
		}
		mounts, volumes, storageErr := plainCfg.Storage(ctx)
		if storageErr != nil {
			return nil, nil, nil, nil, errors.Wrap(storageErr, "storage")
		}
		extras, extrasErr := plainCfg.ExtraResources(ctx)
		if extrasErr != nil {
			return nil, nil, nil, nil, errors.Wrap(extrasErr, "extra resources")
		}
		inits, initsErr := plainCfg.InitContainers(ctx)
		if initsErr != nil {
			return nil, nil, nil, nil, errors.Wrap(initsErr, "init containers")
		}
		allMounts = append(allMounts, mounts...)
		allVols = append(allVols, volumes...)
		allExtras = append(allExtras, extras...)
		allInits = append(allInits, inits...)
	}

	// 不需要渲染的文件走直接 ConfigMap 挂载（无 init container）
	if len(buildResult.DirectParams) > 0 {
		directCfg := buildDirectConfigMap(
			fmt.Sprintf("%s-plain-direct", appModel.Workload.Name),
			buildResult.DirectParams,
		)
		mounts, volumes, storageErr := directCfg.Storage(ctx)
		if storageErr != nil {
			return nil, nil, nil, nil, errors.Wrap(storageErr, "direct storage")
		}
		extras, extrasErr := directCfg.ExtraResources(ctx)
		if extrasErr != nil {
			return nil, nil, nil, nil, errors.Wrap(extrasErr, "direct extra resources")
		}
		allMounts = append(allMounts, mounts...)
		allVols = append(allVols, volumes...)
		allExtras = append(allExtras, extras...)
	}

	return allMounts, allVols, allExtras, allInits, nil
}

// directConfigMapResult 直接 ConfigMap 挂载的产物，无 init container。
type directConfigMapResult struct {
	mounts    []corev1.VolumeMount
	volumes   []corev1.Volume
	configMap corev1.ConfigMap
}

// Storage 返回直接 ConfigMap 挂载的 volume mounts 和 volumes。
func (d *directConfigMapResult) Storage(_ context.Context) ([]corev1.VolumeMount, []corev1.Volume, error) {
	return d.mounts, d.volumes, nil
}

// ExtraResources 返回 ConfigMap 作为额外资源。
func (d *directConfigMapResult) ExtraResources(_ context.Context) ([]unstructured.Unstructured, error) {
	if d.configMap.Name == "" {
		return nil, nil
	}
	return plugin.ToUnstructured(&d.configMap)
}

// buildDirectConfigMap 为不需要 env var 渲染的 plain 文件构建直接 ConfigMap 挂载。
// 内容直接写入 ConfigMap data，主容器通过 subPath 挂载到目标路径，无需 init container。
func buildDirectConfigMap(configMapName string, files []cfgrender.ConfigFileParams) *directConfigMapResult {
	volumeName := configMapName

	cm := corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: configMapName},
		Data:       make(map[string]string, len(files)),
	}

	items := make([]corev1.KeyToPath, 0, len(files))
	mounts := make([]corev1.VolumeMount, 0, len(files))

	for i, file := range files {
		key := fmt.Sprintf("%02d-%s", i, file.FileName)
		cm.Data[key] = file.FileContent

		items = append(items, corev1.KeyToPath{
			Key:  key,
			Path: key,
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: filepath.Join(file.FilePath, file.FileName),
			SubPath:   key,
		})
	}

	vol := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				Items:                items,
			},
		},
	}

	return &directConfigMapResult{
		mounts:    mounts,
		volumes:   []corev1.Volume{vol},
		configMap: cm,
	}
}

func buildSensitiveEnvVarValues(appEnvVars envvartypes.EnvVariableList) map[string]string {
	sensitiveValues := make(map[string]string)
	// Deduplicate with the same priority rule as ToKubeObjs (list is low→high priority;
	// last wins). Otherwise an earlier sensitive source could remain after a later
	// non-sensitive override became the effective value.
	for _, item := range appEnvVars.ToDeduplicatedList() {
		if !item.IsSensitive {
			continue
		}
		sensitiveValues[item.Key] = item.Value
	}
	return sensitiveValues
}

func shouldEnableDevModeInEnv(envType string, effectiveAppSpec *appspec.AppSpec) bool {
	if effectiveAppSpec == nil || effectiveAppSpec.DevMode == nil || effectiveAppSpec.DevMode.Enabled == nil {
		return false
	}
	if !*effectiveAppSpec.DevMode.Enabled {
		return false
	}
	return bkmsenv.IsValidEnvType(envType) && !bkmsenv.IsProductionType(bkmsenv.Type(envType))
}

// validateVolumeMountPaths 校验单个容器的 VolumeMounts 中不存在重复的 MountPath。
// 多个来源（framework plugin、plain files、dev mode、BSCP、components）的挂载最终
// 都汇聚到同一个 Container，重复 MountPath 会导致 Pod 创建失败或后者静默覆盖前者。
func validateVolumeMountPaths(containerName string, mounts []corev1.VolumeMount) error {
	seen := make(map[string]string, len(mounts))
	for _, m := range mounts {
		if prev, exists := seen[m.MountPath]; exists {
			return fmt.Errorf(
				"container %q: duplicate MountPath %q: volume %q conflicts with volume %q",
				containerName, m.MountPath, m.Name, prev,
			)
		}
		seen[m.MountPath] = m.Name
	}
	return nil
}

// mergeUserMetadata merges user-defined labels/annotations into the given ObjectMeta. System-managed
// keys are rejected upfront by the labels/annotations section validators, so user entries only add
// to (never collide with) the system-managed metadata. Empty user maps leave the ObjectMeta untouched.
func mergeUserMetadata(meta *metav1.ObjectMeta, labels, annotations map[string]string) {
	if len(labels) > 0 {
		if meta.Labels == nil {
			meta.Labels = make(map[string]string, len(labels))
		}
		maps.Copy(meta.Labels, labels)
	}
	if len(annotations) > 0 {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string, len(annotations))
		}
		maps.Copy(meta.Annotations, annotations)
	}
}

// Build the main container specification.
func (b *Builder) buildContainerSpec(
	appModel *appmodel.AppModel,
	appEnvVars envvartypes.EnvVariableList,
) (corev1.Container, error) {
	empty := corev1.Container{}

	if appModel.Workload.Image == "" {
		return empty, errors.New("workload image is not specified")
	}

	c := corev1.Container{
		Name:    defaults.WorkloadMainContainerName,
		Image:   appModel.Workload.Image,
		Command: appModel.Workload.Command,
		Args:    appModel.Workload.Args,
	}

	c.Env = appEnvVars.ToKubeObjs()

	livenessProbe, err := buildProbe(appModel.Workload.LivenessProbe)
	if err != nil {
		return empty, errors.Wrap(err, "building liveness probe")
	}
	readinessProbe, err := buildProbe(appModel.Workload.ReadinessProbe)
	if err != nil {
		return empty, errors.Wrap(err, "building readiness probe")
	}
	startupProbe, err := buildProbe(appModel.Workload.StartupProbe)
	if err != nil {
		return empty, errors.Wrap(err, "building startup probe")
	}

	if livenessProbe != nil {
		c.LivenessProbe = livenessProbe
	}
	if readinessProbe != nil {
		c.ReadinessProbe = readinessProbe
	}
	if startupProbe != nil {
		c.StartupProbe = startupProbe
	}

	if appModel.Workload.ImagePullPolicy != "" {
		c.ImagePullPolicy = corev1.PullPolicy(appModel.Workload.ImagePullPolicy)
	}

	lifecycle, err := buildLifecycle(appModel.Workload.Lifecycle)
	if err != nil {
		return empty, errors.Wrap(err, "building lifecycle")
	}
	if lifecycle != nil {
		c.Lifecycle = lifecycle
	}

	return c, nil
}

// ListComponents lists the components used by the application.
//
// For components that reference workspace-level components via RefWorkspaceCompName,
// their definitions are loaded from the workspace and substituted here.
func (b *Builder) ListComponents(
	ctx context.Context,
	env *envmodel.Environment,
	appModel *appmodel.AppModel,
) ([]*component.Component, error) {
	comps := slices.Clone(appModel.Components)
	return b.resolveWorkspaceComps(ctx, env, comps)
}

// resolveWorkspaceComps resolves components that reference workspace-level components.
// For components with RefWorkspaceCompName set, it loads the actual component definition
// from the workspace and substitutes it.
//
// Returns only valid components that should be used:
//   - Instance components (non-reference) are always included
//   - Reference components are resolved and included only if the workspace component
//     is available in the current environment (based on ScopeType and ScopeEnvNames)
func (b *Builder) resolveWorkspaceComps(
	ctx context.Context,
	env *envmodel.Environment,
	comps []*component.Component,
) ([]*component.Component, error) {
	// Check if there are any referenced workspace components
	hasRefs := false
	for _, comp := range comps {
		if comp.RefWorkspaceCompName != "" {
			hasRefs = true
			break
		}
	}
	if !hasRefs {
		// No referenced workspace components, return directly
		return comps, nil
	}

	// Load the workspace components and build a lookup map
	workspaceComps, err := b.workspaceCompsStore.ListByWorkspace(ctx, env.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "getting workspace comps")
	}
	workspaceCompMap := make(map[string]*workspace.Component, len(workspaceComps))
	for _, comp := range workspaceComps {
		workspaceCompMap[comp.Name] = comp
	}

	// Build result list with only valid components
	result := make([]*component.Component, 0, len(comps))
	for _, comp := range comps {
		// Instance component (non-reference), include directly
		if comp.RefWorkspaceCompName == "" {
			result = append(result, comp)
			continue
		}

		// Reference component, resolve from workspace
		workspaceComp, exists := workspaceCompMap[comp.RefWorkspaceCompName]
		if !exists {
			return nil, errors.Errorf("referenced workspace component not found: %s", comp.RefWorkspaceCompName)
		}

		// Skip if the workspace component is not available in current environment
		if !workspaceComp.IsAvailableInEnv(env.Name) {
			log.Infof(ctx, "workspace component %s [%s/%s] is not available in environment %s, skip",
				workspaceComp.Name, workspaceComp.ScopeType, strings.Join(workspaceComp.ScopeEnvNames, ","), env.Name)
			continue
		}

		// Substitute the reference with resolved component definition
		resolved := &component.Component{
			Name: workspaceComp.Name,
			ComponentInst: component.ComponentInst{
				Type:       workspaceComp.Type,
				Version:    workspaceComp.Version,
				Properties: workspaceComp.Properties,
			},
		}
		result = append(result, resolved)
	}

	return result, nil
}

// applyComponents applies component patchers and specs to the GameDeployment.
//
// Returns:
// - The main workload resource after applying all outputs.
// - Any extra resources generated as unstructured objects.
// - An optional error if any occurs during the process.
func (b *Builder) applyComponents(
	gd tkex.GameDeployment,
	evaluatedComps []*component.EvaluatedComponent,
) (tkex.GameDeployment, []unstructured.Unstructured, error) {
	extraObjs := make([]unstructured.Unstructured, 0)
	for compIndex, evaluated := range evaluatedComps {
		patched, patchErr := component.ApplyGameDeploymentPatchers(gd, evaluated.Patchers)
		if patchErr != nil {
			return gd, nil, errors.Wrapf(patchErr, "applying component[%d] patchers", compIndex)
		}
		gd = patched

		for specIndex, spec := range evaluated.Specs {
			// 需要将 yaml 反序列化得到的 map[string]any 转换成标准的 Unstructured 必须经过这一步。
			// 这里传 *map 给 ToUnstructured，让 converter 递归把 int 等值标准化为 int64，避免 DeepCopy Panic。
			converted, convertErr := runtime.DefaultUnstructuredConverter.ToUnstructured(&spec)
			if convertErr != nil {
				return gd, nil, errors.Wrapf(
					convertErr, "converting component[%d] spec[%d] to unstructured", compIndex, specIndex,
				)
			}
			extraObjs = append(extraObjs, unstructured.Unstructured{Object: converted})
		}
	}

	return gd, extraObjs, nil
}

// ComputeAppModel compute the original AppModel in the given env, the result includes overrides from
// app specs based on environment.
func (b *Builder) ComputeAppModel(
	ctx context.Context,
	env *envmodel.Environment,
) (*appmodel.AppModel, error) {
	// Make a deep copy of the original AppModel to avoid modifying it directly
	var appModel appmodel.AppModel
	if err := copier.CopyWithOption(&appModel, b.appModel, copier.Option{DeepCopy: true}); err != nil {
		return nil, errors.Wrap(err, "copying app model")
	}

	effectiveAppSpec, err := appspec.GetEnvEffective(ctx, b.appSpecStore, b.appModelStore, b.app.ID, env.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "getting effective app spec for app %s env %s", b.app.ID, env.Name)
	}
	appspec.ApplyToAppModel(effectiveAppSpec, &appModel)

	// 检查：不是生产环境，且启用了开发模式
	b.devModeConfig = nil
	if shouldEnableDevModeInEnv(env.Type, effectiveAppSpec) {
		b.devModeConfig = devmode.CreateDevModeConfig(&appModel, env.Type, true)
		if effectiveAppSpec.DevMode.WorkPath != nil {
			b.devModeConfig.WorkPath = *effectiveAppSpec.DevMode.WorkPath
		}
		if effectiveAppSpec.DevMode.MountPath != nil {
			b.devModeConfig.MountPath = *effectiveAppSpec.DevMode.MountPath
		}
	}

	return &appModel, nil
}
