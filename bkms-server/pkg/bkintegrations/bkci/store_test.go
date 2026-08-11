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

package bkci

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ProjectStoreMongo", func() {
	var (
		ctx   context.Context
		store *ProjectStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection("bkci_projects")
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		store, err = NewProjectStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})

	Describe("Create", func() {
		It("should create a project successfully", func() {
			project := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project",
				WorkspaceID: "workspace-001",
				Creator:     "test-user",
			}

			err := store.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.CreatedAt).NotTo(BeZero())
		})

		It("should return error when creating duplicate project ID", func() {
			project := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project",
				WorkspaceID: "workspace-001",
				Creator:     "test-user",
			}

			err := store.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 ID 的项目
			duplicateProject := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project-2",
				WorkspaceID: "workspace-002",
				Creator:     "test-user-2",
			}

			err = store.Create(ctx, duplicateProject)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should return error when creating duplicate project code", func() {
			project := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project",
				WorkspaceID: "workspace-001",
				Creator:     "test-user",
			}

			err := store.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 Code 的项目
			duplicateProject := &Project{
				ID:          "test-project-id-002",
				Code:        "test-project",
				WorkspaceID: "workspace-002",
				Creator:     "test-user-2",
			}

			err = store.Create(ctx, duplicateProject)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should return error when creating duplicate workspace ID", func() {
			project := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project",
				WorkspaceID: "workspace-001",
				Creator:     "test-user",
			}

			err := store.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 WorkspaceID 的项目
			duplicateProject := &Project{
				ID:          "test-project-id-002",
				Code:        "test-project-2",
				WorkspaceID: "workspace-001",
				Creator:     "test-user-2",
			}

			err = store.Create(ctx, duplicateProject)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})
	})

	Describe("GetByWorkspace", func() {
		BeforeEach(func() {
			project := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project",
				WorkspaceID: "workspace-001",
				Creator:     "test-user",
			}
			err := store.Create(ctx, project)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get project by workspace ID successfully", func() {
			project, err := store.GetByWorkspace(ctx, "workspace-001")
			Expect(err).NotTo(HaveOccurred())
			Expect(project).NotTo(BeNil())
			Expect(project.ID).To(Equal("test-project-id-001"))
			Expect(project.Code).To(Equal("test-project"))
			Expect(project.WorkspaceID).To(Equal("workspace-001"))
			Expect(project.Creator).To(Equal("test-user"))
			Expect(project.CreatedAt).NotTo(BeZero())
		})

		It("should return error when workspace not found", func() {
			project, err := store.GetByWorkspace(ctx, "non-existent-workspace")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrProjectNotFound)).To(BeTrue())
			Expect(project).To(BeNil())
		})
	})

	Describe("Multiple Projects", func() {
		It("should allow creating multiple projects with different IDs, codes and workspaces", func() {
			project1 := &Project{
				ID:          "test-project-id-001",
				Code:        "test-project-1",
				WorkspaceID: "workspace-001",
				Creator:     "test-user-1",
			}
			err := store.Create(ctx, project1)
			Expect(err).NotTo(HaveOccurred())

			project2 := &Project{
				ID:          "test-project-id-002",
				Code:        "test-project-2",
				WorkspaceID: "workspace-002",
				Creator:     "test-user-2",
			}
			err = store.Create(ctx, project2)
			Expect(err).NotTo(HaveOccurred())

			// 验证两个项目都能正确获取
			p1, err := store.GetByWorkspace(ctx, "workspace-001")
			Expect(err).NotTo(HaveOccurred())
			Expect(p1.Code).To(Equal("test-project-1"))

			p2, err := store.GetByWorkspace(ctx, "workspace-002")
			Expect(err).NotTo(HaveOccurred())
			Expect(p2.Code).To(Equal("test-project-2"))
		})
	})
})

