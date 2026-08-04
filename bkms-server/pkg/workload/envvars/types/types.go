// Package types 承载 envvars 链路中跨包共享的基础类型（环境变量对象、富列表、
// 来源与冲突信息等）。
//
// 它被抽离为独立子包，是为了让产出环境变量的实现方（如 pkg/depservice/envvars）
// 与上游 pkg/envvars 主包都依赖同一套稳定类型，从而避免两者之间形成循环依赖。
package types

import (
	"sort"

	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
)

// SensitiveValueMask is the masked value used when showing sensitive env vars to users.
const SensitiveValueMask = "******"

// EnvVariableObj represents an environment variable object, it's majorly used for holding
// the built-in environment variables, it can be used as a universal data carrier also.
type EnvVariableObj struct {
	// Key ..
	Key string
	// Value ...
	Value string
	// Placeholder is used for config file rendering (e.g., "__#VAR_PLACEHOLDER#__BKMS_POD_IP__").
	// Only runtime variables (RuntimeVars) have this field set. ToMap() prefers
	// Placeholder over Value, so that config rendering preserves the placeholder
	// for the init container to replace with the actual value at pod startup.
	Placeholder string
	// Description is an optional field to describe the env variable.
	Description string
	// IsBuiltin indicates whether the env variable is bkms built-in.
	IsBuiltin bool
	// IsSensitive indicates whether the env variable is sensitive.
	IsSensitive bool
	// ValueFrom is an optional field to specify the source of the env variable value.
	ValueFrom *corev1.EnvVarSource
}

// ValueForDisplay returns the value that is safe to show in user-facing responses.
func (o EnvVariableObj) ValueForDisplay() string {
	if o.IsSensitive {
		return SensitiveValueMask
	}
	return o.Value
}

// ToKubeObj converts the EnvVariableObj to Kubernetes corev1.EnvVar object.
func (o EnvVariableObj) ToKubeObj() corev1.EnvVar {
	return corev1.EnvVar{
		Name:      o.Key,
		Value:     o.Value,
		ValueFrom: o.ValueFrom,
	}
}

// EnvVariableList represents a list of EnvVariableObj.
type EnvVariableList []EnvVariableObj

// ToKubeObjs converts the EnvVariableList to a list of Kubernetes corev1.EnvVar objects.
func (varList EnvVariableList) ToKubeObjs() []corev1.EnvVar {
	return lo.Map(varList, func(varObj EnvVariableObj, _ int) corev1.EnvVar {
		return varObj.ToKubeObj()
	})
}

// ToMap converts the EnvVariableList to a map of key-value pairs.
func (varList EnvVariableList) ToMap() map[string]string {
	result := make(map[string]string)
	for _, v := range varList {
		// NOTE：运行时变量会设置 Placeholder，用于在配置文件渲染时保留占位符，由 Init Container 在 Pod 启动时替换。
		// 此时需要使用 Placeholder 作为值，而不是 Value。
		if v.Placeholder != "" {
			result[v.Key] = v.Placeholder
		} else {
			result[v.Key] = v.Value
		}
	}
	return result
}

// ToDeduplicatedList returns the effective env var list with duplicate keys removed.
// Later items have higher priority, so only the last item for the same key is kept.
func (varList EnvVariableList) ToDeduplicatedList() EnvVariableList {
	latestIndexByKey := make(map[string]int, len(varList))
	for i, item := range varList {
		latestIndexByKey[item.Key] = i
	}

	result := make(EnvVariableList, 0, len(latestIndexByKey))
	for i, item := range varList {
		// 同一个 key 只保留最后一次出现的记录，因为后面的环境变量层已经在
		// 生效结果里覆盖了前面的值。
		if latestIndexByKey[item.Key] != i {
			continue
		}
		result = append(result, item)
	}
	return result
}

// EnvVariableRichItem wraps EnvVariableObj with source info.
type EnvVariableRichItem struct {
	// Obj is the env variable object.
	Obj EnvVariableObj

	// Source indicates where this env var comes from.
	// INFO: It reuses the ConflictedSource type
	Source ConflictedSource
}

// EnvVariableRichList wraps env vars with source metadata and other information.
type EnvVariableRichList struct {
	Vars []EnvVariableRichItem
}

// GetDataList returns the underlying env vars in the same priority order.
func (l EnvVariableRichList) GetDataList() EnvVariableList {
	return lo.Map(l.Vars, func(item EnvVariableRichItem, _ int) EnvVariableObj {
		return item.Obj
	})
}

// ToDeduplicatedList returns the effective rich env var list with duplicate keys removed.
// Later items have higher priority, so only the last item for the same key is kept.
func (l EnvVariableRichList) ToDeduplicatedList() EnvVariableRichList {
	latestIndexByKey := make(map[string]int, len(l.Vars))
	for i, item := range l.Vars {
		latestIndexByKey[item.Obj.Key] = i
	}

	result := make([]EnvVariableRichItem, 0, len(latestIndexByKey))
	for i, item := range l.Vars {
		if latestIndexByKey[item.Obj.Key] != i {
			continue
		}
		result = append(result, item)
	}
	return EnvVariableRichList{Vars: result}
}

// SortBySourcePriority sorts the Vars in EnvVariableRichList by their source priority.
func (l *EnvVariableRichList) SortBySourcePriority() {
	sort.SliceStable(l.Vars, func(i, j int) bool {
		left := EnvVarSourcePriority(l.Vars[i].Source.Source)
		right := EnvVarSourcePriority(l.Vars[j].Source.Source)
		if left != right {
			return left < right
		}
		// For vars with the same source, sort by key to ensure deterministic order.
		return l.Vars[i].Obj.Key < l.Vars[j].Obj.Key
	})
}
