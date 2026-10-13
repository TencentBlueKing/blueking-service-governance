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

package bkrepo

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkrepo"
	helmchartcred "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// newTestManager 使用随机工作空间创建 ProjectManager，避免不同用例的数据触发唯一索引冲突
func newTestManager() *ProjectManager {
	return NewProjectManager(testutil.NewDB(GinkgoT()), "ws-"+stringx.Random(8), "test-user")
}

// overrideConfig 在全局配置（保留加密密钥等）的副本上修改部分字段，用例结束后自动恢复
func overrideConfig(mutate func(cfg *config.Config)) {
	origin := config.G
	cfg := *origin
	mutate(&cfg)
	config.G = &cfg
	DeferCleanup(func() { config.G = origin })
}

// seedBkciProject 写入工作空间关联的蓝盾项目
func seedBkciProject(ctx context.Context, workspaceID string) *bkci.Project {
	proj := &bkci.Project{
		ID:          stringx.Random(32),
		Code:        "proj-" + stringx.Random(8),
		WorkspaceID: workspaceID,
		Creator:     "test-user",
	}
	Expect(testutil.NewStore(GinkgoT(), bkci.ProjectMeta).Create(ctx, proj)).To(Succeed())
	return proj
}

// seedBkrepoProject 写入工作空间关联的制品库项目
func seedBkrepoProject(ctx context.Context, m *ProjectManager, projectID string) {
	Expect(testutil.NewStore(GinkgoT(), ProjectMeta).Create(ctx, &Project{
		ID:          projectID,
		WorkspaceID: m.workspaceID,
		Username:    m.username,
		Password:    m.password,
		Creator:     m.operator,
	})).To(Succeed())
}

// seedBkrepoRepository 写入工作空间关联的制品库仓库
func seedBkrepoRepository(ctx context.Context, workspaceID, projectID string, repoType RepoType) {
	Expect(testutil.NewStore(GinkgoT(), RepositoryMeta).Create(ctx, &Repository{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        string(repoType),
		Type:        repoType,
		Creator:     "test-user",
	})).To(Succeed())
}

// dockerInitRepos 只包含 DOCKER 仓库的初始化配置
func dockerInitRepos() []config.InitBKRepoConfig {
	return []config.InitBKRepoConfig{
		{
			Name:         "docker",
			Type:         "DOCKER",
			IsPublic:     false,
			Description:  "Docker repository",
			EndpointTmpl: "docker.example.com/%s/%s",
		},
	}
}

// mockImageRegistry creates a mock image registry
func mockImageRegistry() *registry.ImageRegistry {
	// nosec G101
	return &registry.ImageRegistry{
		WorkspaceID:      "test-workspace",
		Type:             registry.ImageRegistryTypeBuiltin,
		Registry:         "docker.example.com",
		Username:         "g_bkms_test_workspace",
		Password:         "test-password-32-chars-long-123",
		BkCICredentialID: "test-credential-id",
	}
}

// mockExistingImageRegistry 镜像源 Store 不在本次 GenericStore 改造范围内，沿用原有的 mock 方式
func mockExistingImageRegistry() {
	mockStore := &registry.ImageRegistryStoreMongo{}
	mockey.Mock(registry.NewImageRegistryStoreMongo).Return(mockStore, nil).Build()
	mockey.Mock(
		(*registry.ImageRegistryStoreMongo).GetByWorkspaceAndType,
	).Return(mockImageRegistry(), nil).Build()
}

var _ = Describe("NewProjectManager", func() {
	It("should create valid ProjectManager instances with correct fields", func() {
		// Test basic instance creation and field assignment
		workspaceID := "test-workspace"
		manager := NewProjectManager(testutil.NewDB(GinkgoT()), workspaceID, "admin")
		Expect(manager).NotTo(BeNil())
		Expect(manager.workspaceID).To(Equal(workspaceID))
		Expect(manager.operator).To(Equal("admin"))
	})

	DescribeTable("should generate correct username format",
		func(workspaceID, expectedUsername string) {
			m := NewProjectManager(testutil.NewDB(GinkgoT()), workspaceID, "")
			Expect(m.username).To(Equal(expectedUsername))
		},
		Entry("workspace with single dash", "test-workspace", "g_bkms_test_workspace"),
		Entry("workspace with numbers", "my-project-123", "g_bkms_my_project_123"),
		Entry("simple workspace name", "simple", "g_bkms_simple"),
		Entry("workspace with multiple dashes", "multi-dash-name", "g_bkms_multi_dash_name"),
	)

	It("should generate random passwords correctly", func() {
		workspaceID := "test-workspace"

		// Test password generation
		manager1 := NewProjectManager(testutil.NewDB(GinkgoT()), workspaceID, "")
		Expect(len(manager1.password)).To(Equal(32))
		Expect(manager1.password).NotTo(BeEmpty())

		// Test different instances generate different passwords
		manager2 := NewProjectManager(testutil.NewDB(GinkgoT()), workspaceID, "")
		Expect(manager1.password).NotTo(Equal(manager2.password))
	})
})

