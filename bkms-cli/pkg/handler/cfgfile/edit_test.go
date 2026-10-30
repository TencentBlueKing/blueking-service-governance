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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("Edit", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("writes the default file with optimistic lock", func() {
		def := plainDef("d1", "a.conf", "/etc")
		expectDefs(cli, def)
		detail := def
		detail.CurrentVersion = 4
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "").Return(&detail, nil)
		cli.EXPECT().
			UpdateAppConfigFileDefContent(mock.Anything, testAppID, "d1", "", client.UpdateAppConfigFileDefContentBody{
				Content: "k=v\n", Description: "update", CurrentVersion: lo.ToPtr(int64(4)),
			}).
			Return(&client.AppConfigFileDefContentUpdateResult{FileID: "d1-default", CurrentVersion: 5}, nil)

		result, err := Edit(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "", EditOptions{
			Content: "k=v\n", Description: "update",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.SwitchedToPerEnv).To(BeFalse())
		Expect(result.CreatedEnvInstance).To(BeFalse())
		Expect(result.UpdateResult.CurrentVersion).To(Equal(int64(5)))
	})

	It("switches a unified file to per-env before writing one environment", func() {
		def := plainDef("d1", "a.conf", "/etc")
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "dev", "prod")
		detail := def
		detail.HasEnvInstance = lo.ToPtr(false)
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "prod").Return(&detail, nil)
		cli.EXPECT().
			UpdateAppConfigFileDef(mock.Anything, testAppID, "d1", client.UpdateAppConfigFileDefBody{
				IsUnifiedConfig: lo.ToPtr(false),
			}).
			Return(&def, nil)
		cli.EXPECT().
			UpdateAppConfigFileDefContent(mock.Anything, testAppID, "d1", "prod", client.UpdateAppConfigFileDefContentBody{
				Content: "k=prod\n",
			}).
			Return(&client.AppConfigFileDefContentUpdateResult{FileID: "d1-prod", CurrentVersion: 1}, nil)

		result, err := Edit(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", EditOptions{Content: "k=prod\n"})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.SwitchedToPerEnv).To(BeTrue())
		Expect(result.CreatedEnvInstance).To(BeTrue())
	})

	It("updates an existing environment instance with its own version", func() {
		def := plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "prod")
		detail := def
		detail.FileID = "d1-prod"
		detail.CurrentVersion = 3
		detail.HasEnvInstance = lo.ToPtr(true)
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "prod").Return(&detail, nil)
		cli.EXPECT().
			UpdateAppConfigFileDefContent(mock.Anything, testAppID, "d1", "prod", client.UpdateAppConfigFileDefContentBody{
				Content: "k=prod\n", CurrentVersion: lo.ToPtr(int64(3)),
			}).
			Return(&client.AppConfigFileDefContentUpdateResult{FileID: "d1-prod", CurrentVersion: 4}, nil)

		result, err := Edit(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", EditOptions{Content: "k=prod\n"})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.SwitchedToPerEnv).To(BeFalse())
		Expect(result.CreatedEnvInstance).To(BeFalse())
	})

	It("still reports the mode switch when writing content fails", func() {
		def := plainDef("d1", "a.conf", "/etc")
		expectDefs(cli, def)
		expectAppType(cli, "trpc")
		expectEnvs(cli, "prod")
		detail := def
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "prod").Return(&detail, nil)
		cli.EXPECT().UpdateAppConfigFileDef(mock.Anything, testAppID, "d1", mock.Anything).Return(&def, nil)
		cli.EXPECT().
			UpdateAppConfigFileDefContent(mock.Anything, testAppID, "d1", "prod", mock.Anything).
			Return(nil, errors.New("boom"))

		result, err := Edit(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", EditOptions{Content: "x"})

		Expect(err).To(MatchError(ContainSubstring("boom")))
		Expect(result).NotTo(BeNil())
		Expect(result.SwitchedToPerEnv).To(BeTrue())
	})
})
