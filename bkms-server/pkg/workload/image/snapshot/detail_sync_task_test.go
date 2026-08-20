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

package snapshot

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
)

var _ = Describe("DetailSyncManager Handle", func() {
	var (
		ctx context.Context
		mgr *DetailSyncManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		mgr = NewDetailSyncManager(&SnapshotStoreMongo{})
	})

	AfterEach(func() {
		mockey.UnPatchAll()
	})

	It("stops retry when credential decrypt fails", func() {
		err := mgr.Handle(ctx, ImageDetailSyncArgs{
			RepoKey:           "repo-1",
			RepoName:          "example/app",
			EncryptedUsername: "not-a-valid-cipher",
		})
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeTrue())
	})

	It("returns nil when SyncDetails succeeds", func() {
		mockey.Mock((*DetailSyncer).SyncDetails).Return(nil).Build()
		err := mgr.Handle(ctx, ImageDetailSyncArgs{RepoKey: "repo-1", RepoName: "example/app"})
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns SyncDetails error for asynq retry", func() {
		mockey.Mock((*DetailSyncer).SyncDetails).Return(errors.New("store down")).Build()
		err := mgr.Handle(ctx, ImageDetailSyncArgs{RepoKey: "repo-1", RepoName: "example/app"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("detail sync for repo-1 failed"))
		Expect(err.Error()).To(ContainSubstring("store down"))
		Expect(errors.Is(err, taskq.ErrStopRetry)).To(BeFalse())
	})
})