var _ = Describe("initProject", func() {
	var (
		ctx       context.Context
		manager   *ProjectManager
		projectID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = newTestManager()
		projectID = "proj-" + stringx.Random(8)
	})

	It("should skip bkrepo API calls when project is already recorded", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			seedBkrepoProject(ctx, manager, projectID)
			// 已有记录时不应再调用制品库 API
			mockey.Mock((*bkrepo.StubApiClient).CreateProject).Return(errors.New("should not be called")).Build()

			err := manager.initProject(ctx, bkrepo.NewStub(manager.operator), projectID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should create and record new project", func() {
		err := manager.initProject(ctx, bkrepo.NewStub(manager.operator), projectID)
		Expect(err).NotTo(HaveOccurred())

		project, err := testutil.NewStore(GinkgoT(), ProjectMeta).Get(ctx, ByWorkspace(manager.workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(project.ID).To(Equal(projectID))
		Expect(project.Username).To(Equal(manager.username))
		Expect(project.Password).To(Equal(manager.password))
	})
})

var _ = Describe("initRepositories", func() {
	var (
		ctx       context.Context
		manager   *ProjectManager
		projectID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = newTestManager()
		projectID = "proj-" + stringx.Random(8)
	})

	It("should skip existing repositories", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) { cfg.BKRepo.InitRepos = dockerInitRepos() })
			seedBkrepoRepository(ctx, manager.workspaceID, projectID, RepoTypeDocker)
			// 已有记录时不应再调用制品库 API
			mockey.Mock((*bkrepo.StubApiClient).CreateRepository).
				Return(errors.New("should not be called")).
				Build()

			err := manager.initRepositories(ctx, bkrepo.NewStub(manager.operator), projectID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should create and record new repositories", func() {
		overrideConfig(func(cfg *config.Config) {
			cfg.BKRepo.InitRepos = []config.InitBKRepoConfig{
				{
					Name:        "docker",
					Type:        "DOCKER",
					IsPublic:    false,
					Description: "Docker repository",
				},
				{
					Name:        "helm",
					Type:        "HELM",
					IsPublic:    false,
					Description: "Helm repository",
				},
			}
		})

		err := manager.initRepositories(ctx, bkrepo.NewStub(manager.operator), projectID)
		Expect(err).NotTo(HaveOccurred())

		repoStore := testutil.NewStore(GinkgoT(), RepositoryMeta)
		helmRepo, err := repoStore.Get(ctx, ByWorkspaceAndType(manager.workspaceID, RepoTypeHelm))
		Expect(err).NotTo(HaveOccurred())
		Expect(helmRepo.ProjectID).To(Equal(projectID))
		Expect(helmRepo.Name).To(Equal("helm"))
		count, err := repoStore.Count(ctx, ByWorkspace(manager.workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(int64(2)))
	})
})

