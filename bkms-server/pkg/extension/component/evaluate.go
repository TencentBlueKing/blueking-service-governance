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

// Package component evaluate.go takes care of evaluating components.
package component

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/yaml.v3"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// EvaluatedComponent is a rendered and parsed component definition.
type EvaluatedComponent struct {
	// Patchers are ordered patches applied to the GameDeployment.
	Patchers []map[string]any
	// Specs are additional Kubernetes resources created alongside the GameDeployment.
	Specs []map[string]any
}

// AppComponentApplier helps applying a component instance to an application.
type AppComponentApplier struct {
	builder      *AppPropertiesBuilder
	compDefStore ComponentDefStore
}

// CreateDefaultApplier creates a default AppComponentApplier instance which uses the
// projects's default db connection.
//
// TODO: Maybe we should introduce a DI framework to manage these dependencies to avoid
// the boilerplate code.
func CreateDefaultApplier() (*AppComponentApplier, error) {
	mongoClient, dbName := database.Client(), database.Name()
	compDefStore, err := NewComponentDefStoreMongo(mongoClient, dbName)
	if err != nil {
		return nil, errors.Wrap(err, "creating component-def store")
	}
	envStore, err := envmodel.NewEnvironmentStoreMongo(mongoClient, dbName)
	if err != nil {
		return nil, errors.Wrap(err, "creating environment store")
	}
	envSvc := env.NewEnvService(envStore)
	return NewAppComponentApplier(compDefStore, envSvc), nil
}

// NewAppComponentApplier creates a new AppComponentApplier instance.
func NewAppComponentApplier(
	compDefStore ComponentDefStore,
	envService *env.EnvService,
) *AppComponentApplier {
	return &AppComponentApplier{
		builder:      NewAppPropertiesBuilder(compDefStore, envService),
		compDefStore: compDefStore,
	}
}

// Evaluate renders and parses the patchers and specs of a component definition.
//
// vars 为渲染变量 map（通常由 envvars.BuildAppEnvVars().ToMap() 构建），可为 nil。
// collector 用于收集未定义环境变量引用，可为 nil。
func (a *AppComponentApplier) Evaluate(
	ctx context.Context,
	app *bkmsapp.Application,
	comp Component,
	envID bson.ObjectID,
	vars map[string]string,
	collector *envvarrefs.Collector,
) (*EvaluatedComponent, error) {
	// Prepare the properties
	props, err := a.builder.Build(ctx, app, comp, envID, vars, collector)
	if err != nil {
		return nil, errors.Wrap(err, "building properties")
	}

	compDef, err := a.compDefStore.Get(ctx, comp.Type, comp.Version)
	if err != nil {
		return nil, errors.Wrap(err, "getting component definition")
	}
	if err = ValidateComponentDef(compDef); err != nil {
		return nil, errors.Wrap(err, "validating component definition")
	}
	return evaluateComponentTemplates(compDef.Patchers, compDef.Specs, props)
}

func evaluateComponentTemplates(
	patchers, specs []string,
	props map[string]any,
) (*EvaluatedComponent, error) {
	result := &EvaluatedComponent{
		Patchers: make([]map[string]any, 0, len(patchers)),
		Specs:    make([]map[string]any, 0, len(specs)),
	}
	for i, patcher := range patchers {
		parsed, err := renderMappingTemplate(patcher, props)
		if err != nil {
			return nil, errors.Wrapf(err, "evaluating patcher[%d]", i)
		}
		result.Patchers = append(result.Patchers, parsed)
	}
	for i, spec := range specs {
		parsed, err := renderMappingTemplate(spec, props)
		if err != nil {
			return nil, errors.Wrapf(err, "evaluating spec[%d]", i)
		}
		result.Specs = append(result.Specs, parsed)
	}
	return result, nil
}

func renderMappingTemplate(fragment string, props map[string]any) (map[string]any, error) {
	rendered, err := render.RenderGoTemplate(fragment, props)
	if err != nil {
		return nil, errors.Wrap(err, "rendering template")
	}
	var result map[string]any
	if err = yaml.Unmarshal([]byte(rendered), &result); err != nil {
		return nil, errors.Wrap(err, "parsing rendered YAML mapping")
	}
	if result == nil {
		return nil, errors.New("rendered YAML is not a mapping")
	}
	return result, nil
}

// AppPropertiesBuilder helps with building the "properties" for an application when
// instantiating components.
type AppPropertiesBuilder struct {
	compDefStore ComponentDefStore
	envService   *env.EnvService
}

// NewAppPropertiesBuilder creates a new AppPropertiesBuilder instance.
func NewAppPropertiesBuilder(
	compDefStore ComponentDefStore,
	envService *env.EnvService,
) *AppPropertiesBuilder {
	return &AppPropertiesBuilder{
		compDefStore: compDefStore,
		envService:   envService,
	}
}

