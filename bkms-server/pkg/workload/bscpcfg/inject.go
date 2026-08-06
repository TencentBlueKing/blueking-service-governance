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
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
)

// MergePodSpec 相关的 sentinel error 变量
var (
	// ErrPodSpecNil podSpec 参数为 nil
	ErrPodSpecNil = errors.New("podSpec must not be nil")

	// ErrMainContainerNotFound 指定的主容器在 PodSpec.Containers 中不存在
	ErrMainContainerNotFound = errors.New("main container not found in PodSpec.Containers")
)

// InjectFromStore 从 Store 获取配置快照并将产物后置注入到 PodSpec 中。
//
// 当指定 app+env 未配置时不做任何修改，直接返回 nil。
func InjectFromStore(
	ctx context.Context,
	store bscpcfg.Store,
	appID, envName, mainContainerName string,
	podSpec *corev1.PodSpec,
) error {
	fragment, err := BuildFromStore(ctx, store, appID, envName)
	if err != nil {
		return err
	}
	return MergePodSpec(podSpec, fragment, mainContainerName)
}

// MergePodSpec 将 fragment 合并到 PodSpec 中。
// 如果 fragment 为 nil，则不做任何修改直接返回。
func MergePodSpec(podSpec *corev1.PodSpec, fragment *PodFragment, mainContainerName string) error {
	if fragment == nil {
		return nil
	}
	if podSpec == nil {
		return ErrPodSpecNil
	}
	// 查找主容器
	mainIdx := lo.IndexOf(
		lo.Map(podSpec.Containers, func(c corev1.Container, _ int) string { return c.Name }),
		mainContainerName,
	)
	if mainIdx == -1 {
		return errors.Wrapf(ErrMainContainerNotFound, "main container name: %s", mainContainerName)
	}
	// 检查 init container 是否已注入
	fragICNames := lo.Map(fragment.InitContainers, func(c corev1.Container, _ int) string { return c.Name })
	alreadyInjected := lo.SomeBy(podSpec.InitContainers, func(ic corev1.Container) bool {
		return lo.Contains(fragICNames, ic.Name)
	})
	if !alreadyInjected {
		podSpec.InitContainers = append(podSpec.InitContainers, fragment.InitContainers...)
	}
	// 检查 sidecar 是否已注入
	fragSidecarNames := lo.Map(fragment.Containers, func(c corev1.Container, _ int) string { return c.Name })
	sidecarAlreadyInjected := lo.SomeBy(podSpec.Containers, func(c corev1.Container) bool {
		return lo.Contains(fragSidecarNames, c.Name)
	})
	if !sidecarAlreadyInjected {
		podSpec.Containers = append(podSpec.Containers, fragment.Containers...)
	}
	// 检查 volume 是否已注入
	fragVolNames := lo.Map(fragment.Volumes, func(v corev1.Volume, _ int) string { return v.Name })
	volumeAlreadyInjected := lo.SomeBy(podSpec.Volumes, func(v corev1.Volume) bool {
		return lo.Contains(fragVolNames, v.Name)
	})
	if !volumeAlreadyInjected {
		podSpec.Volumes = append(podSpec.Volumes, fragment.Volumes...)
	}
	// 检查主容器 volumeMount 是否已注入
	fragMountNames := lo.Map(
		fragment.MainContainerVolumeMounts,
		func(vm corev1.VolumeMount, _ int) string { return vm.Name },
	)
	mountAlreadyInjected := lo.SomeBy(podSpec.Containers[mainIdx].VolumeMounts, func(vm corev1.VolumeMount) bool {
		return lo.Contains(fragMountNames, vm.Name)
	})
	if !mountAlreadyInjected {
		podSpec.Containers[mainIdx].VolumeMounts = append(
			podSpec.Containers[mainIdx].VolumeMounts,
			fragment.MainContainerVolumeMounts...,
		)
	}
	return nil
}
