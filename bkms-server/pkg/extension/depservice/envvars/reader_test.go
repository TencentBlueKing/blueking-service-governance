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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("Reader", func() {
	var (
		ctx      context.Context
		insts    *model.ServiceInstanceStore
		bindings *model.ServiceBindingStore
		reader   *depenvvars.Reader
		app      *bkmsapp.Application
	)

	BeforeEach(func() {
		ctx = context.Background()
		insts = testutil.NewStore(GinkgoT(), model.ServiceInstanceMeta)
		bindings = testutil.NewStore(GinkgoT(), model.ServiceBindingMeta)
		reader = depenvvars.NewReader(testutil.NewDB(GinkgoT()))
		app = &bkmsapp.Application{ID: "app-" + stringx.Random(8), Name: "test-app"}
	})

	AfterEach(func() {
		_, err := bindings.DeleteMany(ctx, model.ByBindingQuery(&model.BindingQueryOptions{AppID: app.ID}))
		Expect(err).NotTo(HaveOccurred())
		_, err = insts.DeleteMany(ctx, store.Where("workspaceID", app.ID))
		Expect(err).NotTo(HaveOccurred())
	})

	createInst := func(status model.InstanceStatus, credentials map[string]any) bson.ObjectID {
		inst := &model.ServiceInstance{
			Name:         "inst-" + stringx.Random(8),
			ServiceName:  "redis",
			PlanName:     "default",
			ProviderType: model.ProviderTypeSystemAllocated,
			ScopeType:    model.ScopeTypeWorkspace,
			// 以应用 ID 作为 workspace，便于按应用清理
			WorkspaceID: app.ID,
			Operator:    "tester",
			Status:      status,
			Credentials: credentials,
		}
		Expect(insts.Create(ctx, inst)).To(Succeed())
		return inst.ID
	}

	createBinding := func(name string, envMap map[string]bson.ObjectID, envVars map[string]string) {
		err := bindings.Create(ctx, &model.ServiceBinding{
			Name:           name,
			AppID:          app.ID,
			WorkspaceID:    "ws-test",
			ServiceName:    "redis",
			EnvInstanceMap: envMap,
			EnvVars:        envVars,
		})
		Expect(err).NotTo(HaveOccurred())
	}

	byKey := func(list envvartypes.EnvVariableRichList) map[string]envvartypes.EnvVariableRichItem {
		m := make(map[string]envvartypes.EnvVariableRichItem, len(list.Vars))
		for _, item := range list.Vars {
			m[item.Obj.Key] = item
		}
		return m
	}

	Describe("ListEnvVarsForApp", func() {
		It("returns rendered vars only for the mapped environment", func() {
			testID := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.1"})
			prodID := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.2"})
			createBinding("session", map[string]bson.ObjectID{
				"test": testID,
				"prod": prodID,
			}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListEnvVarsForApp(ctx, envmodel.Environment{Name: "test"}, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(byKey(list)["REDIS_HOST"].Obj.Value).To(Equal("10.0.0.1"))
			Expect(byKey(list)["REDIS_HOST"].Source.SourceValue).To(Equal("session"))

			list, err = reader.ListEnvVarsForApp(ctx, envmodel.Environment{Name: "prod"}, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(byKey(list)["REDIS_HOST"].Obj.Value).To(Equal("10.0.0.2"))
		})

		It("skips bindings that do not map the environment", func() {
			id := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.1"})
			createBinding("session", map[string]bson.ObjectID{"test": id}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListEnvVarsForApp(ctx, envmodel.Environment{Name: "prod"}, app)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(BeEmpty())
		})

		It("returns no variables for a nil application", func() {
			list, err := reader.ListEnvVarsForApp(ctx, envmodel.Environment{Name: "test"}, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(BeEmpty())
		})
	})

	Describe("ListAppVarsForConflicts", func() {
		It("joins unique rendered values across mapped environments into one item per key", func() {
			testID := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.1"})
			prodID := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.2"})
			createBinding("session", map[string]bson.ObjectID{
				"test": testID,
				"prod": prodID,
			}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListAppVarsForConflicts(ctx, "ws-1", app)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(HaveLen(1))
			Expect(list.Vars[0].Obj.Key).To(Equal("REDIS_HOST"))
			Expect(list.Vars[0].Obj.Value).To(Equal("10.0.0.1, 10.0.0.2"))
			Expect(list.Vars[0].Source).To(Equal(envvartypes.ConflictedSource{
				Source:      envvartypes.EnvVarSourceAppDeps,
				SourceValue: "session",
			}))
		})

		It("deduplicates the same instance mapped to multiple environments", func() {
			id := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.1"})
			createBinding("session", map[string]bson.ObjectID{
				"test": id,
				"prod": id,
			}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListAppVarsForConflicts(ctx, "ws-1", app)
			Expect(err).NotTo(HaveOccurred())
			Expect(byKey(list)["REDIS_HOST"].Obj.Value).To(Equal("10.0.0.1"))
		})

		It("still emits keys when no instance is mapped", func() {
			createBinding("session", map[string]bson.ObjectID{}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListAppVarsForConflicts(ctx, "ws-1", app)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(HaveLen(1))
			Expect(list.Vars[0].Obj.Key).To(Equal("REDIS_HOST"))
			Expect(list.Vars[0].Obj.Value).To(BeEmpty())
		})

		It("skips unavailable instances but keeps values from available ones", func() {
			readyID := createInst(model.AvailableStatus, map[string]any{"REDIS_HOST": "10.0.0.1"})
			pendingID := createInst(model.ProvisioningStatus, map[string]any{"REDIS_HOST": "10.0.0.9"})
			createBinding("session", map[string]bson.ObjectID{
				"test": readyID,
				"prod": pendingID,
			}, map[string]string{
				"REDIS_HOST": "${{env.REDIS_HOST}}",
			})

			list, err := reader.ListAppVarsForConflicts(ctx, "ws-1", app)
			Expect(err).NotTo(HaveOccurred())
			Expect(byKey(list)["REDIS_HOST"].Obj.Value).To(Equal("10.0.0.1"))
		})

		It("returns no variables for a nil application", func() {
			list, err := reader.ListAppVarsForConflicts(ctx, "ws-1", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(list.Vars).To(BeEmpty())
		})
	})
})
