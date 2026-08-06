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

var _ = Describe("ListVersions", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("lists all versions for the selected config file across pages", func() {
		contentV2 := "server:\n  port: 8080\n"
		contentV1 := "server:\n  port: 8081\n"

		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "").
			Return([]client.AppConfigFile{{
				ID:         "default-file",
				Name:       "default",
				EnvName:    "",
				Type:       "normal",
				FileFormat: "yaml",
			}}, nil)
		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("default-file", nil, 1, client.DefaultListAppConfigFileVersionsPageSize),
			).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 2,
				Results: []client.AppConfigFileVersion{{
					ID:              "version-2",
					AppID:           appID,
					AppConfigFileID: "default-file",
					Name:            "default",
					EnvName:         "",
					Type:            "normal",
					FileFormat:      "yaml",
					Version:         2,
					Creator:         "alice",
					CreatedAt:       "2026-07-01T10:00:00Z",
					Content:         &contentV2,
				}},
			}, nil)
		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("default-file", nil, 2, client.DefaultListAppConfigFileVersionsPageSize),
			).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 2,
				Results: []client.AppConfigFileVersion{{
					ID:              "version-1",
					AppID:           appID,
					AppConfigFileID: "default-file",
					Name:            "default",
					EnvName:         "",
					Type:            "normal",
					FileFormat:      "yaml",
					Version:         1,
					Creator:         "bob",
					CreatedAt:       "2026-06-30T10:00:00Z",
					Content:         &contentV1,
				}},
			}, nil)

		result, err := ListVersions(ctx, cli, appID, "", "")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.File.ID).To(Equal("default-file"))
		Expect(result.EnvName).To(Equal(defaultEnvLabel))
		Expect(result.Versions).To(HaveLen(2))

		listOutput, err := result.Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(listOutput).To(HaveLen(2))
		Expect(listOutput[0].EnvName).To(Equal(defaultEnvLabel))
		Expect(listOutput[0].Content).To(Equal(&contentV2))

		jsonOutput, err := output.FormatData(ctx, listOutput, string(output.FormatJson))
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonOutput).To(ContainSubstring(`"id":"version-2"`))
		Expect(jsonOutput).To(ContainSubstring(`"envName":"\u003cdefault\u003e"`))
	})

	It("returns an error when the selected config file cannot be found", func() {
		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "prod").
			Return(nil, nil)

		result, err := ListVersions(ctx, cli, appID, "prod", "")

		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no app config file found"))
		Expect(err.Error()).To(ContainSubstring("prod"))
	})
})
