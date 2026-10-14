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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("GetWorkspaceImageRegistry", func() {
	var (
		ctx            context.Context
		db             *mongo.Database
		workspaceStore *WorkspaceStore
		registries     *bkmsreg.ImageRegistryStore
		ws             *Workspace
		oldConfig      *svccfg.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		oldConfig = svccfg.G
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

		workspaceStore = testutil.NewStore(GinkgoT(), Meta)
		db = testutil.NewDB(GinkgoT())
		registries = testutil.NewStore(GinkgoT(), bkmsreg.ImageRegistryMeta)

		ws = dbfactory.Workspace(ctx, workspaceStore)
		Expect(registries.CreateMany(ctx, []*bkmsreg.ImageRegistry{
			{
				WorkspaceID:      ws.ID,
				Type:             bkmsreg.ImageRegistryTypeBuiltin,
				Registry:         "mirrors.example.com/test",
				BkCICredentialID: "cred-123",
			},
			{
				WorkspaceID:      ws.ID,
				Type:             bkmsreg.ImageRegistryTypeExternal,
				Registry:         "external.registry.com/repo",
				Username:         "external-user",
				Password:         "external-password",
				BkCICredentialID: "cred-456",
			},
		})).To(Succeed())
	})

	AfterEach(func() {
		_, err := registries.DeleteMany(ctx, bkmsreg.ByWorkspace(ws.ID))
		Expect(err).NotTo(HaveOccurred())
		Expect(Delete(ctx, workspaceStore, ws.ID)).To(Succeed())
		svccfg.G = oldConfig
	})

	setRegistryType := func(registryType bkmsreg.ImageRegistryType) {
		ws.ImageRegistryType = registryType
		Expect(Update(ctx, workspaceStore, ws)).To(Succeed())
	}

	It("should return the builtin image registry of the workspace", func() {
		setRegistryType(bkmsreg.ImageRegistryTypeBuiltin)

		registry, err := GetWorkspaceImageRegistry(ctx, db, ws.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(registry.WorkspaceID).To(Equal(ws.ID))
		Expect(registry.Type).To(Equal(bkmsreg.ImageRegistryTypeBuiltin))
		Expect(registry.Registry).To(Equal("mirrors.example.com/test"))
	})

	It("should return the external image registry with decrypted credentials", func() {
		setRegistryType(bkmsreg.ImageRegistryTypeExternal)

		registry, err := GetWorkspaceImageRegistry(ctx, db, ws.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(registry.Type).To(Equal(bkmsreg.ImageRegistryTypeExternal))
		Expect(registry.Username).To(Equal("external-user"))
		Expect(registry.Password).To(Equal("external-password"))
	})

	It("should return error with workspace ID when workspace not found", func() {
		registry, err := GetWorkspaceImageRegistry(ctx, db, "missing-workspace")
		Expect(err).To(MatchError(ErrWorkspaceNotFound))
		Expect(err.Error()).To(ContainSubstring("missing-workspace"))
		Expect(registry).To(BeNil())
	})

	It("should return ErrImageRegistryNotFound when the workspace has no registry of its type", func() {
		_, err := registries.DeleteMany(ctx, bkmsreg.ByWorkspace(ws.ID))
		Expect(err).NotTo(HaveOccurred())
		setRegistryType(bkmsreg.ImageRegistryTypeBuiltin)

		registry, err := GetWorkspaceImageRegistry(ctx, db, ws.ID)
		Expect(err).To(MatchError(bkmsreg.ErrImageRegistryNotFound))
		Expect(err.Error()).To(ContainSubstring("get image registry"))
		Expect(registry).To(BeNil())
	})
})
