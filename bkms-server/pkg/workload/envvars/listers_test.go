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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ListBuiltinVars", func() {
	var diApp *fxtest.App
	var store envvars.ScopedEnvVarStore
	var ctx context.Context
	var workspaceID string
	var otherWorkspaceID string
	var environment envmodel.Environment

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			envvars.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		workspaceID = "test-workspace-" + stringx.Random(6)
		otherWorkspaceID = "test-workspace-" + stringx.Random(6)
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
		Expect(store.DeleteAll(ctx)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	It("should list environment built-in vars without app-related vars", func() {
		_, err := store.Create(ctx, envvars.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			ScopeValue:  "",
			Key:         "WORKSPACE_BUILTIN",
			Value:       "workspace-builtin-value",
			Description: "workspace builtin",
			IsBuiltin:   true,
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = store.Create(ctx, envvars.ScopedEnvVar{
			WorkspaceID: workspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			ScopeValue:  "",
			Key:         "WORKSPACE_DEFINED",
			Value:       "workspace-defined-value",
			IsSensitive: true,
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = store.Create(ctx, envvars.ScopedEnvVar{
			WorkspaceID: otherWorkspaceID,
			ScopeType:   envvartypes.ScopeTypeWorkspace,
			ScopeValue:  "",
			Key:         "OTHER_WORKSPACE_BUILTIN",
			Value:       "other-workspace-builtin-value",
			IsBuiltin:   true,
		})
		Expect(err).NotTo(HaveOccurred())

		builtinVars, err := envvars.ListBuiltinVars(ctx, store, environment, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(envVarKeyValues(builtinVars)).To(Equal([]envVarKeyValue{
			{Key: envvars.EnvVarNameEnvType, Value: "production"},
			{Key: envvars.EnvVarNameEnvName, Value: "prod-env"},
			{Key: envvars.EnvVarNameEnvNS, Value: "prod-ns"},
			{Key: envvars.EnvVarNameEnvCluster, Value: "BCS-K8S-00000"},
			{Key: "WORKSPACE_BUILTIN", Value: "workspace-builtin-value"},
			{Key: envvars.EnvVarNamePodIP, Value: ""},
			{Key: envvars.EnvVarNamePodName, Value: ""},
			{Key: envvars.EnvVarNameNodeIP, Value: ""},
		}))

		varsByKey := envVarByKey(builtinVars)
		Expect(varsByKey[envvars.EnvVarNamePodIP].Placeholder).To(Equal("__#VAR_PLACEHOLDER#__BKMS_POD_IP__"))
		Expect(varsByKey[envvars.EnvVarNamePodIP].ValueFrom.FieldRef.FieldPath).To(Equal("status.podIP"))
		Expect(varsByKey[envvars.EnvVarNamePodName].Placeholder).To(Equal("__#VAR_PLACEHOLDER#__BKMS_POD_NAME__"))
		Expect(varsByKey[envvars.EnvVarNamePodName].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.name"))
		Expect(varsByKey[envvars.EnvVarNameNodeIP].Placeholder).To(Equal("__#VAR_PLACEHOLDER#__BKMS_NODE_IP__"))
		Expect(varsByKey[envvars.EnvVarNameNodeIP].ValueFrom.FieldRef.FieldPath).To(Equal("status.hostIP"))
	})

	It("should append app-related built-in vars when app is provided", func() {
		app := &bkmsapp.Application{
			Name: "demo-app",
			Type: bkmsapp.AppTypeTRPC,
		}

		builtinVars, err := envvars.ListBuiltinVars(ctx, store, environment, app)
		Expect(err).NotTo(HaveOccurred())
		Expect(envVarKeyValues(builtinVars)).To(Equal([]envVarKeyValue{
			{Key: envvars.EnvVarNameEnvType, Value: "production"},
			{Key: envvars.EnvVarNameEnvName, Value: "prod-env"},
			{Key: envvars.EnvVarNameEnvNS, Value: "prod-ns"},
			{Key: envvars.EnvVarNameEnvCluster, Value: "BCS-K8S-00000"},
			{Key: envvars.EnvVarNamePodIP, Value: ""},
			{Key: envvars.EnvVarNamePodName, Value: ""},
			{Key: envvars.EnvVarNameNodeIP, Value: ""},
			{Key: envvars.EnvVarNameAppName, Value: "demo-app"},
			{Key: envvars.EnvVarNameContainerName, Value: defaults.WorkloadMainContainerName},
		}))
	})
})

var _ = Describe("EnvVariableRichList", func() {
	Describe("ToDeduplicatedList", func() {
		It("should keep only the last item for duplicate keys while preserving effective item order", func() {
			vars := envvartypes.EnvVariableRichList{
				Vars: []envvartypes.EnvVariableRichItem{
					{
						Obj:    envvartypes.EnvVariableObj{Key: "SHARED_KEY", Value: "workspace-value"},
						Source: envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceScopedWorkspace},
					},
					{Obj: envvartypes.EnvVariableObj{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"}},
					{
						Obj:    envvartypes.EnvVariableObj{Key: "SHARED_KEY", Value: "envtype-value"},
						Source: envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceScopedEnvType},
					},
					{Obj: envvartypes.EnvVariableObj{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"}},
					{
						Obj:    envvartypes.EnvVariableObj{Key: "SHARED_KEY", Value: "env-value"},
						Source: envvartypes.ConflictedSource{Source: envvartypes.EnvVarSourceScopedEnv},
					},
					{Obj: envvartypes.EnvVariableObj{Key: "ENV_ONLY_KEY", Value: "env-only-value"}},
				},
			}

			deduplicatedVars := vars.ToDeduplicatedList()

			Expect(envVarKeyValues(deduplicatedVars.GetDataList())).To(Equal([]envVarKeyValue{
				{Key: "WORKSPACE_ONLY_KEY", Value: "workspace-only-value"},
				{Key: "ENV_TYPE_ONLY_KEY", Value: "envtype-only-value"},
				{Key: "SHARED_KEY", Value: "env-value"},
				{Key: "ENV_ONLY_KEY", Value: "env-only-value"},
			}))
			Expect(deduplicatedVars.Vars[2].Source.Source).To(Equal(envvartypes.EnvVarSourceScopedEnv))
		})
	})
})

type envVarKeyValue struct {
	Key   string
	Value string
}

// A helper function to extract key-value pairs from EnvVariableList for easier assertion in tests.
func envVarKeyValues(vars envvartypes.EnvVariableList) []envVarKeyValue {
	return lo.Map(vars, func(item envvartypes.EnvVariableObj, _ int) envVarKeyValue {
		return envVarKeyValue{Key: item.Key, Value: item.Value}
	})
}

// A helper function to convert EnvVariableList to a map for easier lookup by key in tests.
func envVarByKey(vars envvartypes.EnvVariableList) map[string]envvartypes.EnvVariableObj {
	result := make(map[string]envvartypes.EnvVariableObj, len(vars))
	for _, item := range vars {
		result[item.Key] = item
	}
	return result
}
