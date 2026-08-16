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

// Package runtimerender builds Kubernetes resources that render runtime-only
// values into application config files before the main container starts.
//
// The package exists for runtime-only Pod values such as BKMS_POD_IP,
// BKMS_POD_NAME, and BKMS_NODE_IP. These values are unknown to the bkms-server
// backend while it builds the workload spec, because Kubernetes only resolves
// them after the Pod is scheduled or created. The backend writes placeholders
// into a ConfigMap template, and the generated init container copies that
// template into an emptyDir volume and replaces the placeholders with values
// injected by the Kubernetes Downward API.
package runtimerender

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// initContainerImage is the image for the init container.
// Only depends on cp and sed commands now.
const initContainerImage = "busybox:1.36"

// ConfigParams describes a config file rendered by an init container at pod startup.
type ConfigParams struct {
	// WorkloadType is used to derive default volume, mount path, and init container names.
	// For example, "trpc" produces "trpc-config-template", "/trpc-config-template", and "trpc-init".
	WorkloadType string
	// ConfigMapName is the ConfigMap resource name that holds the config template.
	ConfigMapName string
	// Files contains all config files rendered into the workload, including the main file
	// and any extra plain files.
	Files []ConfigFileParams
}

// ConfigFileParams describes one rendered config file.
type ConfigFileParams struct {
	// FileName is the config file name.
	FileName string
	// FilePath is the config file directory in the workload container.
	FilePath string
	// FileContent is the config template content written to the ConfigMap.
	FileContent string
}

// Config is the Kubernetes output for init-container based config rendering.
type Config struct {
	// MainContainerMounts are the volume mounts for the main container (reads from emptyDir).
	MainContainerMounts []corev1.VolumeMount
	// Volumes includes both the ConfigMap volume (template) and the emptyDir volume (rendered output).
	Volumes []corev1.Volume
	// ConfigMap is the ConfigMap resource containing the config template.
	ConfigMap corev1.ConfigMap
	// InitContainerSpecs holds the init container that performs runtime rendering.
	InitContainerSpecs []corev1.Container
}

// Storage returns volume mounts and volumes for runtime-rendered config files.
func (c *Config) Storage(
	_ context.Context,
) ([]corev1.VolumeMount, []corev1.Volume, error) {
	if c == nil {
		return nil, nil, nil
	}
	return slices.Clone(c.MainContainerMounts), slices.Clone(c.Volumes), nil
}

// ExtraResources returns the ConfigMap that stores the config template.
func (c *Config) ExtraResources(
	_ context.Context,
) ([]unstructured.Unstructured, error) {
	if c == nil || c.ConfigMap.Name == "" {
		return nil, nil
	}

	extraObjs, err := plugin.ToUnstructured(&c.ConfigMap)
	if err != nil {
		return nil, errors.Wrap(err, "converting runtime rendered config map as unstructured")
	}
	return slices.Clone(extraObjs), nil
}

// InitContainers returns the init containers for runtime config rendering.
// The init container uses sed to replace special placeholders (e.g., __#VAR_PLACEHOLDER#__BKMS_POD_IP__)
// with runtime values injected by Kubernetes Downward API.
func (c *Config) InitContainers(
	_ context.Context,
) ([]corev1.Container, error) {
	if c == nil {
		return nil, nil
	}
	return slices.Clone(c.InitContainerSpecs), nil
}

// BuildConfig builds config resources with init container support for runtime
// variable rendering.
//
// The build produces:
//   - A ConfigMap volume mounted at a temporary path (template source for init container)
//   - An emptyDir volume mounted at the final config path (rendered output)
//   - An init container that runs sed to replace __VAR_NAME__ placeholders with runtime values
//
// Runtime variables (BKMS_POD_IP, BKMS_POD_NAME, BKMS_NODE_IP) are rendered as special
// placeholders (e.g., __#VAR_PLACEHOLDER#__BKMS_POD_IP__) at compile time. The init container then replaces
// these placeholders with actual values from the Kubernetes Downward API at pod startup.
func BuildConfig(params ConfigParams) (*Config, error) {
	names := runtimeRenderNames(params.WorkloadType)
	files := slices.Clone(params.Files)
	if len(files) == 0 {
		return &Config{}, nil
	}
	if err := validateConfigFiles(files); err != nil {
		return nil, err
	}

	configMapVolume := corev1.Volume{
		Name: names.templateVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: params.ConfigMapName},
			},
		},
	}
	renderedVolume := corev1.Volume{
		Name: names.renderedVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	configMap := corev1.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: params.ConfigMapName},
		Data:       make(map[string]string, len(files)),
	}

	mainMounts, configMapItems, commandParts := buildFileMapping(names, files, &configMap)
	configMapVolume.ConfigMap.Items = configMapItems

	initContainer := buildInitContainer(names, commandParts)

	return &Config{
		MainContainerMounts: mainMounts,
		Volumes:             []corev1.Volume{configMapVolume, renderedVolume},
		ConfigMap:           configMap,
		InitContainerSpecs:  []corev1.Container{initContainer},
	}, nil
}

