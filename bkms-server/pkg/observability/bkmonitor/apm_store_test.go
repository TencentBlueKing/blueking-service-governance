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

package bkmonitor_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var _ = Describe("ApmInstConfigStoreMongo", func() {
	var (
		ctx   context.Context
		diApp *fxtest.App
		store ApmInstConfigStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 清理测试数据
		err := testutil.CleanupCollection("bkmonitor_apm_inst_config")
		Expect(err).NotTo(HaveOccurred())

		// 使用 FxModule 统一注入所有依赖
		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("Create", func() {
		It("should create an APM successfully", func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
			}

			id, err := store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))
			Expect(apm.CreatedAt).NotTo(BeZero())
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			apm1 := &ApmInstConfig{
				WorkspaceID: "workspace-a",
				ApmID:       1001,
				Name:        "APM服务1",
				Token:       "token-001",
				Creator:     "user1",
			}
			apm2 := &ApmInstConfig{
				WorkspaceID: "workspace-a",
				ApmID:       1002,
				Name:        "APM服务2",
				Token:       "token-002",
				Creator:     "user2",
			}
			apm3 := &ApmInstConfig{
				WorkspaceID: "workspace-b",
				ApmID:       1003,
				Name:        "APM服务3",
				Token:       "token-003",
				Creator:     "user3",
			}

			_, err := store.Create(ctx, apm1)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Create(ctx, apm2)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Create(ctx, apm3)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should list APMs by workspace ID", func() {
			apms, err := store.List(ctx, "workspace-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(apms).To(HaveLen(2))
			for _, apm := range apms {
				Expect(apm.WorkspaceID).To(Equal("workspace-a"))
			}

			apms, err = store.List(ctx, "workspace-b")
			Expect(err).NotTo(HaveOccurred())
			Expect(apms).To(HaveLen(1))
			Expect(apms[0].ApmID).To(Equal(int64(1003)))
		})
	})

	Describe("GetByApmID", func() {
		BeforeEach(func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
			}
			_, err := store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get APM by APM ID successfully", func() {
			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apm).NotTo(BeNil())
			Expect(apm.ApmID).To(Equal(int64(1001)))
			Expect(apm.Name).To(Equal("测试APM服务"))
		})
	})

	Describe("GetByEnvID", func() {
		var envID bson.ObjectID

		BeforeEach(func() {
			envID = bson.NewObjectID()
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
				AssociatedEnvs: []EnvInfo{
					{EnvID: envID, EnvName: "测试环境"},
				},
			}
			_, err := store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get APM by environment ID successfully", func() {
			apm, err := store.GetByEnvID(ctx, envID)
			Expect(err).NotTo(HaveOccurred())
			Expect(apm).NotTo(BeNil())
			Expect(apm.ApmID).To(Equal(int64(1001)))
			Expect(apm.AssociatedEnvs).To(HaveLen(1))
			Expect(apm.AssociatedEnvs[0].EnvID).To(Equal(envID))
			Expect(apm.AssociatedEnvs[0].EnvName).To(Equal("测试环境"))
		})
	})

	Describe("Update", func() {
		var apmID bson.ObjectID

		BeforeEach(func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
			}
			var err error
			apmID, err = store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update multiple fields successfully", func() {
			newName := "更新后的APM服务"
			newToken := "new-test-token"
			err := store.Update(ctx, apmID, &ApmInstConfigUpdateData{Name: &newName, Token: &newToken})
			Expect(err).NotTo(HaveOccurred())

			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apm.Name).To(Equal("更新后的APM服务"))
			Expect(apm.Token).To(Equal("new-test-token"))
		})
	})

	Describe("Delete", func() {
		var apmID bson.ObjectID

		BeforeEach(func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
			}
			var err error
			apmID, err = store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete APM successfully", func() {
			err := store.Delete(ctx, apmID)
			Expect(err).NotTo(HaveOccurred())

			// 验证APM已被删除
			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrApmInstConfigNotFound)).To(BeTrue())
			Expect(apm).To(BeNil())
		})
	})

	Describe("BindEnv", func() {
		var apmID bson.ObjectID

		BeforeEach(func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
			}
			var err error
			apmID, err = store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add environment to APM successfully", func() {
			envID := bson.NewObjectID()
			err := store.BindEnv(ctx, apmID, envID, "测试环境")
			Expect(err).NotTo(HaveOccurred())

			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apm.AssociatedEnvs).To(HaveLen(1))
			Expect(apm.AssociatedEnvs[0].EnvID).To(Equal(envID))
			Expect(apm.AssociatedEnvs[0].EnvName).To(Equal("测试环境"))
		})

		It("should not add duplicate environment", func() {
			envID := bson.NewObjectID()

			err := store.BindEnv(ctx, apmID, envID, "测试环境")
			Expect(err).NotTo(HaveOccurred())
			// 重复添加相同环境
			err = store.BindEnv(ctx, apmID, envID, "测试环境")
			Expect(err).NotTo(HaveOccurred())

			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			// $addToSet 保证只有一条
			Expect(apm.AssociatedEnvs).To(HaveLen(1))
		})
	})

	Describe("UnbindEnv", func() {
		var (
			apmID bson.ObjectID
			envID bson.ObjectID
		)

		BeforeEach(func() {
			envID = bson.NewObjectID()
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "测试APM服务",
				Token:       "test-token-001",
				Creator:     "test-user",
				AssociatedEnvs: []EnvInfo{
					{EnvID: envID, EnvName: "测试环境"},
				},
			}
			var err error
			apmID, err = store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should remove environment from APM successfully", func() {
			err := store.UnbindEnv(ctx, apmID, envID)
			Expect(err).NotTo(HaveOccurred())

			apm, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(apm.AssociatedEnvs).To(HaveLen(0))
		})
	})

	Describe("UnbindEnvFromAll", func() {
		var (
			envID  bson.ObjectID
			envID2 bson.ObjectID
		)

		BeforeEach(func() {
			envID = bson.NewObjectID()
			envID2 = bson.NewObjectID()
		})

		It("should remove env from a single APM", func() {
			apm := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "APM服务1",
				Token:       "token-001",
				Creator:     "test-user",
				AssociatedEnvs: []EnvInfo{
					{EnvID: envID, EnvName: "测试环境"},
					{EnvID: envID2, EnvName: "测试环境2"},
				},
			}
			_, err := store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())

			err = store.UnbindEnvFromAll(ctx, envID)
			Expect(err).NotTo(HaveOccurred())

			// 验证 envID 已被移除，envID2 仍然保留
			result, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.AssociatedEnvs).To(HaveLen(1))
			Expect(result.AssociatedEnvs[0].EnvID).To(Equal(envID2))
		})

		It("should remove env from multiple APMs (dirty data scenario)", func() {
			// 模拟脏数据：同一个 env 出现在多个 APM 的 associatedEnvs 列表中
			apm1 := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1001,
				Name:        "APM服务1",
				Token:       "token-001",
				Creator:     "test-user",
				AssociatedEnvs: []EnvInfo{
					{EnvID: envID, EnvName: "测试环境"},
				},
			}
			apm2 := &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       1002,
				Name:        "APM服务2",
				Token:       "token-002",
				Creator:     "test-user",
				AssociatedEnvs: []EnvInfo{
					{EnvID: envID, EnvName: "测试环境"},
					{EnvID: envID2, EnvName: "测试环境2"},
				},
			}
			_, err := store.Create(ctx, apm1)
			Expect(err).NotTo(HaveOccurred())
			_, err = store.Create(ctx, apm2)
			Expect(err).NotTo(HaveOccurred())

			// 一步清理所有关联
			err = store.UnbindEnvFromAll(ctx, envID)
			Expect(err).NotTo(HaveOccurred())

			// 验证 APM1 的 associatedEnvs 已清空
			result1, err := store.GetByApmID(ctx, 1001)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.AssociatedEnvs).To(HaveLen(0))

			// 验证 APM2 的 associatedEnvs 中 envID 被移除，envID2 保留
			result2, err := store.GetByApmID(ctx, 1002)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.AssociatedEnvs).To(HaveLen(1))
			Expect(result2.AssociatedEnvs[0].EnvID).To(Equal(envID2))
		})
	})
})
