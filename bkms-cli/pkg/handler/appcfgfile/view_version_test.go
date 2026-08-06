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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client/mocks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

var _ = Describe("ViewVersion", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("views a history version by version number", func() {
		versionNum := int64(7)
		overlayContent := "replicas: 5\n"

		cli.EXPECT().
			ListAppConfigFiles(mock.Anything, appID, "prod").
			Return([]client.AppConfigFile{{
				ID:         "prod-file",
				Name:       "default",
				EnvName:    "prod",
				Type:       "overlay",
				FileFormat: "yaml",
			}}, nil)
		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("prod-file", &versionNum, 1, 0),
			).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 1,
				Results: []client.AppConfigFileVersion{{
					ID:              "version-record-7",
					AppID:           appID,
					AppConfigFileID: "prod-file",
					Version:         versionNum,
				}},
			}, nil)
		cli.EXPECT().
			GetAppConfigFileVersion(mock.Anything, appID, "version-record-7").
			Return(&client.AppConfigFileVersion{
				ID:              "version-record-7",
				AppID:           appID,
				AppConfigFileID: "prod-file",
				Name:            "default",
				EnvName:         "prod",
				Type:            "overlay",
				FileFormat:      "yaml",
				Version:         versionNum,
				Creator:         "alice",
				CreatedAt:       "2026-07-01T10:00:00Z",
				Description:     "rollback check",
				OverlayContent:  &overlayContent,
			}, nil)

		result, err := ViewVersion(ctx, cli, appID, "prod", "", VersionRefOptions{
			Version: &versionNum,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.File.ID).To(Equal("prod-file"))
		Expect(result.EnvName).To(Equal("prod"))
		Expect(result.Version).NotTo(BeNil())
		Expect(result.Version.OverlayContent).To(Equal(&overlayContent))

		viewOutput, err := result.Output()
		Expect(err).NotTo(HaveOccurred())
		Expect(viewOutput.EnvName).To(Equal("prod"))
		Expect(viewOutput.OverlayContent).To(Equal(&overlayContent))

		jsonOutput, err := output.FormatData(ctx, viewOutput, string(output.FormatJson))
		Expect(err).NotTo(HaveOccurred())
		Expect(jsonOutput).To(ContainSubstring(`"id":"version-record-7"`))
		Expect(jsonOutput).To(ContainSubstring(`"overlayContent":"replicas: 5\n"`))
	})

	It("returns an error when neither version nor versionID is specified", func() {
		result, err := ViewVersion(ctx, cli, appID, "", "", VersionRefOptions{})

		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of versionID or version must be specified"))
	})

	It("returns an error when both version and versionID are specified", func() {
		versionNum := int64(7)

		result, err := ViewVersion(ctx, cli, appID, "", "", VersionRefOptions{
			VersionID: "version-record-7",
			Version:   &versionNum,
		})

		Expect(result).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of versionID or version must be specified"))
	})
})

var _ = Describe("resolveVersionID", func() {
	const appID = "demo"

	var (
		ctx context.Context
		cli *mocks.MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
	})

	It("returns versionID directly when provided", func() {
		versionID, err := resolveVersionID(ctx, cli, appID, "default-file", VersionRefOptions{
			VersionID: "version-record-7",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(versionID).To(Equal("version-record-7"))
	})

	It("finds versionID by version number from filtered versions", func() {
		versionNum := int64(7)

		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("default-file", &versionNum, 1, 0),
			).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 2,
				Results: []client.AppConfigFileVersion{
					{ID: "version-record-8", AppConfigFileID: "default-file", Version: 8},
					{ID: "version-record-7", AppConfigFileID: "default-file", Version: 7},
				},
			}, nil)

		versionID, err := resolveVersionID(ctx, cli, appID, "default-file", VersionRefOptions{
			Version: &versionNum,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(versionID).To(Equal("version-record-7"))
	})

	It("returns an error when version number is not found", func() {
		versionNum := int64(7)

		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("default-file", &versionNum, 1, 0),
			).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 1,
				Results: []client.AppConfigFileVersion{
					{ID: "version-record-8", AppConfigFileID: "default-file", Version: 8},
				},
			}, nil)

		versionID, err := resolveVersionID(ctx, cli, appID, "default-file", VersionRefOptions{
			Version: &versionNum,
		})

		Expect(versionID).To(BeEmpty())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no app config file version found"))
	})

	It("returns an error when listing versions fails", func() {
		versionNum := int64(7)

		cli.EXPECT().
			ListAppConfigFileVersions(
				mock.Anything,
				appID,
				matchListVersionsOptions("default-file", &versionNum, 1, 0),
			).
			Return(nil, errors.New("boom"))

		versionID, err := resolveVersionID(ctx, cli, appID, "default-file", VersionRefOptions{
			Version: &versionNum,
		})

		Expect(versionID).To(BeEmpty())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("list app config file versions"))
	})
})

func matchListVersionsOptions(
	fileID string,
	version *int64,
	page, pageSize int,
) interface{} {
	return mock.MatchedBy(func(opts client.ListAppConfigFileVersionsOptions) bool {
		if opts.AppConfigFileID != fileID || opts.Page != page || opts.PageSize != pageSize {
			return false
		}
		if version == nil {
			return opts.Version == nil
		}
		return opts.Version != nil && *opts.Version == *version
	})
}
