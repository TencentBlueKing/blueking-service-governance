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

package appmodel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

var _ = Describe("ResourceKey", func() {
	Describe("String", func() {
		It("should return formatted string with Kind/Name", func() {
			resource := appmodel.ResourceKey{Kind: "GameDeployment", Name: "my-app"}
			Expect(resource.String()).To(Equal("GameDeployment/my-app"))
		})

		It("should handle empty fields", func() {
			Expect(appmodel.ResourceKey{}.String()).To(Equal("/"))
		})
	})
})

var _ = Describe("ResourceKeys", func() {
	Describe("Diff", func() {
		It("should return elements that exist in rs but not in other", func() {
			rs := appmodel.ResourceKeys{
				{Kind: "GameDeployment", Name: "app1"},
				{Kind: "Service", Name: "svc1"},
				{Kind: "ConfigMap", Name: "config1"},
			}
			other := appmodel.ResourceKeys{
				{Kind: "Service", Name: "svc1"},
			}

			diff := rs.Diff(other)
			Expect(diff).To(HaveLen(2))
			Expect(diff).To(ContainElements(
				appmodel.ResourceKey{Kind: "GameDeployment", Name: "app1"},
				appmodel.ResourceKey{Kind: "ConfigMap", Name: "config1"},
			))
		})

		It("should return empty when all elements exist in other", func() {
			rs := appmodel.ResourceKeys{
				{Kind: "GameDeployment", Name: "app1"},
			}
			other := appmodel.ResourceKeys{
				{Kind: "GameDeployment", Name: "app1"},
				{Kind: "Service", Name: "svc1"},
			}

			Expect(rs.Diff(other)).To(BeEmpty())
		})

		It("should diff by Kind and Name combination", func() {
			rs := appmodel.ResourceKeys{
				{Kind: "GameDeployment", Name: "app1"},
				{Kind: "Service", Name: "app1"},
			}
			other := appmodel.ResourceKeys{
				{Kind: "GameDeployment", Name: "app1"},
			}

			diff := rs.Diff(other)
			Expect(diff).To(HaveLen(1))
			Expect(diff[0]).To(Equal(appmodel.ResourceKey{Kind: "Service", Name: "app1"}))
		})
	})
})

var _ = Describe("Record", func() {
	Describe("MainWorkload", func() {
		It("returns WorkloadKind and the matching resource name", func() {
			record := &appmodel.Record{
				WorkloadKind: "Deployment",
				ResourceKeys: appmodel.ResourceKeys{
					{Kind: "Deployment", Name: "demo"},
					{Kind: "Service", Name: "demo"},
				},
			}
			kind, name := record.MainWorkload()
			Expect(kind).To(Equal("Deployment"))
			Expect(name).To(Equal("demo"))
		})

		It("infers Deployment from ResourceKeys when WorkloadKind is empty", func() {
			record := &appmodel.Record{
				ResourceKeys: appmodel.ResourceKeys{
					{Kind: "Service", Name: "demo"},
					{Kind: "Deployment", Name: "demo"},
				},
			}
			kind, name := record.MainWorkload()
			Expect(kind).To(Equal("Deployment"))
			Expect(name).To(Equal("demo"))
		})

		It("falls back to GameDeployment when WorkloadKind is empty", func() {
			record := &appmodel.Record{
				ResourceKeys: appmodel.ResourceKeys{
					{Kind: "GameDeployment", Name: "demo"},
					{Kind: "Service", Name: "demo"},
				},
			}
			kind, name := record.MainWorkload()
			Expect(kind).To(Equal("GameDeployment"))
			Expect(name).To(Equal("demo"))
		})
	})
})

var _ = Describe("BuildAutoDeployExtras", func() {
	Describe("NewBuildAutoDeployExtras", func() {
		It("should encode build auto deploy info into deploy record extras", func() {
			extras := appmodel.NewBuildAutoDeployExtras(&appmodel.BuildAutoDeployInfo{
				Branch:   "release/v1.2.3",
				CommitID: "abc123def456",
			})

			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyDeploySource, appmodel.DeploySourceBuildAutoDeploy))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildBranch, "release/v1.2.3"))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildCommitID, "abc123def456"))
		})

		It("should return nil when build auto deploy info is absent", func() {
			Expect(appmodel.NewBuildAutoDeployExtras(nil)).To(BeNil())
		})
	})
})
