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

package workspace

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	genericstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("WorkspaceStore", func() {
	var store *WorkspaceStore
	var ctx context.Context

	var workspaceA, workspaceB Workspace

	listIDs := func(opts ...genericstore.Opt) []string {
		workspaces, err := store.List(ctx, opts...)
		Expect(err).NotTo(HaveOccurred())
		return lo.Map(workspaces, func(ws Workspace, _ int) string { return ws.ID })
	}

	BeforeEach(func() {
		store = testutil.NewStore(GinkgoT(), Meta)
		ctx = context.Background()
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
		ResetLifecycleHooksForTest()

		workspaceA = Workspace{
			ID:                "workspace-" + stringx.Random(6),
			DisplayName:       "测试工作空间 A",
			Description:       "这是测试工作空间 A 的描述",
			ImageRegistryType: registry.ImageRegistryTypeBuiltin,
			BkSystems: BkSystems{
				BkCIProjectID:      "bk-ci-project-1",
				BkBCSProjectID:     "bk-bcs-project-1",
				BkLogProjectID:     "bk-log-project-1",
				BkMonitorProjectID: "bk-monitor-project-1",
				BkRepoProjectID:    "bk-repo-project-1",
			},
			Creator: "admin",
			Updater: "admin",
		}

		workspaceB = Workspace{
			ID:                "workspace-" + stringx.Random(6),
			DisplayName:       "测试工作空间 B",
			Description:       "这是测试工作空间 B 的描述",
			ImageRegistryType: registry.ImageRegistryTypeExternal,
			BkSystems: BkSystems{
				BkCIProjectID:  "bk-ci-project-2",
				BkBCSProjectID: "bk-bcs-project-2",
			},
			Creator: "user1",
			Updater: "user1",
		}
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())
		ResetLifecycleHooksForTest()
	})

	Context("Create Get Update", func() {
		It("should create, get and update workspace successfully", func() {
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(store.Create(ctx, &workspaceB)).To(Succeed())
			Expect(listIDs()).To(ConsistOf(workspaceA.ID, workspaceB.ID))

			ws, err := store.Get(ctx, ByWorkspaceID(workspaceA.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(ws.DisplayName).To(Equal("测试工作空间 A"))
			Expect(ws.Description).To(Equal("这是测试工作空间 A 的描述"))
			Expect(ws.ImageRegistryType).To(Equal(registry.ImageRegistryTypeBuiltin))
			Expect(ws.BkSystems.BkCIProjectID).To(Equal("bk-ci-project-1"))
			Expect(ws.BkSystems.BkBCSProjectID).To(Equal("bk-bcs-project-1"))
			Expect(ws.Creator).To(Equal("admin"))

			workspaceA.DisplayName = "更新后的工作空间 A"
			workspaceA.Description = "更新后的描述"
			workspaceA.ImageRegistryType = registry.ImageRegistryTypeExternal
			workspaceA.BkSystems.BkCIProjectID = "bk-ci-project-3"
			workspaceA.Updater = "admin2"
			// 创建人不在可变字段中，不应被更新
			workspaceA.Creator = "someone-else"
			Expect(Update(ctx, store, &workspaceA)).To(Succeed())

			ws, err = store.Get(ctx, ByWorkspaceID(workspaceA.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(ws.DisplayName).To(Equal("更新后的工作空间 A"))
			Expect(ws.Description).To(Equal("更新后的描述"))
			Expect(ws.ImageRegistryType).To(Equal(registry.ImageRegistryTypeExternal))
			Expect(ws.BkSystems.BkCIProjectID).To(Equal("bk-ci-project-3"))
			Expect(ws.Updater).To(Equal("admin2"))
			Expect(ws.Creator).To(Equal("admin"))
		})

		It("should return ErrWorkspaceNotFound when getting a missing workspace", func() {
			ws, err := store.Get(ctx, ByWorkspaceID("non-existent-workspace-id"))
			Expect(err).To(MatchError(ErrWorkspaceNotFound))
			Expect(ws).To(BeNil())
		})

		It("should reject a duplicate workspace ID", func() {
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())

			duplicateWorkspace := workspaceA
			duplicateWorkspace.DisplayName = "重复 ID 的工作空间"
			Expect(store.Create(ctx, &duplicateWorkspace)).To(MatchError(genericstore.ErrDuplicate))
		})
	})

	Context("KeywordMatches and InState", func() {
		BeforeEach(func() {
			workspaceA.State = StateReady
			workspaceB.State = StateDisabled
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(store.Create(ctx, &workspaceB)).To(Succeed())
		})

		It("should not filter when keyword is empty", func() {
			Expect(listIDs(KeywordMatches(""))).To(HaveLen(2))
		})

		It("should filter by workspace ID (case-insensitive)", func() {
			keyword := workspaceA.ID[0:15]
			Expect(listIDs(KeywordMatches(strings.ToUpper(keyword)))).To(Equal([]string{workspaceA.ID}))
			Expect(listIDs(KeywordMatches(strings.ToLower(keyword)))).To(Equal([]string{workspaceA.ID}))
		})

		It("should filter by display name", func() {
			Expect(listIDs(KeywordMatches("工作空间 A"))).To(Equal([]string{workspaceA.ID}))
			Expect(listIDs(KeywordMatches("空间 B"))).To(Equal([]string{workspaceB.ID}))
			Expect(listIDs(KeywordMatches("工作空间"))).To(HaveLen(2))
			Expect(listIDs(KeywordMatches("workspace-"))).To(HaveLen(2))
			Expect(listIDs(KeywordMatches("non-existent-keyword-xyz"))).To(BeEmpty())
		})

		It("should filter by state and combine with keyword", func() {
			Expect(listIDs(InState(StateReady))).To(Equal([]string{workspaceA.ID}))
			Expect(listIDs(InState(StateDisabled))).To(Equal([]string{workspaceB.ID}))
			Expect(listIDs(InState(StateProcessing))).To(BeEmpty())
			Expect(listIDs(KeywordMatches("测试工作空间"), InState(StateReady))).To(Equal([]string{workspaceA.ID}))
		})
	})

	Context("ListPage with DefaultOrder", func() {
		It("should page workspaces by updatedAt desc", func() {
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(store.Create(ctx, &workspaceB)).To(Succeed())
			// 把 B 的更新时间提前，避免与 A 落在同一毫秒时按 id 排序导致结果不稳定
			Expect(store.Update(
				ctx, map[string]any{"updatedAt": time.Now().Add(-time.Hour)}, ByWorkspaceID(workspaceB.ID),
			)).To(Succeed())
			// 更新 A 使其成为最近更新的工作空间
			Expect(Update(ctx, store, &workspaceA)).To(Succeed())

			page1, err := store.ListPage(ctx, 1, 1, DefaultOrder())
			Expect(err).NotTo(HaveOccurred())
			Expect(page1.Total).To(Equal(int64(2)))
			Expect(page1.HasNext).To(BeTrue())
			Expect(page1.Items[0].ID).To(Equal(workspaceA.ID))

			page2, err := store.ListPage(ctx, 2, 1, DefaultOrder())
			Expect(err).NotTo(HaveOccurred())
			Expect(page2.HasNext).To(BeFalse())
			Expect(page2.Items[0].ID).To(Equal(workspaceB.ID))
		})
	})

	Context("CountByState", func() {
		It("should count workspaces by lifecycle state", func() {
			workspaceA.ID = "ws-filtered-ready"
			workspaceA.DisplayName = "alpha workspace"
			workspaceA.State = StateReady
			workspaceB.ID = "ws-filtered-disabled"
			workspaceB.DisplayName = "alpha disabled workspace"
			workspaceB.State = StateDisabled
			workspaceC := workspaceA
			workspaceC.ID = "ws-filtered-processing"
			workspaceC.DisplayName = "beta workspace"
			workspaceC.State = StateProcessing
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(store.Create(ctx, &workspaceB)).To(Succeed())
			Expect(store.Create(ctx, &workspaceC)).To(Succeed())

			counts, err := CountByState(ctx, store)
			Expect(err).NotTo(HaveOccurred())
			Expect(counts).To(Equal(map[State]int64{StateReady: 1, StateProcessing: 1, StateDisabled: 1}))

			counts, err = CountByState(ctx, store, KeywordMatches("alpha"))
			Expect(err).NotTo(HaveOccurred())
			Expect(counts).To(Equal(map[State]int64{StateReady: 1, StateProcessing: 0, StateDisabled: 1}))
		})

		It("should ignore undefined states", func() {
			workspaceA.State = State("unknown")
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())

			counts, err := CountByState(ctx, store)
			Expect(err).NotTo(HaveOccurred())
			Expect(counts).To(Equal(map[State]int64{StateReady: 0, StateProcessing: 0, StateDisabled: 0}))
		})
	})

	Context("Delete", func() {
		It("should run pre-delete hooks before deleting the workspace", func() {
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			var hookedID string
			Expect(RegisterPreDeleteHook("record", func(_ context.Context, id string) error {
				// 钩子执行时工作空间仍然存在
				Expect(store.Exists(ctx, ByWorkspaceID(id))).To(BeTrue())
				hookedID = id
				return nil
			})).To(BeTrue())

			Expect(Delete(ctx, store, workspaceA.ID)).To(Succeed())
			Expect(hookedID).To(Equal(workspaceA.ID))
			Expect(store.Exists(ctx, ByWorkspaceID(workspaceA.ID))).To(BeFalse())
		})

		It("should keep the workspace when a pre-delete hook fails", func() {
			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(RegisterPreDeleteHook("failing", func(context.Context, string) error {
				return errors.New("cleanup failed")
			})).To(BeTrue())

			Expect(Delete(ctx, store, workspaceA.ID)).To(MatchError(ContainSubstring("cleanup failed")))
			Expect(store.Exists(ctx, ByWorkspaceID(workspaceA.ID))).To(BeTrue())
		})

		It("should not fail when the workspace does not exist", func() {
			Expect(Delete(ctx, store, "non-existent-workspace-id")).To(Succeed())
		})
	})
})

// registryUpdateFields 更新镜像仓库时允许修改的字段
var registryUpdateFields = genericstore.UpdateSelected("registry", "username", "password", "bkCICredentialID")

var _ = Describe("ImageRegistryStore", func() {
	var store *registry.ImageRegistryStore
	var ctx context.Context

	var workspaceID string
	var registryA, registryB registry.ImageRegistry

	BeforeEach(func() {
		var err error

		// Patch global config to set encrypt secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		store = testutil.NewStore(GinkgoT(), registry.ImageRegistryMeta)

		ctx = context.Background()
		workspaceID = "workspace-" + stringx.Random(6)

		registryA = registry.ImageRegistry{
			WorkspaceID:      workspaceID,
			Type:             registry.ImageRegistryTypeBuiltin,
			Registry:         "docker.io",
			Username:         "blueking",
			Password:         "blueking-passwd",
			BkCICredentialID: "",
		}

		registryB = registry.ImageRegistry{
			WorkspaceID:      workspaceID,
			Type:             registry.ImageRegistryTypeExternal,
			Registry:         "harbor.example.com",
			Username:         "admin",
			Password:         "admin123",
			BkCICredentialID: "credential-123",
		}
	})

	Context("Create Get Update", func() {
		It("should create, get and update image registry successfully", func() {
			// Create registryA
			err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Create registryB
			err = store.Create(ctx, &registryB)
			Expect(err).NotTo(HaveOccurred())

			// List
			regs, err := store.List(ctx, registry.ByWorkspace(workspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(regs).To(HaveLen(2))
			for _, reg := range regs {
				Expect(reg.WorkspaceID).To(Equal(workspaceID))
				Expect(
					reg.Type,
				).To(Or(Equal(registry.ImageRegistryTypeBuiltin), Equal(registry.ImageRegistryTypeExternal)))
				// 密码应该被解密
				Expect(reg.Password).To(Or(Equal("admin123"), Equal("blueking-passwd")))
			}

			// GetByWorkspaceAndType - registryA
			reg, err := store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg).NotTo(BeNil())
			Expect(reg.WorkspaceID).To(Equal(workspaceID))
			Expect(reg.Type).To(Equal(registry.ImageRegistryTypeBuiltin))
			Expect(reg.Registry).To(Equal("docker.io"))
			Expect(reg.Username).To(Equal("blueking"))
			// 密码应该被解密
			Expect(reg.Password).To(Equal("blueking-passwd"))
			Expect(reg.BkCICredentialID).To(Equal(""))

			// GetByWorkspaceAndType - registryB
			reg, err = store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeExternal))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg).NotTo(BeNil())
			Expect(reg.WorkspaceID).To(Equal(workspaceID))
			Expect(reg.Type).To(Equal(registry.ImageRegistryTypeExternal))
			Expect(reg.Registry).To(Equal("harbor.example.com"))
			Expect(reg.Username).To(Equal("admin"))
			Expect(reg.Password).To(Equal("admin123"))
			Expect(reg.BkCICredentialID).To(Equal("credential-123"))

			// Update registryA
			registryA.Registry = "gcr.io"
			registryA.Username = "newUser"
			registryA.Password = "newpassword"
			err = store.Update(
				ctx,
				&registryA,
				registryUpdateFields,
				registry.ByWorkspaceAndType(registryA.WorkspaceID, registryA.Type),
			)
			Expect(err).NotTo(HaveOccurred())

			// Get updated registryA
			reg, err = store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg.Registry).To(Equal("gcr.io"))
			Expect(reg.Username).To(Equal("newUser"))
			Expect(reg.Password).To(Equal("newpassword"))
		})
	})

	Context("Get Non-existent Registry", func() {
		It("should return error when registry not found", func() {
			reg, err := store.Get(
				ctx,
				registry.ByWorkspaceAndType("non-existent-workspace", registry.ImageRegistryTypeBuiltin),
			)
			Expect(err).To(HaveOccurred())
			Expect(reg).To(BeNil())
		})
	})

	Context("Create Duplicate Registry", func() {
		It("should return error when creating registry with duplicate workspaceID and type", func() {
			// Create registryA
			err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Try to create another registry with same workspaceID and type
			duplicateRegistry := registryA
			duplicateRegistry.Registry = "another-registry.com"
			err = store.Create(ctx, &duplicateRegistry)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Multiple Workspaces", func() {
		It("should handle registries from different workspaces correctly", func() {
			workspace2ID := "workspace-" + stringx.Random(6)

			// Create registry for workspace1
			err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Create registry for workspace2 with same type
			registry2 := registry.ImageRegistry{
				WorkspaceID: workspace2ID,
				Type:        registry.ImageRegistryTypeBuiltin,
				Registry:    "quay.io",
				Username:    "user2",
				Password:    "pass2",
			}
			err = store.Create(ctx, &registry2)
			Expect(err).NotTo(HaveOccurred())

			// Get registry for workspace1
			reg1, err := store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg1.Registry).To(Equal("docker.io"))

			// Get registry for workspace2
			reg2, err := store.Get(ctx, registry.ByWorkspaceAndType(workspace2ID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg2.Registry).To(Equal("quay.io"))
		})
	})

	Context("Password Encryption", func() {
		It("should encrypt password when creating and decrypt when getting", func() {
			randomPassword := stringx.Random(16)
			// Create registry with password
			registryWithPassword := registry.ImageRegistry{
				WorkspaceID: workspaceID,
				Type:        registry.ImageRegistryTypeBuiltin,
				Registry:    "docker.io",
				Username:    "blueking",
				Password:    randomPassword,
			}

			err := store.Create(ctx, &registryWithPassword)
			Expect(err).NotTo(HaveOccurred())

			// Get registry - password should be decrypted
			reg, err := store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg.Password).To(Equal(randomPassword))
		})

		It("should handle empty password correctly", func() {
			// Create registry without password
			registryWithoutPassword := registry.ImageRegistry{
				WorkspaceID:      workspaceID,
				Type:             registry.ImageRegistryTypeBuiltin,
				Registry:         "docker.io",
				Username:         "blueking",
				Password:         "",
				BkCICredentialID: "credential-456",
			}

			err := store.Create(ctx, &registryWithoutPassword)
			Expect(err).NotTo(HaveOccurred())

			// Get registry - password should remain empty
			reg, err := store.Get(ctx, registry.ByWorkspaceAndType(workspaceID, registry.ImageRegistryTypeBuiltin))
			Expect(err).NotTo(HaveOccurred())
			Expect(reg.Password).To(Equal(""))
			Expect(reg.BkCICredentialID).To(Equal("credential-456"))
		})
	})

	Context("Input Parameter Immutability", func() {
		It("Create should not modify input parameter", func() {
			originalPassword := "test-password-123"
			registry := registry.ImageRegistry{
				WorkspaceID:      workspaceID,
				Type:             registry.ImageRegistryTypeBuiltin,
				Registry:         "docker.io",
				Username:         "testuser",
				Password:         originalPassword,
				BkCICredentialID: "cred-123",
			}

			// 记录原始值
			originalRegistry := registry

			err := store.Create(ctx, &registry)
			Expect(err).NotTo(HaveOccurred())

			// 验证入参没有被修改
			Expect(registry.WorkspaceID).To(Equal(originalRegistry.WorkspaceID))
			Expect(registry.Type).To(Equal(originalRegistry.Type))
			Expect(registry.Registry).To(Equal(originalRegistry.Registry))
			Expect(registry.Username).To(Equal(originalRegistry.Username))
			Expect(
				registry.Password,
			).To(Equal(originalPassword), "Password should not be encrypted in input parameter")
			Expect(registry.BkCICredentialID).To(Equal(originalRegistry.BkCICredentialID))
		})

		It("Update should not modify input parameter", func() {
			// 先创建一个 registry
			err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// 准备更新数据
			originalPassword := "new-password-456"
			updateRegistry := registry.ImageRegistry{
				WorkspaceID:      workspaceID,
				Type:             registry.ImageRegistryTypeBuiltin,
				Registry:         "gcr.io",
				Username:         "newuser",
				Password:         originalPassword,
				BkCICredentialID: "new-cred-456",
			}

			// 记录原始值
			originalRegistry := updateRegistry

			err = store.Update(
				ctx,
				&updateRegistry,
				registryUpdateFields,
				registry.ByWorkspaceAndType(updateRegistry.WorkspaceID, updateRegistry.Type),
			)
			Expect(err).NotTo(HaveOccurred())

			// 验证入参没有被修改
			Expect(updateRegistry.WorkspaceID).To(Equal(originalRegistry.WorkspaceID))
			Expect(updateRegistry.Type).To(Equal(originalRegistry.Type))
			Expect(updateRegistry.Registry).To(Equal(originalRegistry.Registry))
			Expect(updateRegistry.Username).To(Equal(originalRegistry.Username))
			Expect(
				updateRegistry.Password,
			).To(Equal(originalPassword), "Password should not be encrypted in input parameter")
			Expect(updateRegistry.BkCICredentialID).To(Equal(originalRegistry.BkCICredentialID))
		})
	})
})
