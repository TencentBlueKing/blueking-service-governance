package workspace

import (
	"context"
	"strings"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("WorkspaceStore", func() {
	var store WorkspaceStore
	var ctx context.Context

	var workspaceA, workspaceB Workspace

	BeforeEach(func() {
		var err error

		store, err = NewWorkspaceStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		Expect(cleanupWorkspaceCollection(ctx)).To(Succeed())

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
		Expect(cleanupWorkspaceCollection(ctx)).To(Succeed())
	})

	Context("Create List Get Update", func() {
		It("should create, list, get and update workspace successfully", func() {
			// Create workspaceA
			err := store.Create(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())

			// Create workspaceB
			err = store.Create(ctx, &workspaceB)
			Expect(err).NotTo(HaveOccurred())

			// List - should return 2 workspaces
			workspaces, err := store.List(ctx, &ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(2))

			// Get workspaceA
			ws, err := store.Get(ctx, workspaceA.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ws).NotTo(BeNil())
			Expect(ws.ID).To(Equal(workspaceA.ID))
			Expect(ws.DisplayName).To(Equal("测试工作空间 A"))
			Expect(ws.Description).To(Equal("这是测试工作空间 A 的描述"))
			Expect(ws.ImageRegistryType).To(Equal(registry.ImageRegistryTypeBuiltin))
			Expect(ws.BkSystems.BkCIProjectID).To(Equal("bk-ci-project-1"))
			Expect(ws.BkSystems.BkBCSProjectID).To(Equal("bk-bcs-project-1"))
			Expect(ws.Creator).To(Equal("admin"))
			Expect(ws.CreatedAt).NotTo(BeZero())
			Expect(ws.UpdatedAt).NotTo(BeZero())

			// Update workspaceA
			workspaceA.DisplayName = "更新后的工作空间 A"
			workspaceA.Description = "更新后的描述"
			workspaceA.ImageRegistryType = registry.ImageRegistryTypeExternal
			workspaceA.BkSystems.BkCIProjectID = "bk-ci-project-3"
			workspaceA.Updater = "admin2"
			err = store.Update(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())

			// Get updated workspaceA
			ws, err = store.Get(ctx, workspaceA.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(ws.DisplayName).To(Equal("更新后的工作空间 A"))
			Expect(ws.Description).To(Equal("更新后的描述"))
			Expect(ws.ImageRegistryType).To(Equal(registry.ImageRegistryTypeExternal))
			Expect(ws.BkSystems.BkCIProjectID).To(Equal("bk-ci-project-3"))
			Expect(ws.Updater).To(Equal("admin2"))
		})
	})

	Context("Get Non-existent Workspace", func() {
		It("should return error when workspace not found", func() {
			ws, err := store.Get(ctx, "non-existent-workspace-id")
			Expect(err).To(HaveOccurred())
			Expect(ws).To(BeNil())
		})
	})

	Context("Create Duplicate Workspace ID", func() {
		It("should return error when creating workspace with duplicate ID", func() {
			// Create workspaceA
			err := store.Create(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())

			// Try to create another workspace with same ID
			duplicateWorkspace := workspaceA
			duplicateWorkspace.DisplayName = "重复 ID 的工作空间"
			err = store.Create(ctx, &duplicateWorkspace)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate key error collection"))
		})
	})

	Context("Empty List", func() {
		It("should return empty list when no workspaces exist", func() {
			workspaces, err := store.List(ctx, &ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(BeEmpty())
		})
	})

	Context("List with Keyword Search", func() {
		BeforeEach(func() {
			// Create workspaceA and workspaceB for search tests
			err := store.Create(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())
			err = store.Create(ctx, &workspaceB)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return all workspaces when keyword is empty", func() {
			workspaces, err := store.List(ctx, &ListOptions{Keyword: ""})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(2))
		})

		It("should filter by workspace ID (case-insensitive)", func() {
			// 使用部分 ID 搜索 (转换为大写测试忽略大小写)
			keyword := workspaceA.ID[0:15]
			// 测试大写搜索
			workspaces, err := store.List(ctx, &ListOptions{Keyword: strings.ToUpper(keyword)})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].ID).To(Equal(workspaceA.ID))

			// 测试小写搜索
			workspaces, err = store.List(ctx, &ListOptions{Keyword: strings.ToLower(keyword)})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].ID).To(Equal(workspaceA.ID))
		})

		It("should filter by display name (case-insensitive)", func() {
			// 搜索 "工作空间 A"
			workspaces, err := store.List(ctx, &ListOptions{Keyword: "工作空间 A"})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].DisplayName).To(Equal("测试工作空间 A"))

			// 搜索 "工作空间 B"
			workspaces, err = store.List(ctx, &ListOptions{Keyword: "空间 B"})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].DisplayName).To(Equal("测试工作空间 B"))

			// 搜索 "工作空间" 应该返回两个
			workspaces, err = store.List(ctx, &ListOptions{Keyword: "工作空间"})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(2))
		})

		It("should return empty list when keyword matches nothing", func() {
			workspaces, err := store.List(ctx, &ListOptions{Keyword: "non-existent-keyword-xyz"})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(BeEmpty())
		})

		It("should match partial ID", func() {
			// 使用 workspace- 前缀搜索，应该返回两个工作空间
			workspaces, err := store.List(ctx, &ListOptions{Keyword: "workspace-"})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(2))
		})
	})

	Context("List with State filter", func() {
		BeforeEach(func() {
			workspaceA.State = StateReady
			workspaceB.State = StateDisabled

			err := store.Create(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())
			err = store.Create(ctx, &workspaceB)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return only workspaces matching the specified state", func() {
			stateReady := StateReady
			workspaces, err := store.List(ctx, &ListOptions{State: &stateReady})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].ID).To(Equal(workspaceA.ID))
			Expect(workspaces[0].State).To(Equal(StateReady))
		})

		It("should filter by Disabled state", func() {
			stateDisabled := StateDisabled
			workspaces, err := store.List(ctx, &ListOptions{State: &stateDisabled})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].ID).To(Equal(workspaceB.ID))
			Expect(workspaces[0].State).To(Equal(StateDisabled))
		})

		It("should return empty list when no workspaces match the state", func() {
			stateProcessing := StateProcessing
			workspaces, err := store.List(ctx, &ListOptions{State: &stateProcessing})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(BeEmpty())
		})

		It("should return all workspaces when state is nil", func() {
			workspaces, err := store.List(ctx, &ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(2))
		})

		It("should combine state filter with keyword filter", func() {
			stateReady := StateReady
			// 关键词匹配两个，但 state 过滤后只剩 workspaceA
			workspaces, err := store.List(ctx, &ListOptions{Keyword: "测试工作空间", State: &stateReady})
			Expect(err).NotTo(HaveOccurred())
			Expect(workspaces).To(HaveLen(1))
			Expect(workspaces[0].ID).To(Equal(workspaceA.ID))
		})
	})

	Context("ListWithPagination", func() {
		It("should return paginated workspaces with total count", func() {
			workspaceA.State = StateReady
			workspaceB.State = StateReady
			err := store.Create(ctx, &workspaceA)
			Expect(err).NotTo(HaveOccurred())
			err = store.Create(ctx, &workspaceB)
			Expect(err).NotTo(HaveOccurred())

			pagedStore, ok := store.(*WorkspaceStoreMongo)
			Expect(ok).To(BeTrue())

			page1, total, err := pagedStore.ListWithPagination(ctx, &ListPageOptions{
				ListOptions: ListOptions{},
				Page:        1,
				PageSize:    1,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(page1).To(HaveLen(1))

			page2, total, err := pagedStore.ListWithPagination(ctx, &ListPageOptions{
				ListOptions: ListOptions{},
				Page:        2,
				PageSize:    1,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(page2).To(HaveLen(1))
			Expect(page1[0].ID).NotTo(Equal(page2[0].ID))
		})
	})

	Context("CountByState", func() {
		It("should count workspaces grouped by lifecycle state", func() {
			workspaceA.State = StateReady
			workspaceB.State = StateProcessing
			workspaceC := workspaceA
			workspaceC.ID = "ws-count-by-state-disabled"
			workspaceC.DisplayName = "disabled workspace"
			workspaceC.State = StateDisabled

			Expect(store.Create(ctx, &workspaceA)).To(Succeed())
			Expect(store.Create(ctx, &workspaceB)).To(Succeed())
			Expect(store.Create(ctx, &workspaceC)).To(Succeed())
			defer func() {
				_ = store.Delete(ctx, workspaceA.ID)
				_ = store.Delete(ctx, workspaceB.ID)
				_ = store.Delete(ctx, workspaceC.ID)
			}()

			counts, err := store.CountByState(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[StateReady]).To(Equal(int64(1)))
			Expect(counts[StateProcessing]).To(Equal(int64(1)))
			Expect(counts[StateDisabled]).To(Equal(int64(1)))
		})

		It("should count filtered workspaces grouped by lifecycle state", func() {
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
			defer func() {
				_ = store.Delete(ctx, workspaceA.ID)
				_ = store.Delete(ctx, workspaceB.ID)
				_ = store.Delete(ctx, workspaceC.ID)
			}()

			counts, err := store.CountByState(ctx, &ListOptions{Keyword: "alpha"})
			Expect(err).NotTo(HaveOccurred())
			Expect(counts[StateReady]).To(Equal(int64(1)))
			Expect(counts[StateDisabled]).To(Equal(int64(1)))
			Expect(counts[StateProcessing]).To(Equal(int64(0)))
		})
	})
})

func cleanupWorkspaceCollection(ctx context.Context) error {
	_, err := database.Client().Database(database.Name()).Collection(workspaceCollectionName).
		DeleteMany(ctx, bson.M{})
	return err
}

var _ = Describe("ImageRegistryStore", func() {
	var store registry.ImageRegistryStore
	var ctx context.Context

	var workspaceID string
	var registryA, registryB registry.ImageRegistry

	BeforeEach(func() {
		var err error

		// Patch global config to set encrypt secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		store, err = registry.NewImageRegistryStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

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

	Context("Create GetByWorkspaceAndType Update", func() {
		It("should create, get and update image registry successfully", func() {
			// Create registryA
			registryAID, err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())
			Expect(registryAID).NotTo(Equal(bson.NilObjectID))

			// Create registryB
			registryBID, err := store.Create(ctx, &registryB)
			Expect(err).NotTo(HaveOccurred())
			Expect(registryBID).NotTo(Equal(bson.NilObjectID))

			// List
			regs, err := store.List(ctx, workspaceID)
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
			reg, err := store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeBuiltin)
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
			reg, err = store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeExternal)
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
			err = store.Update(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Get updated registryA
			reg, err = store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeBuiltin)
			Expect(err).NotTo(HaveOccurred())
			Expect(reg.Registry).To(Equal("gcr.io"))
			Expect(reg.Username).To(Equal("newUser"))
			Expect(reg.Password).To(Equal("newpassword"))
		})
	})

	Context("Get Non-existent Registry", func() {
		It("should return error when registry not found", func() {
			reg, err := store.GetByWorkspaceAndType(
				ctx, "non-existent-workspace", registry.ImageRegistryTypeBuiltin,
			)
			Expect(err).To(HaveOccurred())
			Expect(reg).To(BeNil())
		})
	})

	Context("Create Duplicate Registry", func() {
		It("should return error when creating registry with duplicate workspaceID and type", func() {
			// Create registryA
			_, err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Try to create another registry with same workspaceID and type
			duplicateRegistry := registryA
			duplicateRegistry.Registry = "another-registry.com"
			_, err = store.Create(ctx, &duplicateRegistry)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Multiple Workspaces", func() {
		It("should handle registries from different workspaces correctly", func() {
			workspace2ID := "workspace-" + stringx.Random(6)

			// Create registry for workspace1
			_, err := store.Create(ctx, &registryA)
			Expect(err).NotTo(HaveOccurred())

			// Create registry for workspace2 with same type
			registry2 := registry.ImageRegistry{
				WorkspaceID: workspace2ID,
				Type:        registry.ImageRegistryTypeBuiltin,
				Registry:    "quay.io",
				Username:    "user2",
				Password:    "pass2",
			}
			_, err = store.Create(ctx, &registry2)
			Expect(err).NotTo(HaveOccurred())

			// Get registry for workspace1
			reg1, err := store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeBuiltin)
			Expect(err).NotTo(HaveOccurred())
			Expect(reg1.Registry).To(Equal("docker.io"))

			// Get registry for workspace2
			reg2, err := store.GetByWorkspaceAndType(ctx, workspace2ID, registry.ImageRegistryTypeBuiltin)
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

			_, err := store.Create(ctx, &registryWithPassword)
			Expect(err).NotTo(HaveOccurred())

			// Get registry - password should be decrypted
			reg, err := store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeBuiltin)
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

			_, err := store.Create(ctx, &registryWithoutPassword)
			Expect(err).NotTo(HaveOccurred())

			// Get registry - password should remain empty
			reg, err := store.GetByWorkspaceAndType(ctx, workspaceID, registry.ImageRegistryTypeBuiltin)
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

			_, err := store.Create(ctx, &registry)
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
			_, err := store.Create(ctx, &registryA)
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

			err = store.Update(ctx, &updateRegistry)
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
