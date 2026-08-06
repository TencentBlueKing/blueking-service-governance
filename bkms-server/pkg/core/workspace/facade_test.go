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

package workspace_test

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("GetWorkspaceImageRegistry", func() {
	var (
		ctx         context.Context
		workspaceID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		workspaceID = "test-workspace"
	})

	// ==================== Success Scenarios ====================

	Context("when all dependencies work correctly", func() {
		It("should successfully return image registry", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockWorkspaceStoreSuccess()
				mockImageRegistryStoreSuccess()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).NotTo(HaveOccurred())
				Expect(registry).NotTo(BeNil())
				Expect(registry.WorkspaceID).To(Equal(workspaceID))
				Expect(registry.Type).To(Equal(bkmsreg.ImageRegistryTypeBuiltin))
				Expect(registry.Registry).To(Equal("mirrors.example.com/test"))
			})
		})
	})

	Context("when workspace uses external image registry", func() {
		It("should return external registry with credentials", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockWorkspaceStoreWithExternalRegistry()
				mockImageRegistryStoreWithExternal()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).NotTo(HaveOccurred())
				Expect(registry).NotTo(BeNil())
				Expect(registry.Type).To(Equal(bkmsreg.ImageRegistryTypeExternal))
				Expect(registry.Username).To(Equal("external-user"))
			})
		})
	})

	// ==================== Error Scenarios ====================

	Context("when creating workspace store fails", func() {
		It("should return error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
				mockey.Mock(database.Name).Return("test-db").Build()
				mockey.Mock(NewWorkspaceStoreMongo).Return(nil, errors.New("store creation failed")).Build()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("store creation failed"))
				Expect(registry).To(BeNil())
			})
		})
	})

	Context("when workspace not found", func() {
		It("should return error with workspace ID", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockStore := &WorkspaceStoreMongo{}
				mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
				mockey.Mock(database.Name).Return("test-db").Build()
				mockey.Mock(NewWorkspaceStoreMongo).Return(mockStore, nil).Build()
				mockey.Mock((*WorkspaceStoreMongo).Get).Return(nil, ErrWorkspaceNotFound).Build()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get workspace"))
				Expect(err.Error()).To(ContainSubstring(workspaceID))
				Expect(registry).To(BeNil())
			})
		})
	})

	Context("when creating image registry store fails", func() {
		It("should return error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockWorkspaceStoreSuccess()
				mockey.Mock(bkmsreg.NewImageRegistryStoreMongo).Return(
					nil, errors.New("registry store creation failed"),
				).Build()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("registry store creation failed"))
				Expect(registry).To(BeNil())
			})
		})
	})

	Context("when image registry not found", func() {
		It("should return error with 'get image registry' message", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockWorkspaceStoreSuccess()
				mockRegStore := &bkmsreg.ImageRegistryStoreMongo{}
				mockey.Mock(bkmsreg.NewImageRegistryStoreMongo).Return(mockRegStore, nil).Build()
				mockey.Mock((*bkmsreg.ImageRegistryStoreMongo).GetByWorkspaceAndType).
					Return(nil, bkmsreg.ErrImageRegistryNotFound).Build()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get image registry"))
				Expect(registry).To(BeNil())
			})
		})
	})

	Context("when GetByWorkspaceAndType returns database error", func() {
		It("should return wrapped error", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockWorkspaceStoreSuccess()
				mockRegStore := &bkmsreg.ImageRegistryStoreMongo{}
				mockey.Mock(bkmsreg.NewImageRegistryStoreMongo).Return(mockRegStore, nil).Build()
				mockey.Mock((*bkmsreg.ImageRegistryStoreMongo).GetByWorkspaceAndType).
					Return(nil, errors.New("database connection error")).Build()

				// Execute
				registry, err := GetWorkspaceImageRegistry(ctx, workspaceID)

				// Verify
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("get image registry"))
				Expect(err.Error()).To(ContainSubstring("database connection error"))
				Expect(registry).To(BeNil())
			})
		})
	})
})

// ==================== Helper Functions ====================

// mockWorkspace creates a mock workspace with builtin registry type
func mockWorkspace() *Workspace {
	return &Workspace{
		ID:                "test-workspace",
		DisplayName:       "Test Workspace",
		ImageRegistryType: bkmsreg.ImageRegistryTypeBuiltin,
		BkSystems: BkSystems{
			BkCIProjectID: "test-bkci-project",
		},
		Creator: "admin",
	}
}

// mockWorkspaceWithExternalRegistry creates a mock workspace with external registry type
func mockWorkspaceWithExternalRegistry() *Workspace {
	ws := mockWorkspace()
	ws.ImageRegistryType = bkmsreg.ImageRegistryTypeExternal
	return ws
}

// mockImageRegistry creates a mock builtin image registry
func mockImageRegistry() *bkmsreg.ImageRegistry {
	return &bkmsreg.ImageRegistry{
		WorkspaceID:      "test-workspace",
		Type:             bkmsreg.ImageRegistryTypeBuiltin,
		Registry:         "mirrors.example.com/test",
		Username:         "",
		Password:         "",
		BkCICredentialID: "cred-123",
	}
}

// mockExternalImageRegistry creates a mock external image registry
func mockExternalImageRegistry() *bkmsreg.ImageRegistry {
	return &bkmsreg.ImageRegistry{
		WorkspaceID:      "test-workspace",
		Type:             bkmsreg.ImageRegistryTypeExternal,
		Registry:         "external.registry.com/repo",
		Username:         "external-user",
		Password:         "encrypted-password",
		BkCICredentialID: "cred-456",
	}
}

// mockWorkspaceStoreSuccess mocks successful workspace store operations
func mockWorkspaceStoreSuccess() {
	mockStore := &WorkspaceStoreMongo{}
	mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
	mockey.Mock(database.Name).Return("test-db").Build()
	mockey.Mock(NewWorkspaceStoreMongo).Return(mockStore, nil).Build()
	mockey.Mock((*WorkspaceStoreMongo).Get).Return(mockWorkspace(), nil).Build()
}

// mockWorkspaceStoreWithExternalRegistry mocks workspace store returning external registry type
func mockWorkspaceStoreWithExternalRegistry() {
	mockStore := &WorkspaceStoreMongo{}
	mockey.Mock(database.Client).Return(&mongo.Client{}).Build()
	mockey.Mock(database.Name).Return("test-db").Build()
	mockey.Mock(NewWorkspaceStoreMongo).Return(mockStore, nil).Build()
	mockey.Mock((*WorkspaceStoreMongo).Get).Return(mockWorkspaceWithExternalRegistry(), nil).Build()
}

// mockImageRegistryStoreSuccess mocks successful image registry store operations
func mockImageRegistryStoreSuccess() {
	mockRegStore := &bkmsreg.ImageRegistryStoreMongo{}
	mockey.Mock(bkmsreg.NewImageRegistryStoreMongo).Return(mockRegStore, nil).Build()
	mockey.Mock((*bkmsreg.ImageRegistryStoreMongo).GetByWorkspaceAndType).Return(mockImageRegistry(), nil).Build()
}

// mockImageRegistryStoreWithExternal mocks image registry store returning external registry
func mockImageRegistryStoreWithExternal() {
	mockRegStore := &bkmsreg.ImageRegistryStoreMongo{}
	mockey.Mock(bkmsreg.NewImageRegistryStoreMongo).Return(mockRegStore, nil).Build()
	mockey.Mock((*bkmsreg.ImageRegistryStoreMongo).GetByWorkspaceAndType).
		Return(mockExternalImageRegistry(), nil).
		Build()
}
