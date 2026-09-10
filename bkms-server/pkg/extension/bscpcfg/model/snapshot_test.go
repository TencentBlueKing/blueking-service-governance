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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("Snapshot", func() {
	// newValidSnapshot 构造一个满足 Snapshot.Validate 的完整快照。
	newValidSnapshot := func() *model.Snapshot {
		return &model.Snapshot{
			Metadata: &model.Metadata{
				AppID:        "test-app",
				BscpBizID:    "12345",
				MountPath:    "/data/bscp",
				Token:        "test-token",
				FeedAddr:     "bscp-feed.example.com:9500",
				WorkloadName: "test-workload",
			},
			EnvBinding: &model.EnvBinding{
				AppID:     "test-app",
				EnvName:   "dev",
				BscpAppID: "1001",
			},
		}
	}

	Describe("GetBscpAppName", func() {
		Context("when EnvBinding is nil", func() {
			It("should return empty string", func() {
				snap := &model.Snapshot{}
				Expect(snap.GetBscpAppName()).To(Equal(""))
			})
		})

		Context("when EnvBinding is present", func() {
			It("should return the bkms appID", func() {
				snap := &model.Snapshot{
					EnvBinding: &model.EnvBinding{AppID: "test-app"},
				}
				Expect(snap.GetBscpAppName()).To(Equal("test-app"))
			})
		})
	})

	Describe("Validate", func() {
		Context("when snapshot is valid", func() {
			It("should pass", func() {
				Expect(newValidSnapshot().Validate()).NotTo(HaveOccurred())
			})
		})

		Context("when Metadata is nil", func() {
			It("should return error", func() {
				snap := newValidSnapshot()
				snap.Metadata = nil
				Expect(snap.Validate()).To(HaveOccurred())
			})
		})

		Context("when Metadata.MountPath is empty", func() {
			It("should return error", func() {
				snap := newValidSnapshot()
				snap.Metadata.MountPath = ""
				Expect(snap.Validate()).To(HaveOccurred())
			})
		})

		Context("when Metadata.Token is empty", func() {
			It("should return error", func() {
				snap := newValidSnapshot()
				snap.Metadata.Token = ""
				Expect(snap.Validate()).To(HaveOccurred())
			})
		})

		Context("when Metadata.FeedAddr is empty", func() {
			It("should return error", func() {
				snap := newValidSnapshot()
				snap.Metadata.FeedAddr = ""
				Expect(snap.Validate()).To(HaveOccurred())
			})
		})

		Context("when EnvBinding.BscpAppID is empty", func() {
			It("should return error", func() {
				snap := newValidSnapshot()
				snap.EnvBinding.BscpAppID = ""
				Expect(snap.Validate()).To(HaveOccurred())
			})
		})
	})
})
