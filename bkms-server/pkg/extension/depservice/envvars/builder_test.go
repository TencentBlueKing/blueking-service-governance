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

package envvars_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("BuildBindingEnvVars", func() {
	It("renders templates from credentials and skips empty env vars", func() {
		ctx := context.Background()
		vars, err := depenvvars.BuildBindingEnvVars(ctx, nil, map[string]any{
			"REDIS_HOST": "127.0.0.1",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(BeEmpty())

		vars, err = depenvvars.BuildBindingEnvVars(ctx, map[string]string{
			"REDIS_HOST": "${{env.REDIS_HOST}}",
			"REDIS_DSN":  "redis://${{env.REDIS_HOST}}:6379/0",
		}, map[string]any{"REDIS_HOST": "10.0.0.1"})
		Expect(err).NotTo(HaveOccurred())
		byKey := toMap(vars)
		Expect(byKey["REDIS_HOST"].Value).To(Equal("10.0.0.1"))
		Expect(byKey["REDIS_DSN"].Value).To(Equal("redis://10.0.0.1:6379/0"))
	})

	It("returns error on invalid template", func() {
		_, err := depenvvars.BuildBindingEnvVars(context.Background(), map[string]string{
			"BAD_VAR": "${{ unclosed",
		}, map[string]any{"REDIS_HOST": "127.0.0.1"})
		Expect(err).To(HaveOccurred())
	})
})

func toMap(list envvartypes.EnvVariableList) map[string]envvartypes.EnvVariableObj {
	m := make(map[string]envvartypes.EnvVariableObj, len(list))
	for _, v := range list {
		m[v.Key] = v
	}
	return m
}
