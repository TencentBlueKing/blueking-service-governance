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
	"maps"
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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
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
//
// FIXME 这个函数过长（260+ 行），需要后续重构拆分
func (b *Builder) Build(
	ctx context.Context,
	env *envmodel.Environment,
) (*BuildResult, error) {
	// Compute the AppModel with environment overrides applied and use it for building
	// the workload afterward.
	appModel, err := b.ComputeAppModel(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "computing app model")
	}

	// Create and start the workload plugin session.
	//
	// A plugin session can modify the workload building process by providing
	// extra storage, extra resources, etc.
	wlPlugin, err := plugin.GetWorkloadPlugin(appModel.Workload.Type)
	if err != nil {
		return nil, errors.Wrap(err, "resolving workload plugin")
	}

	// Build env vars once, used by both plugin and component rendering.
	// Polaris ServiceLabels use the same map.
	appEnvVars, err := envvars.BuildAppEnvVars(ctx, b.app, appModel, env, b.envVarsReader)
	if err != nil {
		return nil, errors.Wrap(err, "building app env vars")
	}
	sensitiveEnvVarValues := buildSensitiveEnvVarValues(appEnvVars)
	varsMap := appEnvVars.ToMap()
	collector := envvarrefs.NewCollector(varsMap)

	pluginSession, err := wlPlugin.Start(ctx, env, b.app, appModel, plugin.RenderContext{
		EnvVars:   varsMap,
		Collector: collector,
	})
	if err != nil {
		return nil, errors.Wrap(err, "starting workload plugin")
	}

	// extraObjs holds any extra resources, by extra, we mean resources other than the main workload resource,
	// such as ConfigMaps, Secrets, etc.
	extraObjs := make([]unstructured.Unstructured, 0)

	// Build and set the container spec
	containerSpec, err := b.buildContainerSpec(appModel, appEnvVars)
	if err != nil {
		return nil, errors.Wrap(err, "building container spec")
	}

	// Try to build and set the resource requirements
	resourceReq, err := buildResourceRequirements(appModel.Workload.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "building resource requirements")
	}
	if resourceReq != nil {
		containerSpec.Resources = *resourceReq
	}

	// Try to build and set the volume mounts and volumes
	volumeMounts, volumes, err := buildVolumeMounts(appModel.Workload.VolumeMounts)
	if err != nil {
		return nil, errors.Wrap(err, "building volume mounts")
	}

	// Get and append plugin provided storage
	pluginMounts, pluginVolumes, err := pluginSession.Storage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin storage")
	}
	volumeMounts = append(volumeMounts, pluginMounts...)
	volumes = append(volumes, pluginVolumes...)

	// Get and append plugin provided extra resources
	pluginExtraObjs, err := pluginSession.ExtraResources(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin extra resources")
	}
	extraObjs = append(extraObjs, pluginExtraObjs...)

	// Get plugin provided init containers and inject env vars so that
	// runtime variables (e.g., BKMS_POD_IP from Downward API) are available
	// for tools like envsubst inside the init container.
	pluginInitContainers, err := pluginSession.InitContainers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "building plugin init containers")
	}
	for i := range pluginInitContainers {
		pluginInitContainers[i].Env = appEnvVars.ToKubeObjs()
	}

	// Build the dev mode component
	devModeBuilder := devmode.New(b.devModeConfig)
	// 如果不符合条件，不构建，返回值也都为空
	devModeOutput, err := devModeBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "building dev mode component")
	}
	// 如果开发模式启用，应用开发模式配置
	if devModeOutput != nil {
		// 添加 DevMode ConfigMap 到额外资源
		if extraObjs, err = AppendAsUnstructured(extraObjs, devModeOutput.ConfigMap); err != nil {
			return nil, errors.Wrap(err, "appending dev mode config map as unstructured")
		}
		// 添加 Volume 和 VolumeMount
		volumes = append(volumes, devModeOutput.Volume)
		volumeMounts = append(volumeMounts, devModeOutput.VolumeMount)
		// 替换容器启动命令
		containerSpec.Command = devModeOutput.Command
		// 清空 Args，因为 init.sh 会处理启动逻辑
		containerSpec.Args = nil
	}

	if len(volumeMounts) > 0 {
		containerSpec.VolumeMounts = append(containerSpec.VolumeMounts, volumeMounts...)
	}

	// Update the spec to include the default image pull secret.
	// The Secret resource should be created by the "secret" module before any deployments were
	// performed, there's no need to construct the resource here. We only need to reference it in
	// the main workload spec.
	buildCfg, err := b.buildConfigStore.Get(ctx, b.app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get build config")
	}

	secretNames := appModel.Workload.ImagePullSecrets
	if name := secret.ResolveImagePullSecretName(
		env.WorkspaceID, b.app.ID, buildCfg,
	); !lo.Contains(secretNames, name) {
		secretNames = append(secretNames, name)
	}

	// Set the update strategy if specified
	gdStrategy := tkex.GameDeploymentUpdateStrategy{
		MaxUnavailable: lo.ToPtr(intstr.Parse(defaults.MaxUnavailable)),
		MaxSurge:       lo.ToPtr(intstr.Parse(defaults.MaxSurge)),
		// 构造 GameDeployment 时，分区需要设置为 0，确保所有 Pod 都能滚动
		Partition: lo.ToPtr(intstr.FromInt32(0)),
	}
	if strategy := appModel.UpdateStrategy; strategy != nil {
		if strategy.Type != "" {
			gdStrategy.Type = tkex.GameDeploymentUpdateStrategyType(strategy.Type)
		}
		if strategy.MaxUnavailable != nil {
			gdStrategy.MaxUnavailable = lo.ToPtr(intstr.Parse(*strategy.MaxUnavailable))
		}
		if strategy.MaxSurge != nil {
			gdStrategy.MaxSurge = lo.ToPtr(intstr.Parse(*strategy.MaxSurge))
		}
	}

	// `name` if the main name used in for constructing the resource
	// NOTE: Do Name of Workload always exist? Should we use the name of application as fallback?
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
				"controller.kubernetes.io/pod-deletion-cost": strconv.Itoa(defaults.PodDeletionCost),
				// 允许变更 GameDeployment 的 update strategy
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
					InitContainers:                pluginInitContainers,
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

	// Inject user-defined labels/annotations into the Pod Template's metadata.
	mergeUserMetadata(&gd.Spec.Template.ObjectMeta, appModel.Labels, appModel.Annotations)

	// Inject TKE Route ENI (VPC-CNI) annotation when the flag is set by appspec.
	if appModel.TkeRouteEni {
		if gd.Spec.Template.Annotations == nil {
			gd.Spec.Template.Annotations = make(map[string]string)
		}
		gd.Spec.Template.Annotations[tkeRouteEniAnnotationKey] = tkeRouteEniAnnotationValue
	}

	// Inject BSCP configuration management artifacts (initContainer, sidecar, volume, etc.)
	if err = bscpcfg.InjectFromStore(
		ctx, b.bscpCfgStore, b.app.ID, env.Name,
		defaults.WorkloadMainContainerName, &gd.Spec.Template.Spec,
	); err != nil {
		return nil, errors.Wrap(err, "injecting bscp config")
	}

	// Apply the components, the components may modify the main workload resource
	// or generate extra resources.
	comps, err := b.ListComponents(ctx, env, appModel)
	if err != nil {
		return nil, errors.Wrap(err, "listing components")
	}
	compApplier, err := component.CreateDefaultApplier()
	if err != nil {
		return nil, errors.Wrap(err, "creating component applier")
	}

	// Evaluate component patchers and specs in component order.
	evaluatedComps := make([]*component.EvaluatedComponent, 0, len(comps))
	for _, comp := range comps {
		evaluated, evaluateErr := compApplier.Evaluate(ctx, b.app, *comp, env.ID, varsMap, collector)
		if evaluateErr != nil {
			return nil, errors.Wrapf(evaluateErr, "evaluating component %s", comp.Name)
		}
		evaluatedComps = append(evaluatedComps, evaluated)
	}

	// Apply the evaluated components to the main workload resource and collect any extra resources.
	gd, extraCompObjs, err := b.applyComponents(gd, evaluatedComps)
	if err != nil {
		return nil, errors.Wrap(err, "applying components")
	}
	extraObjs = append(extraObjs, extraCompObjs...)

	// Polaris: construct PolarisConfig/Service CRs into extra objects.
	polarisResult, err := b.polarisWorkloadBuilder.Build(
		ctx, b.app, env, varsMap,
		gd.Spec.Template.Spec,
		collector,
	)
	if err != nil {
		return nil, errors.Wrap(err, "applying polaris configs")
	}
	gd.Spec.Template.Spec = polarisResult.PodSpec
	extraObjs = append(extraObjs, polarisResult.ExtraObjects...)

	result := &BuildResult{
		ExtraObjects:          extraObjs,
		SensitiveEnvVarValues: sensitiveEnvVarValues,
		UndefinedEnvVars:      collector.UndefinedEnvVars(),
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

func buildSensitiveEnvVarValues(appEnvVars envvartypes.EnvVariableList) map[string]string {
	sensitiveValues := make(map[string]string)
	for _, item := range appEnvVars {
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
