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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

type memoryInstStore struct {
	byID map[bson.ObjectID]*model.ServiceInstance
}

func (s *memoryInstStore) Create(_ context.Context, inst *model.ServiceInstance) (bson.ObjectID, error) {
	if inst.ID.IsZero() {
		inst.ID = bson.NewObjectID()
	}
	s.byID[inst.ID] = inst
	return inst.ID, nil
}

func (s *memoryInstStore) Get(_ context.Context, id bson.ObjectID) (*model.ServiceInstance, error) {
	inst, ok := s.byID[id]
	if !ok {
		return nil, errors.New("instance not found")
	}
	return inst, nil
}

func (s *memoryInstStore) ListByIDs(_ context.Context, ids []bson.ObjectID) ([]*model.ServiceInstance, error) {
	result := make([]*model.ServiceInstance, 0, len(ids))
	seen := map[bson.ObjectID]struct{}{}
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if inst, ok := s.byID[id]; ok {
			result = append(result, inst)
		}
	}
	return result, nil
}

func (s *memoryInstStore) List(context.Context, *model.SvcInstQueryOptions) ([]*model.ServiceInstance, error) {
	panic("unexpected")
}

func (s *memoryInstStore) Update(context.Context, bson.ObjectID, *model.SvcInstUpdateData) error {
	panic("unexpected")
}

func (s *memoryInstStore) UpdateConfig(context.Context, bson.ObjectID, map[string]any) error {
	panic("unexpected")
}

func (s *memoryInstStore) PatchConfig(context.Context, bson.ObjectID, map[string]any) error {
	panic("unexpected")
}

func (s *memoryInstStore) UpdateCredentials(context.Context, bson.ObjectID, map[string]any) error {
	panic("unexpected")
}

func (s *memoryInstStore) PatchCredentials(context.Context, bson.ObjectID, map[string]any) error {
	panic("unexpected")
}

func (s *memoryInstStore) UpdateStatus(context.Context, bson.ObjectID, model.InstanceStatus, string) error {
	panic("unexpected")
}
func (s *memoryInstStore) Delete(context.Context, bson.ObjectID) error { panic("unexpected") }
func (s *memoryInstStore) DeleteAll(context.Context) error             { panic("unexpected") }

type memoryBindingStore struct {
	items []*model.ServiceBinding
}

func (s *memoryBindingStore) Create(_ context.Context, binding *model.ServiceBinding) (bson.ObjectID, error) {
	if binding.ID.IsZero() {
		binding.ID = bson.NewObjectID()
	}
	s.items = append(s.items, binding)
	return binding.ID, nil
}

func (s *memoryBindingStore) List(
	_ context.Context,
	opts *model.BindingQueryOptions,
) ([]*model.ServiceBinding, error) {
	var result []*model.ServiceBinding
	for _, b := range s.items {
		if opts != nil && opts.AppID != "" && b.AppID != opts.AppID {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (s *memoryBindingStore) Get(context.Context, string, string, string) (*model.ServiceBinding, error) {
	panic("unexpected")
}

func (s *memoryBindingStore) Update(context.Context, string, string, string, *model.ServiceBindingUpdateData) error {
	panic("unexpected")
}

func (s *memoryBindingStore) Delete(context.Context, string, string, string) error {
	panic("unexpected")
}
func (s *memoryBindingStore) DeleteAll(context.Context) error { panic("unexpected") }

var _ = Describe("Reader", func() {
	var (
		ctx          context.Context
		instStore    *memoryInstStore
		bindingStore *memoryBindingStore
		reader       *depenvvars.Reader
		app          *bkmsapp.Application
	)

	BeforeEach(func() {
		ctx = context.Background()
		instStore = &memoryInstStore{byID: map[bson.ObjectID]*model.ServiceInstance{}}
		bindingStore = &memoryBindingStore{}
		reader = depenvvars.NewReader(instStore, bindingStore)
		app = &bkmsapp.Application{ID: "app-1", Name: "test-app"}
	})

	createInst := func(status model.InstanceStatus, credentials map[string]any) bson.ObjectID {
		id, err := instStore.Create(ctx, &model.ServiceInstance{
			Status:      status,
			Credentials: credentials,
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	createBinding := func(name string, envMap map[string]bson.ObjectID, envVars map[string]string) {
		_, err := bindingStore.Create(ctx, &model.ServiceBinding{
			Name:           name,
			AppID:          app.ID,
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
