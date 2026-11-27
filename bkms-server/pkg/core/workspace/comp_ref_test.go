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

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	bkmsworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("ComponentRefResolver", func() {
	var ctx context.Context
	var db *mongo.Database
	var appStore *bkmsapp.ApplicationStore
	var workspaceID string
	var appIDs []string

	// newRefApp 创建一个引用了空间组件 comp 的应用
	newRefApp := func(appType, framework, language string) string {
		app := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{
			WorkspaceID: workspaceID,
			Type:        appType,
			Framework:   framework,
			Language:    language,
		})
		Expect(appmodel.CreateAppModel(ctx, db, &appmodel.AppModel{
			AppID: app.ID,
			Components: []*component.Component{
				{Name: "ref", ComponentRef: component.ComponentRef{RefWorkspaceCompName: "comp"}},
			},
		})).To(Succeed())
		appIDs = append(appIDs, app.ID)
		return app.ID
	}

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db = testutil.NewDB(GinkgoT())
		appStore = store.Of(db, bkmsapp.AppMeta)
		workspaceID = "ws-ref-" + stringx.Random(6)
		appIDs = nil
	})

	AfterEach(func() {
		for _, appID := range appIDs {
			_ = appmodel.DeleteAppModel(ctx, db, appID)
			_ = appStore.DeleteIfExists(ctx, bkmsapp.ByID(appID))
		}
	})

	It("collects references from every AppModel app, including taf and blank", func() {
		trpcApp := newRefApp(appruntime.AppTypeTRPC, "", "")
		tafApp := newRefApp(appruntime.AppTypeTAF, "", "")
		blankApp := newRefApp(appruntime.AppTypeBKMSApp, "blank", "python")

		refs, err := bkmsworkspace.NewComponentRefResolver(db).BuildRefMap(ctx, workspaceID)
		Expect(err).NotTo(HaveOccurred())
		Expect(refs["comp"]).To(ConsistOf(trpcApp, tafApp, blankApp))
	})
})
