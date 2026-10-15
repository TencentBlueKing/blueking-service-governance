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

package tagdeletion

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	helmrelease "helm.sh/helm/v3/pkg/release"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// MongoDB DateTime 精度为毫秒，依赖 createdAt 排序的场景需要等到下一个精度窗口。
const tagDeletionTimestampPrecisionWait = 5 * time.Millisecond

var _ = Describe("TagDeletionServiceRealDB", func() {
	var (
		ctx             context.Context
		diApp           *fxtest.App
		service         *Service
		workspaceStore  *bkmsworkspace.WorkspaceStore
		appStore        *bkmsapp.ApplicationStore
		envSvc          *bkmsenv.EnvService
		cfgStore        *build.ConfigStore
		helmDeployStore *helmdeploy.RecordStore
		db              *mongo.Database
		promotionStore  *promotion.Store
		oldConfig       *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()

		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		oldConfig = config.G
		config.G = &config.Config{
			Encrypt: config.EncryptConfig{Secret: secret},
		}

		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)
		diApp = fxtest.New(
			GinkgoT(),
			bkmsenv.FxModule,
			FxModule,
			fx.Populate(
				&service,
				&envSvc,
			),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		helmDeployStore = store.Of(db, helmdeploy.RecordMeta)
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		cfgStore = testutil.NewStore(GinkgoT(), build.ConfigMeta)
		promotionStore = testutil.NewStore(GinkgoT(), promotion.Meta)
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		if diApp != nil {
			diApp.RequireStop()
		}
		config.G = oldConfig
	})

	It("should report usage when the latest deployed record still uses the tag", func() {
		// Helm 应用：最新部署记录直接命中目标 tag，应返回当前 release 名称。
		fixture := newTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		err := helmDeployStore.Create(ctx, &helmdeploy.Record{
			WorkspaceID: fixture.workspace.ID,
			AppID:       fixture.app.ID,
			EnvName:     fixture.env.Name,
			ReleaseName: "my-release",
			ImageTag:    fixture.inUseTag,
			Status:      helmrelease.StatusDeployed,
			Operator:    "admin",
		})
		Expect(err).NotTo(HaveOccurred())

		usages, err := service.ListImageUsages(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).NotTo(HaveOccurred())
		Expect(usages).To(HaveLen(1))
		Expect(usages[0].AppID).To(Equal(fixture.app.ID))
		Expect(usages[0].AppName).To(Equal(fixture.app.Name))
		Expect(usages[0].EnvName).To(Equal(fixture.env.Name))
		Expect(usages[0].LaneName).To(BeEmpty())
		Expect(usages[0].WorkloadName).To(Equal("my-release"))
		Expect(usages[0].Status).To(Equal(string(helmrelease.StatusDeployed)))
	})

	It("should report app model workload usage when the latest deployed record still uses the tag", func() {
		// AppModel 应用：从 ResourceKeys 中提取 workload 名称，
		// 并忽略 service 等非 workload 资源。
		fixture := newAppModelTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		err := store.Of(db, appmodeldeploy.RecordMeta).Create(ctx, &appmodeldeploy.Record{
			WorkspaceID: fixture.workspace.ID,
			AppID:       fixture.app.ID,
			EnvName:     fixture.env.Name,
			ImageTag:    fixture.inUseTag,
			Status:      appmodeldeploy.StatusDeployed,
			ResourceKeys: appmodeldeploy.ResourceKeys{
				{Kind: k8skind.Deploy, Name: "trpc-workload"},
				{Kind: k8skind.SVC, Name: "service-a"},
			},
			Creator: "admin",
			Updater: "admin",
		})
		Expect(err).NotTo(HaveOccurred())

		usages, err := service.ListImageUsages(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).NotTo(HaveOccurred())
		Expect(usages).To(HaveLen(1))
		Expect(usages[0].AppID).To(Equal(fixture.app.ID))
		Expect(usages[0].AppName).To(Equal(fixture.app.Name))
		Expect(usages[0].EnvName).To(Equal(fixture.env.Name))
		Expect(usages[0].LaneName).To(BeEmpty())
		Expect(usages[0].WorkloadName).To(Equal("trpc-workload"))
		Expect(usages[0].Status).To(Equal(string(appmodeldeploy.StatusDeployed)))
	})

	It("should delete the image even when the latest deployed record still uses the tag", func() {
		// 删除接口不依赖 usage-check 做拦截；
		// 这里只验证远端删除被调用，以及本地快照/晋级记录被清理。
		fixture := newTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		err := helmDeployStore.Create(ctx, &helmdeploy.Record{
			WorkspaceID: fixture.workspace.ID,
			AppID:       fixture.app.ID,
			EnvName:     fixture.env.Name,
			ReleaseName: "my-release",
			ImageTag:    fixture.inUseTag,
			Status:      helmrelease.StatusDeployed,
			Operator:    "admin",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(promotion.Promote(ctx, promotionStore, fixture.app.ID, fixture.repoKey, fixture.inUseTag, "admin")).
			To(Succeed())

		var deletedRepoName string
		var deletedTag string
		// 避免真实访问仓库，只记录 service 传给 registry client 的删除参数。
		mockey.Mock(registry.New).Return(&registry.Client{}).Build()
		mockey.Mock((*registry.Client).DeleteTag).To(
			func(_ *registry.Client, _ context.Context, repoName, tag string) error {
				deletedRepoName = repoName
				deletedTag = tag
				return nil
			},
		).Build()

		err = service.DeleteImageTag(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).NotTo(HaveOccurred())
		Expect(deletedRepoName).To(Equal(fixture.repoName))
		Expect(deletedTag).To(Equal(fixture.inUseTag))

		tags, err := snapshot.ListAllTags(ctx, db, fixture.repoKey)
		Expect(err).NotTo(HaveOccurred())
		Expect(tags).To(Equal([]string{fixture.safeToDeleteTag}))

		isPromoted, err := promotionStore.Exists(
			ctx,
			promotion.ByTag(fixture.app.ID, fixture.repoKey, fixture.inUseTag),
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(isPromoted).To(BeFalse())
	})

	It("should classify delete errors when image repository credential is missing", func() {
		fixture := newTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		Expect(build.UpdateConfig(ctx, cfgStore, &build.Config{
			AppID: fixture.app.ID,
			Image: &build.ImageConfig{
				Name: fixture.repoName,
			},
		})).To(Succeed())

		err := service.DeleteImageTag(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).To(HaveOccurred())
		Expect(stderrors.Is(err, ErrImageRepoAuthRequired)).To(BeTrue())
	})

	It("should classify delete errors when registry returns authentication required", func() {
		fixture := newTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		mockey.Mock(registry.New).Return(&registry.Client{}).Build()
		mockey.Mock((*registry.Client).DeleteTag).To(
			func(_ *registry.Client, _ context.Context, _, _ string) error {
				return pkgerrors.New("delete tag: UNAUTHORIZED: authentication required")
			},
		).Build()

		err := service.DeleteImageTag(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).To(HaveOccurred())
		Expect(stderrors.Is(err, ErrImageRepoAuthRequired)).To(BeTrue())
	})

	It("should report no usage when only historical records use the tag", func() {
		// 先插入一个使用旧 tag 的成功记录，再插入一个更新后的成功记录；
		// 当前生效的是最新记录，因此旧 tag 不应再被报告为占用。
		fixture := newTagDeletionFixture(ctx, workspaceStore, appStore, envSvc, cfgStore, db)
		defer fixture.cleanup(ctx, workspaceStore, appStore, envSvc, cfgStore, promotionStore)

		err := helmDeployStore.Create(ctx, &helmdeploy.Record{
			WorkspaceID: fixture.workspace.ID,
			AppID:       fixture.app.ID,
			EnvName:     fixture.env.Name,
			ReleaseName: "old-release",
			ImageTag:    fixture.inUseTag,
			Status:      helmrelease.StatusDeployed,
			Operator:    "admin",
		})
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(tagDeletionTimestampPrecisionWait)

		err = helmDeployStore.Create(ctx, &helmdeploy.Record{
			WorkspaceID: fixture.workspace.ID,
			AppID:       fixture.app.ID,
			EnvName:     fixture.env.Name,
			ReleaseName: "new-release",
			ImageTag:    fixture.safeToDeleteTag,
			Status:      helmrelease.StatusDeployed,
			Operator:    "admin",
		})
		Expect(err).NotTo(HaveOccurred())

		usages, err := service.ListImageUsages(ctx, fixture.app.ID, fixture.inUseTag)
		Expect(err).NotTo(HaveOccurred())
		Expect(usages).To(BeEmpty())
	})
})

type tagDeletionFixture struct {
	workspace       *bkmsworkspace.Workspace
	app             *bkmsapp.Application
	env             *envmodel.Environment
	repoName        string
	repoKey         string
	inUseTag        string
	safeToDeleteTag string
}

func newTagDeletionFixture(
	ctx context.Context,
	workspaceStore *bkmsworkspace.WorkspaceStore,
	appStore *bkmsapp.ApplicationStore,
	envSvc *bkmsenv.EnvService,
	cfgStore *build.ConfigStore,
	db *mongo.Database,
) *tagDeletionFixture {
	// Helm 场景的最小测试夹具：工作空间、应用、环境、镜像构建配置和两条快照。
	workspace := dbfactory.Workspace(ctx, workspaceStore)
	app := dbfactory.HelmApplication(ctx, &dbfactory.HelmApplicationStores{
		AppStore: appStore,
	}, &dbfactory.HelmApplicationOpts{
		WorkspaceID: workspace.ID,
	})
	env := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
		WorkspaceID: workspace.ID,
		AppIDs:      []string{app.ID},
	})

	repoName := "library/delete-" + stringx.Random(6)
	Expect(cfgStore.Create(ctx, &build.Config{
		AppID: app.ID,
		Image: &build.ImageConfig{
			Name:     repoName,
			Username: "alice",
			Password: "secret",
		},
	})).To(Succeed())

	repoKey := snapshot.GenerateRepoKey(repoName, "alice", "secret")
	Expect(snapshot.UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v2.0.0"})).To(Succeed())

	return &tagDeletionFixture{
		workspace:       workspace,
		app:             app,
		env:             env,
		repoName:        repoName,
		repoKey:         repoKey,
		inUseTag:        "v1.0.0",
		safeToDeleteTag: "v2.0.0",
	}
}

func newAppModelTagDeletionFixture(
	ctx context.Context,
	workspaceStore *bkmsworkspace.WorkspaceStore,
	appStore *bkmsapp.ApplicationStore,
	envSvc *bkmsenv.EnvService,
	cfgStore *build.ConfigStore,
	db *mongo.Database,
) *tagDeletionFixture {
	// AppModel 场景复用同一份返回结构，仅把应用类型切为 TRPC。
	workspace := dbfactory.Workspace(ctx, workspaceStore)
	appName := "test-app-" + stringx.Random(6)
	app := &bkmsapp.Application{
		ID:          appName + stringx.Random(6),
		Name:        appName,
		WorkspaceID: workspace.ID,
		Type:        bkmsapp.AppTypeTRPC,
	}
	Expect(appStore.Create(ctx, app)).To(Succeed())

	env := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{
		WorkspaceID: workspace.ID,
		AppIDs:      []string{app.ID},
	})

	repoName := "library/delete-" + stringx.Random(6)
	Expect(cfgStore.Create(ctx, &build.Config{
		AppID: app.ID,
		Image: &build.ImageConfig{
			Name:     repoName,
			Username: "alice",
			Password: "secret",
		},
	})).To(Succeed())

	repoKey := snapshot.GenerateRepoKey(repoName, "alice", "secret")
	Expect(snapshot.UpsertTags(ctx, db, repoKey, []string{"v1.0.0", "v2.0.0"})).To(Succeed())

	return &tagDeletionFixture{
		workspace:       workspace,
		app:             app,
		env:             env,
		repoName:        repoName,
		repoKey:         repoKey,
		inUseTag:        "v1.0.0",
		safeToDeleteTag: "v2.0.0",
	}
}

