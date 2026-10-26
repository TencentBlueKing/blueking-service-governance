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
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("ConfigStore", func() {
	var configs *ConfigStore
	var ctx context.Context

	var codeRepoAppID, imageAppID string
	var codeRepoBuildCfg, imageBuildCfg Config

	BeforeEach(func() {
		var err error

		// Patch global config to set encrypt secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		configs = testutil.NewStore(GinkgoT(), ConfigMeta)

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
					ExtraFiles: []string{"data/key.pem", "certs"},
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
		It("should return error when build config with same appID already exists", func() {
			// First call should succeed
			err := configs.Create(ctx, &imageBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			// Second call with same name should fail
			err = configs.Create(ctx, &imageBuildCfg)
			Expect(err).To(MatchError(store.ErrDuplicate))
		})

		It("should encrypt the image password in db without touching the caller's object", func() {
			Expect(configs.Create(ctx, &imageBuildCfg)).To(Succeed())
			Expect(imageBuildCfg.Image.Password).To(Equal("password"))

			// 绕过 Decode 钩子直接读取库中的原始文档
			var raw Config
			Expect(configs.Collection().FindOne(ctx, bson.M{"appID": imageAppID}).Decode(&raw)).To(Succeed())
			Expect(raw.Image.Password).NotTo(BeEmpty())
			Expect(raw.Image.Password).NotTo(Equal("password"))

			cfg, err := configs.Get(ctx, ByApp(imageAppID))
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Image.Password).To(Equal("password"))
		})
	})

	Context("UpdateConfig", func() {
		It("should update build config successfully", func() {
			err := configs.Create(ctx, &codeRepoBuildCfg)
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
				ExtraFiles: []string{"data/*.pem"},
			}
			err = UpdateConfig(ctx, configs, &codeRepoBuildCfg)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := configs.Get(ctx, ByApp(codeRepoAppID))
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
				ExtraFiles: []string{"data/*.pem"},
			}))
		})
	})

	Context("UpdateConfig with cleared optional configs", func() {
		It("should remove optional configs that are cleared in the update", func() {
			Expect(configs.Create(ctx, &imageBuildCfg)).To(Succeed())

			imageBuildCfg.SourceType = SourceTypeCodeRepository
			imageBuildCfg.Image = nil
			imageBuildCfg.CodeRepo = codeRepoBuildCfg.CodeRepo
			Expect(UpdateConfig(ctx, configs, &imageBuildCfg)).To(Succeed())

			cfg, err := configs.Get(ctx, ByApp(imageAppID))
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.SourceType).To(Equal(SourceTypeCodeRepository))
			Expect(cfg.Image).To(BeNil())
			Expect(cfg.CodeRepo.RepoURL).To(Equal("https://github.com/octocat/hello-world"))
		})
	})

	Context("Get", func() {
		It("should return ErrConfigNotFound when build config with given appID does not exist", func() {
			cfg, err := configs.Get(ctx, ByApp("non-existent-app-id"))
			Expect(err).To(MatchError(ErrConfigNotFound))
			Expect(cfg).To(BeNil())

			err = UpdateConfig(ctx, configs, &Config{AppID: "non-existent-app-id"})
			Expect(err).To(MatchError(ErrConfigNotFound))
		})
	})
})

