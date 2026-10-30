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

var _ = Describe("versions", func() {
	var (
		ctx context.Context
		cli *mocks.MockClient
		def client.AppConfigFileDef
	)

	// expectEnvDetail 准备 prod 环境的详情：hasInstance 决定生效文件是环境实例还是默认文件。
	expectEnvDetail := func(hasInstance bool, currentVersion int64) {
		expectAppType(cli, "trpc")
		expectEnvs(cli, "prod")
		detail := def
		detail.HasEnvInstance = lo.ToPtr(hasInstance)
		detail.CurrentVersion = currentVersion
		if hasInstance {
			detail.FileID = "d1-prod"
		}
		cli.EXPECT().GetAppConfigFileDef(mock.Anything, testAppID, "d1", "prod").Return(&detail, nil)
	}

	BeforeEach(func() {
		ctx = context.Background()
		cli = mocks.NewMockClient(GinkgoT())
		def = plainDef("d1", "a.conf", "/etc")
		def.IsUnifiedConfig = false
		expectDefs(cli, def)
	})

	It("lists versions of the env instance across pages", func() {
		expectEnvDetail(true, 3)
		cli.EXPECT().
			ListAppConfigFileVersions(mock.Anything, testAppID, mock.MatchedBy(
				func(opts client.ListAppConfigFileVersionsOptions) bool {
					return opts.AppConfigFileID == "d1-prod" && opts.Page == 1
				},
			)).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 3, Results: []client.AppConfigFileVersion{{Version: 3}, {Version: 2}},
			}, nil)
		cli.EXPECT().
			ListAppConfigFileVersions(mock.Anything, testAppID, mock.MatchedBy(
				func(opts client.ListAppConfigFileVersionsOptions) bool { return opts.Page == 2 },
			)).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 3, Results: []client.AppConfigFileVersion{{Version: 1}},
			}, nil)

		result, err := ListVersions(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod")

		Expect(err).NotTo(HaveOccurred())
		out := result.Output()
		Expect(out).To(HaveLen(3))
		Expect(out[0].IsCurrent).To(BeTrue())
		Expect(out[1].IsCurrent).To(BeFalse())
	})

	It("refuses to roll back an env that shares the default file", func() {
		expectEnvDetail(false, 5)

		_, err := RollbackVersion(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod",
			VersionRef{Version: lo.ToPtr(int64(2))}, "")

		Expect(err).To(MatchError(ContainSubstring("has no independent config")))
	})

	It("rolls back by version number with optimistic lock", func() {
		expectEnvDetail(true, 5)
		cli.EXPECT().
			ListAppConfigFileVersions(mock.Anything, testAppID, client.ListAppConfigFileVersionsOptions{
				AppConfigFileID: "d1-prod", Version: lo.ToPtr(int64(2)),
			}).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 1, Results: []client.AppConfigFileVersion{{ID: "v2", Version: 2, AppConfigFileID: "d1-prod"}},
			}, nil)
		cli.EXPECT().
			RollbackAppConfigFileVersion(mock.Anything, testAppID, "v2", client.RollbackAppConfigFileVersionOptions{
				CurrentVersion: lo.ToPtr(int64(5)), Description: "rb",
			}).
			Return(&client.RollbackAppConfigFileVersionResult{ID: "d1-prod", CurrentVersion: 6}, nil)

		result, err := RollbackVersion(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod",
			VersionRef{Version: lo.ToPtr(int64(2))}, "rb")

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Result.CurrentVersion).To(Equal(int64(6)))
	})

	It("rejects a version ID that belongs to another file", func() {
		expectEnvDetail(true, 5)
		cli.EXPECT().
			GetAppConfigFileVersion(mock.Anything, testAppID, "v9").
			Return(&client.AppConfigFileVersion{ID: "v9", AppConfigFileID: "other"}, nil)

		_, err := ViewVersion(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", VersionRef{VersionID: "v9"})

		Expect(err).To(MatchError(ContainSubstring("does not belong")))
	})

	It("refuses to delete the current version", func() {
		expectEnvDetail(true, 5)
		cli.EXPECT().
			GetAppConfigFileVersion(mock.Anything, testAppID, "v5").
			Return(&client.AppConfigFileVersion{ID: "v5", Version: 5, AppConfigFileID: "d1-prod"}, nil)

		_, err := DeleteVersion(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", VersionRef{VersionID: "v5"})

		Expect(err).To(MatchError(ContainSubstring("current version")))
	})

	It("diffs a version against the current version", func() {
		expectEnvDetail(true, 3)
		cli.EXPECT().
			ListAppConfigFileVersions(mock.Anything, testAppID, mock.MatchedBy(
				func(opts client.ListAppConfigFileVersionsOptions) bool { return *opts.Version == 1 },
			)).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 1, Results: []client.AppConfigFileVersion{{ID: "v1", Version: 1}},
			}, nil)
		cli.EXPECT().
			ListAppConfigFileVersions(mock.Anything, testAppID, mock.MatchedBy(
				func(opts client.ListAppConfigFileVersionsOptions) bool { return *opts.Version == 3 },
			)).
			Return(&client.PaginatedAppConfigFileVersions{
				Count: 1, Results: []client.AppConfigFileVersion{{ID: "v3", Version: 3}},
			}, nil)
		cli.EXPECT().
			CompareAppConfigFileVersions(mock.Anything, testAppID, "v1", "v3").
			Return(&client.AppConfigFileVersionComparison{
				Previous: &client.AppConfigFileVersion{Version: 1, Content: lo.ToPtr("a=1\nb=2\n")},
				Current:  &client.AppConfigFileVersion{Version: 3, Content: lo.ToPtr("a=1\nb=3\n")},
			}, nil)

		result, err := DiffVersions(ctx, cli, testAppID, FileRef{Name: "a.conf"}, "prod", 1, nil)
		Expect(err).NotTo(HaveOccurred())
		diff, err := result.UnifiedDiff()

		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(ContainSubstring("--- v1/content"))
		Expect(diff).To(ContainSubstring("+++ v3/content"))
		Expect(diff).To(ContainSubstring("-b=2"))
		Expect(diff).To(ContainSubstring("+b=3"))
		Expect(diff).NotTo(ContainSubstring("overlayContent"))
	})
})
