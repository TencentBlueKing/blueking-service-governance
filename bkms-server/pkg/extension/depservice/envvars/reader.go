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

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Reader 基于 ServiceInstance 列出某个应用在某个环境下生效的环境变量。
//
// 它产出的富列表 (envvartypes.EnvVariableRichList) 会被注入到
// envvars.UnifiedEnvVarsReader 中以接入统一的 envvars 链路。
type Reader struct {
	instStore model.ServiceInstanceStore
}

// NewReader 创建一个依赖服务实例的环境变量读取器。
func NewReader(instStore model.ServiceInstanceStore) *Reader {
	return &Reader{instStore: instStore}
}

// ListEnvVarsForApp 返回 attach 到给定 app 且对该 environment 可见的全部实例所产出的环境变量。
//
// 过滤规则 (全部由 store 层执行):
//   - inst.WorkspaceID == environment.WorkspaceID
//   - 通过 store List 的 EnvName/EnvType 过滤命中三种 scope 之一
//   - 通过 store List 的 AttachedAppID 过滤已 attach 给该 app 的实例
//   - inst.Status == model.AvailableStatus
//
// 实例之间按 store 返回顺序扁平拼接, 同一实例内部, 先内置变量后衍生变量。
// 单个实例处理失败仅记日志后跳过, 不影响其它实例。
func (r *Reader) ListEnvVarsForApp(
	ctx context.Context,
	environment envmodel.Environment,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}

	insts, err := r.instStore.List(ctx, &model.SvcInstQueryOptions{
		WorkspaceID:   environment.WorkspaceID,
		AttachedAppID: app.ID,
		EnvName:       environment.Name,
		EnvType:       environment.Type,
		Status:        model.AvailableStatus,
	})
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list service instances for app")
	}

	return r.collect(ctx, insts)
}

// ListAppVarsForConflicts 返回 workspace 内 attach 到该 app 的全部实例
// (不限定具体 environment) 所产出的环境变量, 用于 app 级别冲突检测。
func (r *Reader) ListAppVarsForConflicts(
	ctx context.Context,
	workspaceID string,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}

	insts, err := r.instStore.List(ctx, &model.SvcInstQueryOptions{
		WorkspaceID:   workspaceID,
		AttachedAppID: app.ID,
		Status:        model.AvailableStatus,
	})
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list service instances for app in workspace")
	}

	return r.collect(ctx, insts)
}

// collect 公共流程: 对 insts 中的实例, 调用 BuildInstanceEnvVars,
// 包装 source 信息后扁平拼接。
// AttachedApps 和 Status 过滤已由 store 层完成, 此处不再重复判断。
func (r *Reader) collect(
	ctx context.Context,
	insts []*model.ServiceInstance,
) (envvartypes.EnvVariableRichList, error) {
	result := envvartypes.EnvVariableRichList{Vars: make([]envvartypes.EnvVariableRichItem, 0)}

	for _, inst := range insts {
		vars, err := BuildInstanceEnvVars(ctx, inst)
		if err != nil {
			// 单实例失败不阻塞整体
			log.Warnf(ctx, "build instance env vars for instance %s: %v", inst.Name, err)
			continue
		}

		source := envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourceAppDeps,
			SourceValue: inst.Name,
		}
		for _, v := range vars {
			result.Vars = append(result.Vars, envvartypes.EnvVariableRichItem{
				Obj:    v,
				Source: source,
			})
		}
	}

	return result, nil
}
