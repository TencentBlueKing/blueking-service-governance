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

// Package bscpcfg 提供配置管理的 Pod 注入装配能力（底层借助 BSCP 渠道下发配置）。
package bscpcfg

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
)

const (
	// InitContainerName bscp-init 容器名称
	InitContainerName = "bscp-init"
	// SidecarContainerName bscp-sidecar 容器名称
	SidecarContainerName = "bscp-sidecar"
	// VolumeName bscp-temp volume 名称
	// 仅 bscp init 和 sidecar 容器共享
	VolumeName = "bscp-temp"
	// ShareVolumeName bscp-share volume 名称（sidecar 和主容器共享）
	ShareVolumeName = "bscp-share"

	// InitImage bscp-init 容器镜像
	InitImage = "mirrors.tencent.com/bscp/bscp-init:latest"
	// SidecarImage bscp-sidecar 容器镜像
	SidecarImage = "mirrors.tencent.com/bscp/bscp-sidecar:latest"

	// fileCacheDisabledArg 禁用文件缓存的启动参数
	fileCacheDisabledArg = "--file-cache-enabled=false"

	// BscpDownloadPath bscp 下载配置文件的临时路径
	// 仅 bscp init 和 sidecar 容器共享
	BscpDownloadPath = "/data/bscp-temp"
	// BscpShareBasePath bscp-share volume 在 bccp sidecar 容器中的固定挂载路径。
	// 后置脚本 rsync 的目标路径必须与此保持一致
	BscpShareBasePath = "/data/bkms/app/cfg"
)

// Params 装配所需的参数
type Params struct {
	// BscpBizID BSCP 业务 ID
	BscpBizID string
	// AppNames 绑定的服务名称列表（允许多个 bscp 配置，使用逗号分隔）
	AppNames string
	// MountPath 业务容器指定的挂载路径
	MountPath string
	// FeedAddr 配置订阅地址
	FeedAddr string
	// Token 服务秘钥
	Token string
}

// PodFragment 装配产出的 pod 片段，待合并到完整 pod 中
type PodFragment struct {
	// WorkloadName 目标 workload 名称
	WorkloadName string
	// WorkloadKind 目标工作负载类型
	WorkloadKind string
	// InitContainers init 容器列表
	InitContainers []corev1.Container
	// Containers sidecar 容器列表
	Containers []corev1.Container
	// Volumes volume 列表
	Volumes []corev1.Volume
	// MainContainerVolumeMounts 主容器 volumeMount
	MainContainerVolumeMounts []corev1.VolumeMount
}

// Build 根据参数装配配置管理所需的 pod 片段。
func Build(params Params) *PodFragment {
	// 构造 bscp init、sidecar 共享环境变量
	bscpEnvVars := []corev1.EnvVar{
		{Name: "biz", Value: params.BscpBizID},
		{Name: "app", Value: params.AppNames},
		{Name: "feed_addrs", Value: params.FeedAddr},
		{Name: "token", Value: params.Token},
		{Name: "temp_dir", Value: BscpDownloadPath},
	}
	// bscp-temp volumeMount
	bscpTempMount := corev1.VolumeMount{
		Name:      VolumeName,
		MountPath: BscpDownloadPath,
	}
	// bscp-sidecar volumeMount
	sidecarShareMount := corev1.VolumeMount{
		Name:      ShareVolumeName,
		MountPath: BscpShareBasePath,
	}
	// 主容器（用户容器） volumeMount
	mainShareMount := corev1.VolumeMount{
		Name:      ShareVolumeName,
		MountPath: params.MountPath,
	}

	return &PodFragment{
		// bscp-initContainer
		InitContainers: []corev1.Container{
			{
				Name:         InitContainerName,
				Image:        InitImage,
				Args:         []string{fileCacheDisabledArg},
				Env:          bscpEnvVars,
				VolumeMounts: []corev1.VolumeMount{bscpTempMount},
			},
		},
		// bscp-sidecarContainer
		Containers: []corev1.Container{
			{
				Name:         SidecarContainerName,
				Image:        SidecarImage,
				Args:         []string{fileCacheDisabledArg},
				Env:          bscpEnvVars,
				VolumeMounts: []corev1.VolumeMount{bscpTempMount, sidecarShareMount},
			},
		},
		// volumes: bscp-temp + bscp-share
		Volumes: []corev1.Volume{
			{
				Name: VolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: ShareVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		// 主容器挂载 bscp-share volume
		MainContainerVolumeMounts: []corev1.VolumeMount{mainShareMount},
	}
}

// BuildFromStore 从 Store 获取配置快照并装配 pod 片段。
//
// 当指定 app+env 未配置时返回 nil, nil，调用方无需额外判断。
func BuildFromStore(
	ctx context.Context,
	store bscpcfg.Store,
	appID, envName string,
) (*PodFragment, error) {
	snapshot, err := store.GetSnapshot(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "getting bscp config snapshot")
	}
	if snapshot == nil {
		return nil, nil
	}

	if err = snapshot.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating bscp config snapshot")
	}

	fragment := Build(Params{
		BscpBizID: snapshot.Metadata.BscpBizID,
		AppNames:  snapshot.GetServiceNames(),
		MountPath: snapshot.Metadata.MountPath,
		FeedAddr:  snapshot.Metadata.FeedAddr,
		Token:     snapshot.Metadata.Token,
	})
	fragment.WorkloadName = snapshot.Metadata.WorkloadName
	fragment.WorkloadKind = snapshot.Metadata.WorkloadKind
	return fragment, nil
}
