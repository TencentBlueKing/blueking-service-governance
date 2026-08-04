package preview_test

import (
	"context"
	"errors"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	coreapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvarsstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	parserpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/parser"
	previewpkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/envfile/preview"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ImportPreviewService", func() {
	var (
		diApp                 *fxtest.App
		ctx                   context.Context
		workspaceID           string
		environment           envmodel.Environment
		scopedEnvVarStore     envvarsstore.ScopedEnvVarStore
		appStore              coreapp.ApplicationStore
		appModelStore         appmodel.AppModelStore
		appConfigFileStore    appcfg.AppConfigFileStore
		appConfigVersionStore appcfg.AppConfigFileVersionStore
		buildConfigStore      build.ConfigStore
		service               *previewpkg.ImportPreviewService
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvarsstore.FxModule,
			depmodel.FxModule,
			coreapp.FxModule,
			appmodel.FxModule,
			appcfg.FxModule,
			build.FxModule,
			fx.Populate(
				&scopedEnvVarStore,
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigVersionStore,
				&buildConfigStore,
			),
		)
		diApp.RequireStart()

		workspaceID = "preview-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
		seedPreviewScopedEnvVars(ctx, scopedEnvVarStore, workspaceID)
		service = previewpkg.NewPreviewService(scopedEnvVarStore, appModelStore)
	})

	AfterEach(func() {
		if scopedEnvVarStore != nil {
			Expect(scopedEnvVarStore.DeleteAll(ctx)).NotTo(HaveOccurred())
		}
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("should preview public import with explicit structured scope and overwrite detection", func() {
		_, err := scopedEnvVarStore.Create(ctx, envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeEnvType,
			ScopeValue:  "development",
			Key:         "SHARED_KEY",
			Value:       "development-value",
		})
		Expect(err).NotTo(HaveOccurred())

		preview, err := service.PreviewPublic(ctx, workspaceID, `
# desc: public default scope
# scopeType: workspace
NEW_PUBLIC_KEY="new-public-value # kept"

# scopeType: envType
# scopeValue: development
SHARED_KEY=override-dev-value
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(2))

		Expect(preview.Items[0]).To(Equal(previewpkg.ImportPreviewItem{
			Key:           "NEW_PUBLIC_KEY",
			Value:         "new-public-value # kept",
			Description:   "public default scope",
			DeclaredScope: previewScope(string(envvartypes.ScopeTypeWorkspace), ""),
			EffectiveScope: previewScope(
				string(envvartypes.ScopeTypeWorkspace),
				"",
			),
			Action:      previewpkg.ImportActionNew,
			EffectScope: previewpkg.ImportEffectScopeApplied,
		}))

		Expect(preview.Items[1]).To(Equal(previewpkg.ImportPreviewItem{
			Key:           "SHARED_KEY",
			Value:         "override-dev-value",
			OriginalValue: "development-value",
			DeclaredScope: previewScope(string(envvartypes.ScopeTypeEnvType), "development"),
			EffectiveScope: previewScope(
				string(envvartypes.ScopeTypeEnvType),
				"development",
			),
			Action:      previewpkg.ImportActionOverwrite,
			EffectScope: previewpkg.ImportEffectScopeApplied,
		}))

		Expect(preview.Summary.Total).To(Equal(2))
		Expect(preview.Summary.New).To(Equal(1))
		Expect(preview.Summary.Overwrite).To(Equal(1))
	})

	It("should treat same key in different public scope as new", func() {
		preview, err := service.PreviewPublic(ctx, workspaceID, `
# scopeType: envType
# scopeValue: development
SHARED_KEY=override-dev-value
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))

		item := preview.Items[0]
		Expect(item).To(Equal(previewpkg.ImportPreviewItem{
			Key:            "SHARED_KEY",
			Value:          "override-dev-value",
			DeclaredScope:  previewScope(string(envvartypes.ScopeTypeEnvType), "development"),
			EffectiveScope: previewScope(string(envvartypes.ScopeTypeEnvType), "development"),
			Action:         previewpkg.ImportActionNew,
			EffectScope:    previewpkg.ImportEffectScopeApplied,
		}))

		Expect(preview.Summary.Total).To(Equal(1))
		Expect(preview.Summary.New).To(Equal(1))
		Expect(preview.Summary.Overwrite).To(Equal(0))
	})

	It("should preview env import without scope metadata", func() {
		preview, err := service.PreviewEnv(ctx, environment, `
ENV_ONLY_KEY=env-override-value
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))

		Expect(preview.Items[0]).To(Equal(previewpkg.ImportPreviewItem{
			Key:            "ENV_ONLY_KEY",
			Value:          "env-override-value",
			OriginalValue:  "env-only-value",
			DeclaredScope:  nil,
			EffectiveScope: previewScope(string(envvartypes.ScopeTypeEnv), environment.Name),
			Action:         previewpkg.ImportActionOverwrite,
			EffectScope:    previewpkg.ImportEffectScopeApplied,
		}))

		Expect(preview.Summary.Overwrite).To(Equal(1))
	})

	It("should treat inherited public vars as new in env import preview", func() {
		preview, err := service.PreviewEnv(ctx, environment, `
WORKSPACE_ONLY_KEY=env-level-value
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))

		Expect(preview.Items[0]).To(Equal(previewpkg.ImportPreviewItem{
			Key:            "WORKSPACE_ONLY_KEY",
			Value:          "env-level-value",
			DeclaredScope:  nil,
			EffectiveScope: previewScope(string(envvartypes.ScopeTypeEnv), environment.Name),
			Action:         previewpkg.ImportActionNew,
			EffectScope:    previewpkg.ImportEffectScopeApplied,
		}))

		Expect(preview.Summary.New).To(Equal(1))
		Expect(preview.Summary.Overwrite).To(Equal(0))
	})

	It("should mask original value for sensitive scoped env var overwrite", func() {
		_, err := scopedEnvVarStore.Create(ctx, envvarsstore.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "SENSITIVE_PUBLIC_KEY",
			Value:       "top-secret",
			IsSensitive: true,
		})
		Expect(err).NotTo(HaveOccurred())

		preview, err := service.PreviewPublic(ctx, workspaceID, `
# scopeType: workspace
SENSITIVE_PUBLIC_KEY=updated-secret
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))

		item := preview.Items[0]
		Expect(item.Action).To(Equal(previewpkg.ImportActionOverwrite))
		Expect(item.OriginalValue).To(Equal(envvartypes.SensitiveValueMask))
	})

	It("should detect existing keys in app import preview without scope metadata", func() {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			WorkspaceID: workspaceID,
			EnvVars: []appmodel.Variable{
				{Key: "APP_DEFINED_KEY", Value: "existing-app-value", Description: "existing app key"},
			},
		})

		preview, err := service.PreviewApp(ctx, app, `
APP_DEFINED_KEY=updated-app-value
BKMS_APP_NAME=custom-app-name
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(2))

		Expect(preview.Items[0]).To(Equal(previewpkg.ImportPreviewItem{
			Key:           "APP_DEFINED_KEY",
			Value:         "updated-app-value",
			OriginalValue: "existing-app-value",
			Action:        previewpkg.ImportActionOverwrite,
			EffectScope:   previewpkg.ImportEffectScopeNone,
		}))

		Expect(preview.Items[1]).To(Equal(previewpkg.ImportPreviewItem{
			Key:         "BKMS_APP_NAME",
			Value:       "custom-app-name",
			Action:      previewpkg.ImportActionNew,
			EffectScope: previewpkg.ImportEffectScopeNone,
		}))

		Expect(preview.Summary.New).To(Equal(1))
		Expect(preview.Summary.Overwrite).To(Equal(1))
	})

	It("should mask original value for sensitive app env var overwrite", func() {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			WorkspaceID: workspaceID,
			EnvVars: []appmodel.Variable{
				{Key: "APP_SECRET_KEY", Value: "existing-app-secret", IsSensitive: true},
			},
		})

		preview, err := service.PreviewApp(ctx, app, `
APP_SECRET_KEY=updated-app-secret
`)
		Expect(err).NotTo(HaveOccurred())
		Expect(preview.Items).To(HaveLen(1))

		item := preview.Items[0]
		Expect(item.Action).To(Equal(previewpkg.ImportActionOverwrite))
		Expect(item.OriginalValue).To(Equal(envvartypes.SensitiveValueMask))
	})

	It("should fail fast on invalid key", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
1BAD=bad-key
GOOD_KEY=good-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("invalid env var key"))
		Expect(err.Error()).To(ContainSubstring("1BAD"))
	})

	It("should fail fast on malformed line", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
GOOD_KEY=good-value
MALFORMED_LINE
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("invalid format"))
	})

	It("should fail fast on duplicate key", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
DUP_KEY=v1
DUP_KEY=v2
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("duplicate key"))
		Expect(err.Error()).To(ContainSubstring("DUP_KEY"))
	})

	It("should fail fast on unsupported metadata directive", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
# owner: team-a
GOOD_KEY=good-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("unsupported metadata directive"))
		Expect(err.Error()).To(ContainSubstring("owner"))
	})

	It("should fail fast when desc directive is left dangling at EOF", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
GOOD_KEY=good-value
# desc: missing assignment
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring(`metadata directive must be followed by KEY=VALUE`))
		Expect(err.Error()).To(ContainSubstring("line 3"))
	})

	It("should fail fast when scopeType directive is left dangling at EOF", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
GOOD_KEY=good-value
# scopeType: workspace
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring(`metadata directive must be followed by KEY=VALUE`))
		Expect(err.Error()).To(ContainSubstring("line 3"))
	})

	It("should reject keys longer than the shared CRUD limit", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
`+strings.Repeat("A", 257)+`=too-long-key
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("env var key"))
		Expect(err.Error()).To(ContainSubstring("must be at most 256 characters"))
	})

	It("should reject values longer than the shared CRUD limit", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
GOOD_KEY=`+strings.Repeat("v", 8193)+`
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("env var value"))
		Expect(err.Error()).To(ContainSubstring("must be at most 8192 characters"))
	})

	It("should reject missing scope metadata in public import", func() {
		_, err := service.PreviewPublic(ctx, workspaceID, `
STAGING_KEY=staging-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("scopeType is required in public import"))
	})

	It("should reject scope metadata in env import", func() {
		_, err := service.PreviewEnv(ctx, environment, `
# scopeType: workspace
ENV_ONLY_KEY=env-override-value
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("env import does not allow scope metadata"))
	})

	It("should reject scope metadata in app import", func() {
		app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appConfigFileStore,
			AppConfigFileVersionStore: appConfigVersionStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			WorkspaceID: workspaceID,
		})

		_, err := service.PreviewApp(ctx, app, `
# scopeType: workspace
BKMS_APP_NAME=custom-app-name
`)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, parserpkg.ErrInvalidEnvFileContent)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("app import does not allow scope metadata"))
	})
})

