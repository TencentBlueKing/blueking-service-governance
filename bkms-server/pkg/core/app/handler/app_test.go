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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
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
