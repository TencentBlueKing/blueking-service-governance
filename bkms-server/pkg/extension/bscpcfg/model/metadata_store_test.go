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

package model_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("MetadataStore", func() {
	var store *model.MetadataStore
	var ctx context.Context
	var testAppID string

	BeforeEach(func() {
		store = testutil.NewStore(GinkgoT(), model.MetadataMeta)
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_ = store.DeleteIfExists(ctx, model.ByApp(testAppID))
	})

	Describe("Create", func() {
		Context("when creating a valid metadata", func() {
			It("should create successfully", func() {
				meta := &model.Metadata{
					AppID:        testAppID,
					Enable:       true,
					MountPath:    "/data/bscp",
					WorkloadKind: "Deployment",
					WorkloadName: "test-workload",
					Operator:     "tester",
				}

				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				// 验证写入
				stored, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.Enable).To(BeTrue())
				Expect(stored.MountPath).To(Equal("/data/bscp"))
				Expect(stored.WorkloadKind).To(Equal("Deployment"))
				Expect(stored.WorkloadName).To(Equal("test-workload"))
				Expect(stored.Operator).To(Equal("tester"))
			})
		})

		Context("when creating duplicate metadata", func() {
			It("should return ErrMetadataAlreadyExists", func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				meta2 := &model.Metadata{
					AppID:     testAppID,
					MountPath: "/data/bscp2",
				}
				err = store.Create(ctx, meta2)
				Expect(err).To(MatchError(model.ErrMetadataAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				meta := &model.Metadata{
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})
		})
	})

	Describe("Get", func() {
		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				_, err := store.Get(ctx, model.ByApp("non-existent-app"))
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})
})