func (f *tagDeletionFixture) cleanup(
	ctx context.Context,
	workspaceStore *bkmsworkspace.WorkspaceStore,
	appStore *bkmsapp.ApplicationStore,
	envSvc *bkmsenv.EnvService,
	cfgStore *build.ConfigStore,
	promotionStore *promotion.Store,
) {
	// 这些测试会落真实数据库，结束后按 app/repoKey 定向清理，避免互相污染。
	db := database.Client().Database(database.Name())

	_, err := db.Collection("app_model_deploy_records").DeleteMany(ctx, bson.M{
		"appID": f.app.ID,
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = db.Collection("helm_deploy_records").DeleteMany(ctx, bson.M{
		"appID": f.app.ID,
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = db.Collection("image_snapshots").DeleteMany(ctx, bson.M{
		"repoKey": f.repoKey,
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = db.Collection("repo_snapshot_statuses").DeleteMany(ctx, bson.M{
		"repoKey": f.repoKey,
	})
	Expect(err).NotTo(HaveOccurred())
	_, err = promotionStore.DeleteMany(ctx, promotion.ByApp(f.app.ID))
	Expect(err).NotTo(HaveOccurred())
	Expect(cfgStore.DeleteIfExists(ctx, build.ByApp(f.app.ID))).To(Succeed())
	Expect(envmodel.RemoveApp(ctx, db, f.env.ID, f.app.ID)).To(Succeed())
	Expect(envSvc.Delete(ctx, f.env.ID)).To(Succeed())
	Expect(appStore.DeleteIfExists(ctx, bkmsapp.ByName(f.workspace.ID, f.app.Name))).To(Succeed())
	Expect(bkmsworkspace.Delete(ctx, workspaceStore, f.workspace.ID)).To(Succeed())
}