// Build builds the properties, the value of the property are evaluated values.
// Placeholders like "{{ .BKMS.ENV.VAR_NAME }}" (legacy) and "${{KEY}}" (new) are evaluated
// to the actual values.
//
// Args:
// - comp: the component instance object
// - envID: the environment ID where the component is being instantiated
// - vars: render variables map (typically from envvars.BuildAppEnvVars().ToMap()), can be nil
// - collector: collects undefined env-var references, can be nil
//
// Return:
// - A map of property name to its evaluated value, can be used for template rendering.
func (b *AppPropertiesBuilder) Build(
	ctx context.Context,
	app *bkmsapp.Application,
	comp Component,
	envID bson.ObjectID,
	vars map[string]string,
	collector *envvarrefs.Collector,
) (map[string]any, error) {
	props, err := b.BuildRaw(ctx, app, comp, envID)
	if err != nil {
		return nil, err
	}

	evaResult, err := renderProps(props, vars, collector, comp.Name)
	if err != nil {
		return nil, errors.Wrap(err, "rendering properties")
	}

	// Rich types: some property type such "MAP" should be unmarshaled from string to map
	return lo.MapEntries(evaResult, func(key string, value PropValue) (string, any) {
		return key, value.ToRichValue()
	}), nil
}

// BuildRaw builds the properties, the value of the property are raw values.
//
// Args:
// - comp: the component instance object
// - envID: the environment ID where the component is being instantiated
func (b *AppPropertiesBuilder) BuildRaw(
	ctx context.Context,
	app *bkmsapp.Application,
	comp Component,
	envID bson.ObjectID,
) (map[string]PropValue, error) {
	compDef, err := b.compDefStore.Get(ctx, comp.Type, comp.Version)
	if err != nil {
		return nil, errors.Wrap(err, "getting component definition")
	}
	result := make(map[string]PropValue)
	// Read the default value defined in the component definition
	for _, propDef := range compDef.Properties {
		result[propDef.Name] = PropValue{
			Ty:    propDef.Type,
			Value: propDef.NormalizedDefaultValue(),
		}
	}
	// Override with the value provided in the component instance
	for propName, propValue := range comp.Properties {
		if _, exists := result[propName]; !exists {
			// Should ignore unknown property which is not defined in the component definition
			continue
		}
		result[propName] = PropValue{
			Ty:    result[propName].Ty,
			Value: propValue,
		}
	}
	// Add the basic built-in properties
	env, err := b.envService.Get(ctx, envID)
	if err != nil {
		return nil, errors.Wrap(err, "getting environment")
	}
	// Merge the result
	for key, value := range buildBasicBuiltin(*app, *env) {
		result[key] = PropValue{
			Ty:    PropTypeString,
			Value: value,
		}
	}

	// name 在部分组件中作为生成 cr 的名称， 需要满足 k8s 的命名规范
	// 部分存量数据的 name 包含大写字符，这里先做小写处理
	result[PropNameName] = PropValue{
		Ty:    PropTypeString,
		Value: strings.ToLower(fmt.Sprintf("%s-%s", app.Name, comp.Name)),
	}
	return result, nil
}

// renderProps render the propValues by evaluating any template placeholders in values.
// It also collects undefined env-var references from rendered string properties.
func renderProps(
	props map[string]PropValue,
	vars map[string]string,
	collector *envvarrefs.Collector,
	sourceName string,
) (map[string]PropValue, error) {
	evaResult := make(map[string]PropValue)
	renderer := render.New(render.SetEnvContext(vars))
	for key, prop := range props {
		strVal, ok := prop.Value.(string)
		if !ok {
			evaResult[key] = prop
			continue
		}

		if !strings.Contains(strVal, "${{") {
			evaResult[key] = prop
			continue
		}

		if err := collector.Collect(strVal, envvarrefs.Source{
			Type: envvarrefs.SourceComponent,
			Name: sourceName,
		}); err != nil {
			return nil, errors.Wrapf(err, "collecting env vars from property %s", key)
		}
		rendered, err := renderer.Render(strVal)
		if err != nil {
			return nil, errors.Wrapf(err, "rendering property %s", key)
		}
		evaResult[key] = PropValue{
			Ty:    prop.Ty,
			Value: rendered,
		}
	}
	return evaResult, nil
}

// PropValue is a simple container type for property
type PropValue struct {
	// ty is the type of the property
	Ty PropType
	// value is the value of the property
	Value any
}

// ToRichValue converts the Value to its rich value representation.
func (v PropValue) ToRichValue() any {
	if v.Ty != PropTypeMap {
		return v.Value
	}

	empty := make(map[string]any)
	if v.Value == nil {
		return empty
	}
	// Return if the value is already a map
	if _, ok := v.Value.(map[string]any); ok {
		return v.Value
	}
	// Return an empty value if the value is not a string
	strVal, ok := v.Value.(string)
	if !ok {
		return empty
	}
	// Try to unmarshal the string to a map, return an empty value even it fails
	var result map[string]any
	err := json.Unmarshal([]byte(strVal), &result)
	if err != nil {
		return empty
	}
	return result
}

// buildBasicBuiltin returns the basic built-in properties for a component.
func buildBasicBuiltin(app bkmsapp.Application, env envmodel.Environment) map[string]string {
	result := make(map[string]string)

	// Part: app
	result[PropNameAppName] = app.Name
	result[PropNameContainerName] = defaults.WorkloadMainContainerName
	result[PropNameEnvName] = env.Name
	// Part env
	if env.Cluster.ClusterID != "" {
		result[PropNameEnvNS] = env.Cluster.Namespace
		result[PropNameEnvCluster] = env.Cluster.ClusterID
	}
	return result
}
