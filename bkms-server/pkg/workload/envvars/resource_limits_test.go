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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("resourceLimitBuiltinVars", func() {
	It("should inject the limit text as builtin env vars", func() {
		Expect(resourceLimitBuiltinVars(map[string]string{
			"cpu":    "100m-200m",
			"memory": "2048Mi",
		})).To(Equal(envvartypes.EnvVariableList{
			{
				Key:         EnvVarNameCPULimit,
				Value:       "200m",
				Description: "The CPU limit configured for the workload",
				IsBuiltin:   true,
			},
			{
				Key:         EnvVarNameMemoryLimit,
				Value:       "2048Mi",
				Description: "The memory limit configured for the workload",
				IsBuiltin:   true,
			},
		}))
	})

	It("should omit a resource that is absent or cannot be parsed", func() {
		Expect(resourceLimitBuiltinVars(nil)).To(BeNil())
		Expect(resourceLimitBuiltinVars(map[string]string{
			"memory": "2Gi",
		})).To(Equal(envvartypes.EnvVariableList{{
			Key:         EnvVarNameMemoryLimit,
			Value:       "2Gi",
			Description: "The memory limit configured for the workload",
			IsBuiltin:   true,
		}}))
		Expect(resourceLimitBuiltinVars(map[string]string{
			"cpu":    "100m-bad",
			"memory": "2Gi",
		})).To(Equal(envvartypes.EnvVariableList{{
			Key:         EnvVarNameMemoryLimit,
			Value:       "2Gi",
			Description: "The memory limit configured for the workload",
			IsBuiltin:   true,
		}}))
	})
})
