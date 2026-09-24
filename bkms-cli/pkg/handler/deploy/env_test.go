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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

// envListResult 保存环境列表测试场景中的 API 返回值。
type envListResult struct {
	envs []client.Env
	err  error
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

		DescribeTable("validate env names against app env list",
			func(data *envListResult, envNames []string, expectErr bool, errSubstrings []string) {
				ctx := context.Background()
				cli := mocks.NewMockClient(GinkgoT())
				cli.EXPECT().ListAppEnvs(ctx, "app-1").Return(data.envs, data.err).Once()
				err := validateEnvNames(ctx, cli, "app-1", envNames)
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
				&envListResult{envs: defaultEnvs}, []string{"prod", "staging"}, false, nil),
			// 单个环境名称存在时应返回 nil
			Entry("returns nil when a single env name exists",
				&envListResult{envs: defaultEnvs}, []string{"prod"}, false, nil),
			Entry("accepts owned feature environments alongside standard environments",
				&envListResult{envs: []client.Env{
					{Name: "staging", Kind: "standard"},
					{Name: "feat-1", Kind: "feature", OwnerAppID: "app-1"},
				}}, []string{"staging", "feat-1"}, false, nil),
			Entry("rejects a feature environment unavailable to this app",
				&envListResult{envs: defaultEnvs}, []string{"other-app-feat-1"}, true, []string{"other-app-feat-1"}),
			// 部分环境名称不存在时应返回错误
			Entry(
				"returns error when some env names do not exist",
				&envListResult{
					envs: defaultEnvs,
				},
				[]string{"prod", "nonexistent"},
				true,
				[]string{"nonexistent", "env(s) not found"},
			),
			// 全部环境名称不存在时应返回错误
			Entry("returns error when all env names do not exist",
				&envListResult{envs: defaultEnvs}, []string{"foo", "bar"}, true, []string{"foo", "bar"}),
			// ListAppEnvs 返回错误时应传播错误
			Entry(
				"propagates error when ListAppEnvs fails",
				&envListResult{
					err: context.DeadlineExceeded,
				},
				[]string{"prod"},
				true,
				[]string{"failed to list envs"},
			),
			// 环境列表为空时所有名称都应不存在
			Entry("returns error when env list is empty",
				&envListResult{envs: []client.Env{}}, []string{"prod"}, true, []string{"prod"}),
		)
	})

	// ==================== validateDeployEnvs ====================
	Describe("validateDeployEnvs", func() {
		defaultEnvs := []client.Env{
			{Name: "prod", Kind: "standard"},
			{Name: "staging", Kind: "standard"},
			{Name: "feat-1", Kind: "feature", OwnerAppID: "app-1"},
		}

		type visibleEnvCase struct {
			envs          []client.Env
			envsErr       error
			app           *client.AppFull
			appErr        error
			expectGetApp  bool
			envNames      []string
			expectErr     bool
			errSubstrings []string
		}

		DescribeTable("validate deploy target envs against existence and visible env names",
			func(tc visibleEnvCase) {
				ctx := context.Background()
				cli := mocks.NewMockClient(GinkgoT())
				cli.EXPECT().ListAppEnvs(ctx, "app-1").Return(tc.envs, tc.envsErr).Once()
				if tc.expectGetApp {
					cli.EXPECT().GetApp(ctx, "app-1").Return(tc.app, tc.appErr).Once()
				}
				app, err := validateDeployEnvs(ctx, cli, "app-1", tc.envNames)
				if !tc.expectErr {
					Expect(err).NotTo(HaveOccurred())
					// 校验通过时返回应用详情，调用方无需再拉一次
					Expect(app).To(Equal(tc.app))
					return
				}
				Expect(err).To(HaveOccurred())
				Expect(app).To(BeNil())
				for _, sub := range tc.errSubstrings {
					Expect(err.Error()).To(ContainSubstring(sub))
				}
			},
			Entry("allows all envs when visibleEnvNames is empty",
				visibleEnvCase{
					envs:         defaultEnvs,
					app:          &client.AppFull{ID: "app-1", VisibleEnvNames: []string{}},
					expectGetApp: true,
					envNames:     []string{"prod", "feat-1"},
				}),
			Entry("allows all envs when visibleEnvNames is nil",
				visibleEnvCase{
					envs:         defaultEnvs,
					app:          &client.AppFull{ID: "app-1"},
					expectGetApp: true,
					envNames:     []string{"prod"},
				}),
			Entry("allows a listed standard env",
				visibleEnvCase{
					envs:         defaultEnvs,
					app:          &client.AppFull{ID: "app-1", VisibleEnvNames: []string{"staging"}},
					expectGetApp: true,
					envNames:     []string{"staging"},
				}),
			Entry("rejects an unlisted standard env",
				visibleEnvCase{
					envs:          defaultEnvs,
					app:           &client.AppFull{ID: "app-1", VisibleEnvNames: []string{"staging"}},
					expectGetApp:  true,
					envNames:      []string{"prod"},
					expectErr:     true,
					errSubstrings: []string{"prod", "visible environment"},
				}),
			Entry("lists every denied env when several are rejected",
				visibleEnvCase{
					envs:          defaultEnvs,
					app:           &client.AppFull{ID: "app-1", VisibleEnvNames: []string{"feat-1"}},
					expectGetApp:  true,
					envNames:      []string{"prod", "staging"},
					expectErr:     true,
					errSubstrings: []string{"prod", "staging", "visible environment"},
				}),
			Entry("allows the app's own feature env when the list is configured",
				visibleEnvCase{
					envs:         defaultEnvs,
					app:          &client.AppFull{ID: "app-1", VisibleEnvNames: []string{"staging"}},
					expectGetApp: true,
					envNames:     []string{"feat-1"},
				}),
			Entry("rejects another app's feature env when the list is configured",
				visibleEnvCase{
					envs: []client.Env{
						{Name: "staging", Kind: "standard"},
						{Name: "feat-b", Kind: "feature", OwnerAppID: "app-2"},
					},
					app:           &client.AppFull{ID: "app-1", VisibleEnvNames: []string{"staging"}},
					expectGetApp:  true,
					envNames:      []string{"feat-b"},
					expectErr:     true,
					errSubstrings: []string{"feat-b", "visible environment"},
				}),
			// 环境不存在时报 not found，不退化成可见环境错误，也不必再拉应用详情
			Entry("reports a missing env before checking visible env names",
				visibleEnvCase{
					envs:          defaultEnvs,
					envNames:      []string{"nonexistent"},
					expectErr:     true,
					errSubstrings: []string{"nonexistent", "env(s) not found"},
				}),
			Entry("propagates ListAppEnvs errors",
				visibleEnvCase{
					envsErr:       context.DeadlineExceeded,
					envNames:      []string{"prod"},
					expectErr:     true,
					errSubstrings: []string{"failed to list envs"},
				}),
			// 拉取应用详情失败时必须报错，不能跳过可见环境校验
			Entry("does not skip the check when GetApp fails",
				visibleEnvCase{
					envs:          defaultEnvs,
					appErr:        context.DeadlineExceeded,
					expectGetApp:  true,
					envNames:      []string{"prod"},
					expectErr:     true,
					errSubstrings: []string{"failed to get app"},
				}),
		)
	})
})
