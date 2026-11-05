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
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("View", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("shows the default file without --env", func() {
		def := plainDef("d1", "a.conf", "/etc")
		expectDefs(cli, def)
		detail := def
		detail.Content = lo.ToPtr("k=v\n")
		detail.EditableContentField = "content"
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "").Return(&detail, nil)

		result, err := View(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.SharesDefault()).To(BeFalse())
		Expect(result.EditableContent()).To(Equal("k=v\n"))
		out := result.Output()
		Expect(out.EnvName).To(Equal(defaultEnvLabel))
		Expect(out.MountEnvs).To(Equal(allEnvsLabel))
		Expect(out.EnvMode).To(Equal(envModeUnified))
	})

	It("marks an environment without its own content as sharing the default", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "prod")
		detail := def
		detail.HasEnvInstance = lo.ToPtr(false)
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "prod").Return(&detail, nil)

		result, err := View(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.SharesDefault()).To(BeTrue())
		Expect(result.Output().EnvMode).To(Equal(envModePerEnv))
	})

	It("returns the overlay content as editable content for overlay files", func() {
		def := client.AppConfigFileDef{ID: "d1", Name: "trpc_go.yaml", ConfigKind: client.ConfigKindFramework}
		expectDefs(cli, def)
		detail := def
		detail.Content = lo.ToPtr("base: 1\n")
		detail.OverlayContent = lo.ToPtr("patch: 1\n")
		detail.EditableContentField = editableContentFieldOverlayContent
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "").Return(&detail, nil)

		result, err := View(ctx, cli, testAppID, FileRef{ID: "d1"}, "")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.EditableContent()).To(Equal("patch: 1\n"))
	})
})
