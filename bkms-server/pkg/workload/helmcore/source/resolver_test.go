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

package repo

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
)

var _ = Describe("ResolveConfig", func() {
	var (
		ctx                context.Context
		workspaceID        string
		db                 *mongo.Database
		bkciProjectStore   *bkci.ProjectStore
		bkrepoProjectStore *bkrepo.ProjectStore
		credentialStore    *credential.Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "ws-" + stringx.Random(8)
		db = testutil.NewDB(GinkgoT())
		bkciProjectStore = store.Of(db, bkci.ProjectMeta)
		bkrepoProjectStore = store.Of(db, bkrepo.ProjectMeta)
		credentialStore = store.Of(db, credential.Meta)
	})

	newGitApp := func() *bkmsapp.Application {
		return &bkmsapp.Application{
			WorkspaceID: workspaceID,
			Name:        "my-app",
			Type:        bkmsapp.AppTypeHelm,
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{RepoType: bkmsapp.HelmSourceRepoTypeGit},
			},
		}
	}

	It("should return HelmRepoConfig directly for HelmRepo type", func() {
		app := &bkmsapp.Application{
			WorkspaceID: workspaceID,
			Name:        "my-app",
			Type:        bkmsapp.AppTypeHelm,
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{
					RepoType: bkmsapp.HelmSourceRepoTypeHelm,
					HelmRepoConfig: &bkmsapp.HelmRepoConfig{
						RepoURL:   "https://helm.example.com/charts",
						ChartName: "my-chart",
						Username:  "helm-user",
						Password:  "helm-pass",
					},
				},
			},
		}

		result, err := ResolveConfig(ctx, db, app)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(app.HelmSpec.HelmSource.HelmRepoConfig))
	})

	It("should resolve config from credential store for Git type", func() {
		projectCode := "proj-" + stringx.Random(8)
		Expect(bkciProjectStore.Create(ctx, &bkci.Project{
			ID: stringx.Random(32), Code: projectCode, WorkspaceID: workspaceID,
		})).To(Succeed())
		Expect(bkrepoProjectStore.Create(ctx, &bkrepo.Project{
			ID: projectCode, WorkspaceID: workspaceID, Username: "repo-user", Password: "repo-pass",
		})).To(Succeed())
		// 凭证已存在时 EnsureCredential 直接跳过，不会调用蓝盾 API
		Expect(credentialStore.Create(ctx, &credential.HelmRepoCredential{
			WorkspaceID:  workspaceID,
			CredentialID: "bkms_helm_repo_credential",
			Username:     "repo-user",
			Password:     "repo-pass",
		})).To(Succeed())

		mockey.PatchConvey("mock GenBuiltinRepoURL", GinkgoT(), func() {
			mockey.Mock((*config.HelmConfig).GenBuiltinRepoURL).
				Return("http://bkrepo.example.com/helm/"+projectCode+"/helm", nil).Build()

			result, err := ResolveConfig(ctx, db, newGitApp())
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RepoURL).To(Equal("http://bkrepo.example.com/helm/" + projectCode + "/helm"))
			Expect(result.ChartName).To(Equal("my-app"))
			Expect(result.Username).To(Equal("repo-user"))
			Expect(result.Password).To(Equal("repo-pass"))
		})
	})

	It("should return ErrProjectNotFound when the workspace has no bkci project", func() {
		_, err := ResolveConfig(ctx, db, newGitApp())
		Expect(errors.Is(err, bkci.ErrProjectNotFound)).To(BeTrue())
	})
})
