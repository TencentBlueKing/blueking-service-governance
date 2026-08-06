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

package bscpcfg

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
)

// MergePodSpecMap 将 fragment 以原生 map 操作的方式合并到 podSpecMap 中。
// 与 MergePodSpec 功能一致，但不对用户原始数据做任何 corev1.PodSpec 反序列化，避免因 k8s 库版本不匹配导致字段丢失。
// 仅对注入片段（PodFragment 中的 Container/Volume/VolumeMount）做 json → map 转换， 这部分由我们自己构造，版本完全可控。
func MergePodSpecMap(podSpecMap map[string]any, fragment *PodFragment, mainContainerName string) error {
	if fragment == nil {
		return nil
	}
	if podSpecMap == nil {
		return ErrPodSpecNil
	}

	// 查找主容器索引
	containers := getSlice(podSpecMap, "containers")
	mainIdx := findContainerIndex(containers, mainContainerName)
	if mainIdx == -1 {
		return errors.Wrapf(ErrMainContainerNotFound, "main container name: %s", mainContainerName)
	}

	// 注入 initContainers
	if len(fragment.InitContainers) > 0 {
		initContainers := getSlice(podSpecMap, "initContainers")
		fragICNames := lo.Map(fragment.InitContainers, func(c corev1.Container, _ int) string { return c.Name })

		if !hasAnyName(initContainers, fragICNames) {
			fragMaps, err := toMapSlice(fragment.InitContainers)
			if err != nil {
				return errors.Wrap(err, "converting fragment initContainers to map")
			}
			podSpecMap["initContainers"] = append(initContainers, fragMaps...)
		}
	}

	// 注入 sidecar containers
	if len(fragment.Containers) > 0 {
		fragSidecarNames := lo.Map(fragment.Containers, func(c corev1.Container, _ int) string { return c.Name })

		if !hasAnyName(containers, fragSidecarNames) {
			fragMaps, err := toMapSlice(fragment.Containers)
			if err != nil {
				return errors.Wrap(err, "converting fragment containers to map")
			}
			containers = append(containers, fragMaps...)
			podSpecMap["containers"] = containers
		}
	}

	// 注入 volumes
	if len(fragment.Volumes) > 0 {
		volumes := getSlice(podSpecMap, "volumes")
		fragVolNames := lo.Map(fragment.Volumes, func(v corev1.Volume, _ int) string { return v.Name })

		if !hasAnyName(volumes, fragVolNames) {
			fragMaps, err := toMapSlice(fragment.Volumes)
			if err != nil {
				return errors.Wrap(err, "converting fragment volumes to map")
			}
			podSpecMap["volumes"] = append(volumes, fragMaps...)
		}
	}

	// 注入主容器 volumeMounts
	if len(fragment.MainContainerVolumeMounts) > 0 {
		mainContainer, ok := containers[mainIdx].(map[string]any)
		if !ok {
			return errors.New("main container is not a map[string]any")
		}

		mounts := getSlice(mainContainer, "volumeMounts")
		fragMountNames := lo.Map(
			fragment.MainContainerVolumeMounts, func(vm corev1.VolumeMount, _ int) string { return vm.Name },
		)

		if !hasAnyName(mounts, fragMountNames) {
			fragMaps, err := toMapSlice(fragment.MainContainerVolumeMounts)
			if err != nil {
				return errors.Wrap(err, "converting fragment volumeMounts to map")
			}
			mainContainer["volumeMounts"] = append(mounts, fragMaps...)
		}
	}

	return nil
}

// toMapSlice 通过 JSON 序列化/反序列化将 []T 转换为 []any（每个元素为 map[string]any）。
func toMapSlice[T any](items []T) ([]any, error) {
	jsonBytes, err := json.Marshal(items)
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	var result []any
	if err = json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, errors.Wrap(err, "json unmarshal")
	}
	return result, nil
}

// getSlice 从 map 中安全获取 key 对应的 []any，不存在或类型不匹配时返回 nil。
func getSlice(m map[string]any, key string) []any {
	slice, _ := m[key].([]any)
	return slice
}

// findContainerIndex 返回 containers 中 name 字段等于指定值的元素索引，未找到返回 -1。
func findContainerIndex(containers []any, name string) int {
	return lo.IndexOf(
		lo.Map(containers, func(item any, _ int) string {
			m, _ := item.(map[string]any)
			n, _ := m["name"].(string)
			return n
		}),
		name,
	)
}

// hasAnyName 检查 items 中是否存在 name 字段命中 names 的元素。
func hasAnyName(items []any, names []string) bool {
	existingNames := lo.FilterMap(items, func(item any, _ int) (string, bool) {
		m, ok := item.(map[string]any)
		if !ok {
			return "", false
		}
		n, _ := m["name"].(string)
		return n, n != ""
	})
	return lo.SomeBy(names, func(name string) bool {
		return lo.Contains(existingNames, name)
	})
}