// buildFileMapping 遍历文件列表，生成 ConfigMap 数据、主容器挂载和 init container sed 命令。
func buildFileMapping(
	names runtimeNames,
	files []ConfigFileParams,
	configMap *corev1.ConfigMap,
) ([]corev1.VolumeMount, []corev1.KeyToPath, []string) {
	mainMounts := make([]corev1.VolumeMount, 0, len(files))
	configMapItems := make([]corev1.KeyToPath, 0, len(files))
	commandParts := make([]string, 0, len(files))
	for i, file := range files {
		fileAlias := runtimeRenderFileAlias(i, file)

		configMap.Data[fileAlias] = file.FileContent
		configMapItems = append(configMapItems, corev1.KeyToPath{
			Key:  fileAlias,
			Path: fileAlias,
		})
		mainMounts = append(mainMounts, corev1.VolumeMount{
			Name:      names.renderedVolumeName,
			MountPath: filepath.Join(file.FilePath, file.FileName),
			SubPath:   fileAlias,
		})

		templateFilePath := filepath.Join(names.templateMountPath, fileAlias)
		renderedFilePath := filepath.Join(names.renderedMountPath, fileAlias)
		commandParts = append(commandParts, BuildSedCommand(templateFilePath, renderedFilePath))
	}
	return mainMounts, configMapItems, commandParts
}

// buildInitContainer 构建执行运行时变量替换的 init container。
func buildInitContainer(names runtimeNames, commandParts []string) corev1.Container {
	return corev1.Container{
		Name:  names.initContainerName,
		Image: initContainerImage,
		Command: []string{
			"sh", "-c",
			strings.Join(commandParts, " && "),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      names.templateVolumeName,
				MountPath: names.templateMountPath,
			},
			{
				Name:      names.renderedVolumeName,
				MountPath: names.renderedMountPath,
			},
		},
	}
}

// runtimeRenderFileAlias 为运行时渲染链路生成内部文件名。
// 这里不直接使用最终挂载路径，而是使用扁平 alias：
// 1. 中间模板目录和渲染目录只需要稳定唯一的文件标识；
// 2. 避免把容器内真实目录结构复制到临时目录中；
// 3. 最终挂载位置仍由 main container 的 MountPath + SubPath 决定。
func runtimeRenderFileAlias(index int, file ConfigFileParams) string {
	return fmt.Sprintf("%02d-%s", index, file.FileName)
}

// validateConfigFiles 校验一组待渲染文件的最终挂载目标是否冲突。
func validateConfigFiles(files []ConfigFileParams) error {
	targetPaths := make(map[string]struct{}, len(files))
	for _, file := range files {
		// 冲突判断基于主容器内的最终落点，而不是中间 alias。
		targetPath := filepath.Join(file.FilePath, file.FileName)
		if _, exists := targetPaths[targetPath]; exists {
			return errors.Errorf("duplicate config mount path %q", targetPath)
		}
		targetPaths[targetPath] = struct{}{}
	}
	return nil
}

type runtimeNames struct {
	templateVolumeName string
	renderedVolumeName string
	templateMountPath  string
	renderedMountPath  string
	initContainerName  string
}

func runtimeRenderNames(workloadType string) runtimeNames {
	return runtimeNames{
		templateVolumeName: fmt.Sprintf("%s-config-template", workloadType),
		renderedVolumeName: fmt.Sprintf("%s-config-rendered", workloadType),
		templateMountPath:  fmt.Sprintf("/%s-config-template", workloadType),
		renderedMountPath:  fmt.Sprintf("/%s-config-rendered", workloadType),
		initContainerName:  fmt.Sprintf("%s-init", workloadType),
	}
}

// BuildSedCommand constructs a shell command that copies the template config file
// and applies sed replacements for all runtime variable placeholders.
//
// The generated command looks like:
//
//	cp /config-template/file /config-rendered/file &&
//	sed -i 's/__#VAR_PLACEHOLDER#__BKMS_POD_IP__/'"$BKMS_POD_IP"'/g' /config-rendered/file &&
//	sed -i 's/__#VAR_PLACEHOLDER#__BKMS_POD_NAME__/'"$BKMS_POD_NAME"'/g' /config-rendered/file &&
//	sed -i 's/__#VAR_PLACEHOLDER#__BKMS_NODE_IP__/'"$BKMS_NODE_IP"'/g' /config-rendered/file
func BuildSedCommand(templatePath, renderedPath string) string {
	parts := []string{
		fmt.Sprintf("cp '%s' '%s'", templatePath, renderedPath),
	}
	for _, rv := range envvars.RuntimeVars {
		placeholder := envvars.RuntimeVarPlaceholder(rv.Name)
		// Use single-quote-break-single-quote pattern to safely inject the env var value:
		// sed -i 's/__PLACEHOLDER__/'"$VAR_NAME"'/g' file
		parts = append(parts, fmt.Sprintf(
			"sed -i 's/%s/'\"$%s\"'/g' '%s'",
			placeholder, rv.Name, renderedPath,
		))
	}
	return strings.Join(parts, " && ")
}
