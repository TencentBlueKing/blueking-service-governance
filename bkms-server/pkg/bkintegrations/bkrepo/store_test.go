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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
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
		err := testutil.CleanupMetaColl(ProjectMeta)
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		s = testutil.NewStore(GinkgoT(), ProjectMeta)
	})

	Describe("Create", func() {
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
	})

	Describe("password encryption", func() {
		var secret string

		BeforeEach(func() {
			var err error
			secret, err = crypto.GenerateKey(32)
			Expect(err).NotTo(HaveOccurred())
			overrideConfig(func(cfg *config.Config) { cfg.Encrypt.Secret = secret })
		})

		rawPassword := func(id string) string {
			raw, err := s.Collection().FindOne(ctx, bson.M{"id": id}).Raw()
			Expect(err).NotTo(HaveOccurred())
			return raw.Lookup("password").StringValue()
		}

		// insertRaw 绕过 Store 直接写入原始文档，用于验证读取路径的解密
		insertRaw := func(id, password string) {
			_, err := s.Collection().InsertOne(ctx, bson.M{
				"id": id, "workspaceID": "ws-" + id, "username": "admin", "password": password,
			})
			Expect(err).NotTo(HaveOccurred())
		}

		It("should store the AES ciphertext of the password without touching the caller's object", func() {
			project := &Project{ID: "p1", WorkspaceID: "ws-p1", Username: "admin", Password: "password123"}
			Expect(s.Create(ctx, project)).To(Succeed())
			Expect(project.Password).To(Equal("password123"))

			stored := rawPassword("p1")
			Expect(stored).NotTo(Equal("password123"))
			Expect(crypto.AESDecrypt(secret, stored)).To(Equal("password123"))
		})

		It("should decrypt the stored ciphertext on Get and List", func() {
			ciphertext, err := crypto.AESEncrypt(secret, "password123")
			Expect(err).NotTo(HaveOccurred())
			insertRaw("p1", ciphertext)

			got, err := s.Get(ctx, store.Where("id", "p1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Password).To(Equal("password123"))

			list, err := s.List(ctx, ByWorkspace("ws-p1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Password).To(Equal("password123"))
		})

		It("should keep an empty password empty in the database", func() {
			Expect(s.Create(ctx, &Project{ID: "p1", WorkspaceID: "ws-p1", Username: "admin"})).To(Succeed())
			Expect(rawPassword("p1")).To(BeEmpty())

			got, err := s.Get(ctx, store.Where("id", "p1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Password).To(BeEmpty())
		})

		It("should fail to read a password that is not valid ciphertext", func() {
			insertRaw("p1", "plain-password")

			_, err := s.Get(ctx, store.Where("id", "p1"))
			Expect(err).To(MatchError(ContainSubstring("decrypt password")))

			_, err = s.List(ctx, ByWorkspace("ws-p1"))
			Expect(err).To(MatchError(ContainSubstring("decrypt password")))
		})
	})

	Describe("Get", func() {
		It("should return error when project not found", func() {
			project, err := s.Get(ctx, store.Where("id", "non-existent-project"))
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
		err := testutil.CleanupMetaColl(RepositoryMeta)
		Expect(err).NotTo(HaveOccurred())

		// 创建 store 实例
		s = testutil.NewStore(GinkgoT(), RepositoryMeta)
	})

	Describe("Create", func() {
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

		It("should return error when repository not found", func() {
			repo, err := s.Get(ctx, ByWorkspaceAndType("non-existent-workspace", RepoTypeDocker))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrRepositoryNotFound)).To(BeTrue())
			Expect(repo).To(BeNil())
		})
	})
})
