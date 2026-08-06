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

package workload

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// buildVolumeMounts builds volume mounts and volumes from the given VolumeMounts definition.
// Only HostPath type is supported currently.
func buildVolumeMounts(volumeMounts appmodel.VolumeMounts) ([]corev1.VolumeMount, []corev1.Volume, error) {
	if len(volumeMounts.HostPath) == 0 {
		return nil, nil, nil
	}

	mounts := make([]corev1.VolumeMount, 0, len(volumeMounts.HostPath))
	volumes := make([]corev1.Volume, 0, len(volumeMounts.HostPath))

	for i, hostPath := range volumeMounts.HostPath {
		hostPathType, err := parseHostPathType(hostPath.Type)
		if err != nil {
			return nil, nil, err
		}

		name := fmt.Sprintf("hostpath-%d", i)
		mounts = append(mounts, corev1.VolumeMount{
			Name:      name,
			MountPath: strings.TrimSpace(hostPath.MountPath),
		})
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: strings.TrimSpace(hostPath.HostPath),
					Type: &hostPathType,
				},
			},
		})
	}

	return mounts, volumes, nil
}

// parseHostPathType parses the given string to corev1.HostPathType, it return error if the type is invalid.
func parseHostPathType(value string) (corev1.HostPathType, error) {
	// Use DirectoryOrCreate as default type
	if value == "" {
		return corev1.HostPathDirectoryOrCreate, nil
	}
	switch corev1.HostPathType(value) {
	case corev1.HostPathDirectoryOrCreate,
		corev1.HostPathDirectory,
		corev1.HostPathFileOrCreate,
		corev1.HostPathFile,
		corev1.HostPathSocket,
		corev1.HostPathCharDev,
		corev1.HostPathBlockDev:
		return corev1.HostPathType(value), nil
	default:
		return "", errors.Errorf("invalid host path type %q", value)
	}
}
