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
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/resourcevalue"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// resourceLimitBuiltinVars builds builtin env vars from a workload resource map.
// The map value is "{requests}" or "{requests}-{limits}". Both sides are parsed
// by resourcevalue.ParseResourceBounds; only the original limit text is injected.
func resourceLimitBuiltinVars(resources map[string]string) envvartypes.EnvVariableList {
	if len(resources) == 0 {
		return nil
	}

	specs := []struct {
		resourceName string
		envName      string
		description  string
	}{
		{string(corev1.ResourceCPU), EnvVarNameCPULimit, "The CPU limit configured for the workload"},
		{string(corev1.ResourceMemory), EnvVarNameMemoryLimit, "The memory limit configured for the workload"},
	}

	vars := make(envvartypes.EnvVariableList, 0, len(specs))
	for _, spec := range specs {
		raw, ok := resources[spec.resourceName]
		if !ok {
			continue
		}
		bounds, err := resourcevalue.ParseResourceBounds(raw)
		if err != nil {
			continue
		}
		vars = append(vars, envvartypes.EnvVariableObj{
			Key:         spec.envName,
			Value:       bounds.Limit,
			Description: spec.description,
			IsBuiltin:   true,
		})
	}
	if len(vars) == 0 {
		return nil
	}
	return vars
}
