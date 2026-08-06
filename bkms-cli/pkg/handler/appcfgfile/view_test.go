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
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

var _ = Describe("View", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	Context("when environment-specific config is disabled", func() {
		It("uses the single default file returned by ListAppConfigFiles", func() {
			content := "server:\n  port: 8080\n"
			files := []client.AppConfigFile{{
				ID:         "default-file",
				Name:       "default",
				EnvName:    "",
				Type:       "normal",
				FileFormat: "yaml",
			}}
			cli.EXPECT().
				ListAppConfigFiles(mock.Anything, appID, "").
				Return(files, nil)
			cli.EXPECT().
				GetAppConfigFileDetails(mock.Anything, appID, "default-file").
				Return(&client.AppConfigFileDetails{
					Content:        &content,
					CurrentVersion: 3,
					Updater:        "alice",
					UpdatedAt:      "2026-06-29T01:02:03Z",
				}, nil)

			result, err := View(ctx, cli, appID, "", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.File.ID).To(Equal("default-file"))
			Expect(result.EnvName).To(Equal(defaultEnvLabel))
			Expect(result.Content).To(Equal(&content))
			Expect(result.OverlayContent).To(BeNil())

			viewOutput, err := result.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(viewOutput.EnvName).To(Equal(defaultEnvLabel))
			Expect(viewOutput.Content).To(Equal(&content))
			Expect(viewOutput.OverlayContent).To(BeNil())
			Expect(viewOutput.CurrentVersion).To(Equal(int64(3)))

			jsonOutput, err := output.FormatData(ctx, viewOutput, string(output.FormatJson))
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonOutput).To(ContainSubstring(`"envName":"\u003cdefault\u003e"`))
		})

		It("returns an error when ListAppConfigFiles returns no default file", func() {
			cli.EXPECT().
				ListAppConfigFiles(mock.Anything, appID, "").
				Return(nil, nil)

			result, err := View(ctx, cli, appID, "", "")

			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no app config file found"))
			Expect(err.Error()).To(ContainSubstring("default"))
		})
	})

	Context("when environment-specific config is enabled", func() {
		It("uses the single env file returned by ListAppConfigFiles", func() {
			overlayContent := "patches:\n- replicas: 5\n"
			files := []client.AppConfigFile{{
				ID:         "prod-file",
				Name:       "default",
				EnvName:    "prod",
				Type:       "overlay",
				FileFormat: "yaml",
			}}
			cli.EXPECT().
				ListAppConfigFiles(mock.Anything, appID, "prod").
				Return(files, nil)
			cli.EXPECT().
				GetAppConfigFileDetails(mock.Anything, appID, "prod-file").
				Return(&client.AppConfigFileDetails{
					OverlayContent: &overlayContent,
					CurrentVersion: 7,
					Updater:        "bob",
					UpdatedAt:      "2026-06-29T02:03:04Z",
				}, nil)

			result, err := View(ctx, cli, appID, "prod", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.File.ID).To(Equal("prod-file"))
			Expect(result.EnvName).To(Equal("prod"))
			Expect(result.Content).To(BeNil())
			Expect(result.OverlayContent).To(Equal(&overlayContent))

			viewOutput, err := result.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(viewOutput.EnvName).To(Equal("prod"))
			Expect(viewOutput.Content).To(BeNil())
			Expect(viewOutput.OverlayContent).To(Equal(&overlayContent))

			jsonOutput, err := output.FormatData(ctx, viewOutput, string(output.FormatJson))
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonOutput).To(ContainSubstring(`"envName":"prod"`))
			Expect(jsonOutput).To(ContainSubstring(`"overlayContent":"patches:\n- replicas: 5\n"`))
			Expect(jsonOutput).NotTo(ContainSubstring(`"content"`))
		})

		It("returns an error when ListAppConfigFiles returns no env file", func() {
			cli.EXPECT().
				ListAppConfigFiles(mock.Anything, appID, "prod").
				Return(nil, nil)

			result, err := View(ctx, cli, appID, "prod", "")

			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no app config file found"))
			Expect(err.Error()).To(ContainSubstring("prod"))
		})
	})
})

var _ = Describe("findCfgFileBy", func() {
	It("selects one file matching the requested env", func() {
		file, err := findCfgFileBy([]client.AppConfigFile{
			{ID: "default-file", Name: "default", EnvName: ""},
			{ID: "prod-file", Name: "default", EnvName: "prod"},
		}, "prod", "")

		Expect(err).NotTo(HaveOccurred())
		Expect(file.ID).To(Equal("prod-file"))
	})

	It("returns an error when no file matches", func() {
		file, err := findCfgFileBy([]client.AppConfigFile{
			{ID: "prod-file", Name: "default", EnvName: "prod"},
		}, "qa", "")

		Expect(file).To(Equal(client.AppConfigFile{}))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no app config file found"))
		Expect(err.Error()).To(ContainSubstring("qa"))
	})

	It("selects one Helm app-level file by name when multiple default-env files exist", func() {
		// Helm apps keep EnvName empty and can own multiple config files, so --name
		// is needed to choose the intended app-level file.
		file, err := findCfgFileBy([]client.AppConfigFile{
			{ID: "values-file", Name: "values", EnvName: ""},
			{ID: "values-prod-file", Name: "values-prod", EnvName: ""},
		}, "", "values-prod")

		Expect(err).NotTo(HaveOccurred())
		Expect(file.ID).To(Equal("values-prod-file"))
	})

	It("returns an error when no file matches the requested name", func() {
		file, err := findCfgFileBy([]client.AppConfigFile{
			{ID: "values-file", Name: "values", EnvName: ""},
		}, "", "values-qa")

		Expect(file).To(Equal(client.AppConfigFile{}))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("values-qa"))
	})

	It("returns an error when multiple files match", func() {
		file, err := findCfgFileBy([]client.AppConfigFile{
			{ID: "prod-file", Name: "default", EnvName: "prod"},
			{ID: "prod-file-2", Name: "default-copy", EnvName: "prod"},
		}, "prod", "")

		Expect(file).To(Equal(client.AppConfigFile{}))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple app config files"))
		Expect(err.Error()).To(ContainSubstring("default(prod-file)"))
		Expect(err.Error()).To(ContainSubstring("default-copy(prod-file-2)"))
	})
})