var _ = Describe("initImageRegistryCredentials", func() {
	var (
		ctx     context.Context
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = newTestManager()
	})

	It("should skip creation when image registry already exists", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) { cfg.BKRepo.InitRepos = dockerInitRepos() })
			mockExistingImageRegistry()

			err := manager.initImageRegistryCredentials(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should return error when DOCKER repository not found in config", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = []config.InitBKRepoConfig{
					{
						Name:         "helm",
						Type:         "HELM",
						EndpointTmpl: "helm.example.com/%s/%s",
					},
				}
			})

			err := manager.initImageRegistryCredentials(ctx, "test-project")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("generate image registry from config"))
			Expect(err.Error()).To(ContainSubstring("DOCKER repository not found"))
		})
	})

	It("should create credential and record image registry when it does not exist", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = dockerInitRepos()
				cfg.Development.UseStubBkCI = true
				cfg.Development.UseStubBkRepo = true
			})

			mockStore := &registry.ImageRegistryStoreMongo{}
			mockey.Mock(registry.NewImageRegistryStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock(
				(*registry.ImageRegistryStoreMongo).GetByWorkspaceAndType,
			).Return(nil, registry.ErrImageRegistryNotFound).Build()
			mockey.Mock((*registry.ImageRegistryStoreMongo).Create).Return(bson.NilObjectID, nil).Build()
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()

			err := manager.initImageRegistryCredentials(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("initHelmRepoCredentials", func() {
	var (
		ctx     context.Context
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = newTestManager()
	})

	It("should initialize and record helm repo credentials", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) { cfg.Development.UseStubBkCI = true })
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()

			err := manager.initHelmRepoCredentials(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())

			cred, err := testutil.NewStore(GinkgoT(), helmchartcred.Meta).
				Get(ctx, helmchartcred.ByWorkspace(manager.workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(cred.Username).To(Equal(manager.username))
			Expect(cred.Password).To(Equal(manager.password))
		})
	})

	It("should return error when EnsureCredential fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(helmchartcred.EnsureCredential).Return(errors.New("ensure credential error")).Build()

			err := manager.initHelmRepoCredentials(ctx, "test-project")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ensure credential error"))
		})
	})
})

var _ = Describe("Initialize", func() {
	var (
		ctx     context.Context
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = newTestManager()
	})

	It("should successfully complete the full initialization flow", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = dockerInitRepos()
				cfg.Development.UseStubBkCI = true
				cfg.Development.UseStubBkRepo = true
			})
			bkciProj := seedBkciProject(ctx, manager.workspaceID)
			mockExistingImageRegistry()
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()

			err := manager.Initialize(ctx)
			Expect(err).NotTo(HaveOccurred())

			// 制品库项目、仓库与 Helm 凭证均已落库
			project, err := testutil.NewStore(GinkgoT(), ProjectMeta).Get(ctx, ByWorkspace(manager.workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(project.ID).To(Equal(bkciProj.Code))
			_, err = testutil.NewStore(GinkgoT(), RepositoryMeta).
				Get(ctx, ByWorkspaceAndType(manager.workspaceID, RepoTypeDocker))
			Expect(err).NotTo(HaveOccurred())
			_, err = testutil.NewStore(GinkgoT(), helmchartcred.Meta).
				Get(ctx, helmchartcred.ByWorkspace(manager.workspaceID))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should return error when bkci project not found", func() {
		err := manager.Initialize(ctx)
		Expect(errors.Is(err, bkci.ErrProjectNotFound)).To(BeTrue())
	})

	It("should return error when initProject fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) { cfg.Development.UseStubBkRepo = true })
			seedBkciProject(ctx, manager.workspaceID)
			mockey.Mock((*bkrepo.StubApiClient).CreateProject).Return(errors.New("project api error")).Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init bkrepo project"))
			Expect(err.Error()).To(ContainSubstring("project api error"))
		})
	})

	It("should return error when initRepositories fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = dockerInitRepos()
				cfg.Development.UseStubBkRepo = true
			})
			seedBkciProject(ctx, manager.workspaceID)
			mockey.Mock((*bkrepo.StubApiClient).CreateRepository).
				Return(errors.New("repository api error")).
				Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init bkrepo repositories"))
			Expect(err.Error()).To(ContainSubstring("repository api error"))
		})
	})

	It("should return error when initImageRegistryCredentials fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = []config.InitBKRepoConfig{}
				cfg.Development.UseStubBkRepo = true
			})
			seedBkciProject(ctx, manager.workspaceID)

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init image registry credentials"))
			Expect(err.Error()).To(ContainSubstring("DOCKER repository not found"))
		})
	})

	It("should return error when initHelmRepoCredentials fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			overrideConfig(func(cfg *config.Config) {
				cfg.BKRepo.InitRepos = dockerInitRepos()
				cfg.Development.UseStubBkCI = true
				cfg.Development.UseStubBkRepo = true
			})
			seedBkciProject(ctx, manager.workspaceID)
			mockExistingImageRegistry()
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			mockey.Mock((*bkciapi.StubApiClient).CreateCredential).
				Return(errors.New("helm credential api error")).
				Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init helm repo credentials"))
			Expect(err.Error()).To(ContainSubstring("helm credential api error"))
		})
	})
})
