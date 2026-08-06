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

package deploy

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// mockClient 用于测试的 mock client
type mockClient struct {
	client.Client
	envs []client.Env
	err  error
}

func (m *mockClient) ListEnvs(_ context.Context, _ string) ([]client.Env, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.envs, nil
}

var _ = Describe("Env", func() {
	// ==================== parseEnvNames ====================
	Describe("parseEnvNames", func() {
		DescribeTable("parse env names from comma-separated string",
			func(input string, expected []string) {
				result := parseEnvNames(input)
				Expect(result).To(Equal(expected))
			},
			// 单个环境名称
			Entry("single env name", "prod", []string{"prod"}),
			// 多个环境名称
			Entry("multiple env names", "prod,staging,test", []string{"prod", "staging", "test"}),
			// 含空格的环境名称
			Entry("env names with spaces", "prod, staging , test", []string{"prod", "staging", "test"}),
			// 含空字符串（前导逗号）
			Entry("leading comma produces empty string", ",prod,staging", []string{"prod", "staging"}),
			// 含空字符串（尾部逗号）
			Entry("trailing comma produces empty string", "prod,staging,", []string{"prod", "staging"}),
			// 含空字符串（连续逗号）
			Entry("consecutive commas produce empty strings", "prod,,staging", []string{"prod", "staging"}),
			// 重复环境名称
			Entry("duplicate env names are deduplicated", "prod,staging,prod", []string{"prod", "staging"}),
			// 全部为空
			Entry("all empty segments", ",,", []string{}),
			// 空字符串
			Entry("empty string", "", []string{}),
			// 仅空格
			Entry("whitespace only", "  ", []string{}),
			// 空格和逗号
			Entry("spaces and commas only", " , , ", []string{}),
		)
	})

	// ==================== validateEnvNames ====================
	Describe("validateEnvNames", func() {
		defaultEnvs := []client.Env{
			{Name: "prod"},
			{Name: "staging"},
			{Name: "test"},
		}

		DescribeTable("validate env names against workspace env list",
			func(cli *mockClient, envNames []string, expectErr bool, errSubstrings []string) {
				err := validateEnvNames(context.Background(), cli, "ws-1", envNames)
				if !expectErr {
					Expect(err).NotTo(HaveOccurred())
					return
				}
				Expect(err).To(HaveOccurred())
				for _, sub := range errSubstrings {
					Expect(err.Error()).To(ContainSubstring(sub))
				}
			},
			// 所有环境名称都存在时应返回 nil
			Entry("returns nil when all env names exist",
				&mockClient{envs: defaultEnvs}, []string{"prod", "staging"}, false, nil),
			// 单个环境名称存在时应返回 nil
			Entry("returns nil when a single env name exists",
				&mockClient{envs: defaultEnvs}, []string{"prod"}, false, nil),
			// 部分环境名称不存在时应返回错误
			Entry(
				"returns error when some env names do not exist",
				&mockClient{
					envs: defaultEnvs,
				},
				[]string{"prod", "nonexistent"},
				true,
				[]string{"nonexistent", "env(s) not found"},
			),
			// 全部环境名称不存在时应返回错误
			Entry("returns error when all env names do not exist",
				&mockClient{envs: defaultEnvs}, []string{"foo", "bar"}, true, []string{"foo", "bar"}),
			// ListEnvs 返回错误时应传播错误
			Entry("propagates error when ListEnvs fails",
				&mockClient{err: context.DeadlineExceeded}, []string{"prod"}, true, []string{"failed to list envs"}),
			// 环境列表为空时所有名称都应不存在
			Entry("returns error when env list is empty",
				&mockClient{envs: []client.Env{}}, []string{"prod"}, true, []string{"prod"}),
		)
	})
})
