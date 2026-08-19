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

package build

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ConfigStore", func() {
	var store ConfigStore
	var ctx context.Context

	var codeRepoAppID, imageAppID string
	var codeRepoBuildCfg, imageBuildCfg Config

	BeforeEach(func() {
		var err error

		// Patch global config to set encrypt secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		// Create a new ConfigStore
		store, err = NewConfigStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		codeRepoAppID = "code-repo-app-id-" + stringx.Random(6)
		imageAppID = "image-app-id-" + stringx.Random(6)

		// 源码来源应用构建配置
		codeRepoBuildCfg = Config{
			AppID:        codeRepoAppID,
			SourceType:   SourceTypeCodeRepository,
			PipelineType: "Dockerfile",
			CodeRepo: &RepositoryConfig{
				Type:          RepositoryTypeGitHub,
				RepoAlias:     "octocat/hello-world",
				RepoURL:       "https://github.com/octocat/hello-world",
				DefaultBranch: "main",
				SourceDir:     "",
				Dockerfile:    "",
				DockerBuildArgs: map[string]string{
					"buildNo": "1",
				},
				ImageBuildMode: ImageBuildModePlatform,
				PlatformBuildConfig: &PlatformBuildConfig{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					Commands: &BuildCommands{
						PreBuild:   []string{"go mod download"},
						Build:      []string{"go build -o app ./cmd/server"},
						RuntimeEnv: []string{"apt-get update"},
						Start:      "./app",
					},
				},
			},
		}

		// 镜像来源应用构建配置
		imageBuildCfg = Config{
			AppID:        imageAppID,
			SourceType:   SourceTypeImageRegistry,
			PipelineType: "Dockerfile",
			Image: &ImageConfig{
				Name:     "hub.bktencent.com/bkapps/echo-server",
				Username: "admin",
				Password: "password",
			},
		}
	})

	Context("Create", func() {
		It("should create and get build config successfully", func() {
			err := store.Create(ctx, &codeRepoBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			// Get the application back from the database
			cfg, err := store.Get(ctx, codeRepoAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.AppID).To(Equal(codeRepoAppID))
			Expect(cfg.CodeRepo.DockerBuildArgs).To(Equal(map[string]string{
				"buildNo": "1",
			}))
			Expect(cfg.CodeRepo.EffectiveImageBuildMode()).To(Equal(ImageBuildModePlatform))
			Expect(cfg.CodeRepo.PlatformBuildConfig).To(Equal(&PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					PreBuild:   []string{"go mod download"},
					Build:      []string{"go build -o app ./cmd/server"},
					RuntimeEnv: []string{"apt-get update"},
					Start:      "./app",
				},
			}))
		})

		It("should return error when build config with same appID already exists", func() {
			// First call should succeed
			err := store.Create(ctx, &imageBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same name should fail
			err = store.Create(ctx, &imageBuildCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("build config with the same appID already exists"))
		})
	})

	Context("Update", func() {
		It("should update build config successfully", func() {
			err := store.Create(ctx, &codeRepoBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			codeRepoBuildCfg.CodeRepo.SourceDir = "test-dir"
			codeRepoBuildCfg.CodeRepo.Dockerfile = "Dockerfile"
			codeRepoBuildCfg.CodeRepo.DefaultBranch = "dev"
			codeRepoBuildCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.25",
				RunnerImage:  "ubuntu:24.04",
				Commands: &BuildCommands{
					PreBuild:   []string{"go env -w GOPROXY=https://goproxy.cn"},
					Build:      []string{"make build"},
					RuntimeEnv: []string{"apt-get install -y ca-certificates"},
					Start:      "./server",
				},
			}
			err = store.Update(ctx, &codeRepoBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := store.Get(ctx, codeRepoAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.CodeRepo.SourceDir).To(Equal("test-dir"))
			Expect(cfg.CodeRepo.Dockerfile).To(Equal("Dockerfile"))
			Expect(cfg.CodeRepo.DefaultBranch).To(Equal("dev"))
			Expect(cfg.CodeRepo.PlatformBuildConfig).To(Equal(&PlatformBuildConfig{
				BuilderImage: "golang:1.25",
				RunnerImage:  "ubuntu:24.04",
				Commands: &BuildCommands{
					PreBuild:   []string{"go env -w GOPROXY=https://goproxy.cn"},
					Build:      []string{"make build"},
					RuntimeEnv: []string{"apt-get install -y ca-certificates"},
					Start:      "./server",
				},
			}))
		})
	})

	Context("Get", func() {
		It("should return error when build config with given appName does not exist", func() {
			cfg, err := store.Get(ctx, "non-existent-app-id")
			Expect(err).To(HaveOccurred())
			Expect(cfg).To(BeNil())
		})
	})
})