func previewScope(scopeType, scopeValue string) *previewpkg.ImportPreviewScope {
	return &previewpkg.ImportPreviewScope{
		Type:  scopeType,
		Value: scopeValue,
	}
}

func seedPreviewScopedEnvVars(ctx context.Context, store envvarsstore.ScopedEnvVarStore, workspaceID string) {
	_, err := store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeWorkspace,
		ScopeValue:  "",
		Key:         "SHARED_KEY",
		Value:       "workspace-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeWorkspace,
		ScopeValue:  "",
		Key:         "WORKSPACE_ONLY_KEY",
		Value:       "workspace-only-value",
		IsBuiltin:   true,
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeEnvType,
		ScopeValue:  "production",
		Key:         "SHARED_KEY",
		Value:       "envtype-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.Create(ctx, envvarsstore.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeEnvType,
		ScopeValue:  "production",
		Key:         "ENV_TYPE_ONLY_KEY",
		Value:       "envtype-only-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = store.CreateSimpleEnvScopeVar(
		ctx,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"},
		"SHARED_KEY",
		"env-value",
		"",
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = store.CreateSimpleEnvScopeVar(
		ctx,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"},
		"ENV_ONLY_KEY",
		"env-only-value",
		"",
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = store.CreateSimpleEnvScopeVar(
		ctx,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "other-env"},
		"OTHER_ENV_KEY",
		"ignored",
		"",
	)
	Expect(err).NotTo(HaveOccurred())
}
