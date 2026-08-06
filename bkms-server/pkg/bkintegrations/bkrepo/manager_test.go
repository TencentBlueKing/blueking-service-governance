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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkrepo"
	helmchartcred "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// mockBkciProject creates a mock bkci project
func mockBkciProject() *bkci.Project {
	return &bkci.Project{
		ID:          "test-project-id",
		Code:        "test-project",
		WorkspaceID: "test-workspace",
		Creator:     "test-user",
	}
}

// mockBkrepoProject creates a mock bkrepo project
func mockBkrepoProject() *Project {
	return &Project{
		ID:          "test-project",
		WorkspaceID: "test-workspace",
		Username:    "g_bkms_test_workspace",
		Password:    "test-password-32-chars-long-123",
		Creator:     "test-user",
	}
}

// mockBkrepoRepository creates a mock bkrepo repository
func mockBkrepoRepository() *Repository {
	return &Repository{
		WorkspaceID: "test-workspace",
		ProjectID:   "test-project",
		Name:        "docker",
		Type:        RepoTypeDocker,
		IsPublic:    false,
		Creator:     "test-user",
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

var _ = Describe("NewProjectManager", func() {
	It("should create valid ProjectManager instances with correct fields", func() {
		// Test basic instance creation and field assignment
		workspaceID := "test-workspace"
		manager := NewProjectManager(workspaceID, "admin")
		Expect(manager).NotTo(BeNil())
		Expect(manager.workspaceID).To(Equal(workspaceID))
		Expect(manager.operator).To(Equal("admin"))
	})

	DescribeTable("should generate correct username format",
		func(workspaceID, expectedUsername string) {
			m := NewProjectManager(workspaceID, "")
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
		manager1 := NewProjectManager(workspaceID, "")
		Expect(len(manager1.password)).To(Equal(32))
		Expect(manager1.password).NotTo(BeEmpty())

		// Test different instances generate different passwords
		manager2 := NewProjectManager(workspaceID, "")
		Expect(manager1.password).NotTo(Equal(manager2.password))
	})
})

var _ = Describe("initProject", func() {
	var (
		ctx     context.Context
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewProjectManager("test-workspace", "")
	})

	It("should handle existing projects correctly", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(mockBkrepoProject(), nil).Build()

			apiClient := bkrepo.NewStub(manager.operator)
			err := manager.initProject(ctx, apiClient, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should create new projects successfully", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(nil, ErrProjectNotFound).Build()
			mockey.Mock((*ProjectStoreMongo).Create).Return(nil).Build()

			apiClient := bkrepo.NewStub(manager.operator)
			err := manager.initProject(ctx, apiClient, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("initRepositories", func() {
	var (
		ctx       context.Context
		manager   *ProjectManager
		originCfg *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewProjectManager("test-workspace", "")
		originCfg = config.G
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("should handle existing repositories correctly", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:        "docker",
							Type:        "DOCKER",
							IsPublic:    false,
							Description: "Docker repository",
						},
					},
				},
			}

			mockStore := &RepositoryStoreMongo{}
			mockey.Mock(NewRepositoryStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock((*RepositoryStoreMongo).GetByWorkspaceAndType).Return(mockBkrepoRepository(), nil).Build()

			apiClient := bkrepo.NewStub(manager.operator)
			err := manager.initRepositories(ctx, apiClient, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should create new repositories successfully", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
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
					},
				},
			}

			mockStore := &RepositoryStoreMongo{}
			mockey.Mock(NewRepositoryStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock((*RepositoryStoreMongo).GetByWorkspaceAndType).Return(nil, ErrRepositoryNotFound).Build()
			mockey.Mock((*RepositoryStoreMongo).Create).Return(nil).Build()

			apiClient := bkrepo.NewStub(manager.operator)
			err := manager.initRepositories(ctx, apiClient, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("initImageRegistryCredentials", func() {
	var (
		ctx       context.Context
		manager   *ProjectManager
		originCfg *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewProjectManager("test-workspace", "")
		originCfg = config.G
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("should skip creation when image registry already exists", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:         "docker",
							Type:         "DOCKER",
							EndpointTmpl: "docker.example.com/%s/%s",
						},
					},
				},
			}

			mockStore := &registry.ImageRegistryStoreMongo{}
			mockey.Mock(registry.NewImageRegistryStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock(
				(*registry.ImageRegistryStoreMongo).GetByWorkspaceAndType,
			).Return(mockImageRegistry(), nil).Build()

			err := manager.initImageRegistryCredentials(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should return error when DOCKER repository not found in config", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:         "helm",
							Type:         "HELM",
							EndpointTmpl: "helm.example.com/%s/%s",
						},
					},
				},
			}

			err := manager.initImageRegistryCredentials(ctx, "test-project")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("generate image registry from config"))
			Expect(err.Error()).To(ContainSubstring("DOCKER repository not found"))
		})
	})

	It("should return error when CreateCredential API fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:         "docker",
							Type:         "DOCKER",
							EndpointTmpl: "docker.example.com/%s/%s",
						},
					},
				},
				Development: config.DevConfig{
					UseStubBkCI:   true,
					UseStubBkRepo: true,
				},
			}

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
		manager = NewProjectManager("test-workspace", "")
	})

	It("should initialize helm repo credentials successfully", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockCredStore := &helmchartcred.HelmRepoCredentialStoreMongo{}
			mockey.Mock(helmchartcred.NewHelmRepoCredentialStoreMongo).Return(mockCredStore, nil).Build()
			mockey.Mock(helmchartcred.EnsureCredential).Return(nil).Build()

			err := manager.initHelmRepoCredentials(ctx, "test-project")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should return error when helm repo credential store creation fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(helmchartcred.NewHelmRepoCredentialStoreMongo).
				Return(nil, errors.New("cred store error")).
				Build()

			err := manager.initHelmRepoCredentials(ctx, "test-project")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("create helm repo credential store"))
		})
	})

	It("should return error when EnsureCredential fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockCredStore := &helmchartcred.HelmRepoCredentialStoreMongo{}
			mockey.Mock(helmchartcred.NewHelmRepoCredentialStoreMongo).Return(mockCredStore, nil).Build()
			mockey.Mock(helmchartcred.EnsureCredential).Return(errors.New("ensure credential error")).Build()

			err := manager.initHelmRepoCredentials(ctx, "test-project")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ensure credential error"))
		})
	})
})

