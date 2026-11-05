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

package cfgfile

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("PlanUpdate", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("lists all env instances when switching back to unified config", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
		expectEnvInstances(cli, "d1", "dev", "prod")

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{
			IsUnifiedConfig: lo.ToPtr(true),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(*plan.Body.IsUnifiedConfig).To(BeTrue())
		Expect(plan.RemovedEnvInstances).To(ConsistOf("dev", "prod"))
	})

	It("skips an env mode that is already in effect", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{
			IsUnifiedConfig: lo.ToPtr(true),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(plan.HasChanges()).To(BeFalse())
	})

	It("lists env instances outside the new mount envs", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
		expectEnvs(cli, "dev", "test", "prod")
		expectEnvInstances(cli, "d1", "dev", "prod")

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{
			MountEnvNames: []string{"dev", " test ", "dev"},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(*plan.Body.MountedEnvNames).To(Equal([]string{"dev", "test"}))
		Expect(plan.RemovedEnvInstances).To(Equal([]string{"prod"}))
	})

	It("rejects unknown mount envs", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))
		expectEnvs(cli, "dev")

		_, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{
			MountEnvNames: []string{"dev", "prd"},
		})

		Expect(err).To(MatchError(ContainSubstring("envs prd not found")))
	})

	It("keeps a file already mounted on all envs untouched for --mount-all-envs", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{MountAllEnvs: true})

		Expect(err).NotTo(HaveOccurred())
		Expect(plan.HasChanges()).To(BeFalse())
	})

	It("mounts on all current envs for --mount-all-envs", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.MountedEnvNames = []string{"dev"}
		expectDefs(cli, def)
		expectEnvs(cli, "dev", "prod")
		expectEnvInstances(cli, "d1")

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{MountAllEnvs: true})

		Expect(err).NotTo(HaveOccurred())
		Expect(*plan.Body.MountedEnvNames).To(Equal([]string{"dev", "prod"}))
		Expect(plan.RemovedEnvInstances).To(BeEmpty())
	})

	It("does not query env instances for non-destructive changes", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))

		plan, err := PlanUpdate(ctx, cli, testAppID, FileRef{Name: "a.conf"}, UpdateOptions{
			NewName:            lo.ToPtr("b.conf"),
			EnableEnvVarRender: lo.ToPtr(true),
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(plan.HasChanges()).To(BeTrue())
		Expect(plan.RemovedEnvInstances).To(BeEmpty())
	})
})

var _ = Describe("PlanReset", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("rejects unified files", func() {
		expectDefs(cli, plainDef("d1", "a.conf", "/etc"))
		expectAppType(cli, "trpc")
		expectEnvs(cli, "prod")

		_, err := PlanReset(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod")

		Expect(err).To(MatchError(ContainSubstring("unified config")))
	})

	It("reports whether the env has its own content", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "dev", "prod")
		expectEnvInstances(cli, "d1", "dev")

		plan, err := PlanReset(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(plan.HasEnvInstance).To(BeFalse())
	})
})
