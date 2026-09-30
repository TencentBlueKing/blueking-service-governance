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

	Describe("GetReleaseMode", func() {
		It("defaults to releasing all instances for legacy records", func() {
			Expect((&appmodel.Record{}).GetReleaseMode()).To(Equal(appmodel.ReleaseModeAll))
		})

		It("returns the stored release mode", func() {
			record := &appmodel.Record{ReleaseMode: appmodel.ReleaseModeBatch}
			Expect(record.GetReleaseMode()).To(Equal(appmodel.ReleaseModeBatch))
		})
	})

	Describe("IsBatchReleasing", func() {
		DescribeTable("checks release mode and status",
			func(mode appmodel.ReleaseMode, status appmodel.Status, expected bool) {
				record := &appmodel.Record{ReleaseMode: mode, Status: status}
				Expect(record.IsBatchReleasing()).To(Equal(expected))
			},
			Entry("batch release deploying", appmodel.ReleaseModeBatch, appmodel.StatusDeploying, true),
			Entry("legacy record deploying", appmodel.ReleaseMode(""), appmodel.StatusDeploying, false),
			Entry("release all deploying", appmodel.ReleaseModeAll, appmodel.StatusDeploying, false),
			Entry("batch release deployed", appmodel.ReleaseModeBatch, appmodel.StatusDeployed, false),
			Entry("batch release rolled back", appmodel.ReleaseModeBatch, appmodel.StatusRolledBack, false),
			Entry("batch release polling broken", appmodel.ReleaseModeBatch, appmodel.StatusPollingBroken, false),
		)

		It("returns false when there is no record", func() {
			var record *appmodel.Record
			Expect(record.IsBatchReleasing()).To(BeFalse())
		})
	})
})

var _ = Describe("Status", func() {
	DescribeTable("IsStable",
		func(status appmodel.Status, expected bool) {
			Expect(status.IsStable()).To(Equal(expected))
		},
		Entry("deploying", appmodel.StatusDeploying, false),
		Entry("uninstalling", appmodel.StatusUninstalling, false),
		Entry("deployed", appmodel.StatusDeployed, true),
		Entry("rolled back", appmodel.StatusRolledBack, true),
		Entry("failed", appmodel.StatusFailed, true),
		Entry("polling timeout", appmodel.StatusPollingTimeout, true),
	)
})

var _ = Describe("BuildAutoDeployExtras", func() {
	Describe("NewBuildAutoDeployExtras", func() {
		It("should encode build auto deploy info into deploy record extras", func() {
			extras := appmodel.NewBuildAutoDeployExtras(&appmodel.BuildAutoDeployInfo{
				Branch:       "release/v1.2.3",
				RevisionType: "branch",
				Revision:     "release/v1.2.3",
				CommitID:     "abc123def456",
			})

			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyDeploySource, appmodel.DeploySourceBuildAutoDeploy))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildRevisionType, "branch"))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildRevision, "release/v1.2.3"))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildBranch, "release/v1.2.3"))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildCommitID, "abc123def456"))
		})

		It("should not store a commit revision in the legacy branch extra", func() {
			extras := appmodel.NewBuildAutoDeployExtras(&appmodel.BuildAutoDeployInfo{
				RevisionType: "commit",
				Revision:     "a1b2c3d4e5f6",
				CommitID:     "a1b2c3d4e5f6",
			})

			Expect(extras).NotTo(HaveKey(appmodel.ExtraKeyBuildBranch))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildRevisionType, "commit"))
			Expect(extras).To(HaveKeyWithValue(appmodel.ExtraKeyBuildRevision, "a1b2c3d4e5f6"))
		})

		It("should return nil when build auto deploy info is absent", func() {
			Expect(appmodel.NewBuildAutoDeployExtras(nil)).To(BeNil())
		})
	})
})
