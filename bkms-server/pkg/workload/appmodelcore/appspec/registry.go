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

package appspec

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sectiondriver"
	annotationssection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/annotations"
	devmodesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/devmode"
	labelssection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/labels"
	lifecyclesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/lifecycle"
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
	resourcessection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/resources"
	tkerouteenisection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/tke_route_eni"
	updatestrategysection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/updatestrategy"
)

// AppSpecSectionID is the stable identifier for a top-level AppSpec section.
type AppSpecSectionID string

const (
	// AppSpecSectionResources 资源规格
	AppSpecSectionResources AppSpecSectionID = "resources"
	// AppSpecSectionUpdateStrategy 更新策略
	AppSpecSectionUpdateStrategy AppSpecSectionID = "updateStrategy"
	// AppSpecSectionDevMode 开发模式
	AppSpecSectionDevMode AppSpecSectionID = "devMode"
	// AppSpecSectionLifecycle 生命周期钩子
	AppSpecSectionLifecycle AppSpecSectionID = "lifecycle"
	// AppSpecSectionProbe 容器探针
	AppSpecSectionProbe AppSpecSectionID = "probes"
	// AppSpecSectionLabels 自定义标签
	AppSpecSectionLabels AppSpecSectionID = "labels"
	// AppSpecSectionAnnotations 自定义注解
	AppSpecSectionAnnotations AppSpecSectionID = "annotations"
	// AppSpecSectionTkeRouteEni TKE Route ENI (VPC-CNI) 网络模式
	AppSpecSectionTkeRouteEni AppSpecSectionID = "tkeRouteEni"
)

// SectionWriteMode controls how a section write resolves input against current state.
type SectionWriteMode string

const (
	SectionWriteModePatch   SectionWriteMode = "patch"
	SectionWriteModeReplace SectionWriteMode = "replace"
)

// registeredSection 代表一类应用配置块（配置域），比如资源配置、更新策略等。
// 每类配置块独立管理自己的数据存取逻辑，相互不影响。
//
// 这是 runtime 层的统一桥接接口，供 merge/clone/store/validator/appmodel 这些通用流程遍历调用。
type registeredSection interface {
	id() AppSpecSectionID

	// mergeTo 负责合并两份数据并设置到给定的 AppSpec 中。
	mergeTo(base, override, merged *AppSpec)
	// cloneTo 负责克隆一份新数据并设置到给定的 AppSpec 中。
	cloneTo(src, dst *AppSpec)

	// appendWholeUpdate 定义了 section 在整块替换/删除场景下如何生成 MongoDB 更新语句。
	appendWholeUpdate(set, unset *bson.D, spec *AppSpec)
	// appendPatch 定义了如何将配置块数据转换成 MongoDB 更新语句的一部分，此函数定义了部分更新（Patch）的语义。
	appendPatch(set *bson.D, spec *AppSpec)
	// registerValidation 负责注册配置块数据的验证规则。
	registerValidation(v *validator.Validate)

	// fill 和 apply 定义了数据如何在 AppModel 和 AppSpec 之间转换，并非所有配置块都支持这个转换。
	fillFromAppModel(spec *AppSpec, appModel *appmodel.AppModel)
	applyToAppModel(spec *AppSpec, appModel *appmodel.AppModel)
}

// SectionHandle 是 section 的一个具体绑定，负责将某个 section driver 挂载到 AppSpec 的具体字段上。
// 它本身不定义领域语义，只负责字段定位和调用 driver 的通用桥接。
type SectionHandle[T any] struct {
	// sectionID 是 section 的唯一标识。
	sectionID AppSpecSectionID

	// getRaw 明确了如何从 AppSpec 中获取对应 section 字段的原始值。
	getRaw func(*AppSpec) *T

	// setRaw 明确了如何在 AppSpec 中设置对应 section 字段的值。
	setRaw func(*AppSpec, *T)

	// driver 定义了这个 section 的领域操作逻辑。
	driver sectiondriver.Driver[T]
}

// registeredSection interface implementation
//
// 这一组方法是 SectionHandle 作为 runtime registry 条目时对外暴露的统一行为，
// 供 merge/clone/store/validator/appmodel 这些通用流程遍历调用。
func (h SectionHandle[T]) id() AppSpecSectionID {
	return h.sectionID
}

func (h SectionHandle[T]) mergeTo(base, override, merged *AppSpec) {
	h.setRaw(merged, h.driver.Merge(h.getRaw(base), h.getRaw(override)))
}

func (h SectionHandle[T]) cloneTo(src, dst *AppSpec) {
	h.setCloned(dst, h.getRaw(src))
}

func (h SectionHandle[T]) appendWholeUpdate(set, unset *bson.D, spec *AppSpec) {
	value := h.getCloned(spec)
	if value == nil {
		*unset = append(*unset, bson.E{Key: h.sectionID.String(), Value: ""})
		return
	}
	*set = append(*set, bson.E{Key: h.sectionID.String(), Value: value})
}

func (h SectionHandle[T]) appendPatch(set *bson.D, spec *AppSpec) {
	h.driver.AppendPatch(set, h.getRaw(spec))
}

func (h SectionHandle[T]) registerValidation(v *validator.Validate) {
	h.driver.RegisterValidation(v)
}

func (h SectionHandle[T]) fillFromAppModel(spec *AppSpec, appModel *appmodel.AppModel) {
	if h.driver.FromAppModel == nil {
		return
	}
	h.setCloned(spec, h.driver.FromAppModel(appModel))
}