var _ = Describe("DBPipelineTemplateStore", func() {
	var (
		ctx   context.Context
		store *DBPipelineTemplateStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection("bkci_pipeline_templates")
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		store, err = NewDBPipelineTemplateStore(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})

	Describe("Upsert", func() {
		It("should upsert a pipeline template successfully", func() {
			template := &PipelineTemplate{
				ID:          "0348a1df-44de-29b2-1c94-2d42841c009d",
				Type:        "build-template",
				Version:     "1.0.0",
				Name:        "构建模板",
				Description: "用于构建的流水线模板",
			}

			err := store.Upsert(ctx, template)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetByType", func() {
		BeforeEach(func() {
			template := &PipelineTemplate{
				ID:          "0348a1df-44de-29b2-1c94-2d42841c009d",
				Type:        "build-template",
				Version:     "1.0.0",
				Name:        "构建模板",
				Description: "用于构建的流水线模板",
				Stages: []map[string]any{
					{
						"id":   "stage-1",
						"name": "stage-1",
						"containers": []map[string]any{
							{
								"@type":     "trigger",
								"classType": "trigger",
								"name":      "trigger",
							},
						},
					},
				},
			}
			err := store.Upsert(ctx, template)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get template by type successfully", func() {
			template, err := store.GetByType(ctx, "build-template")
			Expect(err).NotTo(HaveOccurred())
			Expect(template).NotTo(BeNil())
			Expect(template.ID).To(Equal("0348a1df-44de-29b2-1c94-2d42841c009d"))
			Expect(template.Type).To(Equal("build-template"))
			Expect(template.Version).To(Equal("1.0.0"))
			Expect(template.Name).To(Equal("构建模板"))
			Expect(template.Description).To(Equal("用于构建的流水线模板"))
			Expect(template.Stages).To(HaveLen(1))
		})

		It("should return error when template type not found", func() {
			template, err := store.GetByType(ctx, "non-existent-template")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrPipelineTemplateNotFound)).To(BeTrue())
			Expect(template).To(BeNil())
		})
	})
})

