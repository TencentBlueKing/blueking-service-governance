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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ServiceBindingStoreMongo", func() {
	var (
		ctx   context.Context
		store ServiceBindingStore
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		store, err = NewServiceBindingStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
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
		id, err := store.Create(ctx, newBinding("app-1", "session", map[string]bson.ObjectID{
			"prod": instID,
		}))
		Expect(err).NotTo(HaveOccurred())
		Expect(id.IsZero()).To(BeFalse())

		got, err := store.Get(ctx, "app-1", "redis", "session")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.ID).To(Equal(id))
		Expect(got.EnvInstanceMap["prod"]).To(Equal(instID))
		Expect(got.InstanceIDs).To(Equal([]bson.ObjectID{instID}))

		list, err := store.List(ctx, &BindingQueryOptions{AppID: "app-1", ServiceName: "redis"})
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
	})

	It("rejects duplicate names in the same app and service", func() {
		_, err := store.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())
		_, err = store.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).To(MatchError(ErrBindingNameExists))
	})

	It("allows the same name under a different app", func() {
		_, err := store.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())
		_, err = store.Create(ctx, newBinding("app-2", "session", nil))
		Expect(err).NotTo(HaveOccurred())
	})

	It("lists by instance id using derived instanceIDs", func() {
		instA := bson.NewObjectID()
		instB := bson.NewObjectID()
		_, err := store.Create(ctx, newBinding("app-1", "session", map[string]bson.ObjectID{
			"prod": instA, "test": instA,
		}))
		Expect(err).NotTo(HaveOccurred())
		_, err = store.Create(ctx, newBinding("app-1", "cache", map[string]bson.ObjectID{
			"prod": instB,
		}))
		Expect(err).NotTo(HaveOccurred())

		list, err := store.List(ctx, &BindingQueryOptions{InstanceID: instA})
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Name).To(Equal("session"))
	})

	It("updates env mapping and env vars", func() {
		_, err := store.Create(ctx, newBinding("app-1", "session", nil))
		Expect(err).NotTo(HaveOccurred())

		instID := bson.NewObjectID()
		Expect(store.Update(ctx, "app-1", "redis", "session", &ServiceBindingUpdateData{
			EnvInstanceMap: map[string]bson.ObjectID{"prod": instID},
			EnvVars:        map[string]string{"REDIS_DSN": "redis://x"},
			Description:    "updated",
		})).To(Succeed())

		got, err := store.Get(ctx, "app-1", "redis", "session")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.EnvInstanceMap["prod"]).To(Equal(instID))
		Expect(got.EnvVars["REDIS_DSN"]).To(Equal("redis://x"))
		Expect(got.Description).To(Equal("updated"))
		Expect(got.InstanceIDs).To(Equal([]bson.ObjectID{instID}))
	})

	It("deletes a binding", func() {
		_, err := store.Create(ctx, newBinding("app-1", "session-"+stringx.Random(4), nil))
		Expect(err).NotTo(HaveOccurred())
		name := "to-del"
		_, err = store.Create(ctx, newBinding("app-1", name, nil))
		Expect(err).NotTo(HaveOccurred())
		Expect(store.Delete(ctx, "app-1", "redis", name)).To(Succeed())
		_, err = store.Get(ctx, "app-1", "redis", name)
		Expect(AsNotFoundError(err)).To(BeTrue())
	})
})
