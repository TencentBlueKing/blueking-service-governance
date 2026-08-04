package workspace

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

var (
	lifecycleHooksMu sync.RWMutex

	// createHooksByName 用于按名称去重和查找 Hook；createHookNames 保存注册顺序，
	// 避免直接遍历 map 时因顺序不确定而改变创建 Hook 的执行顺序。
	createHooksByName = map[string]CreateHook{}
	createHookNames   []string

	// deleteHooksByName 用于按名称去重和查找 Hook；deleteHookNames 保存注册顺序，
	// 避免直接遍历 map 时因顺序不确定而改变删除 Hook 的执行顺序。
	deleteHooksByName = map[string]DeleteHook{}
	deleteHookNames   []string
)

// CreateHook runs after a workspace has been persisted.
type CreateHook func(ctx context.Context, workspace Workspace) error

// DeleteHook runs before a workspace is removed.
type DeleteHook func(ctx context.Context, workspaceID string) error

// RegisterCreateHook registers a named workspace create hook.
func RegisterCreateHook(name string, hook CreateHook) bool {
	lifecycleHooksMu.Lock()
	defer lifecycleHooksMu.Unlock()

	if _, exists := createHooksByName[name]; exists {
		return false
	}
	createHooksByName[name] = hook
	createHookNames = append(createHookNames, name)
	return true
}

// RegisterDeleteHook registers a named workspace delete hook.
func RegisterDeleteHook(name string, hook DeleteHook) bool {
	lifecycleHooksMu.Lock()
	defer lifecycleHooksMu.Unlock()

	if _, exists := deleteHooksByName[name]; exists {
		return false
	}
	deleteHooksByName[name] = hook
	deleteHookNames = append(deleteHookNames, name)
	return true
}

// IsCreateHookRegistered reports whether a named create hook is registered.
func IsCreateHookRegistered(name string) bool {
	lifecycleHooksMu.RLock()
	defer lifecycleHooksMu.RUnlock()

	_, exists := createHooksByName[name]
	return exists
}

// IsDeleteHookRegistered reports whether a named delete hook is registered.
func IsDeleteHookRegistered(name string) bool {
	lifecycleHooksMu.RLock()
	defer lifecycleHooksMu.RUnlock()

	_, exists := deleteHooksByName[name]
	return exists
}

// ResetLifecycleHooksForTest removes all registered workspace lifecycle hooks.
func ResetLifecycleHooksForTest() {
	lifecycleHooksMu.Lock()
	defer lifecycleHooksMu.Unlock()

	createHooksByName = map[string]CreateHook{}
	createHookNames = nil
	deleteHooksByName = map[string]DeleteHook{}
	deleteHookNames = nil
}

func runCreateHooks(ctx context.Context, workspace Workspace) error {
	lifecycleHooksMu.RLock()
	names := append([]string(nil), createHookNames...)
	hooks := make([]CreateHook, len(names))
	for index, name := range names {
		hooks[index] = createHooksByName[name]
	}
	lifecycleHooksMu.RUnlock()

	for index, name := range names {
		if err := hooks[index](ctx, workspace); err != nil {
			return errors.Wrapf(err, "run workspace create hook %s", name)
		}
	}
	return nil
}

func runDeleteHooks(ctx context.Context, workspaceID string) error {
	lifecycleHooksMu.RLock()
	names := append([]string(nil), deleteHookNames...)
	hooks := make([]DeleteHook, len(names))
	for index, name := range names {
		hooks[index] = deleteHooksByName[name]
	}
	lifecycleHooksMu.RUnlock()

	for index, name := range names {
		if err := hooks[index](ctx, workspaceID); err != nil {
			return errors.Wrapf(err, "run workspace delete hook %s", name)
		}
	}
	return nil
}
