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

package admin

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

var _ = Describe("WorkspaceAdminService", func() {
	var (
		ctx            context.Context
		diApp          *fxtest.App
		recordStore    Store
		workspaceStore bkmsworkspace.WorkspaceStore
		roleMgr        *stubRoleManager
		service        *Service
		ws             *bkmsworkspace.Workspace
	)

	BeforeEach(func() {
		ctx = context.Background()
		roleMgr = &stubRoleManager{roleID: "fake-admin-role"}

		diApp = fxtest.New(
			GinkgoT(),
			bkmsworkspace.FxModule,
			FxModule,
			fx.Provide(
				func() perm.Manager { return roleMgr },
			),
			fx.Populate(&service, &workspaceStore, &recordStore),
		)
		diApp.RequireStart()

		Expect(cleanupWorkspaceAdminCollections()).To(Succeed())
		ws = dbfactory.Workspace(ctx, workspaceStore)
	})

	AfterEach(func() {
		Expect(cleanupWorkspaceAdminCollections()).To(Succeed())
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("should report target user role membership by role code and username", func() {
		roleMgr.members = []string{"developer-user"}

		status, err := service.GetRoleStatus(ctx, ws.ID, "developer", "developer-user")
		Expect(err).NotTo(HaveOccurred())
		Expect(status.HasRole).To(BeTrue())
	})

	It("should grant admin as temporary when isTemporary is true", func() {
		before := time.Now()
		status, err := service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.HasRole).To(BeTrue())

		record, err := recordStore.GetLatestActiveGrant(ctx, ws.ID, "temp-user")
		Expect(err).NotTo(HaveOccurred())
		Expect(record.IsRecycled).To(BeFalse())
		Expect(record.ExpiresAt).To(BeTemporally("~", before.Add(2*time.Hour), 3*time.Second))
	})

	It("should grant admin as permanent when isTemporary is false", func() {
		status, err := service.GrantAdmin(ctx, ws.ID, "perm-user", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.HasRole).To(BeTrue())
		Expect(roleMgr.members).To(ContainElement("perm-user"))

		_, err = recordStore.GetLatestActiveGrant(ctx, ws.ID, "perm-user")
		Expect(err).To(MatchError(ErrRecordNotFound))
	})

	It("should reject grant when current user is already a workspace admin", func() {
		roleMgr.members = []string{"admin"}

		_, err := service.GrantAdmin(ctx, ws.ID, "admin", true)
		Expect(err).To(MatchError(ErrWorkspaceAdminAlreadyExists))
	})

	It("should reject duplicate temporary grant once user already has admin role", func() {
		_, err := service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).NotTo(HaveOccurred())

		_, err = service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).To(MatchError(ErrWorkspaceAdminAlreadyExists))
	})

	It("should reject temporary grant when local temp admin record exists without remote admin role", func() {
		now := time.Now()
		record := &WorkspaceTempAdmin{
			WorkspaceID: ws.ID,
			Username:    "temp-user",
			ExpiresAt:   now.Add(2 * time.Hour),
			IsRecycled:  false,
			Creator:     "temp-user",
			CreatedAt:   now,
			UpdatedAt:   now,
			Updater:     "temp-user",
		}
		Expect(recordStore.Create(ctx, record)).To(Succeed())

		_, err := service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).To(MatchError(ErrTempAdminAlreadyExists))
		Expect(roleMgr.addCalls).To(Equal(0))
		Expect(roleMgr.members).NotTo(ContainElement("temp-user"))
	})

	It("should revoke admin and recycle temp grant history when needed", func() {
		_, err := service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).NotTo(HaveOccurred())

		status, err := service.RevokeAdmin(ctx, ws.ID, "temp-user")
		Expect(err).NotTo(HaveOccurred())
		Expect(status.HasRole).To(BeFalse())

		_, err = recordStore.GetLatestActiveGrant(ctx, ws.ID, "temp-user")
		Expect(err).To(MatchError(ErrRecordNotFound))

		record := loadLatestRecord(ctx, recordStore, ws.ID, "temp-user")
		Expect(record.IsRecycled).To(BeTrue())
		Expect(record.Updater).To(Equal("temp-user"))
	})

	It("should revoke workspace admin even without a temporary admin record", func() {
		roleMgr.members = []string{"admin"}

		status, err := service.RevokeAdmin(ctx, ws.ID, "admin")
		Expect(err).NotTo(HaveOccurred())
		Expect(status.HasRole).To(BeFalse())

		_, err = recordStore.GetLatestActiveGrant(ctx, ws.ID, "admin")
		Expect(err).To(MatchError(ErrRecordNotFound))
	})

	It("should reconcile expired records into recycled state", func() {
		_, err := service.GrantAdmin(ctx, ws.ID, "temp-user", true)
		Expect(err).NotTo(HaveOccurred())

		record, err := recordStore.GetLatestActiveGrant(ctx, ws.ID, "temp-user")
		Expect(err).NotTo(HaveOccurred())
		record.ExpiresAt = time.Now().Add(-time.Hour)
		Expect(recordStore.Update(ctx, record)).To(Succeed())

		err = service.CleanupExpiredGrants(ctx)
		Expect(err).NotTo(HaveOccurred())

		_, err = recordStore.GetLatestActiveGrant(ctx, ws.ID, "temp-user")
		Expect(err).To(MatchError(ErrRecordNotFound))

		record = loadLatestRecord(ctx, recordStore, ws.ID, "temp-user")
		Expect(record.IsRecycled).To(BeTrue())
	})
})