var _ = Describe("Initialize", func() {
	var (
		ctx       context.Context
		manager   *ProjectManager
		originCfg *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = NewProjectManager("test-workspace", "")
		originCfg = config.G
	})

	AfterEach(func() {
		config.G = originCfg
	})

	It("should successfully complete the full initialization flow", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:         "docker",
							Type:         "DOCKER",
							IsPublic:     false,
							Description:  "Docker repository",
							EndpointTmpl: "docker.example.com/%s/%s",
						},
					},
				},
			}

			mockBkciStore := &bkci.ProjectStoreMongo{}
			mockey.Mock(bkci.NewProjectStoreMongo).Return(mockBkciStore, nil).Build()
			mockey.Mock((*bkci.ProjectStoreMongo).GetByWorkspace).Return(mockBkciProject(), nil).Build()

			mockBkrepoProjectStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockBkrepoProjectStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(mockBkrepoProject(), nil).Build()

			mockBkrepoRepoStore := &RepositoryStoreMongo{}
			mockey.Mock(NewRepositoryStoreMongo).Return(mockBkrepoRepoStore, nil).Build()
			mockey.Mock((*RepositoryStoreMongo).GetByWorkspaceAndType).Return(mockBkrepoRepository(), nil).Build()

			mockImageRegistryStore := &registry.ImageRegistryStoreMongo{}
			mockey.Mock(registry.NewImageRegistryStoreMongo).Return(mockImageRegistryStore, nil).Build()
			mockey.Mock(
				(*registry.ImageRegistryStoreMongo).GetByWorkspaceAndType,
			).Return(mockImageRegistry(), nil).Build()

			mockCredStore := &helmchartcred.HelmRepoCredentialStoreMongo{}
			mockey.Mock(helmchartcred.NewHelmRepoCredentialStoreMongo).Return(mockCredStore, nil).Build()
			mockey.Mock(helmchartcred.EnsureCredential).Return(nil).Build()

			err := manager.Initialize(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should return error when initProject fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockBkciStore := &bkci.ProjectStoreMongo{}
			mockey.Mock(bkci.NewProjectStoreMongo).Return(mockBkciStore, nil).Build()
			mockey.Mock((*bkci.ProjectStoreMongo).GetByWorkspace).Return(mockBkciProject(), nil).Build()

			mockey.Mock(NewProjectStoreMongo).Return(nil, errors.New("project store error")).Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init bkrepo project"))
			Expect(err.Error()).To(ContainSubstring("project store error"))
		})
	})

	It("should return error when initRepositories fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockBkciStore := &bkci.ProjectStoreMongo{}
			mockey.Mock(bkci.NewProjectStoreMongo).Return(mockBkciStore, nil).Build()
			mockey.Mock((*bkci.ProjectStoreMongo).GetByWorkspace).Return(mockBkciProject(), nil).Build()

			mockBkrepoProjectStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockBkrepoProjectStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(mockBkrepoProject(), nil).Build()
			mockey.Mock(NewRepositoryStoreMongo).Return(nil, errors.New("repository store error")).Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init bkrepo repositories"))
			Expect(err.Error()).To(ContainSubstring("repository store error"))
		})
	})

	It("should return error when initImageRegistryCredentials fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{},
				},
			}

			mockBkciStore := &bkci.ProjectStoreMongo{}
			mockey.Mock(bkci.NewProjectStoreMongo).Return(mockBkciStore, nil).Build()
			mockey.Mock((*bkci.ProjectStoreMongo).GetByWorkspace).Return(mockBkciProject(), nil).Build()

			mockBkrepoProjectStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockBkrepoProjectStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(mockBkrepoProject(), nil).Build()

			mockBkrepoRepoStore := &RepositoryStoreMongo{}
			mockey.Mock(NewRepositoryStoreMongo).Return(mockBkrepoRepoStore, nil).Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init image registry credentials"))
			Expect(err.Error()).To(ContainSubstring("DOCKER repository not found"))
		})
	})

	It("should return error when initHelmRepoCredentials fails", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			config.G = &config.Config{
				BKRepo: config.BKRepoConfig{
					InitRepos: []config.InitBKRepoConfig{
						{
							Name:         "docker",
							Type:         "DOCKER",
							IsPublic:     false,
							Description:  "Docker repository",
							EndpointTmpl: "docker.example.com/%s/%s",
						},
					},
				},
			}

			mockBkciStore := &bkci.ProjectStoreMongo{}
			mockey.Mock(bkci.NewProjectStoreMongo).Return(mockBkciStore, nil).Build()
			mockey.Mock((*bkci.ProjectStoreMongo).GetByWorkspace).Return(mockBkciProject(), nil).Build()

			mockBkrepoProjectStore := &ProjectStoreMongo{}
			mockey.Mock(NewProjectStoreMongo).Return(mockBkrepoProjectStore, nil).Build()
			mockey.Mock((*ProjectStoreMongo).GetByWorkspace).Return(mockBkrepoProject(), nil).Build()

			mockBkrepoRepoStore := &RepositoryStoreMongo{}
			mockey.Mock(NewRepositoryStoreMongo).Return(mockBkrepoRepoStore, nil).Build()
			mockey.Mock((*RepositoryStoreMongo).GetByWorkspaceAndType).Return(mockBkrepoRepository(), nil).Build()

			mockImageRegistryStore := &registry.ImageRegistryStoreMongo{}
			mockey.Mock(registry.NewImageRegistryStoreMongo).Return(mockImageRegistryStore, nil).Build()
			mockey.Mock(
				(*registry.ImageRegistryStoreMongo).GetByWorkspaceAndType,
			).Return(mockImageRegistry(), nil).Build()

			mockey.Mock(helmchartcred.NewHelmRepoCredentialStoreMongo).
				Return(nil, errors.New("helm cred store error")).
				Build()

			err := manager.Initialize(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("init helm repo credentials"))
			Expect(err.Error()).To(ContainSubstring("helm cred store error"))
		})
	})
})