var _ = Describe("RecordStore", func() {
	var db *mongo.Database
	var records *RecordStore
	var ctx context.Context

	var workspaceID, appID string
	var buildRecordA, buildRecordB Record
	var buildID1, buildID2 string

	listPage := func(keyword string, page, size int64) *store.Page[Record] {
		result, err := records.ListPage(
			ctx, page, size, ByApp(appID), RecordKeywordMatches(keyword), store.Desc("createdAt"),
		)
		Expect(err).NotTo(HaveOccurred())
		return result
	}

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, RecordMeta)

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
			// 早于 B 创建，避免两条记录落在同一毫秒导致排序不稳定
			CreatedAt: time.Now().Add(-time.Minute),
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
			Expect(CreateRecord(ctx, db, &buildRecordA)).To(Succeed())
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())

			// List：按创建时间倒序，第一页是后创建的 B
			page := listPage("", 1, 1)
			Expect(page.Total).To(Equal(int64(2)))
			Expect(page.HasNext).To(BeTrue())
			Expect(page.Items).To(HaveLen(1))
			Expect(page.Items[0].BuildID).To(Equal(buildID2))

			// Update
			buildRecordA.Status = StatusRunning
			buildRecordA.Extras = map[string]string{"commitID": "abc"}
			Expect(UpdateRecord(ctx, records, &buildRecordA)).To(Succeed())

			// Get
			r, err := records.Get(ctx, ByBuild(appID, "b-123"))
			Expect(err).NotTo(HaveOccurred())
			Expect(r.AppID).To(Equal(appID))
			Expect(r.BuildID).To(Equal(buildID1))
			Expect(r.Status).To(Equal(StatusRunning))
			Expect(r.Extras).To(Equal(map[string]string{"commitID": "abc"}))
			Expect(r.Operator).To(Equal("admin"))
			Expect(r.CreatedAt).NotTo(BeZero())
		})

		It("should return ErrRecordNotFound when build record does not exist", func() {
			_, err := records.Get(ctx, ByBuild(appID, "missing-build"))
			Expect(err).To(MatchError(ErrRecordNotFound))

			err = UpdateRecord(ctx, records, &Record{AppID: appID, BuildID: "missing-build"})
			Expect(err).To(MatchError(ErrRecordNotFound))
		})

		It("should return error when build record with same index already exists", func() {
			// First call should succeed
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())

			// Second call with same appID & buildID should fail
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(MatchError(store.ErrDuplicate))
		})

		It("list with keyword match artifact", func() {
			Expect(CreateRecord(ctx, db, &buildRecordA)).To(Succeed())
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())

			page := listPage("v1.0.0", 1, 1)
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.HasNext).To(BeFalse())
			Expect(page.Items).To(HaveLen(1))
		})

		It("list with keyword match operator", func() {
			Expect(CreateRecord(ctx, db, &buildRecordA)).To(Succeed())
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())

			page := listPage("ADMIN", 1, 1)
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.Items).To(HaveLen(1))
			Expect(page.Items[0].Operator).To(Equal("admin"))
		})
	})

	Context("Image digest", func() {
		newSucceeded := func(buildID, tag, digest string, endedAt time.Time) *Record {
			return &Record{
				WorkspaceID: workspaceID,
				AppID:       appID,
				BuildID:     buildID,
				Params:      map[string]string{pipelineparam.ImageTag: tag},
				Status:      StatusSuccess,
				ImageDigest: digest,
				EndedAt:     endedAt,
			}
		}

		It("should update image digest without touching other fields", func() {
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())

			Expect(UpdateRecordImageDigest(ctx, records, appID, buildID2, "sha256:aaa")).To(Succeed())

			r, err := records.Get(ctx, ByBuild(appID, buildID2))
			Expect(err).NotTo(HaveOccurred())
			Expect(r.ImageDigest).To(Equal("sha256:aaa"))
			Expect(r.Status).To(Equal(StatusSuccess))
			Expect(r.Artifact).To(Equal(buildRecordB.Artifact))
		})

		It("should keep the image digest recorded first", func() {
			Expect(CreateRecord(ctx, db, &buildRecordB)).To(Succeed())
			Expect(UpdateRecordImageDigest(ctx, records, appID, buildID2, "sha256:first")).To(Succeed())

			err := UpdateRecordImageDigest(ctx, records, appID, buildID2, "sha256:second")
			Expect(err).To(MatchError(ErrRecordNotFound))
			r, err := records.Get(ctx, ByBuild(appID, buildID2))
			Expect(err).NotTo(HaveOccurred())
			Expect(r.ImageDigest).To(Equal("sha256:first"))
		})

		It("should return ErrRecordNotFound when updating digest of a missing record", func() {
			err := UpdateRecordImageDigest(ctx, records, appID, "missing-build", "sha256:aaa")
			Expect(err).To(MatchError(ErrRecordNotFound))
		})

		It("should match the latest succeeded record by both tag and digest", func() {
			now := time.Now()
			Expect(CreateRecord(ctx, db, newSucceeded("b-old", "v1", "sha256:aaa", now.Add(-time.Hour)))).To(Succeed())
			Expect(CreateRecord(ctx, db, newSucceeded("b-new", "v1", "sha256:aaa", now))).To(Succeed())
			// 同 digest 不同 tag（构建缓存全部命中）不能串到 v1 上
			Expect(CreateRecord(ctx, db, newSucceeded("b-v2", "v2", "sha256:aaa", now.Add(time.Hour)))).To(Succeed())
			failed := newSucceeded("b-failed", "v1", "sha256:aaa", now.Add(2*time.Hour))
			failed.Status = StatusFailed
			Expect(CreateRecord(ctx, db, failed)).To(Succeed())
			otherApp := newSucceeded("b-other-app", "v1", "sha256:aaa", now.Add(3*time.Hour))
			otherApp.AppID = appID + "-other"
			Expect(CreateRecord(ctx, db, otherApp)).To(Succeed())

			result, err := LatestSucceededRecordsByImages(ctx, records, appID, map[string]string{"v1": "sha256:aaa"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["v1"].BuildID).To(Equal("b-new"))
		})

		It("should not match when the tag was overwritten by another image", func() {
			Expect(CreateRecord(ctx, db, newSucceeded("b-1", "v1", "sha256:aaa", time.Now()))).To(Succeed())

			pushed := map[string]string{"v1": "sha256:pushed"}
			result, err := LatestSucceededRecordsByImages(ctx, records, appID, pushed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should pair tag and digest within the same record", func() {
			now := time.Now()
			// tag 与 digest 各自落在当前页集合内，但不是同一对
			Expect(CreateRecord(ctx, db, newSucceeded("b-cross", "v1", "sha256:bbb", now))).To(Succeed())
			Expect(CreateRecord(ctx, db, newSucceeded("b-v2", "v2", "sha256:bbb", now.Add(-time.Hour)))).To(Succeed())

			result, err := LatestSucceededRecordsByImages(
				ctx, records, appID, map[string]string{"v1": "sha256:aaa", "v2": "sha256:bbb"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["v2"].BuildID).To(Equal("b-v2"))
		})

		It("should skip images whose digest is not synced yet", func() {
			Expect(CreateRecord(ctx, db, newSucceeded("b-1", "v1", "", time.Now()))).To(Succeed())

			result, err := LatestSucceededRecordsByImages(ctx, records, appID, map[string]string{"v1": ""})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})
})

var _ = Describe("CreateRecord Auto-increment Num", func() {
	var db *mongo.Database
	var ctx context.Context

	newRecord := func(appID, buildPrefix string) *Record {
		return &Record{
			WorkspaceID: "ws-num-test",
			AppID:       appID,
			BuildID:     buildPrefix + stringx.Random(6),
			Status:      StatusRunning,
			Operator:    "admin",
		}
	}

	BeforeEach(func() {
		db = testutil.NewDB(GinkgoT())
		ctx = context.Background()
	})

	It("should assign Num=1 for the first build record of an AppID", func() {
		record := newRecord("num-test-app-"+stringx.Random(6), "build-first-")
		Expect(CreateRecord(ctx, db, record)).To(Succeed())
		Expect(record.Num).To(Equal(int64(1)))
	})

	It("should auto-increment Num for consecutive builds of the same AppID", func() {
		appID := "num-test-app-" + stringx.Random(6)

		for i := int64(1); i <= 5; i++ {
			record := newRecord(appID, "build-seq-")
			Expect(CreateRecord(ctx, db, record)).To(Succeed())
			Expect(record.Num).To(Equal(i))
		}
	})

	It("should maintain independent Num sequences for different AppIDs", func() {
		appIDA := "num-test-appA-" + stringx.Random(6)
		appIDB := "num-test-appB-" + stringx.Random(6)

		// 为 AppID A 创建 3 条记录
		for i := int64(1); i <= 3; i++ {
			record := newRecord(appIDA, "buildA-")
			Expect(CreateRecord(ctx, db, record)).To(Succeed())
			Expect(record.Num).To(Equal(i))
		}

		// 为 AppID B 创建 2 条记录，序号应从 1 开始独立递增
		for i := int64(1); i <= 2; i++ {
			record := newRecord(appIDB, "buildB-")
			Expect(CreateRecord(ctx, db, record)).To(Succeed())
			Expect(record.Num).To(Equal(i))
		}

		// 再为 AppID A 创建一条，序号应为 4
		recordA4 := newRecord(appIDA, "buildA-")
		Expect(CreateRecord(ctx, db, recordA4)).To(Succeed())
		Expect(recordA4.Num).To(Equal(int64(4)))
	})
})
