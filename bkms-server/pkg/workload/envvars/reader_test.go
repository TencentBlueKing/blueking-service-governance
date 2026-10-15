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

package envvars_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model/initdata"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("UnifiedEnvVarsReader", func() {
	var (
		diApp       *fxtest.App
		db          *mongo.Database
		ctx         context.Context
		workspaceID string
		environment envmodel.Environment
		reader      *envvars.UnifiedEnvVarsReader
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(&reader),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())

		workspaceID = "test-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(envvars.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	expectedEnvBgVarSourceKeyValues := func() []envVarSourceKeyValue {
		return []envVarSourceKeyValue{
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameEnvCluster, Value: "BCS-K8S-00000"},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameEnvName, Value: "prod-env"},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameEnvNS, Value: "prod-ns"},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameEnvType, Value: "production"},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameNodeIP, Value: ""},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNamePodIP, Value: ""},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNamePodName, Value: ""},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
			{Source: envvartypes.EnvVarSourceScopedWorkspace, Key: "SHARED_KEY", Value: "workspace-value"},
			{Source: envvartypes.EnvVarSourceScopedEnvType, Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
			{Source: envvartypes.EnvVarSourceScopedEnvType, Key: "SHARED_KEY", Value: "envtype-value"},
		}
	}

	It("keeps standard and feature environment variable scopes isolated", func() {
		seedScopedEnvVars(ctx, db, workspaceID)
		featureEnv := environment
		featureEnv.Name = "feat-env"
		featureEnv.Kind = envmodel.EnvironmentKindFeature
		featureEnv.OwnerAppID = "feature-app"
		app := &bkmsapp.Application{ID: featureEnv.OwnerAppID, WorkspaceID: workspaceID, Type: bkmsapp.AppTypeTRPC}

		_, err := envvars.CreateSimpleEnvScopeVar(ctx, db, featureEnv, "SHARED_KEY", "feature-value", "")
		Expect(err).NotTo(HaveOccurred())
		vars, err := envvars.BuildAppEnvVars(ctx, app, nil, &featureEnv, reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars.ToMap()).To(HaveKeyWithValue("SHARED_KEY", "feature-value"))
		Expect(vars.ToMap()).NotTo(HaveKey("ENV_ONLY_KEY"))

		sourceVars, err := envvars.BuildAppEnvVars(ctx, app, nil, &environment, reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(sourceVars.ToMap()).To(HaveKeyWithValue("SHARED_KEY", "env-value"))
	})

	It("should list env background vars sorted by source priority", func() {
		seedScopedEnvVars(ctx, db, workspaceID)

		bgVars, err := reader.ListEnvBgVars(ctx, environment)
		Expect(err).NotTo(HaveOccurred())

		Expect(envVarSourceKeyValues(bgVars)).To(Equal(expectedEnvBgVarSourceKeyValues()))
	})

	It("should list app background vars and include env scoped vars", func() {
		seedScopedEnvVars(ctx, db, workspaceID)

		bgVars, err := reader.ListAppBgVars(ctx, environment, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		expectedValues := append(
			expectedEnvBgVarSourceKeyValues(),
			envVarSourceKeyValue{
				Source: envvartypes.EnvVarSourceScopedEnv,
				Key:    "ENV_ONLY_KEY",
				Value:  "env-only-value",
			},
			envVarSourceKeyValue{Source: envvartypes.EnvVarSourceScopedEnv, Key: "SHARED_KEY", Value: "env-value"},
		)
		Expect(envVarSourceKeyValues(bgVars)).To(Equal(expectedValues))
	})

	It("should list app background vars with app-level builtins when app is provided", func() {
		seedScopedEnvVars(ctx, db, workspaceID)

		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
		bgVars, err := reader.ListAppBgVars(ctx, environment, testApp, nil)
		Expect(err).NotTo(HaveOccurred())

		// App-level builtin vars (BKMS_APP_NAME, BKMS_CONTAINER_NAME) are sorted
		// within the builtin source group by key, so they appear at the top.
		appBuiltinVars := []envVarSourceKeyValue{
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameAppName, Value: "test-app"},
			{Source: envvartypes.EnvVarSourceBuiltin, Key: envvars.EnvVarNameContainerName, Value: "main"},
		}
		envScopedVars := []envVarSourceKeyValue{
			{Source: envvartypes.EnvVarSourceScopedEnv, Key: "ENV_ONLY_KEY", Value: "env-only-value"},
			{Source: envvartypes.EnvVarSourceScopedEnv, Key: "SHARED_KEY", Value: "env-value"},
		}
		expectedValues := append(
			appBuiltinVars,
			append(expectedEnvBgVarSourceKeyValues(), envScopedVars...)...,
		)
		Expect(envVarSourceKeyValues(bgVars)).To(Equal(expectedValues))
	})

	It("should list resource limit builtins in app background vars when app model is provided", func() {
		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
		am := &appmodel.AppModel{Workload: appmodel.Workload{
			Resources: map[string]string{"cpu": "100m-200m", "memory": "2Gi"},
		}}

		bgVars, err := reader.ListAppBgVars(ctx, environment, testApp, am)
		Expect(err).NotTo(HaveOccurred())
		builtin := envvartypes.EnvVarSourceBuiltin
		Expect(envVarSourceKeyValues(bgVars)).To(ContainElements(
			envVarSourceKeyValue{Source: builtin, Key: envvars.EnvVarNameCPULimit, Value: "200m"},
			envVarSourceKeyValue{Source: builtin, Key: envvars.EnvVarNameMemoryLimit, Value: "2Gi"},
		))
	})

	It("should list available env vars with duplicate keys deduplicated by priority", func() {
		seedScopedEnvVars(ctx, db, workspaceID)

		envVars, err := reader.ListEnvVars(ctx, environment)
		Expect(err).NotTo(HaveOccurred())

		Expect(envVarKeyValues(envVars)).To(Equal([]envVarKeyValue{
			{Key: envvars.EnvVarNameEnvCluster, Value: "BCS-K8S-00000"},
			{Key: envvars.EnvVarNameEnvName, Value: "prod-env"},
			{Key: envvars.EnvVarNameEnvNS, Value: "prod-ns"},
			{Key: envvars.EnvVarNameEnvType, Value: "production"},
			{Key: envvars.EnvVarNameNodeIP, Value: ""},
			{Key: envvars.EnvVarNamePodIP, Value: ""},
			{Key: envvars.EnvVarNamePodName, Value: ""},
			{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
			{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
			{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
			{Key: "SHARED_KEY", Value: "env-value"},
		}))
	})
	It("should list deployable app vars from scoped env vars and app model vars", func() {
		seedScopedEnvVars(ctx, db, workspaceID)

		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
		am := &appmodel.AppModel{
			Workload: appmodel.Workload{
				EnvVars: []appmodel.Variable{
					{Key: "SHARED_KEY", Value: "app-value"},
					{Key: "APP_ONLY_KEY", Value: "app-only-value"},
				},
			},
		}

		vars, err := reader.ListVars(ctx, environment, testApp, am)
		Expect(err).NotTo(HaveOccurred())

		Expect(envVarKeyValues(vars)).To(Equal([]envVarKeyValue{
			{Key: envvars.EnvVarNameAppName, Value: "test-app"},
			{Key: envvars.EnvVarNameContainerName, Value: "main"},
			{Key: envvars.EnvVarNameEnvCluster, Value: "BCS-K8S-00000"},
			{Key: envvars.EnvVarNameEnvName, Value: "prod-env"},
			{Key: envvars.EnvVarNameEnvNS, Value: "prod-ns"},
			{Key: envvars.EnvVarNameEnvType, Value: "production"},
			{Key: envvars.EnvVarNameNodeIP, Value: ""},
			{Key: envvars.EnvVarNamePodIP, Value: ""},
			{Key: envvars.EnvVarNamePodName, Value: ""},
			{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
			{Key: "SHARED_KEY", Value: "workspace-value"},
			{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
			{Key: "SHARED_KEY", Value: "envtype-value"},
			{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
			{Key: "SHARED_KEY", Value: "env-value"},
			{Key: "SHARED_KEY", Value: "app-value"},
			{Key: "APP_ONLY_KEY", Value: "app-only-value"},
		}))
		Expect(vars.ToMap()["SHARED_KEY"]).To(Equal("app-value"))
	})

	DescribeTable("should inject CPU and memory limits as builtin env vars",
		func(resources map[string]string, appEnvVars []appmodel.Variable, want map[string]string, absent []string) {
			testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
			am := &appmodel.AppModel{
				Workload: appmodel.Workload{
					Resources: resources,
					EnvVars:   appEnvVars,
				},
			}

			vars, err := reader.ListVars(ctx, environment, testApp, am)
			Expect(err).NotTo(HaveOccurred())

			effective := vars.ToMap()
			for key, value := range want {
				Expect(effective).To(HaveKeyWithValue(key, value))
			}
			for _, key := range absent {
				Expect(effective).NotTo(HaveKey(key))
			}

			overridden := lo.SliceToMap(appEnvVars, func(item appmodel.Variable) (string, struct{}) {
				return item.Key, struct{}{}
			})
			byKey := envVarByKey(vars.ToDeduplicatedList())
			for key := range want {
				if _, ok := overridden[key]; ok {
					continue
				}
				Expect(byKey[key].IsBuiltin).To(BeTrue())
				Expect(byKey[key].ValueFrom).To(BeNil())
				Expect(byKey[key].Placeholder).To(BeEmpty())
			}
		},
		Entry("keeps the configured limit unit",
			map[string]string{"cpu": "100m-200m", "memory": "512Mi-2Gi"},
			nil,
			map[string]string{"BKMS_CPU_LIMIT": "200m", "BKMS_MEMORY_LIMIT": "2Gi"},
			nil,
		),
		Entry("preserves a single quantity as both request and limit",
			map[string]string{"cpu": "0.2", "memory": "2048Mi"},
			nil,
			map[string]string{"BKMS_CPU_LIMIT": "0.2", "BKMS_MEMORY_LIMIT": "2048Mi"},
			nil,
		),
		Entry("injects only the resource that is configured",
			map[string]string{"memory": "256Mi"},
			nil,
			map[string]string{"BKMS_MEMORY_LIMIT": "256Mi"},
			[]string{"BKMS_CPU_LIMIT"},
		),
		Entry("omits both variables when resources are unset",
			nil,
			nil,
			nil,
			[]string{"BKMS_CPU_LIMIT", "BKMS_MEMORY_LIMIT"},
		),
		Entry("skips a limit that is not a kubernetes quantity",
			map[string]string{"cpu": "100m-bad", "memory": "2Gi"},
			nil,
			map[string]string{"BKMS_MEMORY_LIMIT": "2Gi"},
			[]string{"BKMS_CPU_LIMIT"},
		),
		Entry("lets an app env var override the builtin limit",
			map[string]string{"cpu": "100m-200m", "memory": "2Gi"},
			[]appmodel.Variable{{Key: "BKMS_CPU_LIMIT", Value: "custom"}},
			map[string]string{"BKMS_CPU_LIMIT": "custom", "BKMS_MEMORY_LIMIT": "2Gi"},
			nil,
		),
	)

	It("lets scoped public vars and app vars override builtin resource limits", func() {
		seedScopedEnvVars(ctx, db, workspaceID)
		err := store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			Key:         "BKMS_CPU_LIMIT",
			Value:       "from-workspace",
		})
		Expect(err).NotTo(HaveOccurred())
		_, err = envvars.CreateSimpleEnvScopeVar(
			ctx, db,
			environment,
			"BKMS_MEMORY_LIMIT",
			"from-env",
			"",
		)
		Expect(err).NotTo(HaveOccurred())

		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
		am := &appmodel.AppModel{
			Workload: appmodel.Workload{
				Resources: map[string]string{"cpu": "100m-200m", "memory": "512Mi-2Gi"},
				EnvVars:   []appmodel.Variable{{Key: "BKMS_CPU_LIMIT", Value: "from-app"}},
			},
		}

		vars, err := reader.ListVars(ctx, environment, testApp, am)
		Expect(err).NotTo(HaveOccurred())
		Expect(vars.ToMap()).To(HaveKeyWithValue("BKMS_CPU_LIMIT", "from-app"))
		Expect(vars.ToMap()).To(HaveKeyWithValue("BKMS_MEMORY_LIMIT", "from-env"))
	})
})

var _ = Describe("BuildEnvConflictedInfoByKeys", func() {
	var (
		diApp       *fxtest.App
		db          *mongo.Database
		ctx         context.Context
		workspaceID string
		environment envmodel.Environment
		reader      *envvars.UnifiedEnvVarsReader
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(&reader),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())

		workspaceID = "test-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
		seedScopedEnvVars(ctx, db, workspaceID)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(envvars.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	It("should build conflict info for env background view", func() {
		conflicts, err := reader.BuildEnvConflictedInfoByKeys(
			ctx,
			[]string{"SHARED_KEY", "MISSING_KEY", "SHARED_KEY"},
			environment,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts["SHARED_KEY"]
		Expect(info.OverrideConflicted).To(BeTrue())
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{
				Source:      envvartypes.EnvVarSourceScopedWorkspace,
				SourceValue: workspaceID,
			},
			{
				Source:      envvartypes.EnvVarSourceScopedEnvType,
				SourceValue: "production",
			},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：workspace-value, envtype-value"))
	})

	It("should mask sensitive values in conflict detail", func() {
		err := store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			ScopeValue:  "",
			Key:         "SENSITIVE_KEY",
			Value:       "sensitive-value",
			IsSensitive: true,
		})
		Expect(err).NotTo(HaveOccurred())

		conflicts, err := reader.BuildEnvConflictedInfoByKeys(
			ctx,
			[]string{"SENSITIVE_KEY"},
			environment,
		)
		Expect(err).NotTo(HaveOccurred())

		info := conflicts["SENSITIVE_KEY"]
		Expect(info.ConflictedDetail).To(Equal("冲突值为：******"))
	})

	It("should display empty runtime builtin value in conflict detail", func() {
		conflicts, err := reader.BuildEnvConflictedInfoByKeys(
			ctx,
			[]string{envvars.EnvVarNamePodIP},
			environment,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts[envvars.EnvVarNamePodIP]
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{Source: envvartypes.EnvVarSourceBuiltin},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：--"))
	})

	It("should return empty map when no valid key is provided", func() {
		conflicts, err := reader.BuildEnvConflictedInfoByKeys(
			ctx,
			[]string{""},
			environment,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(BeEmpty())
	})
})

var _ = Describe("BuildAppConflictedInfoByKeys", func() {
	var (
		diApp       *fxtest.App
		db          *mongo.Database
		ctx         context.Context
		workspaceID string
		reader      *envvars.UnifiedEnvVarsReader
		testApp     *bkmsapp.Application
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(&reader),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())

		workspaceID = "test-workspace-" + stringx.Random(6)
		testApp = &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}
		seedScopedEnvVars(ctx, db, workspaceID)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(envvars.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	It("should build conflict info against all scoped vars in the workspace", func() {
		conflicts, err := reader.BuildAppConflictedInfoByKeys(ctx, []string{"SHARED_KEY"}, workspaceID, testApp, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts["SHARED_KEY"]
		Expect(info.OverrideConflicted).To(BeTrue())
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{
				Source:      envvartypes.EnvVarSourceScopedWorkspace,
				SourceValue: workspaceID,
			},
			{
				Source:      envvartypes.EnvVarSourceScopedEnvType,
				SourceValue: "production",
			},
			{
				Source:      envvartypes.EnvVarSourceScopedEnv,
				SourceValue: "prod-env",
			},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：workspace-value, envtype-value, env-value"))
	})

	It("should include scoped vars from other environments in the same workspace", func() {
		conflicts, err := reader.BuildAppConflictedInfoByKeys(
			ctx, []string{"OTHER_ENV_KEY"}, workspaceID, testApp, nil,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts["OTHER_ENV_KEY"]
		Expect(info.OverrideConflicted).To(BeTrue())
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{
				Source:      envvartypes.EnvVarSourceScopedEnv,
				SourceValue: "other-env",
			},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：ignored"))
	})

	It("should include app-level builtin vars", func() {
		conflicts, err := reader.BuildAppConflictedInfoByKeys(
			ctx,
			[]string{envvars.EnvVarNameAppName},
			workspaceID,
			testApp,
			nil,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts[envvars.EnvVarNameAppName]
		Expect(info.OverrideConflicted).To(BeTrue())
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{Source: envvartypes.EnvVarSourceBuiltin},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：test-app"))
	})

	It("should include resource limit builtin vars when app model is provided", func() {
		am := &appmodel.AppModel{Workload: appmodel.Workload{
			Resources: map[string]string{"cpu": "100m-200m"},
		}}
		conflicts, err := reader.BuildAppConflictedInfoByKeys(
			ctx,
			[]string{envvars.EnvVarNameCPULimit},
			workspaceID,
			testApp,
			am,
		)
		Expect(err).NotTo(HaveOccurred())

		info := conflicts[envvars.EnvVarNameCPULimit]
		Expect(info.OverrideConflicted).To(BeTrue())
		Expect(info.ConflictedSources).To(Equal([]envvartypes.ConflictedSource{
			{Source: envvartypes.EnvVarSourceBuiltin},
		}))
		Expect(info.ConflictedDetail).To(Equal("冲突值为：200m"))
	})
})

// A helper function to seed scoped env vars for testing.
func seedScopedEnvVars(ctx context.Context, db *mongo.Database, workspaceID string) {
	err := store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeWorkspace,
		ScopeValue:  "",
		Key:         "SHARED_KEY",
		Value:       "workspace-value",
	})
	Expect(err).NotTo(HaveOccurred())

	err = store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeWorkspace,
		ScopeValue:  "",
		Key:         "WORKSPACE_ONLY_KEY",
		Value:       "workspace-only-value",
		IsBuiltin:   true,
	})
	Expect(err).NotTo(HaveOccurred())

	err = store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeEnvType,
		ScopeValue:  "production",
		Key:         "SHARED_KEY",
		Value:       "envtype-value",
	})
	Expect(err).NotTo(HaveOccurred())

	err = store.Of(db, envvars.ScopedEnvVarMeta).Create(ctx, &envvars.ScopedEnvVar{
		WorkspaceID: workspaceID,
		ScopeType:   envvartypes.ScopeTypeEnvType,
		ScopeValue:  "production",
		Key:         "ENV_TYPE_ONLY_KEY",
		Value:       "envtype-only-value",
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = envvars.CreateSimpleEnvScopeVar(
		ctx, db,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"},
		"SHARED_KEY",
		"env-value",
		"",
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = envvars.CreateSimpleEnvScopeVar(
		ctx, db,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "prod-env"},
		"ENV_ONLY_KEY",
		"env-only-value",
		"",
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = envvars.CreateSimpleEnvScopeVar(
		ctx, db,
		envmodel.Environment{WorkspaceID: workspaceID, Name: "other-env"},
		"OTHER_ENV_KEY",
		"ignored",
		"",
	)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("UnifiedEnvVarsReader with depInstance (real depenvvars.Reader + fake.Provider)", func() {
	var (
		diApp        *fxtest.App
		db           *mongo.Database
		insts        *depmodel.ServiceInstanceStore
		bindingStore *depmodel.ServiceBindingStore
		ctx          context.Context
		workspaceID  string
		environment  envmodel.Environment
		reader       *envvars.UnifiedEnvVarsReader
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(&reader),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		insts = testutil.NewStore(GinkgoT(), depmodel.ServiceInstanceMeta)
		bindingStore = testutil.NewStore(GinkgoT(), depmodel.ServiceBindingMeta)

		workspaceID = "test-workspace-" + stringx.Random(6)
		environment = envmodel.Environment{
			WorkspaceID: workspaceID,
			Name:        "prod-env",
			Type:        "production",
			Cluster: envmodel.BizCluster{
				ClusterID: "BCS-K8S-00000",
				Namespace: "prod-ns",
			},
		}
		seedScopedEnvVars(ctx, db, workspaceID)

		// Seed fake service definition (含 providerType=user-defined, 对应 fake.Provider)
		Expect(initdata.Do(db)).To(Succeed())
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(envvars.ScopedEnvVarMeta)).NotTo(HaveOccurred())
		Expect(testutil.CleanupMetaColl(depmodel.ServiceInstanceMeta)).NotTo(HaveOccurred())
		Expect(testutil.CleanupMetaColl(depmodel.ServiceBindingMeta)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	// seedDepSvcBinding 创建实例并建立绑定：EnvVars 默认把 Credentials 键按 ${{env.KEY}} 直出。
	seedDepSvcBinding := func(
		appID string,
		envNames []string,
		credentials map[string]any,
		extraEnvVars map[string]string,
	) {
		instName := "fake-inst-" + stringx.Random(6)
		inst := &depmodel.ServiceInstance{
			Name:         instName,
			WorkspaceID:  workspaceID,
			ServiceName:  "fake",
			PlanName:     "default",
			ProviderType: "user-defined",
			ScopeType:    depmodel.ScopeTypeWorkspace,
			Credentials:  credentials,
			Status:       depmodel.AvailableStatus,
			Operator:     "test-operator",
		}
		Expect(insts.Create(ctx, inst)).To(Succeed())
		id := inst.ID

		envVars := make(map[string]string, len(credentials)+len(extraEnvVars))
		for k := range credentials {
			envVars[k] = "${{env." + k + "}}"
		}
		for k, v := range extraEnvVars {
			envVars[k] = v
		}
		envMap := make(map[string]bson.ObjectID, len(envNames))
		for _, name := range envNames {
			envMap[name] = id
		}
		err := bindingStore.Create(ctx, &depmodel.ServiceBinding{
			Name:           "binding-" + stringx.Random(6),
			AppID:          appID,
			WorkspaceID:    workspaceID,
			ServiceName:    "fake",
			EnvInstanceMap: envMap,
			EnvVars:        envVars,
		})
		Expect(err).NotTo(HaveOccurred())
	}

	It("should include depInstance vars in ListAppBgVars sorted by priority", func() {
		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}

		// Credentials 中的键值对会作为环境变量输出
		seedDepSvcBinding(
			testApp.ID,
			[]string{environment.Name},
			map[string]any{"DB_HOST": "mysql.internal", "DB_PORT": 3306, "DB_PWD": "secret"},
			nil,
		)

		bgVars, err := reader.ListAppBgVars(ctx, environment, testApp, nil)
		Expect(err).NotTo(HaveOccurred())

		items := envVarSourceKeyValues(bgVars)

		// depInstance(优先级45) 应出现在 scopedEnv(优先级40) 之后
		scopedEnvIdx := -1
		depInstIdx := -1
		for i, item := range items {
			if item.Source == envvartypes.EnvVarSourceScopedEnv && scopedEnvIdx == -1 {
				scopedEnvIdx = i
			}
			if item.Source == envvartypes.EnvVarSourceAppDeps && depInstIdx == -1 {
				depInstIdx = i
			}
		}
		Expect(scopedEnvIdx).To(BeNumerically(">=", 0))
		Expect(depInstIdx).To(BeNumerically(">", scopedEnvIdx))

		// 验证 Credentials 输出的变量确实存在
		Expect(items).To(ContainElement(envVarSourceKeyValue{
			Source: envvartypes.EnvVarSourceAppDeps,
			Key:    "DB_HOST",
			Value:  "mysql.internal",
		}))
		Expect(items).To(ContainElement(envVarSourceKeyValue{
			Source: envvartypes.EnvVarSourceAppDeps,
			Key:    "DB_PWD",
			Value:  "secret",
		}))
	})

	It("should not return depInstance vars when app is nil in ListAppBgVars", func() {
		seedDepSvcBinding(
			"app-1",
			[]string{environment.Name},
			map[string]any{"DB_HOST": "mysql.internal"},
			nil,
		)

		bgVars, err := reader.ListAppBgVars(ctx, environment, nil, nil)
		Expect(err).NotTo(HaveOccurred())

		items := envVarSourceKeyValues(bgVars)
		for _, item := range items {
			Expect(item.Source).NotTo(Equal(envvartypes.EnvVarSourceAppDeps))
		}
	})

	It("should only return binding vars for the given app", func() {
		app1 := &bkmsapp.Application{ID: "app-1", Name: "test-app-1", Type: bkmsapp.AppTypeTRPC}
		app2 := &bkmsapp.Application{ID: "app-2", Name: "test-app-2", Type: bkmsapp.AppTypeTRPC}

		seedDepSvcBinding(
			app1.ID,
			[]string{environment.Name},
			map[string]any{"APP1_VAR": "value-for-app1"},
			nil,
		)
		seedDepSvcBinding(
			app2.ID,
			[]string{environment.Name},
			map[string]any{"APP2_VAR": "value-for-app2"},
			nil,
		)

		bgVars, err := reader.ListAppBgVars(ctx, environment, app1, nil)
		Expect(err).NotTo(HaveOccurred())

		items := envVarSourceKeyValues(bgVars)
		Expect(items).To(ContainElement(envVarSourceKeyValue{
			Source: envvartypes.EnvVarSourceAppDeps, Key: "APP1_VAR", Value: "value-for-app1",
		}))
		Expect(items).NotTo(ContainElement(envVarSourceKeyValue{
			Source: envvartypes.EnvVarSourceAppDeps, Key: "APP2_VAR",
		}))
	})

	It("should include depInstance vars in ListVars and they can be overridden by appModel vars", func() {
		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}

		seedDepSvcBinding(
			testApp.ID,
			[]string{environment.Name},
			map[string]any{"DB_HOST": "dep-value", "OVERRIDE_ME": "dep-original"},
			nil,
		)

		am := &appmodel.AppModel{
			Workload: appmodel.Workload{
				EnvVars: []appmodel.Variable{
					{Key: "OVERRIDE_ME", Value: "app-override"},
				},
			},
		}

		vars, err := reader.ListVars(ctx, environment, testApp, am)
		Expect(err).NotTo(HaveOccurred())

		Expect(vars.ToMap()["OVERRIDE_ME"]).To(Equal("app-override"))
		Expect(vars.ToMap()["DB_HOST"]).To(Equal("dep-value"))
	})

	It("should include depInstance vars in BuildAppConflictedInfoByKeys", func() {
		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}

		seedDepSvcBinding(
			testApp.ID,
			[]string{environment.Name},
			map[string]any{"SHARED_KEY": "dep-instance-value"},
			nil,
		)

		conflicts, err := reader.BuildAppConflictedInfoByKeys(ctx, []string{"SHARED_KEY"}, workspaceID, testApp, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(conflicts).To(HaveLen(1))

		info := conflicts["SHARED_KEY"]
		Expect(lo.ContainsBy(info.ConflictedSources, func(cs envvartypes.ConflictedSource) bool {
			return cs.Source == envvartypes.EnvVarSourceAppDeps
		})).To(BeTrue())
		Expect(info.OverrideConflicted).To(BeTrue())
	})

	It("should only inject vars for envs mapped on the binding", func() {
		testApp := &bkmsapp.Application{ID: "app-1", Name: "test-app", Type: bkmsapp.AppTypeTRPC}

		seedDepSvcBinding(
			testApp.ID,
			[]string{"test-env"},
			map[string]any{"TEST_ONLY_VAR": "test-env-value"},
			nil,
		)

		// production 环境未出现在 EnvInstanceMap 中，不应注入
		bgVars, err := reader.ListAppBgVars(ctx, environment, testApp, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(envVarSourceKeyValues(bgVars)).NotTo(ContainElement(envVarSourceKeyValue{Key: "TEST_ONLY_VAR"}))

		// test 环境已映射，应注入
		testEnv := envmodel.Environment{WorkspaceID: workspaceID, Name: "test-env", Type: "test"}
		bgVars, err = reader.ListAppBgVars(ctx, testEnv, testApp, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(envVarSourceKeyValues(bgVars)).To(ContainElement(envVarSourceKeyValue{
			Source: envvartypes.EnvVarSourceAppDeps, Key: "TEST_ONLY_VAR", Value: "test-env-value",
		}))
	})
})

type envVarSourceKeyValue struct {
	Source envvartypes.EnvVarSource
	Key    string
	Value  string
}

// envVarSourceKeyValues extracts source-key-value tuples for easier full-list assertions in tests.
func envVarSourceKeyValues(list envvartypes.EnvVariableRichList) []envVarSourceKeyValue {
	return lo.Map(list.Vars, func(item envvartypes.EnvVariableRichItem, _ int) envVarSourceKeyValue {
		return envVarSourceKeyValue{
			Source: item.Source.Source,
			Key:    item.Obj.Key,
			Value:  item.Obj.Value,
		}
	})
}