func (h SectionHandle[T]) applyToAppModel(spec *AppSpec, appModel *appmodel.AppModel) {
	if h.driver.ApplyToAppModel == nil {
		return
	}
	h.driver.ApplyToAppModel(h.getRaw(spec), appModel)
}

// local helper methods
//
// 这一组方法只在 SectionHandle 内部复用，不属于 registeredSection 的对外语义。
func (h SectionHandle[T]) getCloned(spec *AppSpec) *T {
	return h.driver.Clone(h.getRaw(spec))
}

func (h SectionHandle[T]) setCloned(spec *AppSpec, value *T) {
	h.setRaw(spec, h.driver.Clone(value))
}

// ResourcesSection is the typed handle for the resources block.
var (
	ResourcesSection = SectionHandle[ResourcesSpec]{
		sectionID: AppSpecSectionResources,
		getRaw: func(spec *AppSpec) *ResourcesSpec {
			return spec.Resources
		},
		setRaw: func(spec *AppSpec, value *ResourcesSpec) {
			spec.Resources = value
		},
		driver: resourcessection.Driver,
	}
	// UpdateStrategySection is the typed handle for the update strategy block.
	UpdateStrategySection = SectionHandle[UpdateStrategySpec]{
		sectionID: AppSpecSectionUpdateStrategy,
		getRaw: func(spec *AppSpec) *UpdateStrategySpec {
			return spec.UpdateStrategy
		},
		setRaw: func(spec *AppSpec, value *UpdateStrategySpec) {
			spec.UpdateStrategy = value
		},
		driver: updatestrategysection.Driver,
	}
	// DevModeSection is the typed handle for the dev mode block.
	DevModeSection = SectionHandle[DevModeSpec]{
		sectionID: AppSpecSectionDevMode,
		getRaw: func(spec *AppSpec) *DevModeSpec {
			return spec.DevMode
		},
		setRaw: func(spec *AppSpec, value *DevModeSpec) {
			spec.DevMode = value
		},
		driver: devmodesection.Driver,
	}
	// LifecycleSection is the typed handle for the lifecycle block.
	LifecycleSection = SectionHandle[LifecycleSpec]{
		sectionID: AppSpecSectionLifecycle,
		getRaw: func(spec *AppSpec) *LifecycleSpec {
			return spec.Lifecycle
		},
		setRaw: func(spec *AppSpec, value *LifecycleSpec) {
			spec.Lifecycle = value
		},
		driver: lifecyclesection.Driver,
	}
	// ProbeSection is the typed handle for the probe block.
	ProbeSection = SectionHandle[ProbeSpec]{
		sectionID: AppSpecSectionProbe,
		getRaw: func(spec *AppSpec) *ProbeSpec {
			return spec.Probes
		},
		setRaw: func(spec *AppSpec, value *ProbeSpec) {
			spec.Probes = value
		},
		driver: probesection.Driver,
	}
	// LabelsSection is the typed handle for the labels block.
	LabelsSection = SectionHandle[LabelsSpec]{
		sectionID: AppSpecSectionLabels,
		getRaw: func(spec *AppSpec) *LabelsSpec {
			return spec.Labels
		},
		setRaw: func(spec *AppSpec, value *LabelsSpec) {
			spec.Labels = value
		},
		driver: labelssection.Driver,
	}
	// AnnotationsSection is the typed handle for the annotations block.
	AnnotationsSection = SectionHandle[AnnotationsSpec]{
		sectionID: AppSpecSectionAnnotations,
		getRaw: func(spec *AppSpec) *AnnotationsSpec {
			return spec.Annotations
		},
		setRaw: func(spec *AppSpec, value *AnnotationsSpec) {
			spec.Annotations = value
		},
		driver: annotationssection.Driver,
	}
	// TkeRouteEniSection is the typed handle for the tkeRouteEni block.
	TkeRouteEniSection = SectionHandle[TkeRouteEniSpec]{
		sectionID: AppSpecSectionTkeRouteEni,
		getRaw: func(spec *AppSpec) *TkeRouteEniSpec {
			return spec.TkeRouteEni
		},
		setRaw: func(spec *AppSpec, value *TkeRouteEniSpec) {
			spec.TkeRouteEni = value
		},
		driver: tkerouteenisection.Driver,
	}
)

// registeredSections 定义目前项目所支持的所有 sections。
// 新增 section 时，除了补充 AppSpec 字段和 section driver，也要把对应绑定加到这里。
var registeredSections = []registeredSection{
	ResourcesSection,
	UpdateStrategySection,
	DevModeSection,
	LifecycleSection,
	ProbeSection,
	LabelsSection,
	AnnotationsSection,
	TkeRouteEniSection,
}

// sectionsByID 是一个辅助变量，用于根据 section ID 快速查找对应的 section 实现。
var sectionsByID = func() map[AppSpecSectionID]registeredSection {
	ret := make(map[AppSpecSectionID]registeredSection, len(registeredSections))
	for _, section := range registeredSections {
		ret[section.id()] = section
	}
	return ret
}()

// getSection 根据 section ID 获取对应的 section 实现。
func getSection(id AppSpecSectionID) (registeredSection, bool) {
	section, ok := sectionsByID[id]
	return section, ok
}

func (id AppSpecSectionID) String() string {
	return string(id)
}
