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

package hooks_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworktmpl"
	frameworktmplhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/frameworktmpl/hooks"
)

var _ = Describe("Framework config template lifecycle hooks", func() {
	var ctx context.Context
	var db *mongo.Database
	var workspaceStore *bkmsworkspace.WorkspaceStore
	var workspace *bkmsworkspace.Workspace

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		bkmsworkspace.ResetLifecycleHooksForTest()

		db = testutil.NewDB(GinkgoT())
		workspaceStore = store.Of(db, bkmsworkspace.Meta)
		frameworktmplhooks.RegisterPreDeleteHooks(db)
	})

	AfterEach(func() {
		if workspace != nil {
			Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspace.ID)).To(Succeed())
		}
		Expect(testutil.CleanupMetaColl(frameworktmpl.TemplateMeta)).To(Succeed())
		bkmsworkspace.ResetLifecycleHooksForTest()
	})

	It("removes templates when their workspace is deleted", func() {
		workspace = dbfactory.Workspace(ctx, workspaceStore)
		workspaceID := workspace.ID
		_, err := frameworktmpl.Create(ctx, db, workspaceID, frameworktmpl.CreateParams{
			Framework:  "trpc",
			Language:   "go",
			Definition: frameworktmpl.Definition{Name: "tpl", FileName: "trpc_go.yaml", Content: "a: 1\n"},
			IsDefault:  true,
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(bkmsworkspace.Delete(ctx, workspaceStore, workspaceID)).To(Succeed())
		workspace = nil

		templates, err := store.Of(db, frameworktmpl.TemplateMeta).List(ctx, frameworktmpl.ByWorkspace(workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(templates).To(BeEmpty())
	})
})
