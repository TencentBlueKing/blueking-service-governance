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

package polaris_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

var _ = Describe("PolarisEnvStateManager", func() {
	var (
		ctx              context.Context
		diApp            *fxtest.App
		appStore         bkmsapp.ApplicationStore
		envService       *env.EnvService
		store            polaris.PolarisConfigStore
		manager          *polaris.PolarisEnvStateManager
		app              *bkmsapp.Application
		environment      *bkmsenv.Environment
		otherEnvironment *bkmsenv.Environment
		thirdEnvironment *bkmsenv.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			polaris.FxModule,
			fx.Populate(
				&appStore,
				&envService,
				&store,
				&manager,
			),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		otherEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		thirdEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	createConfig := func(
		name string, scopeEnvNames []string, states map[string]polaris.PolarisEnvState,
	) *polaris.PolarisConfig {
		config := newTestConfig(app.ID, name, scopeEnvNames, states)
		Expect(store.Create(ctx, config)).To(Succeed())
		for envName, state := range states {
			update := polaris.PolarisEnvStateUpdate{AppliedFields: state.AppliedFields}
			if state.LastError != "" {
				errorMessage := state.LastError
				update.LastError = &errorMessage
			}
			Expect(store.UpsertEnvState(ctx, app.ID, name, envName, update)).To(Succeed())
		}
		stored, err := store.Get(ctx, app.ID, name)
		Expect(err).NotTo(HaveOccurred())
		return stored
	}

	updateAndPrepareDynamicApply := func(
		config *polaris.PolarisConfig, update *polaris.ConfigUpdateData,
	) (*polaris.PolarisConfig, []string) {
		Expect(store.Update(ctx, app.ID, config.Name, update)).To(Succeed())
		updated, err := store.Get(ctx, app.ID, config.Name)
		Expect(err).NotTo(HaveOccurred())
		envNames, err := manager.PrepareDynamicApply(ctx, updated)
		Expect(err).NotTo(HaveOccurred())
		return updated, envNames
	}

	Describe("PrepareDynamicApply", func() {
		It("should preserve applied fields when deployment-required fields differ", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := createConfig(
				"cfg-key-change",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)
			instanceKey := "k2"

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{InstanceKey: &instanceKey},
			)
			Expect(updated.InstanceKey).To(Equal("k2"))
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(Equal(applied))
			Expect(envNames).To(BeEmpty())
		})

		It("should leave an undeployed environment without applied fields", func() {
			config := createConfig(
				"cfg-undeployed",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(nil)},
			)
			servicePort := int32(9090)

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{ServicePort: &servicePort},
			)
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(BeNil())
			Expect(envNames).To(BeEmpty())
		})

		It("should wait for deploy before creating a state for a newly scoped environment", func() {
			applied := redeployFields("k0", "t1", 8080)
			config := createConfig(
				"cfg-add-env",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{
					Scope: &polaris.PatchPolarisScope{
						ScopeType:     component.ScopeTypeEnvironment,
						ScopeEnvNames: []string{environment.Name, otherEnvironment.Name},
					},
				},
			)
			Expect(updated.EnvStates).To(HaveLen(1))
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(Equal(applied))
			Expect(updated.EnvStates).NotTo(HaveKey(otherEnvironment.Name))
			Expect(envNames).To(BeEmpty())
		})

		It("should retain a deployed environment when it leaves scope", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := createConfig(
				"cfg-remove-env",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: envState(applied)},
			)

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{
					Scope: &polaris.PatchPolarisScope{ScopeType: component.ScopeTypeEnvironment},
				},
			)
			Expect(updated.EnvStates).To(HaveKey(environment.Name))
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(Equal(applied))
			Expect(envNames).To(BeEmpty())
		})

		It("should remove undeployed environments and retain deployed ones outside scope", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := createConfig(
				"cfg-remove-undeployed",
				[]string{environment.Name, otherEnvironment.Name, thirdEnvironment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name:      envState(nil),
					otherEnvironment.Name: envState(nil),
					thirdEnvironment.Name: envState(applied),
				},
			)

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{
					Scope: &polaris.PatchPolarisScope{ScopeType: component.ScopeTypeEnvironment},
				},
			)
			Expect(updated.EnvStates).NotTo(HaveKey(environment.Name))
			Expect(updated.EnvStates).NotTo(HaveKey(otherEnvironment.Name))
			Expect(updated.GetEnvState(thirdEnvironment.Name).AppliedFields).To(Equal(applied))
			Expect(envNames).To(BeEmpty())
		})

		It("should reuse a deployed environment when it returns to scope", func() {
			state := envState(redeployFields("k0", "t1", 8080))
			config := createConfig(
				"cfg-readd-env", nil, map[string]polaris.PolarisEnvState{environment.Name: state},
			)

			updated, envNames := updateAndPrepareDynamicApply(
				config,
				&polaris.ConfigUpdateData{
					Scope: &polaris.PatchPolarisScope{
						ScopeType:     component.ScopeTypeEnvironment,
						ScopeEnvNames: []string{environment.Name},
					},
				},
			)
			Expect(updated.GetEnvState(environment.Name).AppliedFields).To(Equal(state.AppliedFields))
			Expect(envNames).To(BeEmpty())
		})

		It("should return an environment whose deployed fields match the config", func() {
			config := createConfig(
				"cfg-direct-apply",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name: envState(redeployFields("k1", "t1", 8080)),
				},
			)

			envNames, err := manager.PrepareDynamicApply(ctx, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(envNames).To(ConsistOf(environment.Name))
		})
	})

	Describe("RecordDynamicApplyResult", func() {
		It("should record an error and clear it after a successful apply", func() {
			config := createConfig(
				"cfg-apply-result",
				[]string{environment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name: envState(redeployFields("k1", "t1", 8080)),
				},
			)

			Expect(manager.RecordDynamicApplyResult(
				ctx, app.ID, config.Name, environment.Name, errors.New("apply failed"),
			)).To(Succeed())
			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.GetEnvState(environment.Name).LastError).To(Equal("apply failed"))

			Expect(manager.RecordDynamicApplyResult(
				ctx, app.ID, config.Name, environment.Name, nil,
			)).To(Succeed())
			stored, err = store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.GetEnvState(environment.Name).LastError).To(BeEmpty())
		})
	})

	Describe("ReconcileAfterDeploy", func() {
		It("should record applied fields and clear the previous error", func() {
			state := envState(nil)
			state.LastError = "previous error"
			config := createConfig(
				"cfg-deployed", []string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: state},
			)
			otherState := envState(nil)
			otherState.LastError = "keep"
			other := createConfig(
				"cfg-other-env", []string{otherEnvironment.Name},
				map[string]polaris.PolarisEnvState{otherEnvironment.Name: otherState},
			)

			Expect(manager.ReconcileAfterDeploy(ctx, app, environment)).To(Succeed())

			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			updatedState := stored.GetEnvState(environment.Name)
			Expect(updatedState.AppliedFields).To(Equal(redeployFields("k1", "t1", 8080)))
			Expect(updatedState.LastError).To(BeEmpty())

			otherStored, err := store.Get(ctx, app.ID, other.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(otherStored.GetEnvState(otherEnvironment.Name).LastError).To(Equal("keep"))
		})

		It("should remove an out-of-scope environment state", func() {
			state := envState(redeployFields("k1", "t1", 8080))
			config := createConfig(
				"cfg-out-of-scope", nil, map[string]polaris.PolarisEnvState{environment.Name: state},
			)

			Expect(manager.ReconcileAfterDeploy(ctx, app, environment)).To(Succeed())
			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
		})
	})

	Describe("ReconcileAfterUninstall", func() {
		It("should remove the uninstalled environment from every config", func() {
			state := envState(redeployFields("k1", "t1", 8080))
			first := createConfig(
				"cfg-uninstall-first", []string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: state},
			)
			second := createConfig(
				"cfg-uninstall-second", []string{environment.Name},
				map[string]polaris.PolarisEnvState{environment.Name: state},
			)

			Expect(manager.ReconcileAfterUninstall(ctx, app, environment.Name)).To(Succeed())
			stored, err := store.Get(ctx, app.ID, first.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
			stored, err = store.Get(ctx, app.ID, second.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
		})

		It("should retain states for other environments", func() {
			applied := redeployFields("k1", "t1", 8080)
			config := createConfig(
				"cfg-retain-other-env",
				[]string{environment.Name, otherEnvironment.Name},
				map[string]polaris.PolarisEnvState{
					environment.Name:      envState(applied),
					otherEnvironment.Name: envState(applied),
				},
			)

			Expect(manager.ReconcileAfterUninstall(ctx, app, environment.Name)).To(Succeed())
			stored, err := store.Get(ctx, app.ID, config.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.EnvStates).NotTo(HaveKey(environment.Name))
			Expect(stored.GetEnvState(otherEnvironment.Name).AppliedFields).To(Equal(applied))
		})
	})
})
