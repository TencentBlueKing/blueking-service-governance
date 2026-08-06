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

package bkiam

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WorkspaceData DTO", func() {
	It("preserves workspace data and optional integration fields", func() {
		data := WorkspaceData{
			WorkspaceID:   "ws-001",
			WorkspaceName: "workspace name",
			BKCI:          &BKCIOptions{ProjectID: "ci-project", ProjectName: "CI Project"},
			BCS:           &BCSOptions{ProjectID: "bcs-project", ProjectName: "BCS Project"},
			BKMonitor:     &BKMonitorOptions{SpaceID: "monitor-space", SpaceName: "Monitor Space"},
			BKLog:         &BKLogOptions{SpaceID: "log-space", SpaceName: "Log Space"},
			BKRepo:        &BKRepoOptions{ProjectID: "repo-project", ProjectName: "Repo Project"},
			BSCP: &BSCPOptions{
				BizID:   "biz-id",
				BizName: "Biz Name",
				Services: []BSCPService{
					{ID: "svc-1", Name: "service one"},
					{ID: "svc-2", Name: "service two"},
				},
			},
		}

		Expect(data.WorkspaceID).To(Equal("ws-001"))
		Expect(data.WorkspaceName).To(Equal("workspace name"))
		Expect(data.BKCI.ProjectID).To(Equal("ci-project"))
		Expect(data.BCS.ProjectName).To(Equal("BCS Project"))
		Expect(data.BKMonitor.SpaceID).To(Equal("monitor-space"))
		Expect(data.BKLog.SpaceName).To(Equal("Log Space"))
		Expect(data.BKRepo.ProjectID).To(Equal("repo-project"))
		Expect(data.BSCP.BizID).To(Equal("biz-id"))
		Expect(data.BSCP.Services).To(Equal([]BSCPService{
			{ID: "svc-1", Name: "service one"},
			{ID: "svc-2", Name: "service two"},
		}))
	})

	It("keeps nil optional integrations distinguishable from empty integrations", func() {
		data := WorkspaceData{WorkspaceID: "ws-empty", WorkspaceName: "empty workspace"}

		Expect(data.BKCI).To(BeNil())
		Expect(data.BCS).To(BeNil())
		Expect(data.BKMonitor).To(BeNil())
		Expect(data.BKLog).To(BeNil())
		Expect(data.BKRepo).To(BeNil())
		Expect(data.BSCP).To(BeNil())

		data.BSCP = &BSCPOptions{BizID: "biz-id", BizName: "Biz Name"}
		Expect(data.BSCP).NotTo(BeNil())
		Expect(data.BSCP.Services).To(BeNil())
	})
})
