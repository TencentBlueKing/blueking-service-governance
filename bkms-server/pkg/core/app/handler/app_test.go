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

package handler

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("newAppIDSuffix", func() {
	It("should return a valid suffix", func() {
		suffix := newAppIDSuffix()
		Expect(suffix[0]).To(Equal(byte('-')))
		Expect(len(suffix) <= 7).To(BeTrue())
	})
})

var _ = Describe("collectBKCIRepositoriesForCreateApp", func() {
	It("collects both build and helm git repositories for helm-based apps", func() {
		buildConfig := &build.Config{
			SourceType: build.SourceTypeCodeRepository,
			CodeRepo: &build.RepositoryConfig{
				RepoURL:   "https://git.example.com/build.git",
				RepoAlias: "build-repo",
			},
		}
		helmSpec := &bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				RepoType: bkmsapp.HelmSourceRepoTypeGit,
				GitRepoConfig: &bkmsapp.GitRepoConfig{
					RepoURL:   "https://git.example.com/chart.git",
					RepoAlias: "chart-repo",
				},
			},
		}

		repos := collectBKCIRepositoriesForCreateApp(bkmsapp.AppTypeAgones, helmSpec, buildConfig)

		Expect(repos).To(Equal([]bkci.RepositoryInitSpec{
			{URL: "https://git.example.com/build.git", Alias: "build-repo"},
			{URL: "https://git.example.com/chart.git", Alias: "chart-repo"},
		}))
	})

	It("ignores helm repositories for non-helm apps", func() {
		helmSpec := &bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				RepoType: bkmsapp.HelmSourceRepoTypeGit,
				GitRepoConfig: &bkmsapp.GitRepoConfig{
					RepoURL:   "https://git.example.com/chart.git",
					RepoAlias: "chart-repo",
				},
			},
		}

		repos := collectBKCIRepositoriesForCreateApp(bkmsapp.AppTypeTRPC, helmSpec, nil)

		Expect(repos).To(BeNil())
	})
})

var _ = Describe("validateVisibleEnvs", func() {
	var envs []envmodel.Environment

	BeforeEach(func() {
		parentID := bson.NewObjectID()
		envs = []envmodel.Environment{
			{Name: "deployed", Kind: envmodel.EnvironmentKindStandard, AppIDs: []string{"app-a"}},
			{ID: parentID, Name: "parent", Kind: envmodel.EnvironmentKindStandard},
			{Name: "optional", Kind: envmodel.EnvironmentKindStandard},
			{Name: "other-app", Kind: envmodel.EnvironmentKindStandard, AppIDs: []string{"app-b"}},
			{
				Name:        "feature",
				Kind:        envmodel.EnvironmentKindFeature,
				OwnerAppID:  "app-a",
				SourceEnvID: parentID,
			},
		}
	})

	DescribeTable("rejects hiding required environments or selecting invalid names",
		func(names []string, message string) {
			err := validateVisibleEnvs("app-a", names, envs)

			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(err.Error()).To(ContainSubstring(message))
		},
		Entry("an environment with an app deployment", []string{"parent"},
			`environment "deployed" has a deployment and must remain visible`),
		Entry("a feature environment parent without a deployment", []string{"deployed"},
			`environment "parent" is the parent of a feature environment and must remain visible`),
		Entry("an unknown environment name", []string{"deployed", "parent", "missing"},
			"invalid visible environment name(s): missing"),
		Entry("a feature environment name", []string{"deployed", "parent", "feature"},
			"invalid visible environment name(s): feature"),
	)

	It("allows hiding unused environments and environments deployed by another app", func() {
		Expect(validateVisibleEnvs("app-a", []string{"deployed", "parent"}, envs)).To(Succeed())
	})

	It("allows clearing the list because all environments remain visible", func() {
		Expect(validateVisibleEnvs("app-a", []string{}, envs)).To(Succeed())
	})

	It("allows hiding environments after uninstall and feature environment deletion", func() {
		envs[0].AppIDs = nil
		envs = envs[:4]

		Expect(validateVisibleEnvs("app-a", []string{"optional"}, envs)).To(Succeed())
	})
})
