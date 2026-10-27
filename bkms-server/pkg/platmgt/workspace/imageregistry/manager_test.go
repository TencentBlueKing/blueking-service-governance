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

package imageregistry

import (
	"context"
	"net/http"

	"github.com/bytedance/mockey"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("Manager", func() {
	const (
		external = bkmsreg.ImageRegistryTypeExternal
		registry = "mirrors.example.com/team-ns"
		username = "team-user"
		password = "team-pass" //nolint:gosec // G101: 测试用的假密码
	)

	var (
		ctx            context.Context
		diApp          *fxtest.App
		manager        *Manager
		workspaceStore *bkmsworkspace.WorkspaceStore
		registryStore  *bkmsreg.ImageRegistryStore
		ws             *bkmsworkspace.Workspace

		probeErr       error
		probeHook      func()
		createdCredIDs []string
		deletedCredIDs []string
	)

	getExternal := func() (*bkmsreg.ImageRegistry, error) {
		return registryStore.Get(ctx, bkmsreg.ByWorkspaceAndType(ws.ID, bkmsreg.ImageRegistryTypeExternal))
	}

	mustGetExternal := func() *bkmsreg.ImageRegistry {
		reg, err := getExternal()
		Expect(err).NotTo(HaveOccurred())
		return reg
	}

	getRegistryType := func() bkmsreg.ImageRegistryType {
		got, err := workspaceStore.Get(ctx, bkmsworkspace.ByWorkspaceID(ws.ID))
		Expect(err).NotTo(HaveOccurred())
		return got.ImageRegistryType
	}

	BeforeEach(func() {
		ctx = auth.WithUser(context.Background(), auth.User{ID: "plat-admin"})
		probeErr, probeHook = nil, nil
		createdCredIDs, deletedCredIDs = nil, nil

		diApp = fxtest.New(GinkgoT(), FxModule, fx.Populate(&manager))
		diApp.RequireStart()
		workspaceStore = testutil.NewStore(GinkgoT(), bkmsworkspace.Meta)
		registryStore = testutil.NewStore(GinkgoT(), bkmsreg.ImageRegistryMeta)
		// 新建空间的生效类型为空，按平台仓库处理
		ws = dbfactory.Workspace(ctx, workspaceStore)

		mockey.Mock((*infrasreg.Client).CheckPushPermission).
			To(func(_ *infrasreg.Client, _ context.Context, _ string) error {
				if probeHook != nil {
					probeHook()
				}
				return probeErr
			}).Build()
		// suite 打开了蓝盾 stub，这里只记录凭证的创建与删除
		mockey.Mock((*bkciapi.StubApiClient).CreateCredential).
			To(func(_ *bkciapi.StubApiClient, _ context.Context, _, credID, _, _, _ string) error {
				createdCredIDs = append(createdCredIDs, credID)
				return nil
			}).Build()
		mockey.Mock((*bkciapi.StubApiClient).DeleteCredential).
			To(func(_ *bkciapi.StubApiClient, _ context.Context, _, credID string) error {
				deletedCredIDs = append(deletedCredIDs, credID)
				return nil
			}).Build()
		mockey.Mock(audit.AddOperationRecordAsync).Return().Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(testutil.CleanupMetaColl(bkmsreg.ImageRegistryMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(bkmsworkspace.Meta)).To(Succeed())
		diApp.RequireStop()
	})

	Describe("switching from builtin to external", func() {
		params := UpdateParams{Type: external, Registry: registry, Username: username, Password: password}

		It("should create the credential, the external record and switch the type", func() {
			_, err := manager.Update(ctx, ws.ID, params)
			Expect(err).NotTo(HaveOccurred())

			Expect(getRegistryType()).To(Equal(external))
			reg := mustGetExternal()
			Expect(reg.Password).To(Equal(password))
			Expect(createdCredIDs).To(Equal([]string{reg.BkCICredentialID}))
		})

		It("should require password", func() {
			_, err := manager.Update(ctx, ws.ID, UpdateParams{Type: external, Registry: registry, Username: username})
			Expect(err).To(MatchError(ErrPasswordRequired))
		})

		It("should reject without side effects when probe fails", func() {
			probeErr = &transport.Error{StatusCode: http.StatusUnauthorized}

			_, err := manager.Update(ctx, ws.ID, params)
			Expect(err).To(MatchError(ErrRegistryAuthFailed))
			Expect(createdCredIDs).To(BeEmpty())
			_, err = getExternal()
			Expect(err).To(MatchError(bkmsreg.ErrImageRegistryNotFound))
		})

		It("should allow another update after a failed one", func() {
			probeErr = &transport.Error{StatusCode: http.StatusUnauthorized}
			_, err := manager.Update(ctx, ws.ID, params)
			Expect(err).To(MatchError(ErrRegistryAuthFailed))

			probeErr = nil
			_, err = manager.Update(ctx, ws.ID, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(getRegistryType()).To(Equal(external))
		})

		It("should clean up the record and credential when switching the type fails", func() {
			// 探测期间空间被删除，最后一步更新生效类型会失败
			probeHook = func() {
				Expect(workspaceStore.Delete(ctx, bkmsworkspace.ByWorkspaceID(ws.ID))).To(Succeed())
			}

			_, err := manager.Update(ctx, ws.ID, params)
			Expect(err).To(MatchError(bkmsworkspace.ErrWorkspaceNotFound))
			Expect(deletedCredIDs).To(Equal(createdCredIDs))
			_, err = getExternal()
			Expect(err).To(MatchError(bkmsreg.ErrImageRegistryNotFound))
		})
	})

	Describe("registry address", func() {
		DescribeTable("should reject an address that would not stay on the given registry",
			func(address string) {
				_, err := manager.Update(ctx, ws.ID, UpdateParams{
					Type: external, Registry: address, Username: username, Password: password,
				})
				Expect(err).To(MatchError(ErrInvalidRegistryAddress))
				Expect(createdCredIDs).To(BeEmpty())
			},
			Entry("short host is parsed as Docker Hub", "harbor/ns"),
			Entry("scheme prefix", "https://mirrors.example.com/ns"),
			Entry("uppercase namespace", "mirrors.example.com/TeamNs"),
			Entry("missing namespace", "mirrors.example.com"),
		)
	})

	Describe("with an existing external record", func() {
		const oldCredID = "old-cred"

		BeforeEach(func() {
			Expect(registryStore.Create(ctx, &bkmsreg.ImageRegistry{
				WorkspaceID: ws.ID, Type: bkmsreg.ImageRegistryTypeExternal,
				Registry: registry, Username: username, Password: password, BkCICredentialID: oldCredID,
			})).To(Succeed())
		})

		// 空间先前切回过平台仓库，外部仓库记录仍保留
		It("should overwrite the leftover record and reuse its credential when switching to external again", func() {
			_, err := manager.Update(ctx, ws.ID, UpdateParams{
				Type: external, Registry: "mirrors.example.com/new-ns", Username: username,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(getRegistryType()).To(Equal(external))
			Expect(mustGetExternal().Registry).To(Equal("mirrors.example.com/new-ns"))
			Expect(createdCredIDs).To(BeEmpty())
		})

		Context("when the external registry is effective", func() {
			BeforeEach(func() {
				Expect(workspaceStore.Update(ctx, map[string]any{"imageRegistryType": external},
					bkmsworkspace.ByWorkspaceID(ws.ID))).To(Succeed())
			})

			It("should keep the password and credential when the password is empty", func() {
				_, err := manager.Update(ctx, ws.ID, UpdateParams{
					Type: external, Registry: "mirrors.example.com/new-ns", Username: username,
				})
				Expect(err).NotTo(HaveOccurred())

				reg := mustGetExternal()
				Expect(reg.Registry).To(Equal("mirrors.example.com/new-ns"))
				Expect(reg.Password).To(Equal(password))
				Expect(reg.BkCICredentialID).To(Equal(oldCredID))
				Expect(createdCredIDs).To(BeEmpty())
			})

			It("should create a new credential without deleting the old one when the password changes", func() {
				_, err := manager.Update(ctx, ws.ID, UpdateParams{
					Type: external, Registry: registry, Username: username, Password: "new-pass",
				})
				Expect(err).NotTo(HaveOccurred())

				reg := mustGetExternal()
				Expect(reg.Password).To(Equal("new-pass"))
				Expect(createdCredIDs).To(Equal([]string{reg.BkCICredentialID}))
				Expect(deletedCredIDs).To(BeEmpty())
			})

			It("should switch back to builtin and keep the external record", func() {
				Expect(registryStore.Create(ctx, &bkmsreg.ImageRegistry{
					WorkspaceID: ws.ID, Type: bkmsreg.ImageRegistryTypeBuiltin, Registry: "bkrepo.example.com/ns",
				})).To(Succeed())

				reg, err := manager.Update(ctx, ws.ID, UpdateParams{Type: bkmsreg.ImageRegistryTypeBuiltin})
				Expect(err).NotTo(HaveOccurred())
				Expect(reg.Registry).To(Equal("bkrepo.example.com/ns"))

				Expect(getRegistryType()).To(Equal(bkmsreg.ImageRegistryTypeBuiltin))
				Expect(mustGetExternal().BkCICredentialID).To(Equal(oldCredID))
			})
		})
	})
})
