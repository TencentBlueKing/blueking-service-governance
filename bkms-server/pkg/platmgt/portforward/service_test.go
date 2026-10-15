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

package portforward

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Service", func() {
	var (
		ctx         context.Context
		service     *Service
		whitelist   *Store
		envSvc      *bkmsenv.EnvService
		workspaceID string
	)

	// newEnvID 创建一个指定类型的环境，返回其 ID
	newEnvID := func(envType bkmsenv.Type) string {
		env := dbfactory.EnvWithOpts(ctx, envSvc, &dbfactory.EnvOpts{WorkspaceID: workspaceID, Type: string(envType)})
		return env.ID.Hex()
	}

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "ws-" + stringx.Random(8)

		diApp := fxtest.New(GinkgoT(), bkmsenv.FxModule, fx.Populate(&envSvc))
		diApp.RequireStart()
		DeferCleanup(diApp.RequireStop)

		db := testutil.NewDB(GinkgoT())
		service = NewService(db)
		whitelist = store.Of(db, Meta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
	})

	Describe("Add", func() {
		It("should add env IDs to whitelist", func() {
			env1, env2 := newEnvID(bkmsenv.TypeTest), newEnvID(bkmsenv.TypeTest)
			Expect(service.Add(ctx, []string{env1, env2})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf(env1, env2))
		})

		It("should skip env IDs already in whitelist and duplicated in input", func() {
			env1, env2, env3 := newEnvID(bkmsenv.TypeTest), newEnvID(bkmsenv.TypeTest), newEnvID(bkmsenv.TypeTest)
			Expect(service.Add(ctx, []string{env1, env2})).To(Succeed())
			Expect(service.Add(ctx, []string{env2, env3, env3})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf(env1, env2, env3))
		})

		It("should handle empty input", func() {
			Expect(service.Add(ctx, []string{})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(BeEmpty())
		})

		It("should reject production environments", func() {
			err := service.Add(ctx, []string{newEnvID(bkmsenv.TypeProduction)})
			Expect(err).To(MatchError(ErrProductionEnvNotAllowed))
		})

		It("should reject environments that do not exist", func() {
			err := service.Add(ctx, []string{bson.NewObjectID().Hex()})
			Expect(err).To(MatchError(ErrEnvNotFound))
		})
	})

	Describe("CheckPermission", func() {
		BeforeEach(func() {
			// 添加测试白名单记录
			Expect(whitelist.CreateMany(ctx, []*WhitelistEntry{{EnvID: "env-id-aaa"}, {EnvID: "env-id-bbb"}})).
				To(Succeed())
		})

		It("should allow access when envID is in whitelist", func() {
			Expect(service.CheckPermission(ctx, "env-id-aaa")).To(Succeed())
			Expect(service.CheckPermission(ctx, "env-id-bbb")).To(Succeed())
		})

		It("should deny access when envID not in whitelist", func() {
			err := service.CheckPermission(ctx, "env-id-ccc")
			Expect(err).To(Equal(ErrPermissionDenied))
		})

		It("should deny access when whitelist is empty", func() {
			// 清空白名单
			Expect(service.Remove(ctx, []string{"env-id-aaa", "env-id-bbb"})).To(Succeed())

			err := service.CheckPermission(ctx, "env-id-aaa")
			Expect(err).To(Equal(ErrPermissionDenied))
		})
	})

	Describe("Remove", func() {
		BeforeEach(func() {
			Expect(whitelist.CreateMany(ctx, []*WhitelistEntry{{EnvID: "env-1"}, {EnvID: "env-2"}, {EnvID: "env-3"}})).
				To(Succeed())
		})

		It("should remove env IDs from whitelist", func() {
			Expect(service.Remove(ctx, []string{"env-2"})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-3"))
		})

		It("should ignore env IDs not in whitelist and empty input", func() {
			Expect(service.Remove(ctx, []string{"env-nonexist"})).To(Succeed())
			Expect(service.Remove(ctx, []string{})).To(Succeed())

			envIDs, err := service.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(envIDs).To(ConsistOf("env-1", "env-2", "env-3"))
		})
	})
})
