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

package envvars

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Reader 列出某个应用在某个环境下由依赖服务绑定产出的环境变量。
type Reader struct {
	instStore    model.ServiceInstanceStore
	bindingStore model.ServiceBindingStore
}

// NewReader 创建一个依赖服务环境变量读取器。
func NewReader(instStore model.ServiceInstanceStore, bindingStore model.ServiceBindingStore) *Reader {
	return &Reader{instStore: instStore, bindingStore: bindingStore}
}

// ListEnvVarsForApp 返回该应用在指定环境下生效的依赖服务环境变量。
// 只渲当前环境映射的实例，其它环境的同名 key 不会出现。
// 同一环境多份绑定声明了相同 key 时全部保留、此处不去重，注入时由上游按后写覆盖。
// TODO: 确认如何覆盖「多绑定同 key」的冲突检查。
func (r *Reader) ListEnvVarsForApp(
	ctx context.Context,
	environment envmodel.Environment,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}
	return r.collectFromBindings(ctx, app.ID, environment.Name)
}

// ListAppVarsForConflicts 返回应用下全部绑定声明的环境变量 key，用于应用级冲突检测。
// 每份绑定的每个 key 只出一条；value 为各已映射环境中能渲出的去重结果。
func (r *Reader) ListAppVarsForConflicts(
	ctx context.Context,
	_ string,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}
	return r.collectBindingKeysForConflicts(ctx, app.ID)
}

func (r *Reader) listAppBindings(ctx context.Context, appID string) ([]*model.ServiceBinding, error) {
	if r.bindingStore == nil {
		return nil, nil
	}
	bindings, err := r.bindingStore.List(ctx, &model.BindingQueryOptions{AppID: appID})
	if err != nil {
		return nil, errors.Wrap(err, "list service bindings for app")
	}
	return bindings, nil
}

func (r *Reader) instancesByID(
	ctx context.Context,
	ids []bson.ObjectID,
) (map[bson.ObjectID]*model.ServiceInstance, error) {
	if r.instStore == nil || len(ids) == 0 {
		return map[bson.ObjectID]*model.ServiceInstance{}, nil
	}
	insts, err := r.instStore.ListByIDs(ctx, ids)
	if err != nil {
		return nil, errors.Wrap(err, "list service instances by ids")
	}
	return lo.SliceToMap(insts, func(inst *model.ServiceInstance) (bson.ObjectID, *model.ServiceInstance) {
		return inst.ID, inst
	}), nil
}

// collectFromBindings 按指定环境从绑定产出实际注入的变量。
func (r *Reader) collectFromBindings(
	ctx context.Context,
	appID, envName string,
) (envvartypes.EnvVariableRichList, error) {
	result := envvartypes.EnvVariableRichList{Vars: make([]envvartypes.EnvVariableRichItem, 0)}

	bindings, err := r.listAppBindings(ctx, appID)
	if err != nil {
		return result, err
	}

	instIDs := lo.FilterMap(bindings, func(b *model.ServiceBinding, _ int) (bson.ObjectID, bool) {
		instID, ok := b.EnvInstanceMap[envName]
		return instID, ok && !instID.IsZero()
	})
	insts, err := r.instancesByID(ctx, instIDs)
	if err != nil {
		return result, err
	}

	for _, binding := range bindings {
		instID, ok := binding.EnvInstanceMap[envName]
		if !ok || instID.IsZero() {
			continue
		}
		inst, ok := insts[instID]
		if !ok {
			log.Warnf(ctx, "instance %s for binding %s not found", instID.Hex(), binding.Name)
			continue
		}
		if inst.Status != model.AvailableStatus {
			continue
		}

		vars, err := BuildBindingEnvVars(ctx, binding.EnvVars, inst.Credentials)
		if err != nil {
			log.Warnf(ctx, "build binding env vars for %s: %v", binding.Name, err)
			continue
		}
		result.Vars = append(result.Vars, wrapBindingVars(binding.Name, vars)...)
	}
	return result, nil
}

// collectBindingKeysForConflicts 按绑定聚合 key，不按环境拆成多条源。
func (r *Reader) collectBindingKeysForConflicts(
	ctx context.Context,
	appID string,
) (envvartypes.EnvVariableRichList, error) {
	result := envvartypes.EnvVariableRichList{Vars: make([]envvartypes.EnvVariableRichItem, 0)}

	bindings, err := r.listAppBindings(ctx, appID)
	if err != nil {
		return result, err
	}

	instIDs := lo.Filter(lo.FlatMap(bindings, func(b *model.ServiceBinding, _ int) []bson.ObjectID {
		return lo.Values(b.EnvInstanceMap)
	}), func(id bson.ObjectID, _ int) bool {
		return !id.IsZero()
	})
	insts, err := r.instancesByID(ctx, instIDs)
	if err != nil {
		return result, err
	}

	for _, binding := range bindings {
		if len(binding.EnvVars) == 0 {
			continue
		}
		valuesByKey := r.renderBindingValuesAcrossInstances(ctx, binding, insts)
		keys := lo.Keys(binding.EnvVars)
		sort.Strings(keys)
		source := envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourceAppDeps,
			SourceValue: binding.Name,
		}
		for _, key := range keys {
			vals := lo.Uniq(lo.Filter(valuesByKey[key], func(v string, _ int) bool {
				return v != ""
			}))
			sort.Strings(vals)
			result.Vars = append(result.Vars, envvartypes.EnvVariableRichItem{
				Obj: envvartypes.EnvVariableObj{
					Key:   key,
					Value: strings.Join(vals, ", "),
				},
				Source: source,
			})
		}
	}
	return result, nil
}

// renderBindingValuesAcrossInstances 用每台已映射且可用的实例各渲一次 EnvVars。
// 同一实例只渲一次；不可用或渲染失败则跳过。返回 key → 各次渲出的值（此处不去重）。
func (r *Reader) renderBindingValuesAcrossInstances(
	ctx context.Context,
	binding *model.ServiceBinding,
	insts map[bson.ObjectID]*model.ServiceInstance,
) map[string][]string {
	valuesByKey := make(map[string][]string, len(binding.EnvVars))
	seenInst := make(map[bson.ObjectID]struct{}, len(binding.EnvInstanceMap))

	for _, instID := range binding.EnvInstanceMap {
		if instID.IsZero() {
			continue
		}
		if _, seen := seenInst[instID]; seen {
			continue
		}
		seenInst[instID] = struct{}{}

		inst, ok := insts[instID]
		if !ok {
			log.Warnf(ctx, "instance %s for binding %s not found", instID.Hex(), binding.Name)
			continue
		}
		if inst.Status != model.AvailableStatus {
			continue
		}

		vars, err := BuildBindingEnvVars(ctx, binding.EnvVars, inst.Credentials)
		if err != nil {
			log.Warnf(ctx, "build binding env vars for %s: %v", binding.Name, err)
			continue
		}
		for _, v := range vars {
			valuesByKey[v.Key] = append(valuesByKey[v.Key], v.Value)
		}
	}
	return valuesByKey
}

func wrapBindingVars(bindingName string, vars envvartypes.EnvVariableList) []envvartypes.EnvVariableRichItem {
	source := envvartypes.ConflictedSource{
		Source:      envvartypes.EnvVarSourceAppDeps,
		SourceValue: bindingName,
	}
	items := make([]envvartypes.EnvVariableRichItem, 0, len(vars))
	for _, v := range vars {
		items = append(items, envvartypes.EnvVariableRichItem{Obj: v, Source: source})
	}
	return items
}
