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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("ProjectStore", func() {
	var (
		ctx context.Context
		s   *ProjectStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection(ProjectMeta.Collection)
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		s = testutil.NewStore(GinkgoT(), ProjectMeta)
	})

	Describe("Create", func() {
		It("should create a project successfully", func() {
			project := &Project{
				ID:          "test-project",
				WorkspaceID: "workspace-001",
				Username:    "admin",
				Password:    "password123",
				Creator:     "test-user",
			}

			err := s.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.CreatedAt).NotTo(BeZero())
		})

		It("should return error when creating duplicate project", func() {
			project := &Project{
				ID:          "test-project",
				WorkspaceID: "workspace-001",
				Username:    "admin",
				Password:    "password123",
				Creator:     "test-user",
			}

			err := s.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 workspaceID 的项目
			duplicateProject := &Project{
				ID:          "test-project-2",
				WorkspaceID: "workspace-001",
				Username:    "admin2",
				Password:    "password456",
				Creator:     "test-user-2",
			}

			err = s.Create(ctx, duplicateProject)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())
		})

		It("should encrypt password before storing", func() {
			project := &Project{
				ID:          "test-project",
				WorkspaceID: "workspace-001",
				Username:    "admin",
				Password:    "password123",
				Creator:     "test-user",
			}

			err := s.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())

			// 直接从数据库读取，验证密码已加密
			var rawProject Project
			err = database.DB().Collection(ProjectMeta.Collection).
				FindOne(ctx, bson.M{"id": "test-project"}).Decode(&rawProject)
			Expect(err).NotTo(HaveOccurred())
			Expect(rawProject.Password).NotTo(Equal("password123"))
			// 加密只作用于落库的副本，调用方对象保持明文
			Expect(project.Password).To(Equal("password123"))
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			project := &Project{
				ID:          "test-project",
				WorkspaceID: "workspace-001",
				Username:    "admin",
				Password:    "password123",
				Creator:     "test-user",
			}
			err := s.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get project by ID successfully", func() {
			project, err := s.Get(ctx, store.Where("id", "test-project"))
			Expect(err).NotTo(HaveOccurred())
			Expect(project).NotTo(BeNil())
			Expect(project.ID).To(Equal("test-project"))
			Expect(project.WorkspaceID).To(Equal("workspace-001"))
			Expect(project.Username).To(Equal("admin"))
			Expect(project.Password).To(Equal("password123"))
		})

		It("should return error when project not found", func() {
			project, err := s.Get(ctx, store.Where("id", "non-existent-project"))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrProjectNotFound)).To(BeTrue())
			Expect(project).To(BeNil())
		})
	})

	Describe("GetByWorkspace", func() {
		BeforeEach(func() {
			project := &Project{
				ID:          "test-project",
				WorkspaceID: "workspace-001",
				Username:    "admin",
				Password:    "password123",
				Creator:     "test-user",
			}
			err := s.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get project by workspace ID successfully", func() {
			project, err := s.Get(ctx, ByWorkspace("workspace-001"))
			Expect(err).NotTo(HaveOccurred())
			Expect(project).NotTo(BeNil())
			Expect(project.ID).To(Equal("test-project"))
			Expect(project.WorkspaceID).To(Equal("workspace-001"))
		})

		It("should return error when workspace not found", func() {
			project, err := s.Get(ctx, ByWorkspace("non-existent-workspace"))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrProjectNotFound)).To(BeTrue())
			Expect(project).To(BeNil())
		})
	})
})

var _ = Describe("RepositoryStore", func() {
	var (
		ctx context.Context
		s   *RepositoryStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection(RepositoryMeta.Collection)
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		s = testutil.NewStore(GinkgoT(), RepositoryMeta)
	})

	Describe("Create", func() {
		It("should create a repository successfully", func() {
			repository := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "docker-repo",
				Type:        RepoTypeDocker,
				IsPublic:    false,
				Creator:     "test-user",
			}

			err := s.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())
			Expect(repository.CreatedAt).NotTo(BeZero())
		})

		It("should return error when creating duplicate repository", func() {
			repository := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "docker-repo",
				Type:        RepoTypeDocker,
				IsPublic:    false,
				Creator:     "test-user",
			}

			err := s.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 workspaceID + type 的仓库
			duplicateRepo := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "docker-repo-2",
				Type:        RepoTypeDocker,
				IsPublic:    true,
				Creator:     "test-user-2",
			}

			err = s.Create(ctx, duplicateRepo)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())
		})

		It("should allow creating different types of repositories for same workspace", func() {
			dockerRepo := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "docker-repo",
				Type:        RepoTypeDocker,
				IsPublic:    false,
				Creator:     "test-user",
			}

			err := s.Create(ctx, dockerRepo)
			Expect(err).NotTo(HaveOccurred())

			helmRepo := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "helm-repo",
				Type:        RepoTypeHelm,
				IsPublic:    false,
				Creator:     "test-user",
			}

			err = s.Create(ctx, helmRepo)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetByWorkspaceAndType", func() {
		BeforeEach(func() {
			dockerRepo := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "docker-repo",
				Type:        RepoTypeDocker,
				IsPublic:    false,
				Creator:     "test-user",
				CreatedAt:   time.Now(),
			}
			err := s.Create(ctx, dockerRepo)
			Expect(err).NotTo(HaveOccurred())

			helmRepo := &Repository{
				ProjectID:   "test-project",
				WorkspaceID: "workspace-001",
				Name:        "helm-repo",
				Type:        RepoTypeHelm,
				IsPublic:    true,
				Creator:     "test-user",
				CreatedAt:   time.Now(),
			}
			err = s.Create(ctx, helmRepo)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get docker repository successfully", func() {
			repo, err := s.Get(ctx, ByWorkspaceAndType("workspace-001", RepoTypeDocker))
			Expect(err).NotTo(HaveOccurred())
			Expect(repo).NotTo(BeNil())
			Expect(repo.Name).To(Equal("docker-repo"))
			Expect(repo.Type).To(Equal(RepoTypeDocker))
			Expect(repo.IsPublic).To(BeFalse())
		})

		It("should get helm repository successfully", func() {
			repo, err := s.Get(ctx, ByWorkspaceAndType("workspace-001", RepoTypeHelm))
			Expect(err).NotTo(HaveOccurred())
			Expect(repo).NotTo(BeNil())
			Expect(repo.Name).To(Equal("helm-repo"))
			Expect(repo.Type).To(Equal(RepoTypeHelm))
			Expect(repo.IsPublic).To(BeTrue())
		})

		It("should return error when repository not found", func() {
			repo, err := s.Get(ctx, ByWorkspaceAndType("non-existent-workspace", RepoTypeDocker))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrRepositoryNotFound)).To(BeTrue())
			Expect(repo).To(BeNil())
		})

		It("should return error when repository type not found", func() {
			repo, err := s.Get(ctx, ByWorkspaceAndType("workspace-001", "GENERIC"))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrRepositoryNotFound)).To(BeTrue())
			Expect(repo).To(BeNil())
		})
	})
})