var _ = Describe("RecordStore", func() {
	var store RecordStore
	var ctx context.Context

	var workspaceID, appID string
	var buildRecordA, buildRecordB Record
	var buildID1, buildID2 string

	BeforeEach(func() {
		var err error

		store, err = NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		workspaceID = "test-workspace-" + stringx.Random(6)
		appID = "test-app-" + stringx.Random(6) + "-" + stringx.Random(6)
		buildID1, buildID2 = "b-123", "b-456"

		buildRecordA = Record{
			WorkspaceID: workspaceID,
			AppID:       appID,
			BuildID:     buildID1,
			Params: map[string]string{
				"foo": "bar",
			},
			Status:   StatusFailed,
			Artifact: "",
			Operator: "admin",
		}
		buildRecordB = Record{
			WorkspaceID: workspaceID,
			AppID:       appID,
			BuildID:     buildID2,
			Status:      StatusSuccess,
			Params: map[string]string{
				"foo": "bar",
			},
			Artifact: "hub.bktencent.com/bkapps/helm-application:v1.0.0",
			Operator: "blueking",
		}
	})

	Context("Create List Update Get", func() {
		It("should create, list, update and get build record successfully", func() {
			// Create
			err := store.Create(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			// List
			records, total, err := store.List(ctx, appID, "", 1, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(1))

			// Update
			buildRecordA.Status = StatusRunning
			err = store.Update(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			// Get
			r, err := store.Get(ctx, appID, "b-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(r).NotTo(BeNil())
			Expect(r.AppID).To(Equal(appID))
			Expect(r.BuildID).To(Equal(buildID1))
			Expect(r.Status).To(Equal(StatusRunning))
		})

		It("should return ErrRecordNotFound when build record does not exist", func() {
			_, err := store.Get(ctx, appID, "missing-build")
			Expect(err).To(MatchError(ErrRecordNotFound))

			err = store.Update(ctx, &Record{AppID: appID, BuildID: "missing-build"})
			Expect(err).To(MatchError(ErrRecordNotFound))
		})

		It("should return error when build record with same index already exists", func() {
			// First call should succeed
			err := store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same workspaceID & appName & buildID should fail
			err = store.Create(ctx, &buildRecordB)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"build record with the same appID & buildID already exists",
			))
		})

		It("list with keyword match artifact", func() {
			err := store.Create(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			records, total, err := store.List(ctx, appID, "v1.0.0", 1, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
		})

		It("list with keyword match operator", func() {
			err := store.Create(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			records, total, err := store.List(ctx, appID, "admin", 1, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
		})

		It("should handle regex special characters as literal strings", func() {
			// Test that regex special characters like * and . are treated as literals
			buildRecordA.Artifact = "hub.bktencent.com/app/image:v1.0.0*test"
			buildRecordB.Artifact = "hub.bktencent.com/app/image:v2.0.0.release"

			err := store.Create(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			// Search for "v1.0.0*" should match the literal "*" character
			records, total, err := store.List(ctx, appID, "v1.0.0*", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].Artifact).To(ContainSubstring("v1.0.0*test"))

			// Search for "v2.0.0." should match the literal "." character
			records, total, err = store.List(ctx, appID, "v2.0.0.", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].Artifact).To(ContainSubstring("v2.0.0.release"))
		})

		It("should handle more regex special characters as literal strings", func() {
			// Test more special characters: +, ?, ^, $, [, ], (, )
			buildRecordA.Artifact = "image:v1.0+beta"
			buildRecordB.Artifact = "image:v2.0[test]"

			err := store.Create(ctx, &buildRecordA)
			Expect(err).NotTo(HaveOccurred())

			err = store.Create(ctx, &buildRecordB)
			Expect(err).NotTo(HaveOccurred())

			// Search for "v1.0+" should match the literal "+" character
			records, total, err := store.List(ctx, appID, "v1.0+", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].Artifact).To(ContainSubstring("v1.0+beta"))

			// Search for "[test]" should match the literal "[]" characters
			records, total, err = store.List(ctx, appID, "[test]", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].Artifact).To(ContainSubstring("v2.0[test]"))
		})
	})
})

var _ = Describe("RecordStore Auto-increment Num", func() {
	var store RecordStore
	var ctx context.Context

	BeforeEach(func() {
		var err error
		store, err = NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	It("should assign Num=1 for the first build record of an AppID", func() {
		appID := "num-test-app-" + stringx.Random(6)
		record := Record{
			WorkspaceID: "ws-num-test",
			AppID:       appID,
			BuildID:     "build-first-" + stringx.Random(6),
			Status:      StatusRunning,
			Operator:    "admin",
		}
		err := store.Create(ctx, &record)
		Expect(err).NotTo(HaveOccurred())
		Expect(record.Num).To(Equal(int64(1)))
	})

	It("should auto-increment Num for consecutive builds of the same AppID", func() {
		appID := "num-test-app-" + stringx.Random(6)

		for i := int64(1); i <= 5; i++ {
			record := Record{
				WorkspaceID: "ws-num-test",
				AppID:       appID,
				BuildID:     "build-seq-" + stringx.Random(6),
				Status:      StatusRunning,
				Operator:    "admin",
			}
			err := store.Create(ctx, &record)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Num).To(Equal(i))
		}
	})

	It("should maintain independent Num sequences for different AppIDs", func() {
		appIDA := "num-test-appA-" + stringx.Random(6)
		appIDB := "num-test-appB-" + stringx.Random(6)

		// 为 AppID A 创建 3 条记录
		for i := int64(1); i <= 3; i++ {
			record := Record{
				WorkspaceID: "ws-num-test",
				AppID:       appIDA,
				BuildID:     "buildA-" + stringx.Random(6),
				Status:      StatusRunning,
				Operator:    "admin",
			}
			err := store.Create(ctx, &record)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Num).To(Equal(i))
		}

		// 为 AppID B 创建 2 条记录，序号应从 1 开始独立递增
		for i := int64(1); i <= 2; i++ {
			record := Record{
				WorkspaceID: "ws-num-test",
				AppID:       appIDB,
				BuildID:     "buildB-" + stringx.Random(6),
				Status:      StatusRunning,
				Operator:    "admin",
			}
			err := store.Create(ctx, &record)
			Expect(err).NotTo(HaveOccurred())
			Expect(record.Num).To(Equal(i))
		}

		// 再为 AppID A 创建一条，序号应为 4
		recordA4 := Record{
			WorkspaceID: "ws-num-test",
			AppID:       appIDA,
			BuildID:     "buildA-" + stringx.Random(6),
			Status:      StatusRunning,
			Operator:    "admin",
		}
		err := store.Create(ctx, &recordA4)
		Expect(err).NotTo(HaveOccurred())
		Expect(recordA4.Num).To(Equal(int64(4)))
	})
})
