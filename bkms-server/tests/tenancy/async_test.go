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

package tenancy_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const asyncTestQueue = "tenancy-test"

type createWorkspaceArgs struct {
	WorkspaceID string `json:"workspaceID"`
}

type pollArgs struct {
	Step int `json:"step"`
}

var _ = Describe("Async task tenancy", Ordered, func() {
	var (
		redis     *miniredis.Miniredis
		inspector *asynq.Inspector
		wsStore   *workspace.WorkspaceStore
	)

	BeforeAll(func() {
		var err error
		redis, err = miniredis.Run()
		Expect(err).NotTo(HaveOccurred())
		taskq.InitClient(context.Background(), config.AsynqConfig{
			Redis: config.RedisConfig{Host: redis.Host(), Port: redis.Port()},
			Queue: asyncTestQueue,
		})
		inspector = asynq.NewInspector(asynq.RedisClientOpt{Addr: redis.Addr()})
		wsStore = testutil.NewStore(GinkgoT(), workspace.Meta)
	})

	AfterAll(func() {
		taskq.CloseClient(context.Background())
		Expect(inspector.Close()).To(Succeed())
		redis.Close()
	})

	// consumeNext 取出队列中最早的一个待执行任务，交给 handler 执行，模拟 worker 消费：
	// 任务经过真实的投递与 Redis 序列化，handler 只拿到 payload，拿不到投递方的 ctx
	consumeNext := func(handler asynq.HandlerFunc) {
		pending, err := inspector.ListPendingTasks(asyncTestQueue)
		Expect(err).NotTo(HaveOccurred())
		Expect(pending).NotTo(BeEmpty())
		next := pending[0]
		Expect(inspector.DeleteTask(asyncTestQueue, next.ID)).To(Succeed())
		Expect(handler(context.Background(), asynq.NewTask(next.Type, next.Payload))).To(Succeed())
	}

	// 测的是：租户 B 的请求投递的任务，在 worker 中执行时读写的仍是租户 B 的数据。
	// worker 的 ctx 来自消息而不是请求，租户必须随消息传递并在消费时恢复；
	// 否则任务会因为没有租户而失败，或者把数据写进别的租户
	It("runs a task in the tenant that enqueued it", func() {
		taskType := taskq.NewTaskType[createWorkspaceArgs]("tenancy.create-workspace",
			func(ctx context.Context, args createWorkspaceArgs) error {
				return wsStore.Create(ctx, &workspace.Workspace{
					ID: args.WorkspaceID, DisplayName: args.WorkspaceID, State: workspace.StateReady,
				})
			})

		requestCtx := auth.WithUser(
			store.ScopeToTenant(context.Background(), "tenant-b"), auth.User{ID: "user-b"})
		wsID := "ws-async-" + stringx.Random(6)
		Expect(taskq.Enqueue(requestCtx, taskType.NewTask(createWorkspaceArgs{WorkspaceID: wsID}))).To(Succeed())

		consumeNext(taskType.Handler())

		got, err := wsStore.Get(store.ScopeToTenant(context.Background(), "tenant-b"), workspace.ByWorkspaceID(wsID))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.TenantID).To(Equal("tenant-b"))
		_, err = wsStore.Get(store.ScopeToTenant(context.Background(), "tenant-a"), workspace.ByWorkspaceID(wsID))
		Expect(err).To(MatchError(workspace.ErrWorkspaceNotFound))
	})

	// 测的是：任务在 handler 中再次投递自己（轮询类任务的常见写法）时，后续每一轮仍在原租户下执行。
	// 再次投递用的是 handler 的 ctx，其中的租户来自消息恢复；如果恢复时丢了租户，
	// 第二轮投递会直接失败，轮询链路在第一轮之后就断掉
	It("keeps the tenant across tasks re-enqueued from a handler", func() {
		var seenTenants []string
		var taskType *taskq.TaskType[pollArgs]
		taskType = taskq.NewTaskType[pollArgs]("tenancy.poll", func(ctx context.Context, args pollArgs) error {
			tenantID, _ := tenant.GetTenantID(ctx)
			seenTenants = append(seenTenants, tenantID)
			if args.Step >= 2 {
				return nil
			}
			return taskq.Enqueue(ctx, taskType.NewTask(pollArgs{Step: args.Step + 1}))
		})

		requestCtx := auth.WithUser(
			store.ScopeToTenant(context.Background(), "tenant-b"), auth.User{ID: "user-b"})
		Expect(taskq.Enqueue(requestCtx, taskType.NewTask(pollArgs{Step: 0}))).To(Succeed())
		for range 3 {
			consumeNext(taskType.Handler())
		}

		Expect(seenTenants).To(Equal([]string{"tenant-b", "tenant-b", "tenant-b"}))
		pending, err := inspector.ListPendingTasks(asyncTestQueue)
		Expect(err).NotTo(HaveOccurred())
		Expect(pending).To(BeEmpty())
	})
})