type stubRoleManager struct {
	perm.Manager
	roleID                string
	members               []string
	getRoleErr            error
	getRoleErrByWorkspace map[string]error
	listMembersErr        error
	addErr                error
	addCalls              int
	deleteErr             error
	deleteCalls           int
}

func (m *stubRoleManager) GetRole(_ context.Context, workspaceID, roleCode string) (*role.Role, error) {
	if m.getRoleErr != nil {
		return nil, m.getRoleErr
	}
	if err, ok := m.getRoleErrByWorkspace[workspaceID]; ok {
		return nil, err
	}
	return &role.Role{
		ID:       m.roleID,
		RoleCode: roleCode,
		Scope: role.PermissionScope{
			ResourceType: role.WorkspaceResourceType,
			ResourceID:   workspaceID,
		},
	}, nil
}

func (m *stubRoleManager) ListRoleMembers(_ context.Context, roleID string) ([]string, error) {
	if m.listMembersErr != nil {
		return nil, m.listMembersErr
	}
	if roleID != m.roleID {
		return nil, nil
	}
	return append([]string(nil), m.members...), nil
}

func (m *stubRoleManager) AddRoleForUsers(_ context.Context, _ string, users []string) error {
	m.addCalls++
	if m.addErr != nil {
		return m.addErr
	}
	m.members = append(m.members, users...)
	return nil
}

func (m *stubRoleManager) DeleteRoleForUsers(_ context.Context, _ string, users []string) error {
	m.deleteCalls++
	if m.deleteErr != nil {
		return m.deleteErr
	}
	filtered := m.members[:0]
	for _, member := range m.members {
		shouldDelete := false
		for _, user := range users {
			if member == user {
				shouldDelete = true
				break
			}
		}
		if !shouldDelete {
			filtered = append(filtered, member)
		}
	}
	m.members = filtered
	return nil
}

func cleanupWorkspaceAdminCollections() error {
	if err := testutil.CleanupCollection("workspaces"); err != nil {
		return err
	}
	return testutil.CleanupCollection(collectionName)
}

func loadLatestRecord(
	ctx context.Context,
	store Store,
	workspaceID, username string,
) *WorkspaceTempAdmin {
	mongoStore, ok := store.(*StoreMongo)
	Expect(ok).To(BeTrue())

	var record WorkspaceTempAdmin
	err := mongoStore.collection.FindOne(
		ctx,
		bson.M{
			"workspaceID": workspaceID,
			"username":    username,
		},
		options.FindOne().SetSort(bson.D{{Key: "updatedAt", Value: -1}, {Key: "_id", Value: -1}}),
	).Decode(&record)
	Expect(err).NotTo(HaveOccurred())
	return &record
}
