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

var _ = Describe("Snapshot Validate", func() {
	// validSnapshot 构建一个完整有效的 Snapshot
	validSnapshot := func() *model.Snapshot {
		return &model.Snapshot{
			Metadata: &model.Metadata{
				AppID:        "app-1",
				BscpBizID:    "100",
				MountPath:    "/data/bscp",
				Token:        "test-token",
				FeedAddr:     "feed.example.com:9510",
				WorkloadName: "main",
			},
			EnvBinding: &model.EnvBinding{
				AppID:   "app-1",
				EnvName: "prod",
				Services: []model.ServiceRef{
					{ID: "svc-1", Name: "order-svc"},
				},
			},
		}
	}

	Context("when all fields are valid", func() {
		It("should pass validation", func() {
			snapshot := validSnapshot()
			err := snapshot.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when WorkloadName is empty", func() {
		It("should return validation error", func() {
			snapshot := validSnapshot()
			snapshot.Metadata.WorkloadName = ""

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("WorkloadName"))
		})
	})

	Context("when MountPath is empty", func() {
		It("should return validation error", func() {
			snapshot := validSnapshot()
			snapshot.Metadata.MountPath = ""

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("MountPath"))
		})
	})

	Context("when Token is empty", func() {
		It("should return validation error", func() {
			snapshot := validSnapshot()
			snapshot.Metadata.Token = ""

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Token"))
		})
	})

	Context("when FeedAddr is empty", func() {
		It("should return validation error", func() {
			snapshot := validSnapshot()
			snapshot.Metadata.FeedAddr = ""

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("FeedAddr"))
		})
	})

	Context("when Metadata is nil", func() {
		It("should return validation error", func() {
			snapshot := &model.Snapshot{
				Metadata:   nil,
				EnvBinding: &model.EnvBinding{},
			}

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when EnvBinding.Services is empty", func() {
		It("should return validation error", func() {
			snapshot := validSnapshot()
			snapshot.EnvBinding.Services = []model.ServiceRef{}

			err := snapshot.Validate()
			Expect(err).To(HaveOccurred())
		})
	})
})
