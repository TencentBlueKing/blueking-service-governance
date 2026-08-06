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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("BuildInstanceEnvVars", func() {
	var inst *model.ServiceInstance
	var ctx context.Context

	BeforeEach(func() {
		inst = &model.ServiceInstance{
			Name:        "mysql-instance-1",
			ServiceName: "mysql",
			Credentials: map[string]any{},
		}
		ctx = context.Background()
	})

	It("outputs all credentials as sensitive env vars and renders custom env vars from them", func() {
		inst.Credentials = map[string]any{
			"MYSQL_HOST":     "127.0.0.1",
			"MYSQL_PASSWORD": "blueking",
		}
		inst.CustomEnvVars = map[string]string{
			"MYSQL_DSN": "mysql://${{env.MYSQL_PASSWORD}}@${{env.MYSQL_HOST}}",
		}

		vars, err := depenvvars.BuildInstanceEnvVars(ctx, inst)
		Expect(err).NotTo(HaveOccurred())

		// 3 个 = 2 个 credentials + 1 个衍生
		Expect(vars).To(HaveLen(3))

		byKey := toMap(vars)
		Expect(byKey).To(HaveKey("MYSQL_HOST"))
		Expect(byKey["MYSQL_HOST"].Value).To(Equal("127.0.0.1"))
		Expect(byKey["MYSQL_HOST"].IsSensitive).To(BeTrue())

		Expect(byKey).To(HaveKey("MYSQL_PASSWORD"))
		Expect(byKey["MYSQL_PASSWORD"].Value).To(Equal("blueking"))
		Expect(byKey["MYSQL_PASSWORD"].IsSensitive).To(BeTrue())

		Expect(byKey).To(HaveKey("MYSQL_DSN"))
		Expect(byKey["MYSQL_DSN"].Value).To(Equal("mysql://blueking@127.0.0.1"))
	})

	It("skips credentials output for polaris service", func() {
		inst.ServiceName = "polaris"
		inst.Credentials = map[string]any{
			"POLARIS_TOKEN": "tk-xyz",
		}

		vars, err := depenvvars.BuildInstanceEnvVars(ctx, inst)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(BeEmpty())
	})

	It("returns error on invalid custom env var template", func() {
		inst.Credentials = map[string]any{
			"MYSQL_HOST": "127.0.0.1",
		}
		inst.CustomEnvVars = map[string]string{
			"BAD_VAR": "${{ unclosed",
		}

		_, err := depenvvars.BuildInstanceEnvVars(ctx, inst)
		Expect(err).To(HaveOccurred())
	})

	It("returns empty list when credentials and custom env vars are both empty", func() {
		vars, err := depenvvars.BuildInstanceEnvVars(ctx, inst)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars).To(BeEmpty())
	})
})

func toMap(list envvartypes.EnvVariableList) map[string]envvartypes.EnvVariableObj {
	m := make(map[string]envvartypes.EnvVariableObj, len(list))
	for _, v := range list {
		m[v.Key] = v
	}
	return m
}
