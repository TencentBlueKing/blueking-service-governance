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

package bkci

import (
	"context"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

var _ = Describe("BuildNodeManager", func() {
	var (
		ctx          context.Context
		manager      *BuildNodeManager
		bindingStore *BuildNodeBindingStore
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		Expect(testutil.CleanupMetaColl(BuildNodeBindingMeta)).To(Succeed())
		projectStore, _, _ := setupManagerTestStores()
		Expect(projectStore.Create(ctx, mockProject())).To(Succeed())

		bindingStore = testutil.NewStore(GinkgoT(), BuildNodeBindingMeta)
		manager = NewBuildNodeManager(testutil.NewDB(GinkgoT()), managerTestWorkspaceID)
	})

	// 蓝盾客户端在测试配置下为 stub，只需 mock 当前用户
	withUser := func(fn func()) {
		mockey.PatchConvey("with user", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
			fn()
		})
	}

	It("marks offline and non-linux nodes unselectable and falls back to hostname", func() {
		withUser(func() {
			candidates, err := manager.ListCandidates(ctx)
			Expect(err).NotTo(HaveOccurred())

			byID := lo.KeyBy(candidates, func(c BuildNodeCandidate) string { return c.NodeHashID })
			Expect(byID[cloudbkci.StubAgentNodeLinuxOnline].UnselectableReason).To(BeEmpty())
			Expect(byID[cloudbkci.StubAgentNodeLinuxOffline].UnselectableReason).
				To(Equal(UnselectableReasonOffline))
			Expect(byID[cloudbkci.StubAgentNodeWindows].UnselectableReason).
				To(Equal(UnselectableReasonOSUnsupported))
			Expect(byID[cloudbkci.StubAgentNodeWindows].DisplayName).To(Equal("stub-host-3"))
		})
	})

	It("replaces the existing binding and keeps a single record per workspace", func() {
		withUser(func() {
			// 数据库不做唯一约束，预置两条旧记录模拟并发关联多写出的数据，关联后应只剩新的一条
			// 新建时 updatedAt 取 createdAt，错开时间保证「最近更新」的排序确定
			for i, nodeHashID := range []string{"stale-node", "old-node"} {
				Expect(bindingStore.Create(ctx, &BuildNodeBinding{
					WorkspaceID: managerTestWorkspaceID, ProjectCode: managerTestProjectCode, NodeHashID: nodeHashID,
					CreatedAt: time.Now().Add(time.Duration(i-2) * time.Minute),
				})).To(Succeed())
			}

			res, err := manager.Bind(ctx, cloudbkci.StubAgentNodeLinuxOnline)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Changed).To(BeTrue())
			// 多条时取最近更新的一条作为「更换前」
			Expect(res.Before.NodeHashID).To(Equal("old-node"))

			bindings, err := bindingStore.List(ctx, ByWorkspace(managerTestWorkspaceID))
			Expect(err).NotTo(HaveOccurred())
			Expect(bindings).To(HaveLen(1))
			Expect(bindings[0].NodeHashID).To(Equal(cloudbkci.StubAgentNodeLinuxOnline))
			// 下发流水线依赖 agentHashID，它和 nodeHashID 不是同一个值
			Expect(bindings[0].AgentHashID).To(Equal("stub-agent-1"))

			// 重复提交同一台不算变更
			res, err = manager.Bind(ctx, cloudbkci.StubAgentNodeLinuxOnline)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Changed).To(BeFalse())
		})
	})

	DescribeTable("rejects nodes that cannot run builtin pipelines and keeps no binding",
		func(nodeHashID string, expected error) {
			withUser(func() {
				_, err := manager.Bind(ctx, nodeHashID)
				Expect(err).To(MatchError(expected))

				exists, err := bindingStore.Exists(ctx, ByWorkspace(managerTestWorkspaceID))
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
		},
		Entry("offline", cloudbkci.StubAgentNodeLinuxOffline, ErrBuildNodeOffline),
		Entry("windows", cloudbkci.StubAgentNodeWindows, ErrBuildNodeOSUnsupported),
		Entry("no use permission", cloudbkci.StubAgentNodeNoUsePerm, ErrBuildNodeNoUsePerm),
		Entry("not in project", "missing-node", ErrBuildNodeNotFound),
	)

	It("unbinds idempotently and returns the removed binding", func() {
		withUser(func() {
			_, err := manager.Bind(ctx, cloudbkci.StubAgentNodeLinuxOnline)
			Expect(err).NotTo(HaveOccurred())

			removed, err := manager.Unbind(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(removed.NodeHashID).To(Equal(cloudbkci.StubAgentNodeLinuxOnline))

			removed, err = manager.Unbind(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(removed).To(BeNil())
		})
	})

	It("reports notFound state when the bound node was deleted from bkci", func() {
		withUser(func() {
			Expect(bindingStore.Create(ctx, &BuildNodeBinding{
				WorkspaceID: managerTestWorkspaceID, ProjectCode: managerTestProjectCode, NodeHashID: "deleted-node",
			})).To(Succeed())

			state, err := manager.Get(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(state.ProjectCode).To(Equal(managerTestProjectCode))
			Expect(state.NodeState).To(Equal(BuildNodeStateNotFound))
		})
	})
})
