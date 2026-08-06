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

package appcfgfile

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
)

var _ = Describe("Edit", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("updates Content for the default normal config file", func() {
		newContent := "server:\n  port: 8081\n"
		compiledContent := "server:\n  port: 8081\n"
		files := []client.AppConfigFile{{
			ID:      "default-file",
			Name:    "default",
			EnvName: "",
			Type:    appConfigFileTypeNormal,
		}}
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "").
			Return(files, nil)
		cli.EXPECT().
			GetAppConfigFileDetails(mock.Anything, appID, "default-file").
			Return(&client.AppConfigFileDetails{
				EditableContentField: editableContentFieldContent,
				CurrentVersion:       3,
			}, nil)
		cli.EXPECT().
			UpdateAppConfigFileContent(
				mock.Anything,
				appID,
				"default-file",
				matchContentOptions(newContent, "update default", 3),
			).
			Return(&client.AppConfigFileContentUpdateResult{CompiledContent: compiledContent}, nil)

		result, err := Edit(ctx, cli, appID, "", "", EditOptions{
			Content:     newContent,
			Description: "update default",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.File.ID).To(Equal("default-file"))
		Expect(result.EnvName).To(Equal(defaultEnvLabel))
		Expect(result.Details.CurrentVersion).To(Equal(int64(3)))
		Expect(result.UpdateResult.CompiledContent).To(Equal(compiledContent))
	})

	It("updates OverlayContent for an environment overlay config file", func() {
		newContent := "replicas: 5\n"
		compiledContent := "server:\n  replicas: 5\n"
		files := []client.AppConfigFile{{
			ID:      "prod-file",
			Name:    "default",
			EnvName: "prod",
			Type:    appConfigFileTypeOverlay,
		}}
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "prod").
			Return(files, nil)
		cli.EXPECT().
			GetAppConfigFileDetails(mock.Anything, appID, "prod-file").
			Return(&client.AppConfigFileDetails{
				EditableContentField: editableContentFieldOverlayContent,
				CurrentVersion:       7,
			}, nil)
		cli.EXPECT().
			UpdateAppConfigFileOverlayContent(
				mock.Anything,
				appID,
				"prod-file",
				matchContentOptions(newContent, "update prod", 7),
			).
			Return(&client.AppConfigFileContentUpdateResult{CompiledContent: compiledContent}, nil)

		result, err := Edit(ctx, cli, appID, "prod", "", EditOptions{
			Content:     newContent,
			Description: "update prod",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.File.ID).To(Equal("prod-file"))
		Expect(result.EnvName).To(Equal("prod"))
		Expect(result.Details.CurrentVersion).To(Equal(int64(7)))
		Expect(result.UpdateResult.CompiledContent).To(Equal(compiledContent))
	})

	It("updates OverlayContent when details marks a normal BSCP file overlay-editable", func() {
		newContent := "patches:\n- replicas: 5\n"
		compiledContent := "server:\n  replicas: 5\n"
		files := []client.AppConfigFile{{
			ID:                "bscp-file",
			Name:              "default",
			EnvName:           "",
			Type:              appConfigFileTypeNormal,
			ContentSourceType: "bscp",
		}}
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "").
			Return(files, nil)
		cli.EXPECT().
			GetAppConfigFileDetails(mock.Anything, appID, "bscp-file").
			Return(&client.AppConfigFileDetails{
				EditableContentField: editableContentFieldOverlayContent,
				CurrentVersion:       9,
			}, nil)
		cli.EXPECT().
			UpdateAppConfigFileOverlayContent(
				mock.Anything,
				appID,
				"bscp-file",
				matchContentOptions(newContent, "update bscp overlay", 9),
			).
			Return(&client.AppConfigFileContentUpdateResult{CompiledContent: compiledContent}, nil)

		result, err := Edit(ctx, cli, appID, "", "", EditOptions{
			Content:     newContent,
			Description: "update bscp overlay",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.File.ID).To(Equal("bscp-file"))
		Expect(result.UpdateResult.CompiledContent).To(Equal(compiledContent))
	})

	It("returns an error when details marks the selected file not editable", func() {
		files := []client.AppConfigFile{{ID: "readonly-file", Name: "default", EnvName: ""}}
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "").
			Return(files, nil)
		cli.EXPECT().
			GetAppConfigFileDetails(mock.Anything, appID, "readonly-file").
			Return(&client.AppConfigFileDetails{
				EditableContentField: editableContentFieldNone,
				CurrentVersion:       1,
			}, nil)

		result, err := Edit(ctx, cli, appID, "", "", EditOptions{Content: "replicas: 5\n"})

		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not editable"))
	})

	It("returns an error when no config file matches the requested environment", func() {
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "prod").
			Return(nil, nil)

		result, err := Edit(ctx, cli, appID, "prod", "", EditOptions{Content: "replicas: 5\n"})

		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no app config file found"))
		Expect(err.Error()).To(ContainSubstring("prod"))
	})
})

func matchContentOptions(content, description string, currentVersion int64) interface{} {
	return mock.MatchedBy(func(opts client.AppConfigFileContentOptions) bool {
		return opts.Content == content &&
			opts.Description == description &&
			lo.FromPtr(opts.CurrentVersion) == currentVersion
	})
}