var _ = Describe("PipelineStoreMongo", func() {
	var (
		ctx   context.Context
		store *PipelineStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection("bkci_pipelines")
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		store, err = NewPipelineStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})

	Describe("Create", func() {
		It("should create a pipeline successfully", func() {
			pipeline := &Pipeline{
				ID:              "p-5df30e9fe868af903dff8d375dd7b463",
				Type:            "build",
				WorkspaceID:     "workspace-001",
				ProjectCode:     "test-project",
				Name:            "构建流水线",
				Description:     "用于构建的流水线",
				TemplateVersion: "1.0.0",
				Creator:         "admin",
			}

			err := store.Create(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when creating duplicate pipeline ID", func() {
			pipeline := &Pipeline{
				ID:          "p-5df30e9fe868af903dff8d375dd7b463",
				Type:        "build",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Name:        "构建流水线",
				Description: "用于构建的流水线",
				Creator:     "admin",
			}

			err := store.Create(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 ID 的流水线
			duplicatePipeline := &Pipeline{
				ID:          "p-5df30e9fe868af903dff8d375dd7b463",
				Type:        "deploy",
				WorkspaceID: "workspace-002",
				ProjectCode: "test-project-2",
				Name:        "部署流水线",
				Description: "用于部署的流水线",
				Creator:     "user1",
			}

			err = store.Create(ctx, duplicatePipeline)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should return error when creating duplicate (workspaceID, type)", func() {
			pipeline := &Pipeline{
				ID:          "p-5df30e9fe868af903dff8d375dd7b463",
				Type:        "build",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Name:        "构建流水线",
				Description: "用于构建的流水线",
				Creator:     "admin",
			}

			err := store.Create(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())

			// 尝试在同一个 workspace 下创建相同 type 的流水线
			duplicatePipeline := &Pipeline{
				ID:          "p-f2A2266G-31dF-bbGeDD34g8dg720e82C",
				Type:        "build",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Name:        "构建流水线2",
				Description: "另一个构建流水线",
				Creator:     "user1",
			}

			err = store.Create(ctx, duplicatePipeline)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should allow creating pipelines with same type in different workspaces", func() {
			pipeline1 := &Pipeline{
				ID:          "p-5df30e9fe868af903dff8d375dd7b463",
				Type:        "build",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project-1",
				Name:        "构建流水线1",
				Description: "工作空间1的构建流水线",
				Creator:     "admin",
			}

			err := store.Create(ctx, pipeline1)
			Expect(err).NotTo(HaveOccurred())

			// 在不同的 workspace 下创建相同 type 的流水线应该成功
			pipeline2 := &Pipeline{
				ID:          "p-f2A2266G-31dF-bbGeDD34g8dg720e82C",
				Type:        "build",
				WorkspaceID: "workspace-002",
				ProjectCode: "test-project-2",
				Name:        "构建流水线2",
				Description: "工作空间2的构建流水线",
				Creator:     "user1",
			}

			err = store.Create(ctx, pipeline2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetByWorkspaceAndType", func() {
		BeforeEach(func() {
			pipeline := &Pipeline{
				ID:              "p-5df30e9fe868af903dff8d375dd7b463",
				Type:            "build",
				WorkspaceID:     "workspace-001",
				ProjectCode:     "test-project",
				Name:            "构建流水线",
				Description:     "用于构建的流水线",
				TemplateVersion: "1.0.0",
				Creator:         "admin",
			}
			err := store.Create(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get pipeline by workspaceID and type successfully", func() {
			pipeline, err := store.GetByWorkspaceAndType(ctx, "workspace-001", "build")
			Expect(err).NotTo(HaveOccurred())
			Expect(pipeline).NotTo(BeNil())
			Expect(pipeline.ID).To(Equal("p-5df30e9fe868af903dff8d375dd7b463"))
			Expect(pipeline.Type).To(Equal("build"))
			Expect(pipeline.WorkspaceID).To(Equal("workspace-001"))
			Expect(pipeline.ProjectCode).To(Equal("test-project"))
			Expect(pipeline.Name).To(Equal("构建流水线"))
			Expect(pipeline.Description).To(Equal("用于构建的流水线"))
			Expect(pipeline.TemplateVersion).To(Equal("1.0.0"))
			Expect(pipeline.Creator).To(Equal("admin"))
		})

		It("should return error when pipeline not found", func() {
			pipeline, err := store.GetByWorkspaceAndType(ctx, "workspace-999", "build")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrPipelineNotFound)).To(BeTrue())
			Expect(pipeline).To(BeNil())
		})

		It("should return error when type not found in workspace", func() {
			pipeline, err := store.GetByWorkspaceAndType(ctx, "workspace-001", "deploy")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrPipelineNotFound)).To(BeTrue())
			Expect(pipeline).To(BeNil())
		})
	})

	Describe("UpdateBuiltinTemplateVersion", func() {
		BeforeEach(func() {
			pipeline := &Pipeline{
				ID:              "p-5df30e9fe868af903dff8d375dd7b463",
				Type:            "build",
				WorkspaceID:     "workspace-001",
				ProjectCode:     "test-project",
				Name:            "构建流水线",
				Description:     "用于构建的流水线",
				TemplateVersion: "1.0.0",
				Creator:         "admin",
			}
			err := store.Create(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update template version and display fields successfully", func() {
			pipeline := &Pipeline{
				Type:            "build",
				WorkspaceID:     "workspace-001",
				Name:            "新版构建流水线",
				Description:     "新版描述",
				TemplateVersion: "1.1.0",
			}

			err := store.UpdateBuiltinTemplateVersion(ctx, pipeline)
			Expect(err).NotTo(HaveOccurred())

			updated, err := store.GetByWorkspaceAndType(ctx, "workspace-001", "build")
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.ID).To(Equal("p-5df30e9fe868af903dff8d375dd7b463"))
			Expect(updated.Name).To(Equal("新版构建流水线"))
			Expect(updated.Description).To(Equal("新版描述"))
			Expect(updated.TemplateVersion).To(Equal("1.1.0"))
		})

		It("should return error when pipeline does not exist", func() {
			pipeline := &Pipeline{
				Type:            "deploy",
				WorkspaceID:     "workspace-001",
				Name:            "部署流水线",
				Description:     "部署描述",
				TemplateVersion: "1.1.0",
			}

			err := store.UpdateBuiltinTemplateVersion(ctx, pipeline)
			Expect(errors.Is(err, ErrPipelineNotFound)).To(BeTrue())
		})
	})
})

var _ = Describe("RepositoryStoreMongo", func() {
	var (
		ctx   context.Context
		store *RepositoryStoreMongo
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection("bkci_repositories")
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		store, err = NewRepositoryStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())
	})

	Describe("Create", func() {
		It("should create a repository successfully", func() {
			repository := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Creator:     "admin",
			}

			err := store.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())
			Expect(repository.CreatedAt).NotTo(BeZero())
		})

		It("should return error when creating duplicate repository (id, projectCode)", func() {
			repository := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Creator:     "admin",
			}

			err := store.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())

			// 尝试创建相同 (id, projectCode) 的仓库
			duplicateRepository := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms-2",
				URL:         "https://git.example.com/bkms/bkms-2.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-002",
				ProjectCode: "test-project",
				Creator:     "user1",
			}

			err = store.Create(ctx, duplicateRepository)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should return error when creating duplicate (workspaceID, alias)", func() {
			repository := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Creator:     "admin",
			}

			err := store.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())

			// 尝试在同一个 workspace 下创建相同 alias 的仓库
			duplicateRepository := &Repository{
				ID:          "Ab4Ey",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms-2.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project-2",
				Creator:     "user1",
			}

			err = store.Create(ctx, duplicateRepository)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should allow creating repositories with same alias in different workspaces", func() {
			repository1 := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project-1",
				Creator:     "admin",
			}

			err := store.Create(ctx, repository1)
			Expect(err).NotTo(HaveOccurred())

			// 在不同的 workspace 下创建相同 alias 的仓库应该成功
			repository2 := &Repository{
				ID:          "Ab4Ey",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms-2.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-002",
				ProjectCode: "test-project-2",
				Creator:     "user1",
			}

			err = store.Create(ctx, repository2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow creating repositories with same ID in different projects", func() {
			repository1 := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms-1",
				URL:         "https://git.example.com/bkms/bkms-1.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project-1",
				Creator:     "admin",
			}

			err := store.Create(ctx, repository1)
			Expect(err).NotTo(HaveOccurred())

			// 在不同的项目下创建相同 ID 的仓库应该成功
			repository2 := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms-2",
				URL:         "https://git.example.com/bkms/bkms-2.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-002",
				ProjectCode: "test-project-2",
				Creator:     "user1",
			}

			err = store.Create(ctx, repository2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetByWorkspaceAndAlias", func() {
		BeforeEach(func() {
			repository := &Repository{
				ID:          "Zr3Dx",
				Alias:       "bkms",
				URL:         "https://git.example.com/bkms/bkms.git",
				Type:        "codeGit",
				WorkspaceID: "workspace-001",
				ProjectCode: "test-project",
				Creator:     "admin",
			}
			err := store.Create(ctx, repository)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get repository by workspaceID and alias successfully", func() {
			repository, err := store.GetByWorkspaceAndAlias(ctx, "workspace-001", "bkms")
			Expect(err).NotTo(HaveOccurred())
			Expect(repository).NotTo(BeNil())
			Expect(repository.ID).To(Equal("Zr3Dx"))
			Expect(repository.Alias).To(Equal("bkms"))
			Expect(repository.URL).To(Equal("https://git.example.com/bkms/bkms.git"))
			Expect(repository.Type).To(Equal("codeGit"))
			Expect(repository.WorkspaceID).To(Equal("workspace-001"))
			Expect(repository.ProjectCode).To(Equal("test-project"))
			Expect(repository.Creator).To(Equal("admin"))
			Expect(repository.CreatedAt).NotTo(BeZero())
		})

		It("should return error when repository not found", func() {
			repository, err := store.GetByWorkspaceAndAlias(ctx, "workspace-999", "bkms")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrRepositoryNotFound)).To(BeTrue())
			Expect(repository).To(BeNil())
		})
	})
})
