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
	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// BuilderService holds the stable dependencies required to construct a Builder.
//
// It intentionally excludes app/appModel so the service itself can be provided by fx
// and reused for multiple build requests.
type BuilderService struct {
	envVarsReader          *envvars.UnifiedEnvVarsReader
	workspaceCompsStore    workspace.WorkspaceCompsStore
	polarisWorkloadBuilder *polaris.WorkloadBuilder
	bscpCfgStore           bscpcfg.Store
	appModelStore          appmodel.AppModelStore
	appSpecStore           appspec.AppSpecStore
	buildConfigStore       build.ConfigStore
}

// Builder helps to build workload resources for applications.
type Builder struct {
	*BuilderService

	// The application object
	app      *bkmsapp.Application
	appModel *appmodel.AppModel

	// The dev mode config
	devModeConfig *devmode.Config
}

// BuildResult 包含了一次工作负载构建产生的主工作负载、附加资源和脱敏所需元数据。
// 该结构只在内存中传递，不落库。
type BuildResult struct {
	// WorkloadKind 是主工作负载类型：GameDeployment 或 Deployment。
	WorkloadKind string
	// MainWorkload 是本次构建的主工作负载。
	MainWorkload runtime.Object
	// ExtraObjects 是所有除了主工作负载以外需要一并下发的资源对象。
	ExtraObjects []unstructured.Unstructured
	// SensitiveEnvVarValues 是构建过程中收集到的敏感环境变量值，供调用方后续做脱敏处理。
	SensitiveEnvVarValues map[string]string
	// UndefinedEnvVars 是渲染过程中引用但未定义的环境变量；仅报告，不阻断部署。
	UndefinedEnvVars []envvarrefs.UndefinedEnvVar
}

// workloadMeta 从 MainWorkload 抽出 GameDeployment / Deployment 的公共字段。
// 这两种对象没有统一接口，部署侧需要名称、selector、对象级和 PodTemplate 级 metadata；
// 用这一层避免调用方对 runtime.Object 做类型分支。
type workloadMeta struct {
	name         string
	selector     map[string]string
	objectMeta   *metav1.ObjectMeta
	templateMeta *metav1.ObjectMeta
}

// Name 返回主工作负载的名称。
func (r *BuildResult) Name() string {
	info, _ := r.mainInfo()
	return info.name
}

// GVR 返回主工作负载对应的 GroupVersionResource；未知类型时为空值。
func (r *BuildResult) GVR() schema.GroupVersionResource {
	if r == nil {
		return schema.GroupVersionResource{}
	}
	switch r.WorkloadKind {
	case kind.Deploy:
		return gvr.Deploy
	case kind.GameDeploy:
		return gvr.GameDeploy
	default:
		return schema.GroupVersionResource{}
	}
}

// Selector 返回主工作负载的 matchLabels。
func (r *BuildResult) Selector() map[string]string {
	info, _ := r.mainInfo()
	return info.selector
}

// ObjectMeta 返回主工作负载对象级 metadata，可原地修改。
func (r *BuildResult) ObjectMeta() *metav1.ObjectMeta {
	info, _ := r.mainInfo()
	return info.objectMeta
}

// TemplateMeta 返回主工作负载 PodTemplate 级 metadata，可原地修改。
func (r *BuildResult) TemplateMeta() *metav1.ObjectMeta {
	info, _ := r.mainInfo()
	return info.templateMeta
}

func (r *BuildResult) mainInfo() (workloadMeta, bool) {
	if r == nil || r.MainWorkload == nil {
		return workloadMeta{}, false
	}
	switch obj := r.MainWorkload.(type) {
	case *appsv1.Deployment:
		return workloadMeta{
			name:         obj.Name,
			selector:     labelSelectorMatchLabels(obj.Spec.Selector),
			objectMeta:   &obj.ObjectMeta,
			templateMeta: &obj.Spec.Template.ObjectMeta,
		}, true
	case *tkex.GameDeployment:
		return workloadMeta{
			name:         obj.Name,
			selector:     labelSelectorMatchLabels(obj.Spec.Selector),
			objectMeta:   &obj.ObjectMeta,
			templateMeta: &obj.Spec.Template.ObjectMeta,
		}, true
	default:
		return workloadMeta{}, false
	}
}

// NewBuilderService creates a BuilderService from the dependencies that can be injected by fx.
func NewBuilderService(
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	appDepsVarReader *depenvvars.Reader,
	polarisVarReader *polarisenvvars.Reader,
	workspaceCompsStore workspace.WorkspaceCompsStore,
	polarisConfigStore polaris.PolarisConfigStore,
	bscpCfgStore bscpcfg.Store,
	appModelStore appmodel.AppModelStore,
	appSpecStore appspec.AppSpecStore,
	buildConfigStore build.ConfigStore,
) *BuilderService {
	return &BuilderService{
		envVarsReader:          envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader),
		workspaceCompsStore:    workspaceCompsStore,
		polarisWorkloadBuilder: polaris.NewWorkloadBuilder(polarisConfigStore),
		bscpCfgStore:           bscpCfgStore,
		appModelStore:          appModelStore,
		appSpecStore:           appSpecStore,
		buildConfigStore:       buildConfigStore,
	}
}

// NewBuilder creates a Builder with request-scoped app/appModel inputs.
func NewBuilder(
	builderService *BuilderService,
	app *bkmsapp.Application,
	appModel *appmodel.AppModel,
) *Builder {
	return &Builder{
		BuilderService: builderService,
		app:            app,
		appModel:       appModel,
	}
}
