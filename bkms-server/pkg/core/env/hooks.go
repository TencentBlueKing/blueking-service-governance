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

package env

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var (
	deleteHooksMu     sync.RWMutex
	deleteHooksByName = map[string]DeleteHook{}
	deleteHookNames   []string
)

// DeleteHook 在环境被物理删除前执行，用于清理依赖环境存在的下游资源。
//
// Hook 会拿到即将删除的 Environment 完整对象，因此实现方不需要再次查询环境，也不要只依赖 envID。
// Delete 会按注册顺序串行执行所有 Hook；任意 Hook 返回错误时，删除流程会立即停止并保留环境数据，
// 方便调用方修复清理失败的问题后重试删除。Hook 实现需要保持幂等，因为同一个删除请求可能被重试。
type DeleteHook func(ctx context.Context, environment model.Environment) error

// RegisterDeleteHook 注册环境删除 Hook。
//
// 注册发生在包级全局注册表中，通常由进程 registry 在依赖初始化完成后调用，例如：
//
//	env.RegisterDeleteHook("envvars", cleanupEnvVars)
//
// name 用来避免重复注册。相同 name 已存在时，本函数不会覆盖旧 Hook，并返回 false；首次注册成功返回 true。
func RegisterDeleteHook(name string, hook DeleteHook) bool {
	deleteHooksMu.Lock()
	defer deleteHooksMu.Unlock()

	if _, ok := deleteHooksByName[name]; ok {
		return false
	}
	deleteHooksByName[name] = hook
	deleteHookNames = append(deleteHookNames, name)
	return true
}

// IsDeleteHookRegistered 返回指定名称的删除 Hook 是否已注册。
//
// 该方法主要用于测试，以防御性方式确认关键 Hook 已被正确注册。
func IsDeleteHookRegistered(name string) bool {
	deleteHooksMu.RLock()
	defer deleteHooksMu.RUnlock()

	_, ok := deleteHooksByName[name]
	return ok
}

// ResetDeleteHooksForTest removes registered delete hooks.
//
// It is intended for tests that reset process-wide registries and then rebuild
// hook closures with fresh store dependencies.
func ResetDeleteHooksForTest() {
	deleteHooksMu.Lock()
	defer deleteHooksMu.Unlock()

	deleteHooksByName = map[string]DeleteHook{}
	deleteHookNames = nil
}

// runDeleteHooks 按注册顺序运行所有环境删除 Hook。
//
// 注意：Hook 执行成功后不可回滚（如已清理的作用域变量无法恢复），
// 若后续环境记录删除失败，调用方需感知此风险并决定是否重试。
func runDeleteHooks(ctx context.Context, environment model.Environment) error {
	deleteHooksMu.RLock()
	names := make([]string, len(deleteHookNames))
	copy(names, deleteHookNames)
	hooks := make([]DeleteHook, len(names))
	for i, name := range names {
		hooks[i] = deleteHooksByName[name]
	}
	deleteHooksMu.RUnlock()

	for i, name := range names {
		if err := hooks[i](ctx, environment); err != nil {
			return errors.Wrapf(err, "run environment delete hook %s", name)
		}
	}
	return nil
}
