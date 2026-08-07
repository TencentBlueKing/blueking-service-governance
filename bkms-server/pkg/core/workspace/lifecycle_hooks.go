package workspace

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

var (
	lifecycleHooksMu sync.RWMutex

	// preDeleteHooksByName 用于按名称去重和查找 Hook；preDeleteHookNames 保存注册顺序，
	// 避免直接遍历 map 时因顺序不确定而改变删除 Hook 的执行顺序。
	preDeleteHooksByName = map[string]PreDeleteHook{}
	preDeleteHookNames   []string
)

// PreDeleteHook runs before a workspace is removed.
type PreDeleteHook func(ctx context.Context, workspaceID string) error

// RegisterPreDeleteHook registers a named workspace pre-delete hook.
func RegisterPreDeleteHook(name string, hook PreDeleteHook) bool {
	lifecycleHooksMu.Lock()
	defer lifecycleHooksMu.Unlock()

	if _, exists := preDeleteHooksByName[name]; exists {
		return false
	}
	preDeleteHooksByName[name] = hook
	preDeleteHookNames = append(preDeleteHookNames, name)
	return true
}

// IsPreDeleteHookRegistered reports whether a named pre-delete hook is registered.
func IsPreDeleteHookRegistered(name string) bool {
	lifecycleHooksMu.RLock()
	defer lifecycleHooksMu.RUnlock()

	_, exists := preDeleteHooksByName[name]
	return exists
}

// ResetLifecycleHooksForTest removes all registered workspace lifecycle hooks.
func ResetLifecycleHooksForTest() {
	lifecycleHooksMu.Lock()
	defer lifecycleHooksMu.Unlock()

	preDeleteHooksByName = map[string]PreDeleteHook{}
	preDeleteHookNames = nil
}

func runPreDeleteHooks(ctx context.Context, workspaceID string) error {
	lifecycleHooksMu.RLock()
	names := append([]string(nil), preDeleteHookNames...)
	hooks := make([]PreDeleteHook, len(names))
	for index, name := range names {
		hooks[index] = preDeleteHooksByName[name]
	}
	lifecycleHooksMu.RUnlock()

	for index, name := range names {
		if err := hooks[index](ctx, workspaceID); err != nil {
			return errors.Wrapf(err, "run workspace pre-delete hook %s", name)
		}
	}
	return nil
}
