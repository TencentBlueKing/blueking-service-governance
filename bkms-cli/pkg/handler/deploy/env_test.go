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

	// ==================== resolveEnvNames ====================
	Describe("resolveEnvNames", func() {
		ctx := context.Background()

		It("returns env names when only envName is provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			names, err := resolveEnvNames(ctx, cli, "app-1", "prod,staging", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(Equal([]string{"prod", "staging"}))
		})

		It("returns error when envName and envType are both provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			_, err := resolveEnvNames(ctx, cli, "app-1", "prod", "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mutually exclusive"))
		})

		It("returns error when neither envName nor envType is provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			_, err := resolveEnvNames(ctx, cli, "app-1", "", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required"))
		})

		It("expands env names by type when only envType is provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "dev-1", Type: "development"},
				{Name: "test-1", Type: "test"},
				{Name: "prod-1", Type: "production"},
			}, nil).Once()
			names, err := resolveEnvNames(ctx, cli, "app-1", "", "test,development")
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(Equal([]string{"dev-1", "test-1"}))
		})
	})

	// ==================== ResolveAndValidateEnvNames ====================
	Describe("ResolveAndValidateEnvNames", func() {
		ctx := context.Background()

		It("validates env names when only envName is provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "prod"},
				{Name: "staging"},
			}, nil).Once()
			names, err := ResolveAndValidateEnvNames(ctx, cli, "app-1", "prod,staging", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(Equal([]string{"prod", "staging"}))
		})

		It("returns error when an env name does not exist", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "prod"},
			}, nil).Once()
			_, err := ResolveAndValidateEnvNames(ctx, cli, "app-1", "prod,nonexistent", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("skips name validation when envType is provided", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "dev-1", Type: "development"},
			}, nil).Once()
			names, err := ResolveAndValidateEnvNames(ctx, cli, "app-1", "", "development")
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(Equal([]string{"dev-1"}))
		})
	})

	// ==================== resolveEnvNamesByTypes ====================
	Describe("resolveEnvNamesByTypes", func() {
		ctx := context.Background()

		It("returns error for an invalid env type", func() {
			cli := mocks.NewMockClient(GinkgoT())
			_, err := resolveEnvNamesByTypes(ctx, cli, "app-1", []string{"prod"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid env type"))
		})

		It("returns error when no envs match the given types", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "prod-1", Type: "production"},
			}, nil).Once()
			_, err := resolveEnvNamesByTypes(ctx, cli, "app-1", []string{"test"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no envs found"))
		})

		It("propagates error when ListAppEnvs fails", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return(nil, context.DeadlineExceeded).Once()
			_, err := resolveEnvNamesByTypes(ctx, cli, "app-1", []string{"test"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to list envs"))
		})

		It("includes feature environments whose type matches", func() {
			cli := mocks.NewMockClient(GinkgoT())
			cli.EXPECT().ListAppEnvs(ctx, "app-1").Return([]client.Env{
				{Name: "staging", Type: "staging", Kind: "standard"},
				{Name: "feat-1", Type: "staging", Kind: "feature", OwnerAppID: "app-1"},
			}, nil).Once()
			names, err := resolveEnvNamesByTypes(ctx, cli, "app-1", []string{"staging"})
			Expect(err).NotTo(HaveOccurred())
			Expect(names).To(Equal([]string{"staging", "feat-1"}))
		})
	})

	// ==================== isValidEnvType ====================
	Describe("isValidEnvType", func() {
		DescribeTable("validates env type strings",
			func(input string, expected bool) {
				Expect(isValidEnvType(input)).To(Equal(expected))
			},
			Entry("development", "development", true),
			Entry("test", "test", true),
			Entry("staging", "staging", true),
			Entry("production", "production", true),
			Entry("invalid shorthand", "prod", false),
			Entry("empty string", "", false),
		)
	})
})
