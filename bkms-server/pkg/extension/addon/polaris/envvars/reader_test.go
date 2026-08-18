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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Reader", func() {
	var (
		ctx         context.Context
		diApp       *fxtest.App
		appStore    bkmsapp.ApplicationStore
		envService  *env.EnvService
		store       polaris.PolarisConfigStore
		reader      *polarisenvvars.Reader
		app         *bkmsapp.Application
		environment *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(&appStore, &envService, &store, &reader),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		config := &polaris.PolarisConfig{
			AppID: app.ID,
			Properties: polaris.Properties{
				InstanceKey:      "mypolaris",
				PolarisName:      "test-service",
				PolarisNamespace: "Test",
				PolarisToken:     "test-token-12345",
				ServicePort:      8080,
			},
			ScopeEnvNames: []string{environment.Name},
		}
		Expect(store.Create(ctx, config)).To(Succeed())
	})

	AfterEach(func() {
		Expect(store.DeleteByApp(ctx, app.ID)).To(Succeed())
		diApp.RequireStop()
	})

	It("lists sensitive token and service port variables for a matching environment", func() {
		list, err := reader.ListEnvVarsForApp(ctx, *environment, app)
		Expect(err).NotTo(HaveOccurred())

		byKey := make(map[string]envvartypes.EnvVariableObj, len(list.Vars))
		for _, item := range list.Vars {
			byKey[item.Obj.Key] = item.Obj
			Expect(item.Source.Source).To(Equal(envvartypes.EnvVarSourcePolaris))
		}
		Expect(byKey).To(HaveKey("mypolaris_polarisToken"))
		Expect(byKey["mypolaris_polarisToken"].Value).To(Equal("test-token-12345"))
		Expect(byKey["mypolaris_polarisToken"].IsSensitive).To(BeTrue())
		Expect(byKey).To(HaveKey("mypolaris_serviceport"))
		Expect(byKey["mypolaris_serviceport"].Value).To(Equal(strconv.Itoa(8080)))
		Expect(byKey["mypolaris_serviceport"].IsSensitive).To(BeFalse())
	})

	It("returns no variables when the config is outside the environment scope", func() {
		otherEnv := *environment
		otherEnv.Name = "another-env"

		list, err := reader.ListEnvVarsForApp(ctx, otherEnv, app)
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Vars).To(BeEmpty())
	})

	It("includes all app configs when listing variables for conflicts", func() {
		list, err := reader.ListAppVarsForConflicts(ctx, app.WorkspaceID, app)
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Vars).To(HaveLen(2))
	})

	It("returns no variables for a nil application", func() {
		list, err := reader.ListEnvVarsForApp(ctx, *environment, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Vars).To(BeEmpty())
	})

	It("returns no conflict variables for a nil application", func() {
		list, err := reader.ListAppVarsForConflicts(ctx, app.WorkspaceID, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Vars).To(BeEmpty())
	})

	Context("with an immediate-register config in the same environment", func() {
		BeforeEach(func() {
			immediate := &polaris.PolarisConfig{
				AppID: app.ID,
				Name:  "cfg-immediate",
				Properties: polaris.Properties{
					InstanceKey:      "immediatepolaris",
					PolarisName:      "immediate-service",
					PolarisNamespace: "Test",
					PolarisToken:     "immediate-token",
					ServicePort:      9090,
					RegisterMode:     polaris.RegisterModeImmediate,
				},
				ScopeEnvNames: []string{environment.Name},
			}
			Expect(store.Create(ctx, immediate)).To(Succeed())
		})

		It("produces no environment variables of its own", func() {
			list, err := reader.ListEnvVarsForApp(ctx, *environment, app)
			Expect(err).NotTo(HaveOccurred())

			keys := make([]string, 0, len(list.Vars))
			for _, item := range list.Vars {
				keys = append(keys, item.Obj.Key)
			}
			Expect(keys).To(ConsistOf("mypolaris_polarisToken", "mypolaris_serviceport"))
		})

		It("stays out of conflict detection", func() {
			list, err := reader.ListAppVarsForConflicts(ctx, app.WorkspaceID, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(HaveLen(2))
		})
	})
})
