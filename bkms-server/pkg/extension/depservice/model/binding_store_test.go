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

package model

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("ServiceBinding store", func() {
	var (
		ctx      context.Context
		bindings *ServiceBindingStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		bindings = testutil.NewStore(GinkgoT(), ServiceBindingMeta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(ServiceBindingMeta)).To(Succeed())
	})

	newBinding := func(appID, name string, envMap map[string]bson.ObjectID) *ServiceBinding {
		return &ServiceBinding{
			Name:           name,
			AppID:          appID,
			WorkspaceID:    "ws-1",
			ServiceName:    "redis",
			EnvInstanceMap: envMap,
			EnvVars:        map[string]string{"REDIS_HOST": "${{env.REDIS_HOST}}"},
		}
	}

	It("creates, gets and lists a binding", func() {
		instID := bson.NewObjectID()
		binding := newBinding("app-1", "session", map[string]bson.ObjectID{"prod": instID})
		Expect(bindings.Create(ctx, binding)).To(Succeed())
		Expect(binding.InstanceIDs).To(Equal([]bson.ObjectID{instID}))

		got, err := bindings.Get(ctx, ByBindingName("app-1", "redis", "session"))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal(binding.ID))
		Expect(got.EnvInstanceMap["prod"]).To(Equal(instID))
		Expect(got.InstanceIDs).To(Equal([]bson.ObjectID{instID}))

		list, err := bindings.List(ctx, ByBindingQuery(&BindingQueryOptions{AppID: "app-1", ServiceName: "redis"}))
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
	})

	It("rejects duplicate names in the same app and service", func() {
		err := bindings.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())
		err = bindings.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).To(MatchError(ErrBindingNameExists))
	})

	It("allows the same name under a different app", func() {
		err := bindings.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())
		err = bindings.Create(ctx, newBinding("app-2", "session", nil))
		Expect(err).NotTo(HaveOccurred())
	})

	It("lists by instance id using derived instanceIDs", func() {
		instA := bson.NewObjectID()
		instB := bson.NewObjectID()
		err := bindings.Create(ctx, newBinding("app-1", "session", map[string]bson.ObjectID{
			"prod": instA, "test": instA,
		}))
		Expect(err).NotTo(HaveOccurred())
		err = bindings.Create(ctx, newBinding("app-1", "cache", map[string]bson.ObjectID{
			"prod": instB,
		}))
		Expect(err).NotTo(HaveOccurred())

		list, err := bindings.List(ctx, ByBindingQuery(&BindingQueryOptions{InstanceID: instA}))
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Name).To(Equal("session"))
	})

	It("updates env mapping and env vars", func() {
		err := bindings.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())

		instID := bson.NewObjectID()
		Expect(UpdateBinding(ctx, bindings, "app-1", "redis", "session", &ServiceBindingUpdateData{
			EnvInstanceMap: map[string]bson.ObjectID{"prod": instID},
			EnvVars:        map[string]string{"REDIS_DSN": "redis://x"},
			Description:    "updated",
		})).To(Succeed())

		got, err := bindings.Get(ctx, ByBindingName("app-1", "redis", "session"))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.EnvInstanceMap["prod"]).To(Equal(instID))
		Expect(got.EnvVars["REDIS_DSN"]).To(Equal("redis://x"))
		Expect(got.Description).To(Equal("updated"))
		Expect(got.InstanceIDs).To(Equal([]bson.ObjectID{instID}))
	})

	It("deletes a binding", func() {
		err := bindings.Create(ctx, newBinding("app-1", "session-"+stringx.Random(4), nil))
		Expect(err).NotTo(HaveOccurred())
		name := "to-del"
		err = bindings.Create(ctx, newBinding("app-1", name, nil))
		Expect(err).NotTo(HaveOccurred())
		Expect(bindings.Delete(ctx, ByBindingName("app-1", "redis", name))).To(Succeed())
		_, err = bindings.Get(ctx, ByBindingName("app-1", "redis", name))
		Expect(err).To(MatchError(ErrBindingNotFound))
		Expect(AsNotFoundError(err)).To(BeTrue())

		err = UpdateBinding(ctx, bindings, "app-1", "redis", name, &ServiceBindingUpdateData{})
		Expect(AsNotFoundError(err)).To(BeTrue())
	})

	It("rejects invalid bindings", func() {
		binding := newBinding("app-1", "session", nil)
		binding.WorkspaceID = ""
		Expect(bindings.Create(ctx, binding)).To(MatchError(ContainSubstring("validation failed")))
	})
})
