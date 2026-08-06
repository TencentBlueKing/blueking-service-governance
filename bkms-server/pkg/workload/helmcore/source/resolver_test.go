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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
)

// mockBkciProjectStore 是 bkci.ProjectStore 的 mock 实现
type mockBkciProjectStore struct {
	getByWorkspaceFunc func(ctx context.Context, workspaceID string) (*bkci.Project, error)
}

func (m *mockBkciProjectStore) Create(_ context.Context, _ *bkci.Project) error { return nil }

func (m *mockBkciProjectStore) GetByWorkspace(ctx context.Context, workspaceID string) (*bkci.Project, error) {
	if m.getByWorkspaceFunc != nil {
		return m.getByWorkspaceFunc(ctx, workspaceID)
	}
	return nil, bkci.ErrProjectNotFound
}

// mockBkrepoProjectStore 是 bkrepo.ProjectStore 的 mock 实现
type mockBkrepoProjectStore struct {
	getByWorkspaceFunc func(ctx context.Context, workspaceID string) (*bkrepo.Project, error)
}

func (m *mockBkrepoProjectStore) Create(_ context.Context, _ *bkrepo.Project) error { return nil }
func (m *mockBkrepoProjectStore) Get(_ context.Context, _ string) (*bkrepo.Project, error) {
	return nil, bkrepo.ErrProjectNotFound
}

func (m *mockBkrepoProjectStore) GetByWorkspace(ctx context.Context, workspaceID string) (*bkrepo.Project, error) {
	if m.getByWorkspaceFunc != nil {
		return m.getByWorkspaceFunc(ctx, workspaceID)
	}
	return nil, bkrepo.ErrProjectNotFound
}

// mockHelmRepoCredentialStore 是 credential.HelmRepoCredentialStore 的 mock 实现
type mockHelmRepoCredentialStore struct {
	getByWorkspaceFunc func(ctx context.Context, workspaceID string) (*credential.HelmRepoCredential, error)
	createFunc         func(ctx context.Context, cred *credential.HelmRepoCredential) error
}

func (m *mockHelmRepoCredentialStore) GetByWorkspace(
	ctx context.Context,
	workspaceID string,
) (*credential.HelmRepoCredential, error) {
	if m.getByWorkspaceFunc != nil {
		return m.getByWorkspaceFunc(ctx, workspaceID)
	}
	return nil, credential.ErrHelmRepoCredentialNotFound
}

func (m *mockHelmRepoCredentialStore) Create(ctx context.Context, cred *credential.HelmRepoCredential) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, cred)
	}
	return nil
}

var _ = Describe("ResolveConfig", func() {
	var (
		ctx                context.Context
		bkciProjectStore   *mockBkciProjectStore
		bkrepoProjectStore *mockBkrepoProjectStore
		credentialStore    *mockHelmRepoCredentialStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		bkciProjectStore = &mockBkciProjectStore{}
		bkrepoProjectStore = &mockBkrepoProjectStore{}
		credentialStore = &mockHelmRepoCredentialStore{}
	})

	It("should return HelmRepoConfig directly for HelmRepo type", func() {
		app := &bkmsapp.Application{
			WorkspaceID: "ws-1",
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

		result, err := ResolveConfig(
			ctx,
			bkciProjectStore,
			bkrepoProjectStore,
			credentialStore,
			app,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(app.HelmSpec.HelmSource.HelmRepoConfig))
	})

	It("should resolve config from credential store for Git type", func() {
		bkciProjectStore.getByWorkspaceFunc = func(_ context.Context, _ string) (*bkci.Project, error) {
			return &bkci.Project{Code: "test-project"}, nil
		}
		bkrepoProjectStore.getByWorkspaceFunc = func(_ context.Context, _ string) (*bkrepo.Project, error) {
			return &bkrepo.Project{Username: "repo-user", Password: "repo-pass"}, nil
		}
		// nosec G101
		credentialStore.getByWorkspaceFunc = func(_ context.Context, _ string) (*credential.HelmRepoCredential, error) {
			return &credential.HelmRepoCredential{
				WorkspaceID:  "ws-1",
				CredentialID: "bkms_helm_repo_credential",
				Username:     "repo-user",
				Password:     "repo-pass",
			}, nil
		}
		app := &bkmsapp.Application{
			WorkspaceID: "ws-1",
			Name:        "my-app",
			Type:        bkmsapp.AppTypeHelm,
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{
					RepoType: bkmsapp.HelmSourceRepoTypeGit,
				},
			},
		}

		mockey.PatchConvey("mock GenRepoEndpoint", GinkgoT(), func() {
			mockey.Mock((*config.HelmConfig).GenBuiltinRepoURL).
				Return("http://bkrepo.example.com/helm/test-project/helm", nil).Build()

			result, err := ResolveConfig(
				ctx,
				bkciProjectStore,
				bkrepoProjectStore,
				credentialStore,
				app,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RepoURL).To(Equal("http://bkrepo.example.com/helm/test-project/helm"))
			Expect(result.ChartName).To(Equal("my-app"))
			Expect(result.Username).To(Equal("repo-user"))
			Expect(result.Password).To(Equal("repo-pass"))
		})
	})

	It("should return error when store fails", func() {
		bkciProjectStore.getByWorkspaceFunc = func(_ context.Context, _ string) (*bkci.Project, error) {
			return nil, bkci.ErrProjectNotFound
		}
		app := &bkmsapp.Application{
			WorkspaceID: "ws-1",
			Type:        bkmsapp.AppTypeHelm,
			HelmSpec: &bkmsapp.HelmSpec{
				HelmSource: &bkmsapp.HelmSource{RepoType: bkmsapp.HelmSourceRepoTypeGit},
			},
		}

		_, err := ResolveConfig(ctx, bkciProjectStore, bkrepoProjectStore, credentialStore, app)
		Expect(err).To(HaveOccurred())
	})
})
