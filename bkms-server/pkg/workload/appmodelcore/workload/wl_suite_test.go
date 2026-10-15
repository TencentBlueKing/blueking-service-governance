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

package workload_test

import (
	"context"
	"testing"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
)

func TestWorkload(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Workload Suite")
}

var _ = BeforeSuite(func() {
	if err := testutil.SetUpGlobalDatabase(); err != nil {
		panic("failed to set up global database: " + err.Error())
	}

	// init workload plugins, it's used to build workload
	initWorkloadPlugin()
})

var _ = AfterSuite(func() {
	if err := testutil.TeardownGlobalDatabase(); err != nil {
		panic("failed to teardown global database: " + err.Error())
	}
})

// newWorkspaceStore 构造 WorkspaceStore，供测试补齐 app 所属 workspace 使用。
func newWorkspaceStore() *workspace.WorkspaceStore {
	return testutil.NewStore(GinkgoT(), workspace.Meta)
}

// trackedWorkspaces 记录测试创建的 workspace，由 cleanupWorkspaces 统一清理。
var trackedWorkspaces []string

// newWorkspaceID 创建一个真实 workspace 并返回其 ID。
//
// 构建 workload 时会按 app.WorkspaceID 查询 workspace（BSCP 注入需要），
// 因此 app 不能持有悬空的 WorkspaceID。
func newWorkspaceID(ctx context.Context) string {
	ws := dbfactory.Workspace(ctx, newWorkspaceStore())
	trackedWorkspaces = append(trackedWorkspaces, ws.ID)
	return ws.ID
}

// cleanupWorkspaces 删除本测试创建的 workspace，避免残留影响其他 suite。
func cleanupWorkspaces(ctx context.Context) {
	store := newWorkspaceStore()
	for _, id := range trackedWorkspaces {
		_ = workspace.Delete(ctx, store, id)
	}
	trackedWorkspaces = nil
}

func initWorkloadPlugin() {
	db := testutil.NewDB(GinkgoT())
	workload.InitPlugin(
		appcfg.NewMountableFileProvider(db),
		db,
	)
}

func asGameDeployment(result *workload.BuildResult) *tkex.GameDeployment {
	if result == nil {
		return nil
	}
	gd, _ := result.MainWorkload.(*tkex.GameDeployment)
	return gd
}

func asDeployment(result *workload.BuildResult) *appsv1.Deployment {
	if result == nil {
		return nil
	}
	d, _ := result.MainWorkload.(*appsv1.Deployment)
	return d
}
